import { describe, it, expect, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { get } from 'svelte/store';
import VerdictBanner from '../src/lib/components/VerdictBanner.svelte';
import MissionLegStrip from '../src/lib/components/MissionLegStrip.svelte';
import CorridorControlPanel from '../src/lib/components/CorridorControlPanel.svelte';
import CgEnvelope from '../src/lib/components/CgEnvelope.svelte';
import StationWeatherGrid from '../src/lib/components/StationWeatherGrid.svelte';
import HazardAndTracePanel from '../src/lib/components/HazardAndTracePanel.svelte';
import CustomPresetModal from '../src/lib/components/CustomPresetModal.svelte';
import ConnectionHeader from '../src/lib/components/ConnectionHeader.svelte';
import { resetRouterStores, wsRouter, connectionState } from '../src/lib/websocket';
import { resetWeatherStore, activeHazardCount } from '../src/lib/stores/weather';
import {
  resetVerdictStore,
  latestComplianceReport,
  currentVerdictTier
} from '../src/lib/stores/verdict';
import { latestKinematics, defaultKinematics } from '../src/lib/stores/kinematics';
import {
  unitSystem,
  resetUnitStore,
  formatWeight,
  formatArm,
  formatFuelVolume
} from '../src/lib/stores/units';
import { loadSimulationScenario, FIXTURE_SCENARIOS } from '../src/lib/simulation/fixtures';

describe('TraumaTeam Svelte 5 Component & E2E Integration Suite', () => {
  beforeEach(() => {
    resetRouterStores();
    resetWeatherStore();
    resetVerdictStore();
    resetUnitStore();
    latestKinematics.set(structuredClone(defaultKinematics));
  });

  it('VerdictBanner renders STANDBY initially, formats aviation limiting-factor callouts, and preserves machine code tag', async () => {
    const { getByTestId, container } = render(VerdictBanner);
    expect(getByTestId('verdict-readout').textContent?.trim()).toBe('STANDBY');

    // Load NOMINAL_GO
    loadSimulationScenario(wsRouter, 'NOMINAL_GO');
    await Promise.resolve();
    expect(getByTestId('verdict-readout').textContent?.trim()).toBe('APPROVED');
    expect(getByTestId('limiting-factor-label').textContent?.trim()).toBe(
      'NONE — ALL MARGINS NOMINAL'
    );
    expect(getByTestId('limiting-factor').textContent?.trim()).toBe('LIMITING_FACTOR_NONE');
    expect(container.querySelector('.tier-GO')).not.toBeNull();

    // Load MARGINAL_CAUTION
    loadSimulationScenario(wsRouter, 'MARGINAL_CAUTION');
    await Promise.resolve();
    expect(getByTestId('verdict-readout').textContent?.trim()).toBe('MARGINAL');
    expect(container.querySelector('.tier-CAUTION')).not.toBeNull();

    // Load ASPEN_BELOW_MINIMUMS (NO-GO)
    loadSimulationScenario(wsRouter, 'ASPEN_BELOW_MINIMUMS');
    await Promise.resolve();
    expect(getByTestId('verdict-readout').textContent?.trim()).toBe('BELOW_MINIMUMS');
    expect(getByTestId('limiting-factor-label').textContent?.trim()).toBe(
      'CEILING BELOW § 135.609 VFR MINIMUMS'
    );
    expect(getByTestId('limiting-factor').textContent?.trim()).toBe(
      'FAR_135_609_CEILING_BELOW_MINIMUMS'
    );
    expect(container.querySelector('.tier-NO-GO')).not.toBeNull();
  });

  it('CgEnvelope switches between Airbus H145 and Bell 429 OEM envelopes and recalculates TAS, FF, ETE, and CG', async () => {
    const { getByLabelText, container } = render(CgEnvelope);

    // Initial H145 MTOW 3700 KG
    expect(container.textContent).toContain('MTOW 3700 KG');
    expect(container.textContent).toContain('COMPLIANT');

    // Switch airframe select to Bell 429 GlobalRanger
    const select = getByLabelText('Airframe Selector') as HTMLSelectElement;
    await fireEvent.change(select, { target: { value: 'Bell 429 GlobalRanger' } });

    const kBell = get(latestKinematics);
    expect(kBell.airframe).toBe('Bell 429 GlobalRanger');
    // Bell 429 empty CG is 3260 mm, envelope [3120..3480 mm]
    expect(kBell.mass_balance.cg_long_mm).toBeGreaterThan(3120);
    expect(kBell.mass_balance.cg_long_mm).toBeLessThan(3480);
    expect(kBell.mass_balance.fwd_limit_mm).toBeGreaterThanOrEqual(3120);
    expect(kBell.mass_balance.aft_limit_mm).toBeLessThanOrEqual(3480);
    expect(container.textContent).toContain('MTOW 3175 KG');

    // Adjust Fuel slider and verify TAS / Fuel Flow / Gross Weight update dynamically
    const fuelSlider = container.querySelector(
      'input[title="Jet-A Usable Fuel (kg)"]'
    ) as HTMLInputElement;
    expect(fuelSlider).not.toBeNull();
    const prevGw = get(latestKinematics).mass_balance.gross_weight_kg;
    const prevFf = get(latestKinematics).cruise_fuel_flow_kg_hr;

    await fireEvent.input(fuelSlider, { target: { value: '580' } });
    const kAfterFuel = get(latestKinematics);
    expect(kAfterFuel.mass_balance.gross_weight_kg).toBeGreaterThan(prevGw);
    expect(kAfterFuel.cruise_fuel_flow_kg_hr).toBeGreaterThan(prevFf);
  });

  it('StationWeatherGrid renders METAR rows, flight categories, § 135.609 margins, and selects station on click', async () => {
    loadSimulationScenario(wsRouter, 'NOMINAL_GO');
    const { container } = render(StationWeatherGrid);

    expect(container.textContent).toContain('KAPA');
    expect(container.textContent).toContain('KDEN');
    expect(container.textContent).toContain('KLXV');
    expect(container.textContent).toContain('PASS');

    // Click KASE row (index 1 in alphabetical stationList) to view raw METAR strip
    const rows = container.querySelectorAll('tbody tr');
    expect(rows.length).toBeGreaterThanOrEqual(3);
    await fireEvent.click(rows[1]);
    expect(container.textContent).toContain('RAW METAR // KASE');
  });

  it('HazardAndTracePanel switches tabs (Audit Trace, SIGMETS, WS Envelopes, CONSOLE) and executes CLI eval offline', async () => {
    loadSimulationScenario(wsRouter, 'NOMINAL_GO');
    const { getByText, getByTestId, container } = render(HazardAndTracePanel);

    // 1. Audit Trace tab is active by default and displays trace rows
    expect(getByTestId('audit-trace-list').textContent).toContain('14 CFR § 135.609');

    // 2. Switch to CONSOLE tab and execute `eval KBOS KMHT KPWM` while disconnected
    const consoleTabBtn = getByText('CONSOLE');
    await fireEvent.click(consoleTabBtn);

    const cliInput = container.querySelector('input[type="text"]') as HTMLInputElement;
    const form = container.querySelector('form') as HTMLFormElement;
    expect(cliInput).not.toBeNull();

    await fireEvent.input(cliInput, { target: { value: 'eval KBOS KMHT KPWM' } });
    await fireEvent.submit(form);

    // Offline CLI eval fallback should load NEW_ENGLAND_SIGMET and set verdict to DISQUALIFIED
    expect(get(latestComplianceReport)?.verdict).toBe(FIXTURE_SCENARIOS.NEW_ENGLAND_SIGMET.verdict);
    expect(get(currentVerdictTier)).toBe('NO-GO');
    expect(get(activeHazardCount)).toBeGreaterThan(0);

    // 3. Switch to SIGMETS tab and verify active Convective SIGMET 22E is rendered
    const sigmetsTabBtn = container.querySelectorAll('.panel-header-bar button')[1];
    await fireEvent.click(sigmetsTabBtn);
    expect(container.textContent).toContain('CONVECTIVE SIGMET 22E');
  });

  it('CustomPresetModal loads sample JSON and injects custom scenario into stores', async () => {
    const { getByText, container } = render(CustomPresetModal, { props: { isOpen: true } });

    const loadSampleBtn = getByText('[LOAD SAMPLE JSON]');
    await fireEvent.click(loadSampleBtn);

    const textarea = container.querySelector('textarea') as HTMLTextAreaElement;
    expect(textarea.value).toContain('CUSTOM_ROCKY_FLIGHT');

    const applyBtn = getByText('APPLY PRESET TO ENGINE');
    await fireEvent.click(applyBtn);

    expect(get(latestComplianceReport)?.verdict).toBe('APPROVED');
    expect(get(latestComplianceReport)?.explanation).toContain('Custom simulation');
  });

  it('ConnectionHeader displays STATIC SIM indicator and HIGH-DA (KLXV) triggers OGE_HOVER_SCENE_DEFICIT with defuel mitigation in CgEnvelope', async () => {
    connectionState.set('STATIC_SIM');
    const header = render(ConnectionHeader);
    const cg = render(CgEnvelope);

    expect(header.container.textContent).toContain('STATIC SIM');
    expect(header.container.textContent).not.toContain('WS: RECONNECTING');

    const highDaBtn = header.getByText('HIGH-DA (KLXV)');
    await fireEvent.click(highDaBtn);
    await Promise.resolve();

    expect(get(latestComplianceReport)?.verdict).toBe('NO-GO');
    expect(get(latestComplianceReport)?.limiting_factor).toBe('OGE_HOVER_SCENE_DEFICIT');
    expect(get(latestKinematics).limiting_factor).toBe('OGE_HOVER_SCENE_DEFICIT');
    expect(get(latestKinematics).oge_margin_kg).toBeLessThan(0);
    expect(get(latestKinematics).required_fuel_offload_kg).toBeGreaterThan(250);
    expect(cg.container.textContent).toContain('OGE_HOVER_SCENE_DEFICIT');
    expect(cg.getByTestId('kinematics-mitigation').textContent).toContain('DEFUEL');
  });

  it('ConnectionHeader shows Zulu clock & LIVE FEED/STATIC SIM without raw counters, and UNITS toggle converts weights/arms without mutating SI stores', async () => {
    connectionState.set('STATIC_SIM');
    const header = render(ConnectionHeader);
    const cg = render(CgEnvelope);
    const legStrip = render(MissionLegStrip);

    // Raw counters removed; Zulu HH:MMZ clock present
    expect(header.container.textContent).not.toContain('SEQ:');
    expect(header.container.textContent).not.toContain('RTT:');
    expect(header.container.textContent).not.toContain('TOPICS:');
    expect(header.getByTestId('zulu-clock').textContent).toMatch(/\d{2}:\d{2}Z/);
    expect(header.getByTestId('telemetry-mode').textContent).toContain('STATIC SIM');

    connectionState.set('CONNECTED');
    await Promise.resolve();
    expect(header.getByTestId('telemetry-mode').textContent).toContain('LIVE FEED');

    const siGwBefore = get(latestKinematics).mass_balance.gross_weight_kg;
    const siCgBefore = get(latestKinematics).mass_balance.cg_long_mm;
    expect(get(unitSystem)).toBe('KG_MM');
    expect(cg.container.textContent).toContain('MTOW 3700 KG');
    expect(legStrip.getByTestId('mission-fuel-burn').textContent).toContain('KG');

    // Pure conversion sanity
    expect(formatWeight(3700, 'LBS_IN')).toBe('8157 LBS');
    expect(formatArm(2540, 'LBS_IN')).toBe('100.0 IN');
    expect(formatFuelVolume(420, 'KG_MM')).toBe('522 L');
    expect(formatFuelVolume(420, 'LBS_IN')).toBe('138 GAL');

    await fireEvent.click(header.getByTestId('unit-toggle'));
    await Promise.resolve();

    expect(get(unitSystem)).toBe('LBS_IN');
    expect(header.getByTestId('unit-toggle').textContent).toContain('LBS/IN');
    expect(cg.container.textContent).toContain('MTOW 8157 LBS');
    expect(cg.container.textContent).toContain('IN');
    expect(legStrip.getByTestId('mission-fuel-burn').textContent).toContain('LBS');
    expect(legStrip.getByTestId('mission-fuel-burn').textContent).toContain('GAL');

    // Underlying SI values unchanged
    expect(get(latestKinematics).mass_balance.gross_weight_kg).toBe(siGwBefore);
    expect(get(latestKinematics).mass_balance.cg_long_mm).toBe(siCgBefore);
  });

  it('CgEnvelope interactive loadout sliders & 1-click SIMULATE DEFUEL verify § 135.209 reserves and render MissionLegStrip 20/30-min rules', async () => {
    const banner = render(VerdictBanner);
    const legStrip = render(MissionLegStrip);
    const cg = render(CgEnvelope);
    const corridorPanel = render(CorridorControlPanel);

    // Verify tablet-ergonomic EFB touch class on CorridorControlPanel controls
    expect(corridorPanel.container.querySelectorAll('.chk-row.efb-touch').length).toBe(8);
    expect(corridorPanel.container.querySelectorAll('.action-bar button.efb-touch').length).toBe(2);

    // Load HIGH-DA (KLXV) -> triggers OGE_HOVER_SCENE_DEFICIT & 1-click SIMULATE DEFUEL
    loadSimulationScenario(wsRouter, 'HIGH_DA_KLXV');
    await Promise.resolve();

    expect(banner.getByTestId('limiting-factor-label').textContent?.trim()).toBe(
      'OGE HOVER PERFORMANCE DEFICIT AT SCENE'
    );
    expect(legStrip.getByTestId('mission-total-distance').textContent).toContain('71.');
    expect(legStrip.getByTestId('mission-reserve-margin').textContent).toContain('20-MIN VFR');

    const defuelBtn = cg.getByTestId('simulate-defuel-btn');
    expect(defuelBtn.textContent).toContain('SIMULATE DEFUEL (-289 KG)');
    await fireEvent.click(defuelBtn);
    await Promise.resolve();

    // Post-defuel clears OGE deficit & verifies § 135.209 20-min VFR reserve
    expect(get(latestKinematics).is_feasible).toBe(true);
    expect(get(latestKinematics).oge_margin_kg).toBeGreaterThanOrEqual(0);
    expect(get(latestKinematics).reserve_margin_kg).toBeGreaterThan(0);
    expect(banner.getByTestId('verdict-readout').textContent?.trim()).toBe('APPROVED');
    expect(cg.getByTestId('defuel-verification').textContent).toContain('§ 135.209 VERIFIED');
    expect(cg.getByTestId('cg-burn-vector')).not.toBeNull();

    // Drag Crew + Litter and ECMO/Hoist Gear sliders & Scene OGE Hover slider to stress § 135.209 reserve
    const gearSlider = cg.getByTestId('slider-specialty-gear') as HTMLInputElement;
    await fireEvent.input(gearSlider, { target: { value: '120' } });
    expect(get(latestKinematics).specialty_gear_kg).toBe(120);

    const hoverSlider = cg.getByTestId('slider-scene-hover') as HTMLInputElement;
    await fireEvent.input(hoverSlider, { target: { value: '28' } });
    await Promise.resolve();
    expect(get(latestKinematics).scene_hover_minutes).toBe(28);
    expect(get(latestKinematics).is_feasible).toBe(false);

    // Switch to §135.611 IFR scenario -> MissionLegStrip switches to 30-MIN IFR reserve threshold
    loadSimulationScenario(wsRouter, 'IFR_TRANSITION_REQUIRED');
    await Promise.resolve();
    expect(legStrip.getByTestId('mission-reserve-margin').textContent).toContain('30-MIN IFR');
  });
});
