<script lang="ts">
  import { latestComplianceReport, currentVerdictTier, isEvaluating } from '../stores/verdict';
  import { formatLimitingFactorLabel } from '../aviationTerminology';

  function formatMargin(val: number | undefined, unit: string): string {
    if (val === undefined || val === null || Number.isNaN(val)) return '--';
    const sign = val >= 0 ? '+' : '';
    return `${sign}${val.toFixed(unit === 'SM' ? 1 : 0)} ${unit}`;
  }
</script>

<section class="verdict-banner tier-{$currentVerdictTier}" aria-label="Primary Dispatch Verdict">
  <div class="verdict-top">
    <div class="verdict-badge-col">
      <div class="annunciator-label">DISPATCH FEASIBILITY VERDICT</div>
      <div class="verdict-Readout" data-testid="verdict-readout">
        {#if $isEvaluating}
          EVALUATING...
        {:else if $latestComplianceReport}
          {$latestComplianceReport.verdict}
        {:else}
          STANDBY
        {/if}
      </div>
      <div class="tier-sub">
        TIER: {$currentVerdictTier} //
        {$latestComplianceReport
          ? $latestComplianceReport.is_compliant
            ? 'COMPLIANT'
            : 'DISPATCH PROHIBITED'
          : 'AWAITING EVALUATION'}
      </div>
    </div>

    {#if $latestComplianceReport}
      <div class="metrics-matrix">
        <div class="matrix-cell">
          <span class="cell-label">LIMITING FACTOR</span>
          <span class="cell-value factor-callout" data-testid="limiting-factor-label">
            {formatLimitingFactorLabel($latestComplianceReport.limiting_factor)}
          </span>
          <span class="factor-code-tag" data-testid="limiting-factor">
            {$latestComplianceReport.limiting_factor}
          </span>
        </div>

        <div class="matrix-cell">
          <span class="cell-label">WORST CEILING vs REQ</span>
          <span class="cell-value">
            {$latestComplianceReport.worst_ceiling_ft.toFixed(0)} FT /
            {$latestComplianceReport.min_required_ceiling_ft.toFixed(0)} FT
          </span>
          <span
            class="cell-delta"
            class:neg={$latestComplianceReport.ceiling_margin_ft < 0}
            class:pos={$latestComplianceReport.ceiling_margin_ft >= 0}
          >
            ({formatMargin($latestComplianceReport.ceiling_margin_ft, 'FT')})
          </span>
        </div>

        <div class="matrix-cell">
          <span class="cell-label">WORST VISIBILITY vs REQ</span>
          <span class="cell-value">
            {$latestComplianceReport.worst_visibility_sm.toFixed(1)} SM /
            {$latestComplianceReport.min_required_vis_sm.toFixed(1)} SM
          </span>
          <span
            class="cell-delta"
            class:neg={$latestComplianceReport.vis_margin_sm < 0}
            class:pos={$latestComplianceReport.vis_margin_sm >= 0}
          >
            ({formatMargin($latestComplianceReport.vis_margin_sm, 'SM')})
          </span>
        </div>

        <div class="matrix-cell">
          <span class="cell-label">APPLIED REGULATORY BASIS</span>
          <span class="cell-value rule-text">
            {$latestComplianceReport.legal_rule_applied}
          </span>
        </div>
      </div>
    {/if}
  </div>

  {#if $latestComplianceReport}
    <div class="verdict-explanation">
      <span class="exp-prefix">RATIONALE //</span>
      {$latestComplianceReport.explanation}
    </div>
  {/if}
</section>

<style>
  .verdict-banner {
    background: var(--bg-panel);
    border-bottom: 1px solid var(--border-strong);
    border-left: 4px solid var(--text-muted);
    padding: 10px 14px;
  }

  .tier-GO {
    border-left-color: var(--color-go);
    background: linear-gradient(90deg, var(--color-go-bg) 0%, var(--bg-panel) 35%);
  }

  .tier-CAUTION {
    border-left-color: var(--color-caution);
    background: linear-gradient(90deg, var(--color-caution-bg) 0%, var(--bg-panel) 35%);
  }

  .tier-NO-GO {
    border-left-color: var(--color-nogo);
    background: linear-gradient(90deg, var(--color-nogo-bg) 0%, var(--bg-panel) 35%);
  }

  .tier-IFR {
    border-left-color: var(--color-ifr);
    background: linear-gradient(90deg, var(--color-ifr-bg) 0%, var(--bg-panel) 35%);
  }

  .verdict-top {
    display: flex;
    flex-wrap: wrap;
    justify-content: space-between;
    align-items: stretch;
    gap: 16px;
  }

  .verdict-badge-col {
    min-width: 220px;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }

  .annunciator-label {
    font-size: 9px;
    letter-spacing: 0.12em;
    color: var(--text-secondary);
    text-transform: uppercase;
  }

  .verdict-Readout {
    font-size: 22px;
    font-weight: 800;
    letter-spacing: 0.04em;
    line-height: 1.15;
    margin: 2px 0;
  }

  .tier-GO .verdict-Readout {
    color: var(--color-go);
  }
  .tier-CAUTION .verdict-Readout {
    color: var(--color-caution);
  }
  .tier-NO-GO .verdict-Readout {
    color: var(--color-nogo);
  }
  .tier-IFR .verdict-Readout {
    color: var(--color-ifr);
  }

  .tier-sub {
    font-size: 10px;
    color: var(--text-secondary);
  }

  .metrics-matrix {
    flex: 1;
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: 1px;
    background: var(--border-hairline);
    border: 1px solid var(--border-hairline);
  }

  .matrix-cell {
    background: var(--bg-panel);
    padding: 6px 10px;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }

  .cell-label {
    font-size: 9px;
    color: var(--text-muted);
    letter-spacing: 0.08em;
    margin-bottom: 2px;
  }

  .cell-value {
    font-size: 12px;
    font-weight: 700;
    color: var(--text-primary);
  }

  .factor-callout {
    font-size: 11px;
    font-weight: 800;
    line-height: 1.25;
  }

  .factor-code-tag {
    margin-top: 3px;
    display: inline-block;
    width: fit-content;
    font-size: 9px;
    color: var(--text-muted);
    background: var(--bg-void);
    border: 1px solid var(--border-hairline);
    padding: 1px 4px;
    letter-spacing: 0.04em;
    word-break: break-all;
  }

  .rule-text {
    font-size: 11px;
    font-weight: 600;
  }

  .cell-delta {
    font-size: 11px;
    font-weight: 700;
  }

  .cell-delta.pos {
    color: var(--color-go);
  }

  .cell-delta.neg {
    color: var(--color-nogo);
  }

  .verdict-explanation {
    margin-top: 8px;
    padding-top: 6px;
    border-top: 1px solid var(--border-hairline);
    font-size: 11px;
    color: var(--text-primary);
  }

  .exp-prefix {
    color: var(--text-secondary);
    font-weight: 700;
    margin-right: 6px;
  }
</style>
