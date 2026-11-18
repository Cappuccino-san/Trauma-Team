<script lang="ts">
  import { wsRouter } from '../websocket';
  import {
    ServerEvent,
    type ServerEnvelope,
    type METAR,
    type WeatherHazard,
    type ComplianceReport
  } from '../types/envelope';
  import { activeCorridor } from '../stores/verdict';
  import { activeHazards } from '../stores/weather';
  import { activeScenarioKey } from '../simulation/fixtures';

  export let isOpen = false;
  export let onClose: () => void = () => {};

  let activeTab: 'upload' | 'builder' = 'upload';
  let jsonInput = '';
  let statusMessage = '';
  let statusType: 'success' | 'error' = 'success';

  // Interactive Custom Scenario Builder state
  let customName = 'CUSTOM_MISSION';
  let depIcao = 'KDEN';
  let interIcao = 'KLXV';
  let destIcao = 'KASE';
  let verdict: 'APPROVED' | 'CAUTION' | 'BELOW_MINIMUMS' | 'DISQUALIFIED' = 'APPROVED';
  let limitingFactor = 'LIMITING_FACTOR_NONE';
  let ruleApplied = '14 CFR § 135.609 Table 1';
  let ceilingFt = 5000;
  let visSm = 10;
  let windKts = 12;
  let includeSigmet = false;
  let sigmetSummary = 'Severe Mountain Wave Turbulence FL180-FL320';

  const SAMPLE_SCENARIO_JSON = JSON.stringify(
    {
      name: "CUSTOM_ROCKY_FLIGHT",
      corridor: {
        departure: "KDEN",
        intermediates: ["KLXV"],
        destination: "KASE"
      },
      verdict: {
        verdict: "APPROVED",
        limiting_factor: "LIMITING_FACTOR_NONE",
        legal_rule_applied: "14 CFR § 135.609 Table 1 (Mountainous Day)",
        worst_ceiling_ft: 12000,
        worst_visibility_sm: 10,
        min_required_ceiling_ft: 800,
        min_required_vis_sm: 2.0,
        ceiling_margin_ft: 11200,
        vis_margin_sm: 8.0,
        explanation: "Custom simulation: All route corridors clear with valid RFM performance."
      },
      stations: [
        {
          station_id: "KDEN",
          flight_category: "VFR",
          wind_dir_deg: 270,
          wind_speed_kts: 10,
          wind_gust_kts: 0,
          visibility_sm: 10,
          ceiling_agl_ft: 12000,
          has_ceiling: true,
          temperature_c: 18,
          dewpoint_c: 4,
          altimeter_inhg: 30.05,
          raw_text: "KDEN 051200Z 27010KT 10SM BKN120 18/04 A3005"
        },
        {
          station_id: "KASE",
          flight_category: "VFR",
          wind_dir_deg: 320,
          wind_speed_kts: 8,
          wind_gust_kts: 0,
          visibility_sm: 10,
          ceiling_agl_ft: 9000,
          has_ceiling: true,
          temperature_c: 12,
          dewpoint_c: -2,
          altimeter_inhg: 29.85,
          raw_text: "KASE 051200Z 32008KT 10SM SCT090 12/M02 A2985"
        }
      ],
      hazards: []
    },
    null,
    2
  );

  function loadSampleJson() {
    jsonInput = SAMPLE_SCENARIO_JSON;
    statusMessage = 'Sample scenario JSON loaded into editor.';
    statusType = 'success';
  }

  function handleFileUpload(event: Event) {
    const input = event.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;
    const file = input.files[0];
    const reader = new FileReader();
    reader.onload = (e) => {
      const content = e.target?.result as string;
      jsonInput = content;
      statusMessage = `File loaded: ${file.name} (${content.length} bytes)`;
      statusType = 'success';
    };
    reader.readAsText(file);
  }

  function injectParsedScenario(data: any) {
    const now = new Date().toISOString();
    let seq = Date.now() % 100000;
    activeScenarioKey.set(null);

    // 1. Update corridor if provided
    if (data.corridor && data.corridor.departure && data.corridor.destination) {
      activeCorridor.set({
        departure: String(data.corridor.departure).toUpperCase(),
        intermediates: (data.corridor.intermediates || []).map((s: string) => String(s).toUpperCase()),
        destination: String(data.corridor.destination).toUpperCase()
      });
    }

    // 2. Ingest custom stations
    if (Array.isArray(data.stations)) {
      data.stations.forEach((st: any) => {
        seq += 1;
        const metar: METAR = {
          station_id: String(st.station_id || 'UNK').toUpperCase(),
          observation_time_utc: st.observation_time_utc || now,
          wind_dir_deg: st.wind_dir_deg ?? 180,
          wind_speed_kts: st.wind_speed_kts ?? 10,
          wind_gust_kts: st.wind_gust_kts ?? 0,
          is_variable_wind: false,
          visibility_sm: st.visibility_sm ?? 10,
          visibility_less_than: false,
          visibility_greater_than: false,
          ceiling_agl_ft: st.ceiling_agl_ft ?? 10000,
          has_ceiling: (st.ceiling_agl_ft ?? 10000) < 20000,
          cloud_layers: [{ coverage: 'BKN', base_ft_agl: st.ceiling_agl_ft ?? 10000, is_ceiling: true }],
          temperature_c: st.temperature_c ?? 15,
          dewpoint_c: st.dewpoint_c ?? 0,
          altimeter_inhg: st.altimeter_inhg ?? 29.92,
          present_weather: [],
          flight_category: st.flight_category || 'VFR',
          density_altitude_ft: 5000,
          pressure_altitude_ft: 5000,
          raw_text: st.raw_text || `${st.station_id} 000000Z AUTO`
        };
        const env: ServerEnvelope<METAR> = {
          event: ServerEvent.WEATHER_SNAPSHOT,
          timestamp_utc: now,
          seq,
          station_id: metar.station_id,
          payload: metar,
          metadata: { source: 'CUSTOM_SIM_PRESET', feed_type: 'METAR' }
        };
        wsRouter.routeEnvelope(env);
      });
    }

    // 3. Clear previous hazards and ingest new hazards if any
    activeHazards.set([]);
    if (Array.isArray(data.hazards)) {
      data.hazards.forEach((h: any) => {
        seq += 1;
        const hazard: WeatherHazard = {
          hazard_id: h.hazard_id || `HAZ-${seq}`,
          advisory_id: h.advisory_id || 'SIGMET CUSTOM',
          hazard_type: h.hazard_type || 'CONVECTIVE_SIGMET',
          severity: h.severity || 'SEVERE',
          polygon: h.polygon || [],
          min_alt_ft: h.min_alt_ft ?? 0,
          max_alt_ft: h.max_alt_ft ?? 45000,
          valid_from: now,
          valid_until: now,
          summary: h.summary || 'Custom active SIGMET',
          raw_text: h.raw_text || 'SIGMET CUSTOM ACTIVE',
          active: true
        };
        const hEnv: ServerEnvelope<WeatherHazard> = {
          event: ServerEvent.HAZARD_ALERT,
          timestamp_utc: now,
          seq,
          payload: hazard,
          metadata: { source: 'CUSTOM_SIM_PRESET' }
        };
        wsRouter.routeEnvelope(hEnv);
      });
    }

    // 4. Ingest verdict/compliance report
    const v = data.verdict || {};
    seq += 1;
    const rep: ComplianceReport = {
      is_compliant: v.is_compliant ?? (v.verdict === 'APPROVED' || v.verdict === 'GO'),
      verdict: v.verdict || 'APPROVED',
      limiting_factor: v.limiting_factor || 'LIMITING_FACTOR_NONE',
      worst_ceiling_ft: v.worst_ceiling_ft ?? 5000,
      worst_visibility_sm: v.worst_visibility_sm ?? 10,
      min_required_ceiling_ft: v.min_required_ceiling_ft ?? 800,
      min_required_vis_sm: v.min_required_vis_sm ?? 2.0,
      ceiling_margin_ft: v.ceiling_margin_ft ?? 4200,
      vis_margin_sm: v.vis_margin_sm ?? 8.0,
      legal_rule_applied: v.legal_rule_applied || '14 CFR § 135.609 Table 1',
      explanation: v.explanation || `Custom user scenario ${data.name || ''} applied to dispatch engine.`,
      station_reports: (data.stations || []).map((st: any) => ({
        station_id: String(st.station_id || 'UNK').toUpperCase(),
        is_compliant: true,
        actual_ceiling_ft: st.ceiling_agl_ft ?? 5000,
        required_ceiling_ft: 800,
        ceiling_margin_ft: (st.ceiling_agl_ft ?? 5000) - 800,
        actual_vis_sm: st.visibility_sm ?? 10,
        required_vis_sm: 2.0,
        vis_margin_sm: (st.visibility_sm ?? 10) - 2.0,
        category: st.flight_category || 'VFR',
        explanation: 'User preset condition'
      })),
      active_hazards: data.hazards || [],
      evaluation_time_utc: now,
      audit_trace: [
        `[CUSTOM PRESET] Loaded ${data.name || 'User Scenario'}`,
        `[ROUTE] ${data.corridor?.departure || depIcao} -> ${(data.corridor?.intermediates || [interIcao]).join(',')} -> ${data.corridor?.destination || destIcao}`,
        `[RULE] ${v.legal_rule_applied || '14 CFR § 135.609 Table 1'}`,
        `[FINAL VERDICT] ${v.verdict || 'APPROVED'} (${v.limiting_factor || 'LIMITING_FACTOR_NONE'})`
      ]
    };

    const vEnv: ServerEnvelope<ComplianceReport> = {
      event: ServerEvent.MINIMUMS_EVALUATION,
      request_id: `custom-sim-${Date.now()}`,
      timestamp_utc: now,
      seq,
      payload: rep,
      metadata: { evaluator: 'USER_CUSTOM_SIMULATION' }
    };
    wsRouter.routeEnvelope(vEnv);
  }

  function applyJsonPreset() {
    try {
      const parsed = JSON.parse(jsonInput);
      injectParsedScenario(parsed);
      statusMessage = `Successfully applied custom preset "${parsed.name || 'Custom'}"!`;
      statusType = 'success';
      setTimeout(() => {
        onClose();
      }, 700);
    } catch (err) {
      statusMessage = `JSON parse error: ${err instanceof Error ? err.message : String(err)}`;
      statusType = 'error';
    }
  }

  function applyVisualBuilder() {
    const customData = {
      name: customName,
      corridor: {
        departure: depIcao.toUpperCase(),
        intermediates: interIcao ? [interIcao.toUpperCase()] : [],
        destination: destIcao.toUpperCase()
      },
      verdict: {
        verdict,
        limiting_factor: limitingFactor,
        legal_rule_applied: ruleApplied,
        worst_ceiling_ft: ceilingFt,
        worst_visibility_sm: visSm,
        min_required_ceiling_ft: 800,
        min_required_vis_sm: 2.0,
        ceiling_margin_ft: ceilingFt - 800,
        vis_margin_sm: visSm - 2.0,
        explanation: `Custom scenario ${customName}: Ceiling ${ceilingFt}ft, Vis ${visSm}SM, Wind ${windKts}kt.`
      },
      stations: [
        {
          station_id: depIcao.toUpperCase(),
          flight_category: visSm >= 5 && ceilingFt >= 3000 ? 'VFR' : ceilingFt >= 1000 ? 'MVFR' : 'IFR',
          wind_dir_deg: 260,
          wind_speed_kts: windKts,
          wind_gust_kts: 0,
          visibility_sm: visSm,
          ceiling_agl_ft: ceilingFt,
          temperature_c: 15,
          dewpoint_c: 2,
          altimeter_inhg: 29.95,
          raw_text: `${depIcao.toUpperCase()} 000000Z 260${windKts}KT ${visSm}SM BKN${String(Math.round(ceilingFt/100)).padStart(3,'0')}`
        },
        {
          station_id: destIcao.toUpperCase(),
          flight_category: visSm >= 5 && ceilingFt >= 3000 ? 'VFR' : ceilingFt >= 1000 ? 'MVFR' : 'IFR',
          wind_dir_deg: 270,
          wind_speed_kts: windKts,
          wind_gust_kts: 0,
          visibility_sm: visSm,
          ceiling_agl_ft: ceilingFt,
          temperature_c: 12,
          dewpoint_c: 0,
          altimeter_inhg: 29.90,
          raw_text: `${destIcao.toUpperCase()} 000000Z 270${windKts}KT ${visSm}SM BKN${String(Math.round(ceilingFt/100)).padStart(3,'0')}`
        }
      ],
      hazards: includeSigmet
        ? [
            {
              hazard_id: 'SIG-CUSTOM-1',
              advisory_id: 'SIGMET 99E',
              hazard_type: 'CONVECTIVE_SIGMET',
              severity: 'SEVERE',
              summary: sigmetSummary,
              polygon: [
                { latitude: 39.0, longitude: -106.0 },
                { latitude: 40.0, longitude: -106.0 },
                { latitude: 40.0, longitude: -105.0 },
                { latitude: 39.0, longitude: -105.0 },
                { latitude: 39.0, longitude: -106.0 }
              ]
            }
          ]
        : []
    };

    injectParsedScenario(customData);
    statusMessage = `Applied custom scenario "${customName}"!`;
    statusType = 'success';
    setTimeout(() => {
      onClose();
    }, 600);
  }
</script>

{#if isOpen}
  <div
    class="modal-backdrop"
    role="presentation"
    on:click|self={onClose}
    on:keydown={(e) => {
      if (e.key === 'Escape') onClose();
    }}
  >
    <div class="modal-window" role="dialog" aria-modal="true" aria-label="Custom Simulation Preset Studio">
      <div class="modal-header">
        <div class="flex items-center gap-2">
          <span class="modal-tag">CONFIG // SIM PRESET STUDIO</span>
          <span class="text-xs text-gray-400 font-mono">Upload JSON or Build Live Scenario</span>
        </div>
        <button type="button" class="close-btn" on:click={onClose}>✕</button>
      </div>

      <!-- Mode Selector Tabs -->
      <div class="tab-strip">
        <button
          type="button"
          class="tab-btn {activeTab === 'upload' ? 'active' : ''}"
          on:click={() => (activeTab = 'upload')}
        >
          JSON Data Upload / Editor
        </button>
        <button
          type="button"
          class="tab-btn {activeTab === 'builder' ? 'active' : ''}"
          on:click={() => (activeTab = 'builder')}
        >
          Visual Scenario Builder
        </button>
      </div>

      <div class="modal-body">
        {#if activeTab === 'upload'}
          <div class="space-y-3">
            <div class="flex justify-between items-center text-xs">
              <span class="text-gray-300">Upload a `.json` file containing corridor, METARs, & hazards:</span>
              <div class="flex gap-2">
                <label class="action-link cursor-pointer">
                  [CHOOSE FILE]
                  <input type="file" accept=".json" class="hidden" on:change={handleFileUpload} />
                </label>
                <button type="button" class="action-link" on:click={loadSampleJson}>
                  [LOAD SAMPLE JSON]
                </button>
              </div>
            </div>

            <textarea
              bind:value={jsonInput}
              placeholder="Paste custom scenario JSON here, or click [LOAD SAMPLE JSON] above..."
              class="json-editor"
              rows="12"
            ></textarea>

            <div class="flex justify-between items-center pt-2">
              <div class="status-box {statusType}">
                {statusMessage}
              </div>
              <div class="flex gap-2">
                <button type="button" class="btn-sec" on:click={onClose}>CANCEL</button>
                <button type="button" class="btn-pri" on:click={applyJsonPreset}>
                  APPLY PRESET TO ENGINE
                </button>
              </div>
            </div>
          </div>
        {:else}
          <!-- Visual Builder Form -->
          <div class="space-y-3 text-xs">
            <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
              <div>
                <label for="preset-name-inp" class="field-lbl">Preset Name</label>
                <input id="preset-name-inp" type="text" bind:value={customName} class="field-inp" />
              </div>
              <div>
                <label for="preset-dep-inp" class="field-lbl">Departure (ICAO)</label>
                <input id="preset-dep-inp" type="text" bind:value={depIcao} maxlength="4" class="field-inp uppercase" />
              </div>
              <div>
                <label for="preset-inter-inp" class="field-lbl">Intermediate</label>
                <input id="preset-inter-inp" type="text" bind:value={interIcao} maxlength="4" class="field-inp uppercase" />
              </div>
              <div>
                <label for="preset-dest-inp" class="field-lbl">Destination (ICAO)</label>
                <input id="preset-dest-inp" type="text" bind:value={destIcao} maxlength="4" class="field-inp uppercase" />
              </div>
            </div>

            <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
              <div>
                <label for="preset-verdict-sel" class="field-lbl">Dispatch Verdict</label>
                <select id="preset-verdict-sel" bind:value={verdict} class="field-inp">
                  <option value="APPROVED">APPROVED (GO)</option>
                  <option value="CAUTION">CAUTION (MARGINAL)</option>
                  <option value="BELOW_MINIMUMS">BELOW_MINIMUMS (NO-GO)</option>
                  <option value="DISQUALIFIED">DISQUALIFIED (NO-GO)</option>
                </select>
              </div>
              <div>
                <label for="preset-limit-sel" class="field-lbl">Limiting Factor</label>
                <select id="preset-limit-sel" bind:value={limitingFactor} class="field-inp font-mono text-[10px]">
                  <option value="LIMITING_FACTOR_NONE">NONE</option>
                  <option value="ACTIVE_CONVECTIVE_SIGMET">ACTIVE_CONVECTIVE_SIGMET</option>
                  <option value="FAR_135_609_CEILING_BELOW_MINIMUMS">CEILING_BELOW_MINS</option>
                  <option value="FAR_135_609_VISIBILITY_BELOW_MINIMUMS">VIS_BELOW_MINS</option>
                  <option value="FAR_135_209_VFR_RESERVE_VIOLATION">FUEL_RESERVE_VIOLATION</option>
                </select>
              </div>
              <div>
                <label for="preset-ceil-rng" class="field-lbl">Ceiling (FT AGL): {ceilingFt}</label>
                <input id="preset-ceil-rng" type="range" min="300" max="25000" step="100" bind:value={ceilingFt} class="w-full accent-cyan-500" />
              </div>
              <div>
                <label for="preset-vis-rng" class="field-lbl">Visibility (SM): {visSm}</label>
                <input id="preset-vis-rng" type="range" min="0.5" max="10" step="0.5" bind:value={visSm} class="w-full accent-cyan-500" />
              </div>
            </div>

            <div class="p-2.5 bg-slate-900 border border-slate-800 rounded flex flex-col gap-2">
              <label class="flex items-center gap-2 cursor-pointer text-slate-200">
                <input type="checkbox" bind:checked={includeSigmet} class="accent-red-500" />
                <span class="font-bold text-red-400">Inject Severe Convective Weather Hazard (SIGMET Polygon)</span>
              </label>
              {#if includeSigmet}
                <input
                  type="text"
                  bind:value={sigmetSummary}
                  placeholder="SIGMET advisory summary..."
                  class="field-inp text-red-300"
                />
              {/if}
            </div>

            <div class="flex justify-between items-center pt-2">
              <div class="status-box {statusType}">
                {statusMessage}
              </div>
              <div class="flex gap-2">
                <button type="button" class="btn-sec" on:click={onClose}>CANCEL</button>
                <button type="button" class="btn-pri" on:click={applyVisualBuilder}>
                  GENERATE & APPLY PRESET
                </button>
              </div>
            </div>
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.75);
    backdrop-filter: blur(2px);
    z-index: 9999;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 16px;
  }

  .modal-window {
    background: var(--bg-panel, #0c0e12);
    border: 1px solid var(--border-strong, #2d3545);
    border-top: 2px solid var(--color-go, #00e676);
    width: 100%;
    max-width: 780px;
    box-shadow: 0 20px 40px rgba(0, 0, 0, 0.8);
    display: flex;
    flex-direction: column;
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 12px;
    background: var(--bg-elevated, #12151c);
    border-bottom: 1px solid var(--border-hairline, #1e232e);
  }

  .modal-tag {
    font-size: 11px;
    font-weight: 800;
    letter-spacing: 0.08em;
    color: var(--color-go, #00e676);
    font-family: var(--font-mono);
  }

  .close-btn {
    background: transparent;
    border: none;
    color: var(--text-muted);
    font-size: 14px;
    cursor: pointer;
    padding: 2px 6px;
  }

  .close-btn:hover {
    color: #fff;
    background: #222;
  }

  .tab-strip {
    display: flex;
    border-bottom: 1px solid var(--border-hairline, #1e232e);
    background: var(--bg-void, #07080a);
  }

  .tab-btn {
    padding: 6px 14px;
    font-size: 11px;
    font-family: var(--font-mono);
    text-transform: uppercase;
    color: var(--text-muted);
    border: none;
    border-bottom: 2px solid transparent;
    background: transparent;
  }

  .tab-btn.active {
    color: var(--text-primary);
    background: var(--bg-panel);
    border-bottom-color: var(--color-go);
  }

  .modal-body {
    padding: 14px;
  }

  .action-link {
    color: var(--color-go);
    font-family: var(--font-mono);
    background: transparent;
    border: none;
    font-size: 10px;
    text-decoration: underline;
  }

  .json-editor {
    width: 100%;
    background: var(--bg-void, #07080a);
    color: #a5d6a7;
    font-family: var(--font-mono);
    font-size: 11px;
    border: 1px solid var(--border-strong, #2d3545);
    padding: 8px;
    outline: none;
    resize: vertical;
  }

  .field-lbl {
    display: block;
    color: var(--text-muted);
    font-size: 10px;
    margin-bottom: 3px;
    text-transform: uppercase;
  }

  .field-inp {
    width: 100%;
    background: var(--bg-void);
    color: #fff;
    border: 1px solid var(--border-strong);
    padding: 4px 6px;
    font-size: 11px;
    font-family: var(--font-mono);
  }

  .btn-pri {
    background: var(--color-go);
    color: #000;
    font-weight: 800;
    padding: 5px 12px;
    border: none;
    cursor: pointer;
  }

  .btn-sec {
    background: var(--bg-elevated);
    color: var(--text-secondary);
    border: 1px solid var(--border-strong);
    padding: 5px 12px;
    cursor: pointer;
  }

  .status-box {
    font-size: 11px;
    font-family: var(--font-mono);
  }

  .status-box.success {
    color: var(--color-go);
  }

  .status-box.error {
    color: var(--color-nogo);
  }
</style>
