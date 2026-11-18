<script lang="ts">
  import { latestKinematics } from '../stores/kinematics';
  import { activeCorridor, latestComplianceReport } from '../stores/verdict';
  import { unitSystem, formatWeight, formatFuelVolume } from '../stores/units';
  import { computeCorridorDistanceNm } from '../stations';

  $: totalDistanceNm =
    $latestKinematics.total_distance_nm ?? computeCorridorDistanceNm($activeCorridor);
  $: totalEteMinutes = $latestKinematics.total_ete_minutes ?? $latestKinematics.leg_ete_minutes;
  $: groundspeedKts = $latestKinematics.groundspeed_kts;
  $: headwindKts = $latestKinematics.headwind_component_kts;
  $: fuelBurnKg = $latestKinematics.total_fuel_burn_kg ?? $latestKinematics.leg_fuel_burn_kg;
  $: sceneHoverMin = $latestKinematics.scene_hover_minutes ?? 5;

  $: isIfrMode = Boolean(
    $latestKinematics.is_ifr_reserve || $latestComplianceReport?.verdict === 'IFR_REQUIRED'
  );
  $: reserveRequiredMin = isIfrMode ? 30 : ($latestKinematics.reserve_required_minutes ?? 20);
  $: cruiseFfKgHr = $latestKinematics.cruise_fuel_flow_kg_hr || 210;
  $: requiredReserveKg = (reserveRequiredMin / 60) * cruiseFfKgHr;
  $: reserveMarginKg = $latestKinematics.remaining_fuel_kg - requiredReserveKg;
  $: reserveMarginMin = cruiseFfKgHr > 0 ? (reserveMarginKg / cruiseFfKgHr) * 60 : 0;
  $: reserveMarginSign = reserveMarginMin >= 0 ? '+' : '';
  $: reserveTierClass =
    reserveMarginMin < 0
      ? 'res-breach'
      : reserveMarginMin < 5
        ? 'res-tight'
        : 'res-nominal';
</script>

<section
  class="mission-strip"
  data-testid="mission-leg-strip"
  aria-label="7-Leg HEMS Mission Profile and § 135.209 Reserve Bar"
>
  <div class="legs-rail">
    <span class="rail-title">7-LEG PROFILE</span>
    <div class="leg-crumbs" aria-hidden="true">
      <span class="leg-chip">0.WARM</span>
      <span class="leg-sep">›</span>
      <span class="leg-chip">1.OUT</span>
      <span class="leg-sep">›</span>
      <span class="leg-chip leg-active">2.SCENE OGE ({sceneHoverMin}M)</span>
      <span class="leg-sep">›</span>
      <span class="leg-chip">3.XPORT</span>
      <span class="leg-sep">›</span>
      <span class="leg-chip">4.PAD</span>
      <span class="leg-sep">›</span>
      <span class="leg-chip">5.RTB</span>
      <span class="leg-sep">›</span>
      <span class="leg-chip">{isIfrMode ? '6.RESV 30M IFR' : '6.RESV 20M VFR'}</span>
    </div>
  </div>

  <div class="mission-metrics">
    <div class="m-cell" data-testid="mission-total-distance">
      <span class="m-label">TOTAL DISTANCE</span>
      <span class="m-val">{totalDistanceNm.toFixed(1)} NM</span>
    </div>

    <div class="m-cell" data-testid="mission-total-ete">
      <span class="m-label">TOTAL ETE</span>
      <span class="m-val">{totalEteMinutes.toFixed(1)} MIN</span>
    </div>

    <div class="m-cell" data-testid="mission-groundspeed">
      <span class="m-label">CRUISE GROUNDSPEED</span>
      <span class="m-val">
        {Math.round(groundspeedKts)} KT
        <span class="m-sub">
          (TAS {Math.round($latestKinematics.cruise_tas_kts)} / HDW {headwindKts >= 0 ? '+' : ''}{Math.round(headwindKts)} KT)
        </span>
      </span>
    </div>

    <div class="m-cell" data-testid="mission-fuel-burn">
      <span class="m-label">EST. FUEL BURN</span>
      <span class="m-val">
        {formatWeight(fuelBurnKg, $unitSystem)}
        <span class="m-sub">({formatFuelVolume(fuelBurnKg, $unitSystem)})</span>
      </span>
    </div>

    <div class="m-cell m-cell-reserve {reserveTierClass}" data-testid="mission-reserve-margin">
      <span class="m-label">14 CFR § 135.209 RESERVE MARGIN</span>
      <span class="m-val">
        {reserveMarginSign}{Math.round(reserveMarginMin)} MIN
        <span class="m-sub">
          above {reserveRequiredMin}-MIN {isIfrMode ? 'IFR' : 'VFR'} MIN ({formatWeight(reserveMarginKg, $unitSystem, { signed: true })})
        </span>
      </span>
    </div>
  </div>
</section>

<style>
  .mission-strip {
    background: var(--bg-panel);
    border: 1px solid var(--border-strong);
    border-top: none;
    padding: 5px 12px;
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
  }

  .legs-rail {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }

  .rail-title {
    font-size: 9px;
    font-weight: 800;
    letter-spacing: 0.1em;
    color: var(--color-info);
    background: var(--color-info-bg);
    border: 1px solid rgba(0, 229, 255, 0.35);
    padding: 2px 6px;
    white-space: nowrap;
  }

  .leg-crumbs {
    display: flex;
    align-items: center;
    gap: 4px;
    flex-wrap: wrap;
    font-size: 9px;
    color: var(--text-muted);
  }

  .leg-chip {
    background: var(--bg-void);
    border: 1px solid var(--border-hairline);
    padding: 1px 5px;
    color: var(--text-secondary);
    white-space: nowrap;
  }

  .leg-active {
    color: var(--text-primary);
    border-color: var(--color-info);
  }

  .leg-sep {
    color: var(--text-muted);
  }

  .mission-metrics {
    flex: 1;
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(130px, 1fr));
    gap: 1px;
    background: var(--border-hairline);
    border: 1px solid var(--border-hairline);
  }

  .m-cell {
    background: var(--bg-void);
    padding: 4px 8px;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }

  .m-label {
    font-size: 8px;
    color: var(--text-muted);
    letter-spacing: 0.08em;
    font-weight: 700;
  }

  .m-val {
    font-size: 11px;
    font-weight: 800;
    color: var(--text-primary);
  }

  .m-sub {
    font-size: 9px;
    font-weight: 600;
    color: var(--text-secondary);
  }

  .res-nominal .m-val {
    color: var(--color-go);
  }

  .res-tight .m-val {
    color: var(--color-caution);
  }

  .res-breach .m-val {
    color: var(--color-nogo);
  }

  .res-breach {
    background: var(--color-nogo-bg);
  }
</style>
