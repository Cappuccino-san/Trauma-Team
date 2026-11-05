<script lang="ts">
  import { activeCorridor, regulatoryContext } from '../stores/verdict';
  import { wsRouter, connectionState } from '../websocket';
  import { loadSimulationScenario } from '../simulation/fixtures';

  let intermediatesText = $activeCorridor.intermediates.join(', ');

  function syncIntermediates() {
    $activeCorridor.intermediates = intermediatesText
      .split(',')
      .map((s) => s.trim().toUpperCase())
      .filter(Boolean);
  }

  function submitEvaluation() {
    syncIntermediates();
    wsRouter.requestMinimumsEvaluation($activeCorridor, $regulatoryContext);

    // If not connected to live Go backend, evaluate deterministically against local simulation fixtures
    if ($connectionState !== 'CONNECTED') {
      const dest = $activeCorridor.destination.trim().toUpperCase();
      if (dest === 'KASE') {
        if ($regulatoryContext.is_ifr_certified && $regulatoryContext.has_ifr_alternate) {
          loadSimulationScenario(wsRouter, 'IFR_TRANSITION_REQUIRED');
        } else {
          loadSimulationScenario(wsRouter, 'ASPEN_BELOW_MINIMUMS');
        }
      } else if (!$regulatoryContext.is_day && $regulatoryContext.equipment.nvg_equipped) {
        loadSimulationScenario(wsRouter, 'MARGINAL_CAUTION');
      } else {
        loadSimulationScenario(wsRouter, 'NOMINAL_GO');
      }
    }
  }

  function subscribeCorridorStations() {
    syncIntermediates();
    const stations = [
      $activeCorridor.departure,
      ...$activeCorridor.intermediates,
      $activeCorridor.destination
    ].filter(Boolean);
    wsRouter.subscribe(stations, ['minimums', 'hazards']);
  }
</script>

<section class="panel-box" aria-label="Corridor and Regulatory Matrix">
  <div class="panel-header">
    <span>01 // CORRIDOR & 14 CFR § 135.L MATRIX</span>
    <span>EVAL_MINIMUMS</span>
  </div>

  <div class="panel-body">
    <div class="field-grid-3">
      <label class="field-item">
        <span>DEPARTURE ICAO</span>
        <input
          type="text"
          bind:value={$activeCorridor.departure}
          maxlength="4"
          aria-label="Departure Station"
        />
      </label>

      <label class="field-item">
        <span>INTERMEDIATES (CSV)</span>
        <input
          type="text"
          bind:value={intermediatesText}
          on:change={syncIntermediates}
          aria-label="Intermediate Stations"
        />
      </label>

      <label class="field-item">
        <span>DESTINATION ICAO</span>
        <input
          type="text"
          bind:value={$activeCorridor.destination}
          maxlength="4"
          aria-label="Destination Station"
        />
      </label>
    </div>

    <div class="field-grid-2">
      <label class="field-item">
        <span>PLANNED CRUISE (FT MSL)</span>
        <input
          type="number"
          step="100"
          bind:value={$regulatoryContext.planned_cruise_alt_msl}
        />
      </label>

      <label class="field-item">
        <span>MAX OBSTACLE (FT MSL)</span>
        <input
          type="number"
          step="100"
          bind:value={$regulatoryContext.highest_obstacle_msl}
        />
      </label>
    </div>

    <div class="toggles-section">
      <div class="sub-header">OPERATIONAL ENVIRONMENT (§ 135.609 TABLE 1)</div>
      <div class="checkbox-grid">
        <label class="chk-row">
          <input type="checkbox" bind:checked={$regulatoryContext.is_day} />
          <span>DAYLIGHT OPS</span>
        </label>
        <label class="chk-row">
          <input type="checkbox" bind:checked={$regulatoryContext.is_mountainous} />
          <span>MOUNTAINOUS</span>
        </label>
        <label class="chk-row">
          <input type="checkbox" bind:checked={$regulatoryContext.is_local} />
          <span>LOCAL AREA</span>
        </label>
        <label class="chk-row">
          <input type="checkbox" bind:checked={$regulatoryContext.allow_opspec_a021_relief} />
          <span>OPSPEC A021</span>
        </label>
      </div>
    </div>

    <div class="toggles-section">
      <div class="sub-header">AVIONICS & IFR CAPABILITY (§ 135.611)</div>
      <div class="checkbox-grid">
        <label class="chk-row">
          <input type="checkbox" bind:checked={$regulatoryContext.equipment.nvg_equipped} />
          <span>NVG EQUIPPED</span>
        </label>
        <label class="chk-row">
          <input type="checkbox" bind:checked={$regulatoryContext.equipment.htaws_equipped} />
          <span>HTAWS ACTIVE</span>
        </label>
        <label class="chk-row">
          <input type="checkbox" bind:checked={$regulatoryContext.is_ifr_certified} />
          <span>IFR CERTIFIED</span>
        </label>
        <label class="chk-row">
          <input type="checkbox" bind:checked={$regulatoryContext.has_ifr_alternate} />
          <span>IFR ALTERNATE</span>
        </label>
      </div>
    </div>

    <div class="action-bar">
      <button type="button" class="btn-primary" on:click={submitEvaluation}>
        EVALUATE CORRIDOR
      </button>
      <button type="button" on:click={subscribeCorridorStations}>
        SUBSCRIBE ROUTE
      </button>
    </div>
  </div>
</section>

<style>
  .panel-box {
    background: var(--bg-panel);
    border: 1px solid var(--border-hairline);
    display: flex;
    flex-direction: column;
  }

  .panel-body {
    padding: 10px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .field-grid-3 {
    display: grid;
    grid-template-columns: 1fr 1.4fr 1fr;
    gap: 6px;
  }

  .field-grid-2 {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 6px;
  }

  .field-item {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  .field-item span {
    font-size: 9px;
    color: var(--text-secondary);
    letter-spacing: 0.06em;
  }

  .field-item input {
    width: 100%;
    font-weight: 700;
    text-transform: uppercase;
  }

  .toggles-section {
    border-top: 1px solid var(--border-hairline);
    padding-top: 8px;
  }

  .sub-header {
    font-size: 9px;
    color: var(--text-muted);
    letter-spacing: 0.08em;
    margin-bottom: 6px;
  }

  .checkbox-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 6px;
  }

  .chk-row {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    cursor: pointer;
    user-select: none;
  }

  .chk-row input[type='checkbox'] {
    accent-color: var(--color-go);
    width: 13px;
    height: 13px;
    cursor: pointer;
  }

  .action-bar {
    display: grid;
    grid-template-columns: 1.5fr 1fr;
    gap: 6px;
    padding-top: 4px;
  }

  .btn-primary {
    border-color: var(--color-go);
    color: var(--color-go);
  }

  .btn-primary:hover {
    background: var(--color-go);
    color: var(--bg-void);
  }
</style>
