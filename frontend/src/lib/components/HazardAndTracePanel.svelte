<script lang="ts">
  import { activeHazards, activeHazardCount } from '../stores/weather';
  import { latestComplianceReport } from '../stores/verdict';
  import { envelopeStreamLog } from '../websocket';

  type ActiveTab = 'TRACE' | 'HAZARDS' | 'WIRE';
  let currentTab: ActiveTab = 'TRACE';
</script>

<section class="panel-box" aria-label="Audit Trace, SIGMET Advisories, and Envelope Wire Log">
  <div class="panel-header">
    <span>03 // DETERMINISTIC AUDIT TRACE & SIGMET STREAM</span>
    <div class="tab-buttons">
      <button
        type="button"
        class:active-tab={currentTab === 'TRACE'}
        on:click={() => (currentTab = 'TRACE')}
      >
        AUDIT TRACE ({$latestComplianceReport?.audit_trace.length ?? 0})
      </button>
      <button
        type="button"
        class:active-tab={currentTab === 'HAZARDS'}
        class:hazard-warn={$activeHazardCount > 0}
        on:click={() => (currentTab = 'HAZARDS')}
      >
        SIGMETS ({$activeHazardCount})
      </button>
      <button
        type="button"
        class:active-tab={currentTab === 'WIRE'}
        on:click={() => (currentTab = 'WIRE')}
      >
        WS ENVELOPES ({$envelopeStreamLog.length})
      </button>
    </div>
  </div>

  <div class="panel-content">
    {#if currentTab === 'TRACE'}
      <div class="trace-list" data-testid="audit-trace-list">
        {#if $latestComplianceReport && $latestComplianceReport.audit_trace.length > 0}
          {#each $latestComplianceReport.audit_trace as line, idx}
            <div
              class="trace-line"
              class:line-fail={line.includes('FAIL') || line.includes('BELOW_MINIMUMS')}
              class:line-pass={line.includes('PASS') || line.includes('APPROVED')}
            >
              <span class="step-idx">{String(idx + 1).padStart(2, '0')}</span>
              <span class="step-text">{line}</span>
            </div>
          {/each}
        {:else}
          <div class="empty-msg">NO AUDIT TRACE RECORDED</div>
        {/if}
      </div>
    {:else if currentTab === 'HAZARDS'}
      <div class="hazard-list">
        {#if $activeHazards.length > 0}
          {#each $activeHazards as h (h.hazard_id || h.advisory_id)}
            <div class="hazard-card">
              <div class="hazard-top">
                <span class="haz-id">{h.advisory_id}</span>
                <span class="haz-sev">{h.hazard_type} // {h.severity}</span>
                <span class="haz-alt">
                  FL{Math.round(h.min_alt_ft / 100).toString().padStart(3, '0')} -
                  FL{Math.round(h.max_alt_ft / 100).toString().padStart(3, '0')}
                </span>
              </div>
              <div class="haz-summary">{h.summary}</div>
              <code class="haz-raw">{h.raw_text}</code>
            </div>
          {/each}
        {:else}
          <div class="empty-msg">NO ACTIVE SIGMET OR CONVECTIVE ADVISORIES IN CORRIDOR</div>
        {/if}
      </div>
    {:else}
      <div class="wire-list">
        {#each $envelopeStreamLog as item, i (i)}
          <div class="wire-row">
            <span class="wire-dir dir-{item.direction}">{item.direction}</span>
            <span class="wire-seq">{item.seq ? `#${item.seq}` : '--'}</span>
            <span class="wire-evt">{item.eventOrAction}</span>
            <span class="wire-sum">{item.summary}</span>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</section>

<style>
  .panel-box {
    background: var(--bg-panel);
    border: 1px solid var(--border-hairline);
    display: flex;
    flex-direction: column;
  }

  .tab-buttons {
    display: flex;
    gap: 4px;
  }

  .tab-buttons button {
    font-size: 9px;
    padding: 2px 6px;
  }

  .tab-buttons button.active-tab {
    background: var(--text-primary);
    color: var(--bg-void);
    border-color: var(--text-primary);
  }

  .tab-buttons button.hazard-warn {
    border-color: var(--color-nogo);
    color: var(--color-nogo);
  }

  .panel-content {
    padding: 8px 10px;
    max-height: 250px;
    overflow-y: auto;
  }

  .trace-list,
  .hazard-list,
  .wire-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .trace-line {
    display: flex;
    gap: 8px;
    font-size: 11px;
    padding: 3px 6px;
    background: var(--bg-void);
    border-left: 2px solid var(--border-strong);
  }

  .trace-line.line-pass {
    border-left-color: var(--color-go);
  }

  .trace-line.line-fail {
    border-left-color: var(--color-nogo);
    color: var(--color-nogo);
  }

  .step-idx {
    color: var(--text-muted);
    font-weight: 700;
  }

  .hazard-card {
    padding: 6px 8px;
    background: var(--color-nogo-bg);
    border: 1px solid var(--color-nogo);
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  .hazard-top {
    display: flex;
    justify-content: space-between;
    font-size: 10px;
    font-weight: 800;
    color: var(--color-nogo);
  }

  .haz-summary {
    font-size: 11px;
    color: var(--text-primary);
  }

  .haz-raw {
    font-size: 10px;
    color: var(--text-secondary);
  }

  .wire-row {
    display: grid;
    grid-template-columns: 36px 48px 155px 1fr;
    gap: 8px;
    font-size: 10px;
    padding: 2px 4px;
    border-bottom: 1px solid var(--border-hairline);
  }

  .wire-dir {
    font-weight: 800;
  }

  .dir-IN {
    color: var(--color-go);
  }

  .dir-OUT {
    color: var(--color-mvfr);
  }

  .wire-seq {
    color: var(--text-muted);
  }

  .wire-evt {
    font-weight: 700;
    color: var(--text-primary);
  }

  .wire-sum {
    color: var(--text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .empty-msg {
    color: var(--text-muted);
    font-size: 11px;
    padding: 8px 0;
  }
</style>
