<script lang="ts">
  import { verdictStore, hazardStore } from '$lib/stores/verdict';
  import { websocketLogStore } from '$lib/websocket';

  // Track the active tab state natively
  let activeTab: 'audit' | 'sigmets' | 'envelopes' = 'audit';
</script>

<section class="panel-container flex flex-col" aria-label="Audit Trace and SIGMET Console">
  <!-- Tab Navigation Header -->
  <div class="panel-header-bar">
    <span class="panel-title">03 // DETERMINISTIC AUDIT TRACE & TELEMETRY CONSOLE</span>
    <div class="flex space-x-1">
      <button
        type="button"
        class="px-2.5 py-0.5 font-mono text-[10px] uppercase {activeTab === 'audit'
          ? 'bg-gray-700 text-white border-gray-400'
          : 'text-gray-400 hover:text-gray-200'}"
        on:click={() => (activeTab = 'audit')}
      >
        Audit Trace ({$verdictStore?.evaluationHistory?.length || 0})
      </button>
      <button
        type="button"
        class="px-2.5 py-0.5 font-mono text-[10px] uppercase {activeTab === 'sigmets'
          ? 'bg-red-900/60 text-red-300 border-red-500'
          : ($hazardStore?.length || 0) > 0
            ? 'text-red-400 border-red-800 hover:text-red-300'
            : 'text-gray-400 hover:text-gray-200'}"
        on:click={() => (activeTab = 'sigmets')}
      >
        SIGMETS ({$hazardStore?.length || 0})
      </button>
      <button
        type="button"
        class="px-2.5 py-0.5 font-mono text-[10px] uppercase {activeTab === 'envelopes'
          ? 'bg-gray-700 text-white border-gray-400'
          : 'text-gray-400 hover:text-gray-200'}"
        on:click={() => (activeTab = 'envelopes')}
      >
        WS Envelopes ({$websocketLogStore?.length || 0})
      </button>
    </div>
  </div>

  <!-- Dynamic Tab Content -->
  <div class="tab-viewport font-mono text-xs">
    {#if activeTab === 'audit'}
      <!-- Render deterministic FAA rule audit trace -->
      <div class="space-y-1" data-testid="audit-trace-list">
        {#each $verdictStore?.evaluationHistory || [] as traceItem, idx}
          <div
            class="trace-row {traceItem.includes('FAIL') ||
            traceItem.includes('DISQUALIFIED') ||
            traceItem.includes('BELOW_MINIMUMS')
              ? 'trace-fail'
              : traceItem.includes('PASS') || traceItem.includes('APPROVED')
                ? 'trace-pass'
                : ''}"
          >
            <span class="trace-idx">{String(idx + 1).padStart(2, '0')}</span>
            <span class="trace-text">{traceItem}</span>
          </div>
        {:else}
          <div class="text-gray-500 py-2">NO AUDIT TRACE RECORDED</div>
        {/each}
      </div>
    {:else if activeTab === 'sigmets'}
      <!-- Render active SIGMET advisories -->
      <div class="space-y-1.5">
        {#each $hazardStore || [] as hazard}
          <div class="sigmet-item">
            <div class="flex justify-between items-center text-[10px] text-red-400 font-bold">
              <span>{hazard.advisory_id}</span>
              <span>{hazard.hazard_type} // {hazard.severity}</span>
              <span>
                FL{Math.round((hazard.min_alt_ft || 0) / 100)
                  .toString()
                  .padStart(3, '0')} - FL{Math.round((hazard.max_alt_ft || 0) / 100)
                  .toString()
                  .padStart(3, '0')}
              </span>
            </div>
            <div class="text-gray-200 text-[11px]">{hazard.summary}</div>
            {#if hazard.raw_text}
              <code class="text-[10px] text-gray-400 block">{hazard.raw_text}</code>
            {/if}
          </div>
        {:else}
          <div class="text-gray-500 py-2">NO ACTIVE SIGMET ADVISORIES IN CORRIDOR</div>
        {/each}
      </div>
    {:else if activeTab === 'envelopes'}
      <!-- Render inbound/outbound WebSocket envelope wire frames -->
      {#if $websocketLogStore && $websocketLogStore.length > 0}
        <div class="space-y-0.5">
          {#each $websocketLogStore as item}
            <div class="py-1 px-1.5 border-b border-gray-900 flex items-center space-x-2 text-[11px]">
              <span
                class="px-1.5 py-0.2 font-bold text-[9px] {item.direction === 'IN'
                  ? 'text-emerald-400 bg-emerald-950/60 border border-emerald-800'
                  : 'text-cyan-400 bg-cyan-950/60 border border-cyan-800'}">{item.direction}</span
              >
              <span class="text-gray-500 w-10">{item.seq ? `#${item.seq}` : '--'}</span>
              <span class="text-gray-200 font-bold w-40 truncate">{item.eventOrAction}</span>
              <span class="text-gray-400 flex-1 truncate">{item.summary}</span>
            </div>
          {/each}
        </div>
      {:else}
        <div class="text-gray-400 opacity-70 py-2">
          Listening for telemetry frames...
        </div>
      {/if}
    {/if}
  </div>
</section>

<style>
  .panel-container {
    background: var(--bg-panel, #0c0e12);
    border: 1px solid var(--border-hairline, #1e232e);
  }

  .panel-header-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: 6px;
    padding: 5px 10px;
    background: var(--bg-elevated, #12151c);
    border-bottom: 1px solid var(--border-hairline, #1e232e);
  }

  .panel-title {
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0.1em;
    text-transform: uppercase;
    color: var(--text-secondary, #8c96a8);
  }

  .tab-viewport {
    padding: 6px 10px;
    height: 155px;
    overflow-y: auto;
  }

  .trace-row {
    display: flex;
    gap: 8px;
    font-size: 11px;
    padding: 2px 6px;
    background: var(--bg-void, #07080a);
    border-left: 2px solid var(--border-strong, #2d3545);
  }

  .trace-pass {
    border-left-color: var(--color-go, #00e676);
  }

  .trace-fail {
    border-left-color: var(--color-nogo, #ff1744);
    color: var(--color-nogo, #ff1744);
  }

  .trace-idx {
    color: var(--text-muted, #525b6c);
    font-weight: 700;
  }

  .sigmet-item {
    padding: 5px 8px;
    background: rgba(255, 23, 68, 0.09);
    border: 1px solid var(--color-nogo, #ff1744);
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
</style>
