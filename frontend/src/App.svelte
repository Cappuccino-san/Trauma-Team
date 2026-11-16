<script lang="ts">
  import { onMount } from 'svelte';
  import ConnectionHeader from './lib/components/ConnectionHeader.svelte';
  import VerdictBanner from './lib/components/VerdictBanner.svelte';
  import CorridorControlPanel from './lib/components/CorridorControlPanel.svelte';
  import StationWeatherGrid from './lib/components/StationWeatherGrid.svelte';
  import HazardAndTracePanel from './lib/components/HazardAndTracePanel.svelte';
  import RouteMap from './lib/components/RouteMap.svelte';
  import CgEnvelope from './lib/components/CgEnvelope.svelte';
  import CustomPresetModal from './lib/components/CustomPresetModal.svelte';
  import { wsRouter } from './lib/websocket';
  import { loadSimulationScenario } from './lib/simulation/fixtures';

  let showCustomModal = false;
  let activeView: 'all' | 'flight-config' | 'map-wx' | 'audit-console' = 'all';

  onMount(() => {
    // Load initial deterministic NOAA AWC & FAA 14 CFR Part 135 Subpart L simulation state
    loadSimulationScenario(wsRouter, 'NOMINAL_GO');
    // Connect to the live Go WebSocket Gateway (:8080/ws) unless hosted on static GitHub Pages
    if (typeof window !== 'undefined' && !window.location.hostname.endsWith('.github.io')) {
      wsRouter.connect();
    }
  });
</script>

<div class="console-shell">
  <ConnectionHeader onOpenCustomPreset={() => (showCustomModal = true)} />
  <CustomPresetModal isOpen={showCustomModal} onClose={() => (showCustomModal = false)} />

  <!-- Top Workspace Navigation & Panel Index Bar -->
  <nav class="workspace-nav" aria-label="Workspace Panel Focus">
    <div class="nav-left">
      <span class="nav-label">WORKSPACE LAYOUT:</span>
      <button
        type="button"
        class="nav-pill"
        class:active={activeView === 'all'}
        on:click={() => (activeView = 'all')}
      >
        ALL MODULES (01–05)
      </button>
      <button
        type="button"
        class="nav-pill"
        class:active={activeView === 'flight-config'}
        on:click={() => (activeView = 'flight-config')}
      >
        01 // CORRIDOR &amp; 02 // MASS / CG
      </button>
      <button
        type="button"
        class="nav-pill"
        class:active={activeView === 'map-wx'}
        on:click={() => (activeView = 'map-wx')}
      >
        03 // GEODESIC MAP &amp; 04 // METAR MATRIX
      </button>
      <button
        type="button"
        class="nav-pill nav-pill-console"
        class:active={activeView === 'audit-console'}
        on:click={() => (activeView = 'audit-console')}
      >
        05 // AUDIT TRACE &amp; CLI CONSOLE
      </button>
    </div>
    <div class="nav-right">
      <span class="flow-hint">FLOW: [01 CORRIDOR] → [02 MASS/CG] → [03 MAP] → [04 METAR] → [05 CONSOLE]</span>
    </div>
  </nav>

  <!-- Primary Dispatch Feasibility Verdict Banner (Full-Width Command Strip) -->
  <div class="verdict-strip">
    <VerdictBanner />
  </div>

  <!-- Structured 12-Column Avionics Grid (Sequential 01 -> 05) -->
  <main class="grid grid-cols-1 lg:grid-cols-12 gap-2.5 px-2.5 pb-2.5 flex-1 items-start">
    {#if activeView === 'all'}
      <!-- Left Column (col-span-3): 01 Corridor Matrix & 02 Mass/CG Envelope -->
      <div class="col-span-1 lg:col-span-3 flex flex-col gap-2.5">
        <CorridorControlPanel />
        <CgEnvelope />
      </div>

      <!-- Right Column (col-span-9): 03 RouteMap, 04 StationWeatherGrid, 05 HazardAndTracePanel -->
      <div class="col-span-1 lg:col-span-9 flex flex-col gap-2.5">
        <RouteMap />
        <StationWeatherGrid />
        <HazardAndTracePanel />
      </div>
    {:else if activeView === 'flight-config'}
      <div class="col-span-1 lg:col-span-6 flex flex-col gap-2.5">
        <CorridorControlPanel />
      </div>
      <div class="col-span-1 lg:col-span-6 flex flex-col gap-2.5">
        <CgEnvelope />
      </div>
    {:else if activeView === 'map-wx'}
      <div class="col-span-1 lg:col-span-12 flex flex-col gap-2.5">
        <RouteMap />
        <StationWeatherGrid />
      </div>
    {:else if activeView === 'audit-console'}
      <div class="col-span-1 lg:col-span-12 flex flex-col gap-2.5">
        <HazardAndTracePanel />
        <StationWeatherGrid />
      </div>
    {/if}
  </main>

  <footer class="console-footer">
    <span>REGULATORY ENGINE: 14 CFR PART 135 SUBPART L (§§ 135.609 / 135.611 / 135.615)</span>
    <span>FEED: NOAA AWC METAR / TAF / FD WINDS / SIGMET</span>
    <span>PROTOCOL: GO SERVER_ENVELOPE JSON v1</span>
  </footer>
</div>

<style>
  .console-shell {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    background: var(--bg-void);
  }

  .workspace-nav {
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: 8px;
    padding: 5px 10px;
    background: #0a0d13;
    border-bottom: 1px solid var(--border-hairline);
  }

  .nav-left {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 6px;
  }

  .nav-label {
    font-size: 9px;
    font-weight: 700;
    color: var(--text-muted);
    letter-spacing: 0.08em;
    margin-right: 2px;
  }

  .nav-pill {
    font-size: 9px;
    padding: 3px 8px;
    border: 1px solid var(--border-strong);
    background: var(--bg-panel);
    color: var(--text-secondary);
    letter-spacing: 0.06em;
  }

  .nav-pill:hover {
    color: var(--text-primary);
    border-color: var(--text-secondary);
  }

  .nav-pill.active {
    background: rgba(0, 229, 255, 0.12);
    color: var(--color-info);
    border-color: var(--color-info);
    font-weight: 800;
  }

  .nav-pill-console.active {
    background: rgba(0, 230, 118, 0.15);
    color: var(--color-go);
    border-color: var(--color-go);
  }

  .flow-hint {
    font-size: 9px;
    color: var(--text-muted);
    letter-spacing: 0.06em;
  }

  .verdict-strip {
    padding: 8px 10px 6px 10px;
  }

  .console-footer {
    display: flex;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 8px;
    padding: 5px 12px;
    background: var(--bg-panel);
    border-top: 1px solid var(--border-hairline);
    font-size: 9px;
    letter-spacing: 0.08em;
    color: var(--text-muted);
  }
</style>
