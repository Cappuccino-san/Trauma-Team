import { describe, it, expect, beforeEach } from 'vitest';
import { get } from 'svelte/store';
import {
  TraumaTeamWebSocketRouter,
  connectionState,
  lastSeq,
  seqGapCount,
  rttLatencyMs,
  subscribedTopics,
  lastGatewayError,
  resetRouterStores
} from '../src/lib/websocket';
import {
  stationsById,
  stationList,
  activeHazards,
  activeHazardCount,
  resetWeatherStore
} from '../src/lib/stores/weather';
import {
  latestComplianceReport,
  currentVerdictTier,
  evaluationHistory,
  resetVerdictStore,
  defaultCorridor,
  defaultRegulatoryContext
} from '../src/lib/stores/verdict';
import {
  latestKinematics,
  currentGrossWeight,
  currentCgLong,
  currentCgLat,
  isCgCompliant,
  defaultKinematics
} from '../src/lib/stores/kinematics';
import { FIXTURE_METARS, FIXTURE_HAZARDS, FIXTURE_SCENARIOS } from '../src/lib/simulation/fixtures';
import { ServerEvent, type ServerEnvelope, type ClientEnvelope } from '../src/lib/types/envelope';

/**
 * Mock WebSocket for deterministic unit testing of client/server framing
 */
class MockWebSocket {
  public static OPEN = 1;
  public static CONNECTING = 0;
  public static CLOSED = 3;

  public readyState = MockWebSocket.CONNECTING;
  public sentFrames: string[] = [];
  public onopen: (() => void) | null = null;
  public onmessage: ((ev: { data: string }) => void) | null = null;
  public onerror: (() => void) | null = null;
  public onclose: (() => void) | null = null;

  constructor(public url: string) {}

  public send(data: string): void {
    this.sentFrames.push(data);
  }

  public close(): void {
    this.readyState = MockWebSocket.CLOSED;
    if (this.onclose) this.onclose();
  }

  public simulateOpen(): void {
    this.readyState = MockWebSocket.OPEN;
    if (this.onopen) this.onopen();
  }

  public simulateServerFrame(envelope: ServerEnvelope): void {
    if (this.onmessage) {
      this.onmessage({ data: JSON.stringify(envelope) });
    }
  }
}

describe('TraumaTeam Svelte WebSocket Router & Demultiplexing Stores', () => {
  let mockWs: MockWebSocket;
  let router: TraumaTeamWebSocketRouter;

  beforeEach(() => {
    resetRouterStores();
    resetWeatherStore();
    resetVerdictStore();

    router = new TraumaTeamWebSocketRouter({
      url: 'ws://localhost:8080/ws',
      autoReconnect: false,
      heartbeatIntervalMs: 0,
      webSocketFactory: (url: string) => {
        mockWs = new MockWebSocket(url);
        return mockWs as unknown as WebSocket;
      }
    });
  });

  it('connects and automatically sends SUBSCRIBE envelope for default stations and topics', () => {
    router.connect();
    expect(get(connectionState)).toBe('CONNECTING');

    mockWs.simulateOpen();
    expect(get(connectionState)).toBe('CONNECTED');
    expect(mockWs.sentFrames.length).toBe(1);

    const sent = JSON.parse(mockWs.sentFrames[0]) as ClientEnvelope;
    expect(sent.action).toBe('SUBSCRIBE');
    expect(sent.params?.stations).toContain('KAPA');
    expect(sent.params?.stations).toContain('KLXV');
    expect(sent.params?.topics).toContain('minimums');
    expect(sent.params?.topics).toContain('hazards');
  });

  it('demultiplexes WEATHER_SNAPSHOT and WEATHER_UPDATE into stationsById store', () => {
    router.connect();
    mockWs.simulateOpen();

    mockWs.simulateServerFrame({
      event: ServerEvent.WEATHER_SNAPSHOT,
      request_id: 'cond-1',
      timestamp_utc: '2026-11-05T20:00:00Z',
      seq: 1,
      station_id: 'KAPA',
      payload: FIXTURE_METARS.KAPA
    });

    const kapa = get(stationsById)['KAPA'];
    expect(kapa).toBeDefined();
    expect(kapa.metar?.flight_category).toBe('VFR');
    expect(kapa.metar?.density_altitude_ft).toBe(6840);
    expect(get(lastSeq)).toBe(1);

    // Send WEATHER_UPDATE for KLXV
    mockWs.simulateServerFrame({
      event: ServerEvent.WEATHER_UPDATE,
      timestamp_utc: '2026-11-05T20:01:00Z',
      seq: 2,
      station_id: 'KLXV',
      payload: FIXTURE_METARS.KLXV
    });

    const list = get(stationList);
    expect(list.length).toBe(2);
    expect(list.map((s) => s.station_id)).toEqual(['KAPA', 'KLXV']);
    expect(get(lastSeq)).toBe(2);
  });

  it('demultiplexes MINIMUMS_EVALUATION into latestComplianceReport and maps verdict tiers', () => {
    router.connect();
    mockWs.simulateOpen();

    // Send EVALUATE_MINIMUMS request from client
    const reqId = router.requestMinimumsEvaluation(defaultCorridor, defaultRegulatoryContext);
    const outFrame = JSON.parse(mockWs.sentFrames[mockWs.sentFrames.length - 1]) as ClientEnvelope;
    expect(outFrame.action).toBe('EVALUATE_MINIMUMS');
    expect(outFrame.request_id).toBe(reqId);
    expect(outFrame.params?.corridor?.departure).toBe('KAPA');

    // Receive MINIMUMS_EVALUATION response
    mockWs.simulateServerFrame({
      event: ServerEvent.MINIMUMS_EVALUATION,
      request_id: reqId,
      timestamp_utc: '2026-11-05T20:02:00Z',
      seq: 10,
      payload: FIXTURE_SCENARIOS.ASPEN_BELOW_MINIMUMS
    });

    const report = get(latestComplianceReport);
    expect(report).not.toBeNull();
    expect(report?.verdict).toBe('BELOW_MINIMUMS');
    expect(report?.limiting_factor).toBe('FAR_135_609_CEILING_BELOW_MINIMUMS');
    expect(get(currentVerdictTier)).toBe('NO-GO');
    expect(get(evaluationHistory).length).toBe(1);
  });

  it('demultiplexes HAZARD_ALERT into activeHazards store without duplicates', () => {
    router.connect();
    mockWs.simulateOpen();

    mockWs.simulateServerFrame({
      event: ServerEvent.HAZARD_ALERT,
      timestamp_utc: '2026-11-05T20:03:00Z',
      seq: 11,
      payload: FIXTURE_HAZARDS[0]
    });

    expect(get(activeHazardCount)).toBe(1);
    expect(get(activeHazards)[0].advisory_id).toBe('CONVECTIVE SIGMET 15E');

    // Re-ingest same hazard ID with updated summary
    mockWs.simulateServerFrame({
      event: ServerEvent.HAZARD_ALERT,
      timestamp_utc: '2026-11-05T20:04:00Z',
      seq: 12,
      payload: {
        ...FIXTURE_HAZARDS[0],
        summary: 'UPDATED CONVECTIVE LINE'
      }
    });

    expect(get(activeHazardCount)).toBe(1);
    expect(get(activeHazards)[0].summary).toBe('UPDATED CONVECTIVE LINE');
  });

  it('tracks sequence gaps, ACK subscriptions, PING/PONG latency, and ERROR frames', () => {
    router.connect();
    mockWs.simulateOpen();

    // Seq 1
    mockWs.simulateServerFrame({
      event: ServerEvent.ACK,
      request_id: 'sub-1',
      timestamp_utc: '2026-11-05T20:00:00Z',
      seq: 1,
      payload: {
        status: 'SUBSCRIBED',
        active_topics: ['station:KAPA', 'minimums']
      }
    });
    expect(get(subscribedTopics)).toEqual(['station:KAPA', 'minimums']);
    expect(get(seqGapCount)).toBe(0);

    // Jump to Seq 5 (gap of 3: missing 2, 3, 4)
    const sentPingTs = Date.now() - 42;
    const pingId = router.sendPing(sentPingTs);

    mockWs.simulateServerFrame({
      event: ServerEvent.PONG,
      request_id: pingId,
      timestamp_utc: '2026-11-05T20:00:05Z',
      seq: 5,
      payload: {
        client_timestamp: sentPingTs,
        server_epoch_ms: Date.now()
      }
    });

    expect(get(lastSeq)).toBe(5);
    expect(get(seqGapCount)).toBe(3);
    expect(get(rttLatencyMs)).toBeGreaterThanOrEqual(0);

    // Error frame
    mockWs.simulateServerFrame({
      event: ServerEvent.ERROR,
      request_id: 'bad-req',
      timestamp_utc: '2026-11-05T20:00:06Z',
      seq: 6,
      error: 'station must be specified for GET_CONDITIONS'
    });
    expect(get(lastGatewayError)).toBe('station must be specified for GET_CONDITIONS');
  });

  it('routes KINEMATICS_EVALUATION frames to kinematics store and updates derived stores', () => {
    router.connect();
    mockWs.simulateOpen();

    const mockKinematicsPayload = {
      ...defaultKinematics,
      airframe: 'Airbus H145 (BK117 D-2)',
      verdict: 'CAUTION',
      limiting_factor: 'OGE_HOVER_SCENE_DEFICIT',
      mass_balance: {
        ...defaultKinematics.mass_balance,
        gross_weight_kg: 3420,
        cg_long_mm: 3950,
        cg_lat_mm: 25.5,
        is_overall_compliant: true
      }
    };

    mockWs.simulateServerFrame({
      event: ServerEvent.KINEMATICS_EVALUATION,
      request_id: 'kin-req-1',
      timestamp_utc: '2026-11-05T20:01:00Z',
      seq: 7,
      payload: mockKinematicsPayload
    });

    expect(get(latestKinematics).verdict).toBe('CAUTION');
    expect(get(latestKinematics).limiting_factor).toBe('OGE_HOVER_SCENE_DEFICIT');
    expect(get(currentGrossWeight)).toBe(3420);
    expect(get(currentCgLong)).toBe(3950);
    expect(get(currentCgLat)).toBe(25.5);
    expect(get(isCgCompliant)).toBe(true);
  });
});
