<script lang="ts">
  import { stationList, selectedStationId, selectedStation } from '../stores/weather';
  import { latestComplianceReport } from '../stores/verdict';
  import { wsRouter } from '../websocket';
  import type { StationComplianceReport, METAR } from '../types/envelope';

  let queryStationInput = '';

  $: complianceByStation = ($latestComplianceReport?.station_reports ?? []).reduce<
    Record<string, StationComplianceReport>
  >((acc, rep) => {
    acc[rep.station_id.toUpperCase()] = rep;
    return acc;
  }, {});

  function formatWind(m: METAR | null): string {
    if (!m) return '--';
    if (m.wind_speed_kts === 0) return 'CALM';
    const dir = m.is_variable_wind ? 'VRB' : String(Math.round(m.wind_dir_deg)).padStart(3, '0');
    const spd = String(Math.round(m.wind_speed_kts)).padStart(2, '0');
    const gst = m.wind_gust_kts > 0 ? `G${Math.round(m.wind_gust_kts)}` : '';
    return `${dir}°/${spd}${gst}KT`;
  }

  function formatVis(m: METAR | null): string {
    if (!m) return '--';
    const prefix = m.visibility_less_than ? '<' : m.visibility_greater_than ? '>' : '';
    return `${prefix}${m.visibility_sm.toFixed(1)} SM`;
  }

  function formatCeiling(m: METAR | null): string {
    if (!m) return '--';
    if (!m.has_ceiling || m.ceiling_agl_ft >= 20000) return 'UNL';
    return `${Math.round(m.ceiling_agl_ft)} FT`;
  }

  function selectStation(id: string) {
    selectedStationId.set(id);
  }

  function requestSingleStation() {
    const clean = queryStationInput.trim().toUpperCase();
    if (!clean) return;
    wsRouter.requestStationConditions(clean);
    selectedStationId.set(clean);
    queryStationInput = '';
  }
</script>

<section class="panel-box" aria-label="Station Weather Matrix">
  <div class="panel-header">
    <span>02 // CORRIDOR METEOROLOGICAL MATRIX (METAR / DA / § 135.609 MARGINS)</span>
    <div class="header-actions">
      <input
        type="text"
        placeholder="ICAO (e.g. KCOS)"
        maxlength="4"
        bind:value={queryStationInput}
        class="icao-quick-input"
      />
      <button type="button" on:click={requestSingleStation}>QUERY</button>
    </div>
  </div>

  <div class="table-wrap">
    <table class="wx-table">
      <thead>
        <tr>
          <th>ICAO</th>
          <th>CAT</th>
          <th>WIND</th>
          <th>VIS</th>
          <th>CEILING</th>
          <th>TMP/DEW</th>
          <th>ALT (INHG)</th>
          <th>DENSITY ALT</th>
          <th>§135.609 MARGIN</th>
          <th>STATUS</th>
        </tr>
      </thead>
      <tbody>
        {#each $stationList as st (st.station_id)}
          {@const m = st.metar}
          {@const comp = complianceByStation[st.station_id]}
          <tr
            class:selected={$selectedStationId === st.station_id}
            class:violated={comp && !comp.is_compliant}
            on:click={() => selectStation(st.station_id)}
          >
            <td class="col-station">{st.station_id}</td>
            <td>
              {#if m}
                <span class="cat-badge cat-{m.flight_category}">{m.flight_category}</span>
              {:else}
                --
              {/if}
            </td>
            <td>{formatWind(m)}</td>
            <td>{formatVis(m)}</td>
            <td>{formatCeiling(m)}</td>
            <td>
              {#if m}
                {m.temperature_c.toFixed(0)}°C / {m.dewpoint_c.toFixed(0)}°C
              {:else}
                --
              {/if}
            </td>
            <td>{m ? m.altimeter_inhg.toFixed(2) : '--'}</td>
            <td class:high-da={m && m.density_altitude_ft >= 8000}>
              {m ? `${Math.round(m.density_altitude_ft)} FT` : '--'}
            </td>
            <td>
              {#if comp}
                <span class:neg={comp.ceiling_margin_ft < 0} class:pos={comp.ceiling_margin_ft >= 0}>
                  {comp.ceiling_margin_ft >= 0 ? '+' : ''}{comp.ceiling_margin_ft.toFixed(0)} FT
                </span>
                /
                <span class:neg={comp.vis_margin_sm < 0} class:pos={comp.vis_margin_sm >= 0}>
                  {comp.vis_margin_sm >= 0 ? '+' : ''}{comp.vis_margin_sm.toFixed(1)} SM
                </span>
              {:else}
                <span class="muted">UNRATED</span>
              {/if}
            </td>
            <td>
              {#if comp}
                <span class="comp-flag" class:flag-pass={comp.is_compliant} class:flag-fail={!comp.is_compliant}>
                  {comp.is_compliant ? 'PASS' : 'FAIL'}
                </span>
              {:else}
                <span class="muted">--</span>
              {/if}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>

  {#if $selectedStation && $selectedStation.metar}
    <div class="raw-metar-strip">
      <div class="raw-top">
        <span class="raw-label">RAW METAR // {$selectedStation.station_id} (SEQ #{$selectedStation.seq})</span>
        <span class="raw-time">OBS: {$selectedStation.metar.observation_time_utc}</span>
      </div>
      <code class="raw-code">{$selectedStation.metar.raw_text}</code>
      {#if complianceByStation[$selectedStation.station_id]}
        <div class="station-exp">
          {complianceByStation[$selectedStation.station_id].explanation}
        </div>
      {/if}
    </div>
  {/if}
</section>

<style>
  .panel-box {
    background: var(--bg-panel);
    border: 1px solid var(--border-hairline);
    display: flex;
    flex-direction: column;
  }

  .header-actions {
    display: flex;
    gap: 4px;
    align-items: center;
  }

  .icao-quick-input {
    width: 110px;
    padding: 2px 6px;
    font-size: 10px;
    text-transform: uppercase;
  }

  .table-wrap {
    overflow-x: auto;
    max-height: 155px;
    overflow-y: auto;
  }

  .wx-table {
    width: 100%;
    border-collapse: collapse;
    text-align: left;
    font-size: 11px;
  }

  .wx-table th {
    padding: 6px 8px;
    background: var(--bg-void);
    color: var(--text-muted);
    font-size: 9px;
    font-weight: 700;
    letter-spacing: 0.08em;
    border-bottom: 1px solid var(--border-strong);
    white-space: nowrap;
  }

  .wx-table td {
    padding: 7px 8px;
    border-bottom: 1px solid var(--border-hairline);
    white-space: nowrap;
  }

  .wx-table tbody tr {
    cursor: pointer;
    transition: background-color 60ms linear;
  }

  .wx-table tbody tr:hover {
    background: var(--bg-elevated);
  }

  .wx-table tbody tr.selected {
    background: var(--bg-active);
  }

  .wx-table tbody tr.violated {
    background: var(--color-nogo-bg);
  }

  .col-station {
    font-weight: 800;
    letter-spacing: 0.05em;
  }

  .high-da {
    color: var(--color-caution);
    font-weight: 700;
  }

  .pos {
    color: var(--color-go);
  }

  .neg {
    color: var(--color-nogo);
    font-weight: 700;
  }

  .muted {
    color: var(--text-muted);
  }

  .comp-flag {
    padding: 1px 5px;
    font-size: 10px;
    font-weight: 800;
  }

  .flag-pass {
    color: var(--color-go);
    border: 1px solid var(--color-go);
  }

  .flag-fail {
    color: var(--color-nogo);
    border: 1px solid var(--color-nogo);
  }

  .raw-metar-strip {
    padding: 8px 10px;
    background: var(--bg-void);
    border-top: 1px solid var(--border-strong);
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .raw-top {
    display: flex;
    justify-content: space-between;
    font-size: 9px;
    color: var(--text-secondary);
    letter-spacing: 0.08em;
  }

  .raw-code {
    font-family: var(--font-mono);
    font-size: 11px;
    color: var(--text-primary);
    word-break: break-all;
  }

  .station-exp {
    font-size: 10px;
    color: var(--text-secondary);
  }
</style>
