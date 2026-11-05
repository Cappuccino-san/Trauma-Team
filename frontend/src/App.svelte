<script lang="ts">
  import { onMount } from 'svelte';
  import ConnectionHeader from './lib/components/ConnectionHeader.svelte';
  import VerdictBanner from './lib/components/VerdictBanner.svelte';
  import CorridorControlPanel from './lib/components/CorridorControlPanel.svelte';
  import StationWeatherGrid from './lib/components/StationWeatherGrid.svelte';
  import HazardAndTracePanel from './lib/components/HazardAndTracePanel.svelte';
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
  <VerdictBanner />

  <main class="workspace-grid">
    <div class="left-rail">
      <CorridorControlPanel />
    </div>

    <div class="right-main">
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

  .workspace-grid {
    flex: 1;
    display: grid;
    grid-template-columns: 330px 1fr;
    gap: 8px;
    padding: 8px;
    align-items: start;
  }

  .left-rail {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .right-main {
    display: flex;
    flex-direction: column;
    gap: 8px;
    min-width: 0;
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

  @media (max-width: 960px) {
    .workspace-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
