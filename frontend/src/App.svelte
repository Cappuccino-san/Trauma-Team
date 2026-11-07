<script lang="ts">
  import { onMount } from 'svelte';
  import ConnectionHeader from './lib/components/ConnectionHeader.svelte';
  import VerdictBanner from './lib/components/VerdictBanner.svelte';
  import CorridorControlPanel from './lib/components/CorridorControlPanel.svelte';
  import StationWeatherGrid from './lib/components/StationWeatherGrid.svelte';
  import HazardAndTracePanel from './lib/components/HazardAndTracePanel.svelte';
  import RouteMap from './lib/components/RouteMap.svelte';
  import CgEnvelope from './lib/components/CgEnvelope.svelte';
  import { wsRouter } from './lib/websocket';
  import { loadSimulationScenario } from './lib/simulation/fixtures';

  onMount(() => {
    // Load initial deterministic NOAA AWC & FAA 14 CFR Part 135 Subpart L simulation state
    loadSimulationScenario(wsRouter, 'NOMINAL_GO');
    // Connect to the live Go WebSocket Gateway (:8080/ws)
    wsRouter.connect();
  });
</script>

<div class="console-shell">
  <ConnectionHeader />

  <main class="grid grid-cols-1 lg:grid-cols-12 gap-2.5 p-2.5 flex-1 items-start">
    <!-- Left Column: CorridorControlPanel & CgEnvelope (col-span-3) -->
    <div class="col-span-1 lg:col-span-3 col-span-3 lg:row-span-2 flex flex-col gap-2.5">
      <CorridorControlPanel />
      <CgEnvelope />
    </div>

    <!-- Top-Right Wide Section: VerdictBanner & RouteMap (col-span-9) -->
    <div class="col-span-1 lg:col-span-9 col-span-9 flex flex-col gap-2.5">
      <VerdictBanner />
      <RouteMap />
    </div>

    <!-- Bottom-Right Section: StationWeatherGrid & HazardAndTracePanel (col-span-9) -->
    <div class="col-span-1 lg:col-span-9 col-span-9 flex flex-col gap-2.5">
      <StationWeatherGrid />
      <HazardAndTracePanel />
    </div>
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
