<script lang="ts">
  import {
    connectionState,
    lastSeq,
    seqGapCount,
    rttLatencyMs,
    subscribedTopics,
    lastGatewayError,
    wsRouter
  } from '../websocket';
  import { loadSimulationScenario } from '../simulation/fixtures';

  export let onOpenCustomPreset: () => void = () => {};

  let customUrl =
    typeof window !== 'undefined'
      ? `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`
      : 'ws://localhost:8080/ws';

  function toggleConnection() {
    if ($connectionState === 'CONNECTED' || $connectionState === 'CONNECTING') {
      wsRouter.disconnect();
    } else {
      wsRouter.connect(customUrl);
    }
  }

  function triggerPing() {
    wsRouter.sendPing();
  }

  function dismissError() {
    lastGatewayError.set(null);
  }
</script>

<header class="telemetry-bar">
  <div class="brand-cluster">
    <span class="sys-id">TRAUMATEAM // HAA-135.L</span>
    <span class="status-pill status-{$connectionState}">
      <span class="dot"></span>
      WS: {$connectionState}
    </span>
    <span class="metric">SEQ: <strong>{$lastSeq}</strong></span>
    {#if $seqGapCount > 0}
      <span class="metric gap-warn">GAPS: {$seqGapCount}</span>
    {/if}
    <span class="metric">
      RTT: <strong>{$rttLatencyMs !== null ? `${$rttLatencyMs}ms` : '--'}</strong>
    </span>
    <span class="metric topics-metric" title={$subscribedTopics.join(', ')}>
      TOPICS: <strong>{$subscribedTopics.length}</strong>
    </span>
  </div>

  <div class="controls-cluster">
    <div class="preset-group">
      <span class="group-label">SIM PRESETS:</span>
      <button type="button" on:click={() => loadSimulationScenario(wsRouter, 'NOMINAL_GO')}>
        GO (KAPA-KLXV)
      </button>
      <button type="button" on:click={() => loadSimulationScenario(wsRouter, 'MARGINAL_CAUTION')}>
        MARGINAL (NVG)
      </button>
      <button type="button" on:click={() => loadSimulationScenario(wsRouter, 'ASPEN_BELOW_MINIMUMS')}>
        NO-GO (KASE IFR)
      </button>
      <button type="button" on:click={() => loadSimulationScenario(wsRouter, 'IFR_TRANSITION_REQUIRED')}>
        §135.611 IFR
      </button>
      <button type="button" on:click={() => loadSimulationScenario(wsRouter, 'NEW_ENGLAND_SIGMET')}>
        SIGMET (KBOS-KMHT-KPWM)
      </button>
      <button
        type="button"
        class="custom-preset-btn"
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
        class="ws-url-input"
      />
      <button type="button" on:click={toggleConnection}>
        {$connectionState === 'CONNECTED' ? 'DISCONNECT' : 'LIVE WS'}
      </button>
      <button type="button" on:click={triggerPing} disabled={$connectionState !== 'CONNECTED'}>
        PING
      </button>
    </div>
  </div>
  {#if $lastGatewayError}
    <button
      type="button"
      class="gateway-error-pill"
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
    gap: 8px;
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
    padding: 2px 6px;
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

  .metric {
    font-size: 11px;
    color: var(--text-secondary);
    padding-left: 6px;
    border-left: 1px solid var(--border-hairline);
  }

  .metric strong {
    color: var(--text-primary);
  }

  .gap-warn {
    color: var(--color-caution);
  }

  .group-label {
    font-size: 10px;
    color: var(--text-muted);
    letter-spacing: 0.06em;
  }

  .ws-url-input {
    width: 175px;
    font-size: 11px;
  }

  .gateway-error-pill {
    padding: 2px 8px;
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
</style>
