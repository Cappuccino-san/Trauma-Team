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
</script>

<div class="cg-card" aria-label="Center of Gravity Envelope Visualizer">
  <div class="panel-header">
    <span>05 // MASS & CG ENVELOPE (OEM AIRBUS H145 / BELL 429)</span>
    <span class="cg-status" class:compliant={$isCgCompliant} class:violation={!$isCgCompliant}>
      {$isCgCompliant ? 'COMPLIANT' : 'EXCEEDED'}
    </span>
  </div>

  <div class="cg-body">
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
</style>
