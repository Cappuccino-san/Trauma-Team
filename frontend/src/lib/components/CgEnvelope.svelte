<script lang="ts">
  import {
    latestKinematics,
    currentGrossWeight,
    currentCgLong,
    currentCgLat,
    isCgCompliant
  } from '../stores/kinematics';

  // Certified Airbus H145 (BK117 D-2) envelope boundaries:
  // GW range: 1900 kg to 3700 kg
  // Long CG range: 3750 mm to 4250 mm
  // Lat CG range: -100 mm to +100 mm
  const GW_MIN = 1800;
  const GW_MAX = 3800;
  const LONG_MIN = 3700;
  const LONG_MAX = 4250;

  // ViewBox bounds for SVG canvas
  const WIDTH = 340;
  const HEIGHT = 210;
  const PAD_X = 45;
  const PAD_Y = 25;
  const PLOT_W = WIDTH - PAD_X - 15;
  const PLOT_H = HEIGHT - PAD_Y - 25;

  function scaleX(longMm: number): number {
    return PAD_X + ((longMm - LONG_MIN) / (LONG_MAX - LONG_MIN)) * PLOT_W;
  }

  function scaleY(gwKg: number): number {
    return HEIGHT - PAD_Y - ((gwKg - GW_MIN) / (GW_MAX - GW_MIN)) * PLOT_H;
  }

  // Certified H145 CG points from OEM tabular charts
  const CERT_ENVELOPE = [
    { gw: 2140, fwd: 3770, aft: 4180 },
    { gw: 2600, fwd: 3780, aft: 4185 },
    { gw: 3100, fwd: 3810, aft: 4190 },
    { gw: 3585, fwd: 3855, aft: 4195 },
    { gw: 3700, fwd: 3880, aft: 4195 }
  ];

  $: envelopePointsSvg = (() => {
    // Fwd boundary going up, then Aft boundary going down to form closed polygon
    const fwdPath = CERT_ENVELOPE.map((p) => `${scaleX(p.fwd).toFixed(1)},${scaleY(p.gw).toFixed(1)}`);
    const aftPath = [...CERT_ENVELOPE]
      .reverse()
      .map((p) => `${scaleX(p.aft).toFixed(1)},${scaleY(p.gw).toFixed(1)}`);
    return [...fwdPath, ...aftPath].join(' ');
  })();

  $: curX = scaleX($currentCgLong || 3864);
  $: curY = scaleY($currentGrossWeight || 2890);

  // Interactive Kinematics & Mass/Balance Sliders (ported from Traumateam Console artifact)
  let airframe = 'Airbus H145 (BK117 D-2)';
  let fieldElevFt = 5885;
  let oatC = 18.0;
  let qnhInHg = 29.95;

  let fuelKg = 420;
  let payloadKg = 140;
  let crewKg = 192;

  function recalculateInteractiveKinematics() {
    // 1. Atmospheric State (ICAO Standard Atmosphere)
    const p0 = 29.9213;
    const pa = fieldElevFt + 145442.156 * (1.0 - Math.pow(qnhInHg / p0, 0.190263));
    const isaTempC = 15.0 - 1.98 * (pa / 1000.0);
    const da = pa + 118.8 * (oatC - isaTempC);

    // 2. Mass & CG Envelope (Airbus H145 / Bell 429)
    const isH145 = airframe.includes('H145');
    const emptyWeight = isH145 ? 2138.0 : 2045.0;
    const emptyLongArm = isH145 ? 3910.0 : 3890.0;
    const emptyLatArm = -5.0;

    const totalGW = emptyWeight + fuelKg + payloadKg + crewKg;
    const fuelArm = fuelKg <= 200 ? 4010 : 3980;
    const crewArm = 3620;
    const payloadArm = 4080;

    const longMoment =
      emptyWeight * emptyLongArm +
      fuelKg * fuelArm +
      crewKg * crewArm +
      payloadKg * payloadArm;
    const longCG = longMoment / totalGW;

    const latMoment =
      emptyWeight * emptyLatArm + fuelKg * 0.0 + crewKg * -12.0 + payloadKg * -20.0;
    const latCG = latMoment / totalGW;

    const mtow = isH145 ? 3700 : 3402;
    const fwdLimit = totalGW <= 2600 ? 3780 : totalGW <= 3100 ? 3810 : 3855;
    const aftLimit = 4190;
    const isGwOk = totalGW <= mtow;
    const isLongOk = longCG >= fwdLimit && longCG <= aftLimit;
    const isLatOk = latCG >= -80 && latCG <= 80;
    const isCompliant = isGwOk && isLongOk && isLatOk;

    latestKinematics.update((prev) => ({
      ...prev,
      airframe,
      pressure_altitude_ft: pa,
      density_altitude_ft: da,
      mass_balance: {
        ...prev.mass_balance,
        is_gw_compliant: isGwOk,
        is_long_cg_compliant: isLongOk,
        is_lat_cg_compliant: isLatOk,
        is_overall_compliant: isCompliant,
        gross_weight_kg: totalGW,
        cg_long_mm: longCG,
        cg_lat_mm: latCG,
        fwd_limit_mm: fwdLimit,
        aft_limit_mm: aftLimit
      }
    }));
  }
</script>

<div class="cg-card" aria-label="Center of Gravity Envelope Visualizer">
  <div class="panel-header">
    <span>02 // ATMOSPHERE, MASS & CG ENVELOPE (OEM AIRBUS H145 / BELL 429)</span>
    <span class="cg-status" class:compliant={$isCgCompliant} class:violation={!$isCgCompliant}>
      {$isCgCompliant ? 'COMPLIANT' : 'EXCEEDED'}
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
        <span class="tel-label">AIRFRAME</span>
        <select bind:value={airframe} on:change={recalculateInteractiveKinematics} class="airframe-sel">
          <option value="Airbus H145 (BK117 D-2)">H145 (D-2)</option>
          <option value="Bell 429 GlobalRanger">BELL 429</option>
        </select>
      </div>
    </div>

    <!-- Interactive Atmospheric Sliders -->
    <div class="sliders-box">
      <div class="slider-row">
        <div class="slider-lbl">
          <span>Elev: <strong>{fieldElevFt.toLocaleString()} ft</strong></span>
          <span>OAT: <strong>{oatC.toFixed(0)}°C</strong></span>
          <span>QNH: <strong>{qnhInHg.toFixed(2)}"</strong></span>
        </div>
        <div class="slider-grid-3">
          <input type="range" min="0" max="14000" step="50" bind:value={fieldElevFt} on:input={recalculateInteractiveKinematics} title="Field Elevation (ft)" />
          <input type="range" min="-25" max="45" step="1" bind:value={oatC} on:input={recalculateInteractiveKinematics} title="Outside Air Temp (°C)" />
          <input type="range" min="28.50" max="31.00" step="0.01" bind:value={qnhInHg} on:input={recalculateInteractiveKinematics} title="Altimeter QNH (inHg)" />
        </div>
      </div>
    </div>

    <svg viewBox="0 0 {WIDTH} {HEIGHT}" class="cg-svg">
      <!-- Grid Lines -->
      {#each [2000, 2500, 3000, 3500] as gw}
        {@const y = scaleY(gw)}
        <line x1={PAD_X} y1={y} x2={WIDTH - 15} y2={y} stroke="var(--border-hairline)" stroke-width="1" stroke-dasharray="2,2" />
        <text x={PAD_X - 6} y={y + 3} class="axis-tick" text-anchor="end">{gw}</text>
      {/each}

      {#each [3800, 3900, 4000, 4100, 4200] as l}
        {@const x = scaleX(l)}
        <line x1={x} y1={HEIGHT - PAD_Y} x2={x} y2={PAD_Y} stroke="var(--border-hairline)" stroke-width="1" stroke-dasharray="2,2" />
        <text x={x} y={HEIGHT - 10} class="axis-tick" text-anchor="middle">{l}</text>
      {/each}

      <!-- Axis Labels -->
      <text x={WIDTH / 2} y={HEIGHT - 0} class="axis-title" text-anchor="middle">LONGITUDINAL CG (MM)</text>
      <text x={12} y={HEIGHT / 2} class="axis-title" text-anchor="middle" transform="rotate(-90 12 {HEIGHT / 2})">GW (KG)</text>

      <!-- Certified Boundary Polygon -->
      <polygon
        points={envelopePointsSvg}
        fill="rgba(0, 230, 118, 0.08)"
        stroke="var(--color-go)"
        stroke-width="1.5"
      />

      <!-- Current Operating Point Crosshair -->
      <line x1={curX - 8} y1={curY} x2={curX + 8} y2={curY} stroke={$isCgCompliant ? 'var(--color-go)' : 'var(--color-nogo)'} stroke-width="2" />
      <line x1={curX} y1={curY - 8} x2={curX} y2={curY + 8} stroke={$isCgCompliant ? 'var(--color-go)' : 'var(--color-nogo)'} stroke-width="2" />
      <circle cx={curX} cy={curY} r="4" fill="none" stroke={$isCgCompliant ? 'var(--color-go)' : 'var(--color-nogo)'} stroke-width="1.5" />
    </svg>

    <!-- Interactive Fuel, Payload & Crew Sliders -->
    <div class="sliders-box">
      <div class="slider-row">
        <div class="slider-lbl">
          <span>Fuel: <strong>{fuelKg}kg</strong></span>
          <span>Payload: <strong>{payloadKg}kg</strong></span>
          <span>Crew: <strong>{crewKg}kg</strong></span>
        </div>
        <div class="slider-grid-3">
          <input type="range" min="50" max="720" step="5" bind:value={fuelKg} on:input={recalculateInteractiveKinematics} title="Jet-A Usable Fuel (kg)" />
          <input type="range" min="0" max="350" step="5" bind:value={payloadKg} on:input={recalculateInteractiveKinematics} title="Patient + Gear Payload (kg)" />
          <input type="range" min="150" max="420" step="5" bind:value={crewKg} on:input={recalculateInteractiveKinematics} title="Crew Weight (kg)" />
        </div>
      </div>
    </div>

    <div class="cg-telemetry">
      <div class="tel-item">
        <span class="tel-label">GROSS WEIGHT</span>
        <span class="tel-val">{$currentGrossWeight.toFixed(0)} KG</span>
        <span class="tel-sub">MTOW 3700 KG</span>
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
    </div>
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

  .slider-grid-3 input[type='range'] {
    width: 100%;
    height: 4px;
    padding: 0;
    accent-color: var(--color-go);
    cursor: pointer;
  }
</style>
