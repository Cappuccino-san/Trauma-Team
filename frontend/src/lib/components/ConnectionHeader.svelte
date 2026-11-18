<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import {
    connectionState,
    lastGatewayError,
    wsRouter,
    isStaticHostingEnvironment
  } from '../websocket';
  import { loadSimulationScenario, activeScenarioKey } from '../simulation/fixtures';
  import { unitSystem, toggleUnitSystem } from '../stores/units';
  import { formatTelemetryModeLabel, formatZuluClock } from '../aviationTerminology';

  export let onOpenCustomPreset: () => void = () => {};

  let zuluClock = formatZuluClock();
  let clockTimer: ReturnType<typeof setInterval> | null = null;

  onMount(() => {
    zuluClock = formatZuluClock();
    clockTimer = setInterval(() => {
      zuluClock = formatZuluClock();
    }, 1000);
  });

  onDestroy(() => {
    if (clockTimer) {
      clearInterval(clockTimer);
      clockTimer = null;
    }
  });

  let customUrl =
    typeof window !== 'undefined' && !isStaticHostingEnvironment()
      ? `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`
      : 'ws://localhost:8080/ws';

  function toggleConnection() {
    if ($connectionState === 'CONNECTED' || $connectionState === 'CONNECTING') {
      wsRouter.disconnect();
    } else {
      wsRouter.connect(customUrl, { userInitiated: true });
    }
  }

  function dismissError() {
    lastGatewayError.set(null);
  }
</script>

<header class="telemetry-bar">
  <div class="brand-cluster">
    <span class="sys-id">TRAUMATEAM // HAA-135.L</span>
    <span class="status-pill status-{$connectionState}" data-testid="telemetry-mode">
      <span class="dot"></span>
      {formatTelemetryModeLabel($connectionState)}
    </span>
    <span class="metric zulu-clock" data-testid="zulu-clock" title="UTC / Zulu Time">
      ZULU: <strong>{zuluClock}</strong>
    </span>
    <button
      type="button"
      class="efb-touch unit-toggle-btn"
      class:imperial={$unitSystem === 'LBS_IN'}
      data-testid="unit-toggle"
      aria-pressed={$unitSystem === 'LBS_IN'}
      on:click={toggleUnitSystem}
      title="Toggle cockpit weight, fuel & CG display units between KG/MM and LBS/IN"
    >
      UNITS: <strong>{$unitSystem === 'LBS_IN' ? 'LBS/IN' : 'KG/MM'}</strong>
    </button>
  </div>

  <div class="controls-cluster">
    <div class="preset-group">
      <span class="group-label">SIM PRESETS:</span>
      <button
        type="button"
        class="efb-touch"
        class:preset-active={$activeScenarioKey === 'NOMINAL_GO'}
        on:click={() => loadSimulationScenario(wsRouter, 'NOMINAL_GO')}
      >
        GO (KAPA-KLXV)
      </button>
      <button
        type="button"
        class="efb-touch highda-preset-btn"
        class:preset-active={$activeScenarioKey === 'HIGH_DA_KLXV'}
        on:click={() => loadSimulationScenario(wsRouter, 'HIGH_DA_KLXV')}
      >
        HIGH-DA (KLXV)
      </button>
      <button
        type="button"
        class="efb-touch"
        class:preset-active={$activeScenarioKey === 'MARGINAL_CAUTION'}
        on:click={() => loadSimulationScenario(wsRouter, 'MARGINAL_CAUTION')}
      >
        MARGINAL (NVG)
      </button>
      <button
        type="button"
        class="efb-touch"
        class:preset-active={$activeScenarioKey === 'ASPEN_BELOW_MINIMUMS'}
        on:click={() => loadSimulationScenario(wsRouter, 'ASPEN_BELOW_MINIMUMS')}
      >
        NO-GO (KASE IFR)
      </button>
      <button
        type="button"
        class="efb-touch"
        class:preset-active={$activeScenarioKey === 'IFR_TRANSITION_REQUIRED'}
        on:click={() => loadSimulationScenario(wsRouter, 'IFR_TRANSITION_REQUIRED')}
      >
        §135.611 IFR
      </button>
      <button
        type="button"
        class="efb-touch"
        class:preset-active={$activeScenarioKey === 'NEW_ENGLAND_SIGMET'}
        on:click={() => loadSimulationScenario(wsRouter, 'NEW_ENGLAND_SIGMET')}
      >
        SIGMET (KBOS-KMHT-KPWM)
      </button>
      <button
        type="button"
        class="efb-touch custom-preset-btn"
        on:click={onOpenCustomPreset}
        title="Upload custom weather/hazard JSON or build your own simulation preset"
      >
        + CUSTOM / UPLOAD DATA
      </button>
    </div>

    <div class="ws-group">
      <input
        type="text"
        bind:value={customUrl}
        aria-label="WebSocket Gateway URL"
        class="efb-touch ws-url-input"
      />
      <button type="button" class="efb-touch" on:click={toggleConnection}>
        {$connectionState === 'CONNECTED' ? 'DISCONNECT' : 'LIVE WS'}
      </button>
    </div>
  </div>
  {#if $lastGatewayError}
    <button
      type="button"
      class="efb-touch gateway-error-pill"
      on:click={dismissError}
      title="Click to dismiss notice"
    >
      NOTICE: {$lastGatewayError} ×
    </button>
  {/if}
</header>

<style>
  .telemetry-bar {
    display: flex;
    flex-wrap: wrap;
    justify-content: space-between;
    align-items: center;
    gap: 8px;
    padding: 6px 12px;
    background: var(--bg-panel);
    border-bottom: 1px solid var(--border-strong);
  }

  .brand-cluster,
  .controls-cluster,
  .preset-group,
  .ws-group {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-wrap: wrap;
  }

  .sys-id {
    font-weight: 800;
    letter-spacing: 0.08em;
    color: var(--text-primary);
    padding-right: 6px;
    border-right: 1px solid var(--border-strong);
  }

  .status-pill {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    min-height: 30px;
    padding: 4px 8px;
    font-size: 10px;
    font-weight: 700;
    border: 1px solid var(--border-strong);
  }

  .dot {
    width: 6px;
    height: 6px;
    background: currentColor;
  }

  .status-CONNECTED {
    color: var(--color-go);
    border-color: var(--color-go);
    background: var(--color-go-bg);
  }

  .status-CONNECTING,
  .status-RECONNECTING {
    color: var(--color-caution);
    border-color: var(--color-caution);
    background: var(--color-caution-bg);
  }

  .status-DISCONNECTED,
  .status-ERROR {
    color: var(--text-secondary);
    border-color: var(--border-strong);
  }

  .status-STATIC_SIM {
    color: var(--color-info);
    border-color: var(--color-info);
    background: var(--color-info-bg);
  }

  .metric {
    font-size: 11px;
    color: var(--text-secondary);
    padding-left: 8px;
    border-left: 1px solid var(--border-hairline);
  }

  .zulu-clock strong {
    color: var(--color-info);
    font-weight: 800;
    letter-spacing: 0.04em;
  }

  .unit-toggle-btn {
    border-color: var(--color-info);
    color: var(--color-info);
    background: var(--color-info-bg);
    font-size: 10px;
    letter-spacing: 0.06em;
  }

  .unit-toggle-btn.imperial {
    border-color: var(--color-caution);
    color: var(--color-caution);
    background: var(--color-caution-bg);
  }

  .unit-toggle-btn strong {
    font-weight: 800;
  }

  .group-label {
    font-size: 10px;
    color: var(--text-muted);
    letter-spacing: 0.06em;
  }

  .preset-active {
    border-color: var(--color-go);
    box-shadow: inset 0 0 0 1px var(--color-go);
    color: var(--color-go);
    background: var(--color-go-bg);
  }

  .ws-url-input {
    width: 170px;
    font-size: 11px;
  }

  .gateway-error-pill {
    padding: 4px 10px;
    background: var(--color-caution-bg);
    border: 1px solid var(--color-caution);
    color: var(--color-caution);
    font-size: 10px;
    max-width: 260px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .custom-preset-btn {
    color: var(--color-go);
    border-color: var(--color-go);
    background: var(--color-go-bg);
  }

  .custom-preset-btn:hover {
    background: var(--color-go);
    color: #000;
  }

  .highda-preset-btn {
    color: var(--color-nogo);
    border-color: rgba(255, 23, 68, 0.55);
  }

  .highda-preset-btn.preset-active {
    border-color: var(--color-nogo);
    box-shadow: inset 0 0 0 1px var(--color-nogo);
    color: var(--color-nogo);
    background: var(--color-nogo-bg);
  }

  .highda-preset-btn:hover {
    background: var(--color-nogo);
    color: #fff;
  }
</style>
