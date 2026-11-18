<script lang="ts">
  import { onMount } from 'svelte';
  import {
    latestKinematics,
    kinematicsPresetStore,
    currentGrossWeight,
    currentCgLong,
    currentCgLat,
    isCgCompliant,
    type KinematicsPreset
  } from '../stores/kinematics';
  import {
    unitSystem,
    formatWeight,
    formatFuelVolume,
    formatArm,
    armUnitLabel,
    weightUnitLabel
  } from '../stores/units';
  import { wsRouter, connectionState } from '../websocket';
  import { activeCorridor, latestComplianceReport } from '../stores/verdict';
  import { activeScenarioKey } from '../simulation/fixtures';
  import { isWeatherOrAirspaceLimitingFactor } from '../aviationTerminology';
  import { computeCorridorDistanceNm } from '../stations';
  import {
    H145_SPEC,
    BELL429_SPEC,
    getAirframeSpec,
    evaluateLocalMissionKinematics
  } from '../rfmPhysics';

  // Interactive Atmospheric & Pilot Loadout Controls (SI backing units; display follows $unitSystem)
  let airframe = 'Airbus H145 (BK117 D-2)';
  let fieldElevFt = 5885;
  let oatC = 18.0;
  let qnhInHg = 29.95;

  let fuelKg = 420;
  let crewBaseKg = 192;
  let crewAndLitterKg = 283;
  let specialtyGearKg = 49;
  let sceneHoverMinutes = 5;
  let headwindKts = 8;
  let legDistanceNm = 103.9;
  let isIfrMission = false;

  let appliedDefuelOffloadKg: number | null = null;
  let lastAppliedPreset: KinematicsPreset | null = null;

  $: isH145 = ($latestKinematics.airframe || airframe).includes('H145');
  $: activeSpec = getAirframeSpec($latestKinematics.airframe || airframe);
  $: maxFuelSliderKg = Math.round(activeSpec.usableFuelCapacityKg);
  $: mtowDisplay = activeSpec.mtowKg;
  $: isKinematicsCompliant = $isCgCompliant && $latestKinematics.is_feasible;

  // ViewBox bounds for SVG canvas
  const WIDTH = 340;
  const HEIGHT = 215;
  const PAD_X = 48;
  const PAD_Y = 24;
  const PLOT_W = WIDTH - PAD_X - 15;
  const PLOT_H = HEIGHT - PAD_Y - 26;

  function scaleX(longMm: number, minMm: number, maxMm: number): number {
    return PAD_X + ((longMm - minMm) / (maxMm - minMm)) * PLOT_W;
  }

  function scaleY(gwKg: number, minKg: number, maxKg: number): number {
    return HEIGHT - PAD_Y - ((gwKg - minKg) / (maxKg - minKg)) * PLOT_H;
  }

  $: svgGeo = (() => {
    const k = $latestKinematics;
    const h145 = (k.airframe || airframe).includes('H145');
    const spec = getAirframeSpec(k.airframe || airframe);
    const gwMinVal = h145 ? 1800 : 1700;
    const gwMaxVal = h145 ? 3800 : 3300;
    const longMinVal = h145 ? 3700 : 3050;
    const longMaxVal = h145 ? 4250 : 3550;

    const fwdPath = spec.cgEnvelope.map(
      (p) => `${scaleX(p.fwd, longMinVal, longMaxVal).toFixed(1)},${scaleY(p.gw, gwMinVal, gwMaxVal).toFixed(1)}`
    );
    const aftPath = [...spec.cgEnvelope]
      .reverse()
      .map((p) => `${scaleX(p.aft, longMinVal, longMaxVal).toFixed(1)},${scaleY(p.gw, gwMinVal, gwMaxVal).toFixed(1)}`);

    const toGw = k.mass_balance.gross_weight_kg || (h145 ? 2890 : 2620);
    const toLong = k.mass_balance.cg_long_mm || (h145 ? 3864 : 3260);
    const ldgGw = k.landing_gross_weight_kg ?? toGw;
    const ldgLong = k.landing_cg_long_mm ?? toLong;

    const curXVal = scaleX(toLong, longMinVal, longMaxVal);
    const curYVal = scaleY(toGw, gwMinVal, gwMaxVal);
    const ldgXVal = scaleX(ldgLong, longMinVal, longMaxVal);
    const ldgYVal = scaleY(ldgGw, gwMinVal, gwMaxVal);

    return {
      gwTicks: spec.axisBounds.gwTicks.map((gw) => ({ gw, y: scaleY(gw, gwMinVal, gwMaxVal) })),
      longTicks: spec.axisBounds.longTicks.map((l) => ({ l, x: scaleX(l, longMinVal, longMaxVal) })),
      envelopePointsSvg: [...fwdPath, ...aftPath].join(' '),
      curX: curXVal,
      curY: curYVal,
      ldgX: ldgXVal,
      ldgY: ldgYVal,
      burnVectorLength: Math.hypot(ldgXVal - curXVal, ldgYVal - curYVal),
      toGw,
      toLong
    };
  })();

  $: landingGwKg = $latestKinematics.landing_gross_weight_kg ?? $currentGrossWeight;
  $: landingCgLongMm = $latestKinematics.landing_cg_long_mm ?? $currentCgLong;

  $: fwdLimitDisplay = $latestKinematics.mass_balance.fwd_limit_mm || (isH145 ? 3780 : 3120);
  $: aftLimitDisplay = $latestKinematics.mass_balance.aft_limit_mm || (isH145 ? 4200 : 3480);
  $: cgProgressPct = Math.max(
    0,
    Math.min(100, (($currentCgLong - fwdLimitDisplay) / (aftLimitDisplay - fwdLimitDisplay || 1)) * 100)
  );

  // Reserve & defuel verification status driven by live kinematics
  $: effectiveIfrMode = Boolean(
    isIfrMission || $latestKinematics.is_ifr_reserve
  );
  $: reserveReqMin = effectiveIfrMode ? 30 : ($latestKinematics.reserve_required_minutes ?? 20);
  $: reqReserveKg = (reserveReqMin / 60) * ($latestKinematics.cruise_fuel_flow_kg_hr || 210);
  $: liveReserveMarginKg = $latestKinematics.remaining_fuel_kg - reqReserveKg;
  $: liveReserveMarginMin =
    $latestKinematics.cruise_fuel_flow_kg_hr > 0
      ? (liveReserveMarginKg / $latestKinematics.cruise_fuel_flow_kg_hr) * 60
      : 0;
  $: offloadKg = $latestKinematics.required_fuel_offload_kg ?? 0;

  function applyPresetValues(preset: KinematicsPreset) {
    lastAppliedPreset = preset;
    airframe = preset.airframe;
    fieldElevFt = preset.fieldElevFt;
    oatC = preset.oatC;
    qnhInHg = preset.qnhInHg;
    fuelKg = preset.fuelKg;
    crewBaseKg = preset.crewBaseKg ?? preset.crewKg ?? 192;
    crewAndLitterKg =
      preset.crewAndLitterKg ?? preset.crewKg + Math.round((preset.payloadKg ?? 0) * 0.65);
    specialtyGearKg = preset.specialtyGearKg ?? Math.round((preset.payloadKg ?? 0) * 0.35);
    sceneHoverMinutes = preset.sceneHoverMinutes ?? 5;
    headwindKts = preset.headwindKts;
    legDistanceNm = preset.legDistanceNm ?? computeCorridorDistanceNm($activeCorridor);
    isIfrMission = Boolean(preset.isIfr);
    appliedDefuelOffloadKg = null;
  }

  function syncVerdictWithInteractiveKinematics(evaluated: ReturnType<typeof evaluateLocalMissionKinematics>) {
    if ($connectionState === 'CONNECTED' && $activeScenarioKey === null) return;
    const prev = $latestComplianceReport;
    if (!prev || isWeatherOrAirspaceLimitingFactor(prev.limiting_factor)) {
      return;
    }

    const now = new Date().toISOString();
    if (evaluated.is_feasible) {
      const resLabel = `${evaluated.reserve_required_minutes ?? 20}-MIN ${evaluated.is_ifr_reserve ? 'IFR' : 'VFR'}`;
      const resMarginMin = Math.round(evaluated.reserve_margin_minutes ?? 0);
      const mitigatedNote =
        appliedDefuelOffloadKg !== null
          ? `Post-defuel (-${formatWeight(appliedDefuelOffloadKg, $unitSystem)}) OGE margin +${Math.round(evaluated.oge_margin_kg)} kg; ${resLabel} reserve verified (${resMarginMin >= 0 ? '+' : ''}${resMarginMin} min).`
          : `All corridor weather minimums, RFM OGE hover (+${Math.round(evaluated.oge_margin_kg)} kg), and ${resLabel} reserve (${resMarginMin >= 0 ? '+' : ''}${resMarginMin} min) verified.`;

      latestComplianceReport.set({
        ...prev,
        is_compliant: true,
        verdict: prev.verdict === 'IFR_REQUIRED' ? 'IFR_REQUIRED' : 'APPROVED',
        limiting_factor: 'LIMITING_FACTOR_NONE',
        legal_rule_applied:
          prev.verdict === 'IFR_REQUIRED'
            ? prev.legal_rule_applied
            : '14 CFR § 135.609 + RFM OGE + § 135.209 Reserve Verified',
        explanation: mitigatedNote,
        evaluation_time_utc: now,
        audit_trace: [
          ...(prev.audit_trace || []).slice(0, 6),
          `[LOADOUT] Fuel=${Math.round(fuelKg)}kg Crew+Litter=${crewAndLitterKg}kg Gear=${specialtyGearKg}kg Hover=${sceneHoverMinutes}m -> OGE=${evaluated.oge_margin_kg >= 0 ? '+' : ''}${Math.round(evaluated.oge_margin_kg)}kg, Reserve=${resMarginMin >= 0 ? '+' : ''}${resMarginMin}min -> PASS`
        ]
      });
    } else {
      latestComplianceReport.set({
        ...prev,
        is_compliant: false,
        verdict: 'NO-GO',
        limiting_factor: evaluated.limiting_factor as typeof prev.limiting_factor,
        legal_rule_applied:
          evaluated.limiting_factor === 'FAR_135_209_VFR_RESERVE_VIOLATION' ||
          evaluated.limiting_factor === 'IFR_RESERVE_VIOLATION'
            ? `14 CFR § 135.209 (${evaluated.reserve_required_minutes ?? 20}-min Fuel Reserve)`
            : 'RFM Weight, CG & OGE Hover Envelope',
        explanation:
          evaluated.actionable_advisory ||
          `NO-GO (${evaluated.limiting_factor}): Mission loadout exceeds aircraft performance or reserve limits.`,
        evaluation_time_utc: now,
        audit_trace: [
          ...(prev.audit_trace || []).slice(0, 6),
          `[LOADOUT] FAIL (${evaluated.limiting_factor}): ${evaluated.actionable_advisory || 'Limit exceeded'}`
        ]
      });
    }
  }

  function recalculateInteractiveKinematics(userDriven = false) {
    const litterKg = Math.max(0, crewAndLitterKg - crewBaseKg);
    const derivedPayloadKg = litterKg + specialtyGearKg;

    const evaluated = evaluateLocalMissionKinematics({
      airframe,
      fieldElevFt,
      oatC,
      qnhInHg,
      fuelKg,
      payloadKg: derivedPayloadKg,
      crewKg: crewBaseKg,
      crewBaseKg,
      crewAndLitterKg,
      specialtyGearKg,
      sceneHoverMinutes,
      headwindKts,
      legDistanceNm,
      isIfr: isIfrMission
    });
    latestKinematics.set(evaluated);

    if (userDriven) {
      syncVerdictWithInteractiveKinematics(evaluated);
    }

    // When connected to live Go/Rust C-FFI gateway outside deterministic fixtures, dispatch EVALUATE_KINEMATICS
    if ($connectionState === 'CONNECTED' && userDriven && $activeScenarioKey === null) {
      const h145Selected = airframe.includes('H145');
      const airframeId = h145Selected ? 0 : 1;
      const effectiveCrewKg = Math.min(crewBaseKg, crewAndLitterKg);
      const pilotKg = effectiveCrewKg * 0.48;
      const attendantKg = effectiveCrewKg * 0.52;
      const payloads = h145Selected
        ? [
            { station_id: 0, weight_kg: pilotKg },
            { station_id: 2, weight_kg: attendantKg },
            { station_id: 6, weight_kg: litterKg },
            { station_id: 8, weight_kg: specialtyGearKg }
          ]
        : [
            { station_id: 0, weight_kg: pilotKg },
            { station_id: 4, weight_kg: attendantKg },
            { station_id: 6, weight_kg: litterKg },
            { station_id: 9, weight_kg: specialtyGearKg }
          ];

      wsRouter.requestKinematicsEvaluation(
        {
          airframe_id: airframeId,
          station_elevation_ft: fieldElevFt,
          true_course_deg: 270.0,
          leg_distance_nm: legDistanceNm,
          fuel_weight_kg: fuelKg,
          payloads
        },
        $activeCorridor.destination || $activeCorridor.departure || 'KAPA'
      );
    }
  }

  function handleAirframeChange() {
    const targetMax = Math.round((airframe.includes('H145') ? H145_SPEC : BELL429_SPEC).usableFuelCapacityKg);
    if (fuelKg > targetMax) {
      fuelKg = targetMax;
    }
    recalculateInteractiveKinematics(true);
  }

  function handleFuelSliderInput() {
    appliedDefuelOffloadKg = null;
    recalculateInteractiveKinematics(true);
  }

  function applyDefuelMitigation() {
    if (offloadKg <= 0) return;
    appliedDefuelOffloadKg = offloadKg;
    fuelKg = Math.max(0, Math.round(fuelKg - offloadKg));
    recalculateInteractiveKinematics(true);
  }

  $: if ($kinematicsPresetStore && $kinematicsPresetStore !== lastAppliedPreset) {
    applyPresetValues($kinematicsPresetStore);
    recalculateInteractiveKinematics(false);
  }

  onMount(() => {
    if ($kinematicsPresetStore) {
      applyPresetValues($kinematicsPresetStore);
    }
    recalculateInteractiveKinematics(false);
  });
</script>

<div class="cg-card" aria-label="Center of Gravity Envelope Visualizer">
  <div class="panel-header">
    <div class="flex items-center gap-2">
      <span>02 // ATMOSPHERE, LOADOUT &amp; CG ENVELOPE</span>
      <span class="airframe-badge">{isH145 ? 'AIRBUS H145' : 'BELL 429'}</span>
    </div>
    <span class="cg-status" class:compliant={isKinematicsCompliant} class:violation={!isKinematicsCompliant}>
      {isKinematicsCompliant
        ? 'COMPLIANT (IN LIMITS)'
        : $latestKinematics.limiting_factor && $latestKinematics.limiting_factor !== 'NONE'
          ? `${$latestKinematics.limiting_factor} (NO-GO)`
          : 'EXCEEDED (NO-GO)'}
    </span>
  </div>

  <div class="cg-body">
    <!-- Airframe & Atmospheric Readouts (PA / DA) -->
    <div class="atm-bar">
      <div class="atm-cell">
        <span class="tel-label">PRESSURE ALT</span>
        <span class="tel-val text-cyan-400">{Math.round($latestKinematics.pressure_altitude_ft).toLocaleString()} FT</span>
      </div>
      <div class="atm-cell">
        <span class="tel-label">DENSITY ALT</span>
        <span class="tel-val text-amber-400">{Math.round($latestKinematics.density_altitude_ft).toLocaleString()} FT</span>
      </div>
      <div class="atm-cell">
        <span class="tel-label">AIRFRAME SELECT</span>
        <select bind:value={airframe} on:change={handleAirframeChange} class="airframe-sel" aria-label="Airframe Selector">
          <option value="Airbus H145 (BK117 D-2)">H145 (D-2)</option>
          <option value="Bell 429 GlobalRanger">BELL 429</option>
        </select>
      </div>
    </div>

    <!-- Interactive Atmospheric Sliders -->
    <div class="sliders-box">
      <div class="slider-row">
        <div class="slider-lbl">
          <span>Field Elev: <strong class="text-cyan-400">{fieldElevFt.toLocaleString()} ft</strong></span>
          <span>OAT: <strong class="text-amber-400">{oatC.toFixed(0)}°C</strong></span>
          <span>Altimeter: <strong class="text-cyan-400">{qnhInHg.toFixed(2)}"</strong></span>
        </div>
        <div class="slider-grid-3">
          <input type="range" min="0" max="14000" step="50" bind:value={fieldElevFt} on:input={() => recalculateInteractiveKinematics(true)} title="Field Elevation (ft)" />
          <input type="range" min="-25" max="45" step="1" bind:value={oatC} on:input={() => recalculateInteractiveKinematics(true)} title="Outside Air Temp (°C)" />
          <input type="range" min="28.50" max="31.00" step="0.01" bind:value={qnhInHg} on:input={() => recalculateInteractiveKinematics(true)} title="Altimeter QNH (inHg)" />
        </div>
      </div>
    </div>

    <!-- 2D OEM CG Envelope SVG Canvas with Takeoff -> Landing Burn Vector -->
    <div class="svg-container-box">
      <svg viewBox="0 0 {WIDTH} {HEIGHT}" class="cg-svg">
        <defs>
          <marker id="cg-burn-arrow" markerWidth="6" markerHeight="6" refX="4.5" refY="3" orient="auto">
            <path d="M0,0 L6,3 L0,6 Z" fill="#00e5ff" />
          </marker>
        </defs>

        <!-- Grid Lines -->
        {#each svgGeo.gwTicks as tick}
          <line x1={PAD_X} y1={tick.y} x2={WIDTH - 15} y2={tick.y} stroke="var(--border-hairline)" stroke-width="1" stroke-dasharray="2,2" />
          <text x={PAD_X - 6} y={tick.y + 3} class="axis-tick" text-anchor="end">
            {formatWeight(tick.gw, $unitSystem, { includeUnit: false })}
          </text>
        {/each}

        {#each svgGeo.longTicks as tick}
          <line x1={tick.x} y1={HEIGHT - PAD_Y} x2={tick.x} y2={PAD_Y} stroke="var(--border-hairline)" stroke-width="1" stroke-dasharray="2,2" />
          <text x={tick.x} y={HEIGHT - 10} class="axis-tick" text-anchor="middle">
            {formatArm(tick.l, $unitSystem, { decimals: 0, includeUnit: false })}
          </text>
        {/each}

        <!-- Axis Labels -->
        <text x={WIDTH / 2} y={HEIGHT - 1} class="axis-title" text-anchor="middle">
          LONGITUDINAL CG ({armUnitLabel($unitSystem)})
        </text>
        <text x={12} y={HEIGHT / 2} class="axis-title" text-anchor="middle" transform="rotate(-90 12 {HEIGHT / 2})">
          GW ({weightUnitLabel($unitSystem)})
        </text>

        <!-- Certified Boundary Polygon -->
        <polygon
          points={svgGeo.envelopePointsSvg}
          fill={$isCgCompliant ? 'rgba(0, 230, 118, 0.09)' : 'rgba(255, 23, 68, 0.08)'}
          stroke={$isCgCompliant ? 'var(--color-go)' : 'var(--color-nogo)'}
          stroke-width="1.8"
        />

        <!-- Takeoff -> Landing CG Burn Vector -->
        {#if svgGeo.burnVectorLength > 2}
          <line
            x1={svgGeo.curX}
            y1={svgGeo.curY}
            x2={svgGeo.ldgX}
            y2={svgGeo.ldgY}
            stroke="#00e5ff"
            stroke-width="1.8"
            stroke-dasharray="3,2"
            marker-end="url(#cg-burn-arrow)"
            data-testid="cg-burn-vector"
          />
          <circle cx={svgGeo.ldgX} cy={svgGeo.ldgY} r="3.5" fill="#00e5ff" stroke="#05080c" stroke-width="1" />
          <text
            x={Math.min(WIDTH - 42, Math.max(PAD_X + 4, svgGeo.ldgX + 6))}
            y={Math.min(HEIGHT - PAD_Y - 3, Math.max(svgGeo.curY + 13, svgGeo.ldgY + 10))}
            class="axis-tick font-bold"
            fill="#00e5ff"
          >
            LDG
          </text>
        {/if}

        <!-- Takeoff Operating Point Crosshair & Reticle -->
        <line x1={svgGeo.curX - 10} y1={svgGeo.curY} x2={svgGeo.curX + 10} y2={svgGeo.curY} stroke={$isCgCompliant ? 'var(--color-go)' : 'var(--color-nogo)'} stroke-width="2" />
        <line x1={svgGeo.curX} y1={svgGeo.curY - 10} x2={svgGeo.curX} y2={svgGeo.curY + 10} stroke={$isCgCompliant ? 'var(--color-go)' : 'var(--color-nogo)'} stroke-width="2" />
        <circle cx={svgGeo.curX} cy={svgGeo.curY} r="5" fill="none" stroke={$isCgCompliant ? 'var(--color-go)' : 'var(--color-nogo)'} stroke-width="2" />
        <circle cx={svgGeo.curX} cy={svgGeo.curY} r="10" fill="none" stroke={$isCgCompliant ? 'rgba(0, 230, 118, 0.35)' : 'rgba(255, 23, 68, 0.4)'} stroke-width="1" stroke-dasharray="2,2" />

        <!-- Live Coordinate Tag in SVG -->
        <text
          x={Math.min(WIDTH - 108, Math.max(PAD_X + 4, svgGeo.curX + 8))}
          y={Math.max(PAD_Y + 12, svgGeo.curY - 7)}
          class="axis-tick font-bold"
          fill={$isCgCompliant ? '#00e676' : '#ff1744'}
        >
          TO {formatWeight(svgGeo.toGw, $unitSystem)} / {formatArm(svgGeo.toLong, $unitSystem, { decimals: $unitSystem === 'LBS_IN' ? 1 : 0 })}
        </text>
      </svg>
    </div>

    <!-- Live Graphical Margin Meters -->
    <div class="meters-row">
      <div class="meter-box">
        <div class="meter-header">
          <span>MTOW CAPACITY</span>
          <span class="meter-stat" class:text-rose-400={$currentGrossWeight > mtowDisplay} class:text-emerald-400={$currentGrossWeight <= mtowDisplay}>
            {Math.round(($currentGrossWeight / mtowDisplay) * 100)}% ({formatWeight(mtowDisplay - $currentGrossWeight, $unitSystem)} margin)
          </span>
        </div>
        <div class="meter-track">
          <div
            class="meter-fill"
            class:fill-go={$currentGrossWeight <= mtowDisplay * 0.92}
            class:fill-warn={$currentGrossWeight > mtowDisplay * 0.92 && $currentGrossWeight <= mtowDisplay}
            class:fill-danger={$currentGrossWeight > mtowDisplay}
            style="width: {Math.min(100, ($currentGrossWeight / mtowDisplay) * 100)}%"
          ></div>
        </div>
      </div>

      <div class="meter-box">
        <div class="meter-header">
          <span>LONG CG POSITION</span>
          <span class="meter-stat">
            +{formatArm($latestKinematics.mass_balance.fwd_margin_mm, $unitSystem, { decimals: $unitSystem === 'LBS_IN' ? 1 : 0, includeUnit: false })} / +{formatArm($latestKinematics.mass_balance.aft_margin_mm, $unitSystem, { decimals: $unitSystem === 'LBS_IN' ? 1 : 0 })}
          </span>
        </div>
        <div class="meter-track">
          <div class="cg-marker-indicator" style="left: {cgProgressPct}%"></div>
        </div>
      </div>
    </div>

    <!-- Interactive 4-Control Pilot Loadout Deck -->
    <div class="sliders-box" data-testid="loadout-controls">
      <div class="slider-grid-2">
        <label class="loadout-control">
          <div class="loadout-head">
            <span>USABLE FUEL</span>
            <strong class="text-cyan-400">{formatWeight(fuelKg, $unitSystem)} ({formatFuelVolume(fuelKg, $unitSystem)})</strong>
          </div>
          <input
            type="range"
            min="50"
            max={maxFuelSliderKg}
            step="5"
            bind:value={fuelKg}
            on:input={handleFuelSliderInput}
            title="Jet-A Usable Fuel (kg)"
            data-testid="slider-usable-fuel"
          />
        </label>

        <label class="loadout-control">
          <div class="loadout-head">
            <span>CREW + LITTER</span>
            <strong class="text-cyan-400">{formatWeight(crewAndLitterKg, $unitSystem)}</strong>
          </div>
          <input
            type="range"
            min="160"
            max="440"
            step="5"
            bind:value={crewAndLitterKg}
            on:input={() => recalculateInteractiveKinematics(true)}
            title="Flight Crew and Patient Litter Weight (kg)"
            data-testid="slider-crew-litter"
          />
        </label>

        <label class="loadout-control">
          <div class="loadout-head">
            <span>GEAR (ECMO/HOIST)</span>
            <strong class="text-amber-400">{formatWeight(specialtyGearKg, $unitSystem)}</strong>
          </div>
          <input
            type="range"
            min="0"
            max="160"
            step="5"
            bind:value={specialtyGearKg}
            on:input={() => recalculateInteractiveKinematics(true)}
            title="Specialty Medical Gear ECMO / Hoist (kg)"
            data-testid="slider-specialty-gear"
          />
        </label>

        <label class="loadout-control">
          <div class="loadout-head">
            <span>SCENE OGE HOVER</span>
            <strong class="text-cyan-400">{sceneHoverMinutes} MIN</strong>
          </div>
          <input
            type="range"
            min="0"
            max="30"
            step="1"
            bind:value={sceneHoverMinutes}
            on:input={() => recalculateInteractiveKinematics(true)}
            title="Scene Out-of-Ground-Effect Hover Time (min)"
            data-testid="slider-scene-hover"
          />
        </label>
      </div>
    </div>

    <!-- Telemetry Cards (Unit-Converted) -->
    <div class="cg-telemetry">
      <div class="tel-item">
        <span class="tel-label">TAKEOFF GROSS WT</span>
        <span class="tel-val">{formatWeight($currentGrossWeight, $unitSystem)}</span>
        <span class="tel-sub">MTOW {formatWeight(mtowDisplay, $unitSystem)}</span>
      </div>
      <div class="tel-item">
        <span class="tel-label">LANDING GROSS WT</span>
        <span class="tel-val text-cyan-400">{formatWeight(landingGwKg, $unitSystem)}</span>
        <span class="tel-sub">LDG CG {formatArm(landingCgLongMm, $unitSystem)}</span>
      </div>
      <div class="tel-item">
        <span class="tel-label">LONGITUDINAL CG</span>
        <span class="tel-val">{formatArm($currentCgLong, $unitSystem)}</span>
        <span class="tel-sub">FWD {formatArm($latestKinematics.mass_balance.fwd_limit_mm, $unitSystem, { decimals: $unitSystem === 'LBS_IN' ? 1 : 0, includeUnit: false })} / AFT {formatArm($latestKinematics.mass_balance.aft_limit_mm, $unitSystem, { decimals: $unitSystem === 'LBS_IN' ? 1 : 0, includeUnit: false })}</span>
      </div>
      <div class="tel-item">
        <span class="tel-label">LATERAL CG</span>
        <span class="tel-val">{formatArm($currentCgLat, $unitSystem)}</span>
        <span class="tel-sub">L {formatArm($latestKinematics.mass_balance.left_limit_mm, $unitSystem, { decimals: $unitSystem === 'LBS_IN' ? 1 : 0, includeUnit: false })} / R {formatArm($latestKinematics.mass_balance.right_limit_mm, $unitSystem, { decimals: $unitSystem === 'LBS_IN' ? 1 : 0, includeUnit: false })}</span>
      </div>
      <div class="tel-item" data-testid="oge-margin-card">
        <span class="tel-label">OGE HOVER MARGIN</span>
        <span
          class="tel-val"
          class:text-rose-400={$latestKinematics.oge_margin_kg < 0}
          class:text-emerald-400={$latestKinematics.oge_margin_kg >= 0}
        >
          {formatWeight($latestKinematics.oge_margin_kg, $unitSystem, { signed: true })}
        </span>
        <span class="tel-sub">MAX OGE {formatWeight($latestKinematics.max_oge_hover_weight_kg, $unitSystem)}</span>
      </div>
      <div class="tel-item" data-testid="ige-margin-card">
        <span class="tel-label">IGE HOVER MARGIN</span>
        <span
          class="tel-val"
          class:text-rose-400={$latestKinematics.ige_margin_kg < 0}
          class:text-emerald-400={$latestKinematics.ige_margin_kg >= 0}
        >
          {formatWeight($latestKinematics.ige_margin_kg, $unitSystem, { signed: true })}
        </span>
        <span class="tel-sub">MAX IGE {formatWeight($latestKinematics.max_ige_hover_weight_kg, $unitSystem)}</span>
      </div>
    </div>

    {#if $latestKinematics.actionable_advisory}
      <div class="mitigation-banner" data-testid="kinematics-mitigation">
        <div class="mitigation-info">
          <span class="mitigation-label">MITIGATION //</span>
          <span class="mitigation-text">{$latestKinematics.actionable_advisory}</span>
          {#if offloadKg > 0}
            <span class="mitigation-metric">
              (OFFLOAD {formatWeight(offloadKg, $unitSystem)})
            </span>
          {/if}
        </div>
        {#if offloadKg > 0}
          <button
            type="button"
            class="efb-touch simulate-defuel-btn"
            data-testid="simulate-defuel-btn"
            on:click={applyDefuelMitigation}
          >
            SIMULATE DEFUEL (-{formatWeight(offloadKg, $unitSystem)})
          </button>
        {/if}
      </div>
    {/if}

    {#if appliedDefuelOffloadKg !== null}
      <div
        class="defuel-verify-banner"
        class:verify-pass={liveReserveMarginKg >= 0 && $latestKinematics.oge_margin_kg >= 0}
        class:verify-fail={liveReserveMarginKg < 0 || $latestKinematics.oge_margin_kg < 0}
        data-testid="defuel-verification"
      >
        <strong>DEFUEL (-{formatWeight(appliedDefuelOffloadKg, $unitSystem)}) APPLIED //</strong>
        {#if $latestKinematics.oge_margin_kg < 0}
          <span>OGE STILL DEFICIT ({formatWeight($latestKinematics.oge_margin_kg, $unitSystem, { signed: true })}) — OFFLOAD PAYLOAD/GEAR</span>
        {:else if liveReserveMarginKg >= 0}
          <span>
            § 135.209 VERIFIED: {liveReserveMarginMin >= 0 ? '+' : ''}{Math.round(liveReserveMarginMin)} MIN ({formatWeight(liveReserveMarginKg, $unitSystem, { signed: true })}) ABOVE {reserveReqMin}-MIN {effectiveIfrMode ? 'IFR' : 'VFR'} RESERVE
          </span>
        {:else}
          <span>
            § 135.209 RESERVE VIOLATION: {Math.round(liveReserveMarginMin)} MIN ({formatWeight(liveReserveMarginKg, $unitSystem)}) BELOW {reserveReqMin}-MIN {effectiveIfrMode ? 'IFR' : 'VFR'} MIN — REDUCE HOVER OR PAYLOAD
          </span>
        {/if}
      </div>
    {/if}
  </div>
</div>

<style>
  .cg-card {
    background: var(--bg-panel);
    border: 1px solid var(--border-hairline);
    display: flex;
    flex-direction: column;
  }

  .cg-status {
    font-size: 10px;
    font-weight: 700;
    padding: 1px 6px;
    border: 1px solid currentColor;
  }

  .compliant {
    color: var(--color-go);
    background: var(--color-go-bg);
  }

  .violation {
    color: var(--color-nogo);
    background: var(--color-nogo-bg);
  }

  .cg-body {
    padding: 8px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .cg-svg {
    width: 100%;
    height: auto;
    background: var(--bg-void);
    border: 1px solid var(--border-hairline);
  }

  .axis-tick {
    font-family: var(--font-mono);
    font-size: 8px;
    fill: var(--text-muted);
  }

  .axis-title {
    font-family: var(--font-mono);
    font-size: 8px;
    font-weight: 700;
    fill: var(--text-secondary);
    letter-spacing: 0.08em;
  }

  .cg-telemetry {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 6px;
  }

  .tel-item {
    background: var(--bg-void);
    padding: 4px 6px;
    border: 1px solid var(--border-hairline);
    display: flex;
    flex-direction: column;
  }

  .tel-label {
    font-size: 8px;
    color: var(--text-muted);
    letter-spacing: 0.06em;
  }

  .tel-val {
    font-size: 11px;
    font-weight: 700;
    color: var(--text-primary);
  }

  .tel-sub {
    font-size: 8px;
    color: var(--text-secondary);
  }

  .atm-bar {
    display: grid;
    grid-template-columns: 1fr 1fr 1fr;
    gap: 4px;
  }

  .atm-cell {
    background: var(--bg-void);
    padding: 3px 6px;
    border: 1px solid var(--border-hairline);
    display: flex;
    flex-direction: column;
    justify-content: center;
  }

  .airframe-sel {
    font-size: 9px;
    padding: 3px 4px;
    width: 100%;
    font-weight: 700;
  }

  .sliders-box {
    background: var(--bg-void);
    border: 1px solid var(--border-hairline);
    padding: 6px;
  }

  .slider-row {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .slider-lbl {
    display: grid;
    grid-template-columns: 1fr 1fr 1fr;
    font-size: 9px;
    color: var(--text-secondary);
  }

  .slider-lbl strong {
    color: var(--text-primary);
  }

  .slider-grid-3 {
    display: grid;
    grid-template-columns: 1fr 1fr 1fr;
    gap: 6px;
  }

  .slider-grid-2 {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
  }

  .loadout-control {
    display: flex;
    flex-direction: column;
    gap: 2px;
    font-size: 9px;
    color: var(--text-secondary);
  }

  .loadout-head {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 4px;
  }

  .loadout-head strong {
    font-size: 9px;
    font-weight: 800;
    white-space: nowrap;
  }

  .airframe-badge {
    font-size: 9px;
    font-weight: 800;
    font-family: var(--font-mono);
    padding: 1px 5px;
    background: var(--color-info-bg);
    color: var(--color-info);
    border: 1px solid rgba(0, 229, 255, 0.35);
  }

  .svg-container-box {
    position: relative;
    width: 100%;
  }

  .meters-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 6px;
  }

  .meter-box {
    background: var(--bg-void);
    border: 1px solid var(--border-hairline);
    padding: 4px 6px;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  .meter-header {
    display: flex;
    justify-content: space-between;
    font-size: 8px;
    font-weight: 700;
    color: var(--text-muted);
    letter-spacing: 0.06em;
  }

  .meter-stat {
    font-family: var(--font-mono);
    color: var(--text-secondary);
  }

  .meter-track {
    position: relative;
    width: 100%;
    height: 6px;
    background: #06070a;
    border: 1px solid var(--border-hairline);
    overflow: hidden;
  }

  .meter-fill {
    height: 100%;
    transition: width 0.15s ease-out, background-color 0.15s ease-out;
  }

  .fill-go {
    background: var(--color-go);
  }

  .fill-warn {
    background: var(--color-caution);
  }

  .fill-danger {
    background: var(--color-nogo);
  }

  .cg-marker-indicator {
    position: absolute;
    top: 0;
    bottom: 0;
    width: 4px;
    margin-left: -2px;
    background: var(--color-go);
    box-shadow: 0 0 6px var(--color-go);
    transition: left 0.15s ease-out;
  }

  .mitigation-banner {
    background: var(--color-nogo-bg);
    border: 1px solid var(--color-nogo);
    padding: 6px 8px;
    font-size: 9px;
    font-family: var(--font-mono);
    color: var(--text-primary);
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .mitigation-info {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    align-items: center;
  }

  .mitigation-label {
    color: var(--color-nogo);
    font-weight: 800;
  }

  .mitigation-text {
    font-weight: 700;
  }

  .mitigation-metric {
    color: var(--color-caution);
    font-weight: 700;
  }

  .simulate-defuel-btn {
    width: 100%;
    border-color: var(--color-caution);
    background: var(--color-caution-bg);
    color: var(--color-caution);
    font-size: 10px;
    font-weight: 800;
    letter-spacing: 0.08em;
  }

  .simulate-defuel-btn:hover {
    background: var(--color-caution);
    color: #000;
  }

  .defuel-verify-banner {
    padding: 6px 8px;
    font-size: 9px;
    font-family: var(--font-mono);
    border: 1px solid currentColor;
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }

  .verify-pass {
    color: var(--color-go);
    background: var(--color-go-bg);
    border-color: var(--color-go);
  }

  .verify-fail {
    color: var(--color-nogo);
    background: var(--color-nogo-bg);
    border-color: var(--color-nogo);
  }
</style>
