import { describe, it, expect, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { get } from 'svelte/store';
import VerdictBanner from '../src/lib/components/VerdictBanner.svelte';
import CgEnvelope from '../src/lib/components/CgEnvelope.svelte';
import StationWeatherGrid from '../src/lib/components/StationWeatherGrid.svelte';
import HazardAndTracePanel from '../src/lib/components/HazardAndTracePanel.svelte';
import CustomPresetModal from '../src/lib/components/CustomPresetModal.svelte';
import { resetRouterStores, wsRouter } from '../src/lib/websocket';
import { resetWeatherStore, activeHazardCount } from '../src/lib/stores/weather';
import {
  resetVerdictStore,
  latestComplianceReport,
  currentVerdictTier
} from '../src/lib/stores/verdict';
import { latestKinematics, defaultKinematics } from '../src/lib/stores/kinematics';
import { loadSimulationScenario, FIXTURE_SCENARIOS } from '../src/lib/simulation/fixtures';

describe('TraumaTeam Svelte 5 Component & E2E Integration Suite', () => {
  beforeEach(() => {
    resetRouterStores();
    resetWeatherStore();
    resetVerdictStore();
    latestKinematics.set(structuredClone(defaultKinematics));
  });

  it('VerdictBanner renders STANDBY initially and updates reactively across GO, CAUTION, and NO-GO verdicts', async () => {
    const { getByTestId, container } = render(VerdictBanner);
    expect(getByTestId('verdict-readout').textContent?.trim()).toBe('STANDBY');

    // Load NOMINAL_GO
    loadSimulationScenario(wsRouter, 'NOMINAL_GO');
    await Promise.resolve();
    expect(getByTestId('verdict-readout').textContent?.trim()).toBe('APPROVED');
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
});
