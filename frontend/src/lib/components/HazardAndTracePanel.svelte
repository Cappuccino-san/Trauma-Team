<script lang="ts">
  import { verdictStore, hazardStore, activeCorridor, regulatoryContext } from '$lib/stores/verdict';
  import { websocketLogStore, wsRouter, connectionState } from '$lib/websocket';
  import { loadSimulationScenario } from '$lib/simulation/fixtures';

  // Track the active tab state natively
  let activeTab: 'audit' | 'sigmets' | 'envelopes' | 'console' = 'audit';

  let cmdInput = '';
  let consoleHistory: { ts: string; type: 'cmd' | 'out' | 'err'; text: string }[] = [
    {
      ts: new Date().toISOString().slice(11, 19),
      type: 'out',
      text: 'TRAUMATEAM HAA-135.L DISPATCH CONSOLE READY. Type "help" for commands (eval, sub, get, ping, preset, clear).'
    }
  ];

  function pushConsole(type: 'cmd' | 'out' | 'err', text: string) {
    consoleHistory = [
      ...consoleHistory,
      { ts: new Date().toISOString().slice(11, 19), type, text }
    ];
  }

  function executeConsoleCommand() {
    const raw = cmdInput.trim();
    if (!raw) return;
    pushConsole('cmd', `> ${raw}`);
    cmdInput = '';

    const parts = raw.split(/\s+/);
    const cmd = parts[0].toLowerCase();

    if (cmd === 'help') {
      pushConsole(
        'out',
        'Commands: eval [DEP] [INT] [DEST] | get <ICAO> | sub <ICAO...> | preset <go|highda|marginal|nogo|ifr|sigmet> | ping | clear'
      );
    } else if (cmd === 'clear') {
      consoleHistory = [];
    } else if (cmd === 'ping') {
      const id = wsRouter.sendPing();
      pushConsole('out', `Dispatched PING frame (${id})`);
    } else if (cmd === 'get' && parts[1]) {
      const st = parts[1].toUpperCase();
      const id = wsRouter.requestStationConditions(st);
      pushConsole('out', `Requested METAR/TAF conditions for ${st} (${id})`);
    } else if (cmd === 'sub' && parts.length > 1) {
      const sts = parts.slice(1).map((s) => s.toUpperCase());
      const id = wsRouter.subscribe(sts, ['minimums', 'hazards']);
      pushConsole('out', `Subscribed to stations [${sts.join(', ')}] (${id})`);
    } else if (cmd === 'eval') {
      if (parts.length >= 3) {
        const dep = parts[1].toUpperCase();
        const dest = parts[parts.length - 1].toUpperCase();
        const inters = parts.slice(2, -1).map((s) => s.toUpperCase());
        activeCorridor.set({ departure: dep, intermediates: inters, destination: dest });
      }
      const id = wsRouter.requestMinimumsEvaluation($activeCorridor, $regulatoryContext);
      if ($connectionState !== 'CONNECTED') {
        const dep = $activeCorridor.departure.trim().toUpperCase();
        const dest = $activeCorridor.destination.trim().toUpperCase();
        const inters = ($activeCorridor.intermediates || []).map((s) => s.trim().toUpperCase());
        if (dep === 'KBOS' || dest === 'KPWM' || inters.includes('KMHT')) {
          loadSimulationScenario(wsRouter, 'NEW_ENGLAND_SIGMET');
        } else if (dest === 'KASE') {
          if ($regulatoryContext.is_ifr_certified && $regulatoryContext.has_ifr_alternate) {
            loadSimulationScenario(wsRouter, 'IFR_TRANSITION_REQUIRED');
          } else {
            loadSimulationScenario(wsRouter, 'ASPEN_BELOW_MINIMUMS');
          }
        } else if (!$regulatoryContext.is_day && $regulatoryContext.equipment.nvg_equipped) {
          loadSimulationScenario(wsRouter, 'MARGINAL_CAUTION');
        } else {
          loadSimulationScenario(wsRouter, 'NOMINAL_GO', {
            departure: dep,
            intermediates: inters,
            destination: dest
          });
        }
      }
      pushConsole(
        'out',
        `Evaluated corridor ${$activeCorridor.departure} -> ${($activeCorridor.intermediates || []).join(',')} -> ${$activeCorridor.destination} (${id})`
      );
    } else if (cmd === 'preset' && parts[1]) {
      const p = parts[1].toLowerCase();
      const map: Record<string, any> = {
        go: 'NOMINAL_GO',
        highda: 'HIGH_DA_KLXV',
        marginal: 'MARGINAL_CAUTION',
        nogo: 'ASPEN_BELOW_MINIMUMS',
        ifr: 'IFR_TRANSITION_REQUIRED',
        sigmet: 'NEW_ENGLAND_SIGMET'
      };
      if (map[p]) {
        loadSimulationScenario(wsRouter, map[p]);
        pushConsole('out', `Loaded simulation scenario: ${map[p]}`);
      } else {
        pushConsole('err', `Unknown preset "${p}". Use: go, highda, marginal, nogo, ifr, sigmet`);
      }
    } else {
      try {
        const parsed = JSON.parse(raw);
        wsRouter.sendEnvelope(parsed);
        pushConsole('out', `Sent raw JSON envelope: ${parsed.action || 'CUSTOM'}`);
      } catch {
        pushConsole('err', `Unrecognized command "${cmd}". Type "help" for available commands.`);
      }
    }
  }
</script>

<section class="panel-container flex flex-col" aria-label="Audit Trace and SIGMET Console">
  <!-- Tab Navigation Header -->
  <div class="panel-header-bar">
    <span class="panel-title">05 // DETERMINISTIC AUDIT TRACE & TELEMETRY CONSOLE</span>
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
      <button
        type="button"
        class="px-2.5 py-0.5 font-mono text-[10px] uppercase {activeTab === 'console'
          ? 'bg-emerald-900/60 text-emerald-300 border-emerald-500'
          : 'text-emerald-400 border-emerald-800 hover:text-emerald-200'}"
        on:click={() => (activeTab = 'console')}
      >
        CONSOLE
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
    {:else if activeTab === 'console'}
      <!-- Interactive Dispatch & Telemetry CLI Console -->
      <div class="flex flex-col h-full justify-between gap-1.5">
        <div class="flex-1 overflow-y-auto space-y-1 pr-1">
          {#each consoleHistory as entry}
            <div
              class="text-[11px] font-mono {entry.type === 'cmd'
                ? 'text-cyan-300 font-bold'
                : entry.type === 'err'
                  ? 'text-red-400'
                  : 'text-emerald-300'}"
            >
              <span class="text-gray-500 mr-1.5">[{entry.ts}]</span>{entry.text}
            </div>
          {/each}
        </div>
        <form class="flex gap-1.5 pt-1 border-t border-gray-800" on:submit|preventDefault={executeConsoleCommand}>
          <span class="text-emerald-400 font-bold self-center text-xs">$</span>
          <input
            type="text"
            bind:value={cmdInput}
            placeholder="Type command (e.g. eval KBOS KMHT KPWM, get KDEN, preset sigmet, ping, help)..."
            class="flex-1 bg-black text-white border border-gray-700 px-2 py-1 text-xs font-mono"
          />
          <button type="submit" class="px-3 py-1 bg-emerald-950 text-emerald-300 border border-emerald-700 text-[10px]">
            EXEC
          </button>
        </form>
      </div>
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
