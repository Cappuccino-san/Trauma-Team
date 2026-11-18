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
  import { wsRouter, connectionState } from '../websocket';
  import { activeCorridor } from '../stores/verdict';
  import {
    H145_SPEC,
    BELL429_SPEC,
    evaluateLocalMissionKinematics
  } from '../rfmPhysics';

  // Interactive Kinematics & Mass/Balance Sliders
  let airframe = 'Airbus H145 (BK117 D-2)';
  let fieldElevFt = 5885;
  let oatC = 18.0;
  let qnhInHg = 29.95;

  let fuelKg = 420;
  let payloadKg = 140;
  let crewKg = 192;
  let headwindKts = 8;
  let legDistanceNm = 45;
  let lastAppliedPreset: KinematicsPreset | null = null;

  $: isH145 = airframe.includes('H145');
  $: activeSpec = isH145 ? H145_SPEC : BELL429_SPEC;
  $: gwMin = isH145 ? 1800 : 1700;
  $: gwMax = isH145 ? 3800 : 3300;
  $: longMin = isH145 ? 3700 : 3050;
  $: longMax = isH145 ? 4250 : 3550;
  $: mtowDisplay = activeSpec.mtowKg;
  $: gwTicks = activeSpec.axisBounds.gwTicks;
  $: longTicks = activeSpec.axisBounds.longTicks;
  $: isKinematicsCompliant = $isCgCompliant && $latestKinematics.is_feasible;

  // ViewBox bounds for SVG canvas
  const WIDTH = 340;
  const HEIGHT = 210;
  const PAD_X = 45;
  const PAD_Y = 25;
  const PLOT_W = WIDTH - PAD_X - 15;
  const PLOT_H = HEIGHT - PAD_Y - 25;

  function scaleX(longMm: number, minMm: number, maxMm: number): number {
    return PAD_X + ((longMm - minMm) / (maxMm - minMm)) * PLOT_W;
  }

  function scaleY(gwKg: number, minKg: number, maxKg: number): number {
    return HEIGHT - PAD_Y - ((gwKg - minKg) / (maxKg - minKg)) * PLOT_H;
  }

  $: activeEnvelope = activeSpec.cgEnvelope;

  $: envelopePointsSvg = (() => {
    const fwdPath = activeEnvelope.map(
      (p) => `${scaleX(p.fwd, longMin, longMax).toFixed(1)},${scaleY(p.gw, gwMin, gwMax).toFixed(1)}`
    );
    const aftPath = [...activeEnvelope]
      .reverse()
      .map((p) => `${scaleX(p.aft, longMin, longMax).toFixed(1)},${scaleY(p.gw, gwMin, gwMax).toFixed(1)}`);
    return [...fwdPath, ...aftPath].join(' ');
  })();

  $: curX = scaleX($currentCgLong || (isH145 ? 3864 : 3260), longMin, longMax);
  $: curY = scaleY($currentGrossWeight || (isH145 ? 2890 : 2620), gwMin, gwMax);
  $: fwdLimitDisplay = $latestKinematics.mass_balance.fwd_limit_mm || (isH145 ? 3780 : 3120);
  $: aftLimitDisplay = $latestKinematics.mass_balance.aft_limit_mm || (isH145 ? 4200 : 3480);
  $: cgProgressPct = Math.max(0, Math.min(100, (($currentCgLong - fwdLimitDisplay) / (aftLimitDisplay - fwdLimitDisplay || 1)) * 100));

  function applyPresetValues(preset: KinematicsPreset) {
    lastAppliedPreset = preset;
    airframe = preset.airframe;
    fieldElevFt = preset.fieldElevFt;
    oatC = preset.oatC;
    qnhInHg = preset.qnhInHg;
    fuelKg = preset.fuelKg;
    payloadKg = preset.payloadKg;
    crewKg = preset.crewKg;
    headwindKts = preset.headwindKts;
    legDistanceNm = preset.legDistanceNm ?? 45;
  }

  function recalculateInteractiveKinematics() {
    const evaluated = evaluateLocalMissionKinematics({
      airframe,
      fieldElevFt,
      oatC,
      qnhInHg,
      fuelKg,
      payloadKg,
      crewKg,
      headwindKts,
      legDistanceNm
    });
    latestKinematics.set(evaluated);

    // When connected to live Go/Rust C-FFI gateway, also dispatch EVALUATE_KINEMATICS
    if ($connectionState === 'CONNECTED') {
      const h145Selected = airframe.includes('H145');
      const airframeId = h145Selected ? 0 : 1;
      const pilotKg = crewKg * 0.48;
      const attendantKg = crewKg * 0.52;
      const litterKg = payloadKg * 0.65;
      const rackKg = payloadKg * 0.35;
      const payloads = h145Selected
        ? [
            { station_id: 0, weight_kg: pilotKg },
            { station_id: 2, weight_kg: attendantKg },
            { station_id: 6, weight_kg: litterKg },
            { station_id: 8, weight_kg: rackKg }
          ]
        : [
            { station_id: 0, weight_kg: pilotKg },
            { station_id: 4, weight_kg: attendantKg },
            { station_id: 6, weight_kg: litterKg },
            { station_id: 9, weight_kg: rackKg }
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
        $activeCorridor.departure || 'KAPA'
      );
    }
  }

  $: if ($kinematicsPresetStore && $kinematicsPresetStore !== lastAppliedPreset) {
    applyPresetValues($kinematicsPresetStore);
    recalculateInteractiveKinematics();
  }

  onMount(() => {
    if ($kinematicsPresetStore) {
      applyPresetValues($kinematicsPresetStore);
    }
    recalculateInteractiveKinematics();
  });
</script>

<div class="cg-card" aria-label="Center of Gravity Envelope Visualizer">
  <div class="panel-header">
    <div class="flex items-center gap-2">
      <span>02 // ATMOSPHERE, MASS &amp; CG ENVELOPE (OEM AIRBUS H145 / BELL 429)</span>
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
        <select bind:value={airframe} on:change={recalculateInteractiveKinematics} class="airframe-sel" aria-label="Airframe Selector">
          <option value="Airbus H145 (BK117 D-2)">H145 (D-2)</option>
          <option value="Bell 429 GlobalRanger">BELL 429</option>
        </select>
      </div>
    </div>

    <!-- Interactive Atmospheric Sliders with Live Value Tracking -->
    <div class="sliders-box">
      <div class="slider-row">
        <div class="slider-lbl">
          <span>Field Elev: <strong class="text-cyan-400">{fieldElevFt.toLocaleString()} ft</strong></span>
          <span>OAT: <strong class="text-amber-400">{oatC.toFixed(0)}°C</strong></span>
          <span>Altimeter: <strong class="text-cyan-400">{qnhInHg.toFixed(2)}"</strong></span>
        </div>
        <div class="slider-grid-3">
          <input type="range" min="0" max="14000" step="50" bind:value={fieldElevFt} on:input={recalculateInteractiveKinematics} title="Field Elevation (ft)" />
          <input type="range" min="-25" max="45" step="1" bind:value={oatC} on:input={recalculateInteractiveKinematics} title="Outside Air Temp (°C)" />
          <input type="range" min="28.50" max="31.00" step="0.01" bind:value={qnhInHg} on:input={recalculateInteractiveKinematics} title="Altimeter QNH (inHg)" />
        </div>
      </div>
    </div>

    <!-- 2D OEM CG Envelope SVG Canvas -->
    <div class="svg-container-box">
      <svg viewBox="0 0 {WIDTH} {HEIGHT}" class="cg-svg">
        <!-- Grid Lines -->
        {#each gwTicks as gw}
          {@const y = scaleY(gw, gwMin, gwMax)}
          <line x1={PAD_X} y1={y} x2={WIDTH - 15} y2={y} stroke="var(--border-hairline)" stroke-width="1" stroke-dasharray="2,2" />
          <text x={PAD_X - 6} y={y + 3} class="axis-tick" text-anchor="end">{gw}</text>
        {/each}

        {#each longTicks as l}
          {@const x = scaleX(l, longMin, longMax)}
          <line x1={x} y1={HEIGHT - PAD_Y} x2={x} y2={PAD_Y} stroke="var(--border-hairline)" stroke-width="1" stroke-dasharray="2,2" />
          <text x={x} y={HEIGHT - 10} class="axis-tick" text-anchor="middle">{l}</text>
        {/each}

        <!-- Axis Labels -->
        <text x={WIDTH / 2} y={HEIGHT - 0} class="axis-title" text-anchor="middle">LONGITUDINAL CG (MM)</text>
        <text x={12} y={HEIGHT / 2} class="axis-title" text-anchor="middle" transform="rotate(-90 12 {HEIGHT / 2})">GW (KG)</text>

        <!-- Certified Boundary Polygon -->
        <polygon
          points={envelopePointsSvg}
          fill={$isCgCompliant ? 'rgba(0, 230, 118, 0.09)' : 'rgba(255, 23, 68, 0.08)'}
          stroke={$isCgCompliant ? 'var(--color-go)' : 'var(--color-nogo)'}
          stroke-width="1.8"
        />

        <!-- Operating Point Crosshair & Reticle -->
        <line x1={curX - 10} y1={curY} x2={curX + 10} y2={curY} stroke={$isCgCompliant ? 'var(--color-go)' : 'var(--color-nogo)'} stroke-width="2" />
        <line x1={curX} y1={curY - 10} x2={curX} y2={curY + 10} stroke={$isCgCompliant ? 'var(--color-go)' : 'var(--color-nogo)'} stroke-width="2" />
        <circle cx={curX} cy={curY} r="5" fill="none" stroke={$isCgCompliant ? 'var(--color-go)' : 'var(--color-nogo)'} stroke-width="2" />
        <circle cx={curX} cy={curY} r="10" fill="none" stroke={$isCgCompliant ? 'rgba(0, 230, 118, 0.35)' : 'rgba(255, 23, 68, 0.4)'} stroke-width="1" stroke-dasharray="2,2" />

        <!-- Live Coordinate Tag in SVG -->
        <text
          x={Math.min(WIDTH - 90, Math.max(PAD_X + 4, curX + 8))}
          y={Math.max(PAD_Y + 14, curY - 6)}
          class="axis-tick font-bold"
          fill={$isCgCompliant ? '#00e676' : '#ff1744'}
        >
          {Math.round($currentGrossWeight)}kg / {Math.round($currentCgLong)}mm
        </text>
      </svg>
    </div>

    <!-- Live Graphical Margin Meters -->
    <div class="meters-row">
      <!-- Gross Weight / MTOW Meter -->
      <div class="meter-box">
        <div class="meter-header">
          <span>MTOW CAPACITY</span>
          <span class="meter-stat" class:text-rose-400={$currentGrossWeight > mtowDisplay} class:text-emerald-400={$currentGrossWeight <= mtowDisplay}>
            {Math.round(($currentGrossWeight / mtowDisplay) * 100)}% ({Math.round(mtowDisplay - $currentGrossWeight)} kg margin)
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

      <!-- CG Longitudinal Balance Meter -->
      <div class="meter-box">
        <div class="meter-header">
          <span>LONG CG POSITION</span>
          <span class="meter-stat">
            FWD: +{Math.round($latestKinematics.mass_balance.fwd_margin_mm)} mm / AFT: +{Math.round($latestKinematics.mass_balance.aft_margin_mm)} mm
          </span>
        </div>
        <div class="meter-track">
          <div class="cg-marker-indicator" style="left: {cgProgressPct}%"></div>
        </div>
      </div>
    </div>

    <!-- Interactive Fuel, Payload & Crew Sliders with Live Feedback -->
    <div class="sliders-box">
      <div class="slider-row">
        <div class="slider-lbl">
          <span>Fuel: <strong class="text-cyan-400">{fuelKg} kg</strong> ({Math.round(fuelKg / 0.8)} L)</span>
          <span>Payload: <strong class="text-cyan-400">{payloadKg} kg</strong></span>
          <span>Crew: <strong class="text-cyan-400">{crewKg} kg</strong></span>
        </div>
        <div class="slider-grid-3">
          <input type="range" min="50" max="720" step="5" bind:value={fuelKg} on:input={recalculateInteractiveKinematics} title="Jet-A Usable Fuel (kg)" />
          <input type="range" min="0" max="350" step="5" bind:value={payloadKg} on:input={recalculateInteractiveKinematics} title="Patient + Gear Payload (kg)" />
          <input type="range" min="150" max="420" step="5" bind:value={crewKg} on:input={recalculateInteractiveKinematics} title="Crew Weight (kg)" />
        </div>
      </div>
    </div>

    <!-- Telemetry Cards -->
    <div class="cg-telemetry">
      <div class="tel-item">
        <span class="tel-label">GROSS WEIGHT</span>
        <span class="tel-val">{$currentGrossWeight.toFixed(0)} KG</span>
        <span class="tel-sub">MTOW {mtowDisplay} KG</span>
      </div>
      <div class="tel-item">
        <span class="tel-label">LONGITUDINAL CG</span>
        <span class="tel-val">{$currentCgLong.toFixed(1)} MM</span>
        <span class="tel-sub">FWD: {$latestKinematics.mass_balance.fwd_limit_mm.toFixed(0)} / AFT: {$latestKinematics.mass_balance.aft_limit_mm.toFixed(0)}</span>
      </div>
      <div class="tel-item">
        <span class="tel-label">LATERAL CG</span>
        <span class="tel-val">{$currentCgLat.toFixed(1)} MM</span>
        <span class="tel-sub">LEFT: {$latestKinematics.mass_balance.left_limit_mm.toFixed(0)} / RGHT: {$latestKinematics.mass_balance.right_limit_mm.toFixed(0)}</span>
      </div>
      <div class="tel-item">
        <span class="tel-label">CRUISE TAS / FF</span>
        <span class="tel-val">{$latestKinematics.cruise_tas_kts.toFixed(0)} KT / {$latestKinematics.cruise_fuel_flow_kg_hr.toFixed(0)} KG/H</span>
        <span class="tel-sub">ETE: {$latestKinematics.leg_ete_minutes.toFixed(1)} MIN</span>
      </div>
      <div class="tel-item">
        <span class="tel-label">OGE HOVER MARGIN</span>
        <span
          class="tel-val"
          class:text-rose-400={$latestKinematics.oge_margin_kg < 0}
          class:text-emerald-400={$latestKinematics.oge_margin_kg >= 0}
        >
          {$latestKinematics.oge_margin_kg >= 0 ? '+' : ''}{Math.round($latestKinematics.oge_margin_kg)} KG
        </span>
        <span class="tel-sub">MAX OGE {Math.round($latestKinematics.max_oge_hover_weight_kg)} KG</span>
      </div>
    </div>

    {#if $latestKinematics.actionable_advisory}
      <div class="mitigation-banner" data-testid="kinematics-mitigation">
        <span class="mitigation-label">MITIGATION //</span>
        <span class="mitigation-text">{$latestKinematics.actionable_advisory}</span>
        {#if ($latestKinematics.required_fuel_offload_kg ?? 0) > 0}
          <span class="mitigation-metric">
            (OFFLOAD {$latestKinematics.required_fuel_offload_kg} KG)
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
    grid-template-columns: repeat(auto-fit, minmax(130px, 1fr));
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
    padding: 1px 4px;
    width: 100%;
    font-weight: 700;
  }

  .sliders-box {
    background: var(--bg-void);
    border: 1px solid var(--border-hairline);
    padding: 5px 6px;
  }

  .slider-row {
    display: flex;
    flex-direction: column;
    gap: 3px;
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

  .airframe-badge {
    font-size: 9px;
    font-weight: 800;
    font-family: var(--font-mono);
    padding: 1px 5px;
    background: rgba(0, 229, 255, 0.12);
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
    padding: 5px 7px;
    font-size: 9px;
    font-family: var(--font-mono);
    color: var(--text-primary);
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
</style>
