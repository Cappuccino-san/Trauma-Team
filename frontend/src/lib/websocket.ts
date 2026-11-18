import { writable, get } from 'svelte/store';
import {
  ClientAction,
  ServerEvent,
  type ClientEnvelope,
  type ServerEnvelope,
  type ConnectionState,
  type CorridorParams,
  type RegulatoryContextParams,
  type AckPayload,
  type PongPayload
} from './types/envelope';
import { ingestWeatherEnvelope, ingestHazardEnvelope } from './stores/weather';
import { ingestMinimumsEnvelope, isEvaluating } from './stores/verdict';
import { ingestKinematicsEnvelope, kinematicsPresetStore } from './stores/kinematics';

/**
 * Svelte WebSocket Router & Demultiplexer (`websocket.ts`)
 * Connects to the Go `weather_ingest` gateway (`/ws`), handles bidirectional
 * `ClientEnvelope` / `ServerEnvelope` framing, sequence monotonicity tracking,
 * RTT heartbeat measurement (`PING` / `PONG`), and automatic reconnection.
 */

export interface EnvelopeLogEntry {
  direction: 'IN' | 'OUT';
  timestamp: string;
  eventOrAction: string;
  seq?: number;
  requestId?: string;
  stationId?: string;
  summary: string;
}

export const connectionState = writable<ConnectionState>('DISCONNECTED');
export const lastSeq = writable<number>(0);
export const seqGapCount = writable<number>(0);
export const rttLatencyMs = writable<number | null>(null);
export const subscribedTopics = writable<string[]>([]);
export const lastGatewayError = writable<string | null>(null);
export const envelopeStreamLog = writable<EnvelopeLogEntry[]>([]);
export const websocketLogStore = envelopeStreamLog;

export interface WebSocketRouterOptions {
  url?: string;
  autoReconnect?: boolean;
  maxReconnectDelayMs?: number;
  heartbeatIntervalMs?: number;
  webSocketFactory?: (url: string) => WebSocket;
}

export interface ConnectOptions {
  userInitiated?: boolean;
  autoReconnect?: boolean;
}

/**
 * Returns true when running from a static hosting origin (e.g., GitHub Pages or file://)
 * where no live `/ws` Go telemetry gateway exists on `window.location.host`.
 */
export function isStaticHostingEnvironment(): boolean {
  if (typeof window === 'undefined') return false;
  const host = window.location.hostname.toLowerCase();
  return host.endsWith('.github.io') || window.location.protocol === 'file:';
}

function appendLog(entry: EnvelopeLogEntry): void {
  envelopeStreamLog.update((logs) => [entry, ...logs].slice(0, 60));
}

export class TraumaTeamWebSocketRouter {
  private ws: WebSocket | null = null;
  private url: string;
  private defaultAutoReconnect: boolean;
  private autoReconnect: boolean;
  private maxReconnectDelayMs: number;
  private heartbeatIntervalMs: number;
  private webSocketFactory: (url: string) => WebSocket;

  private reconnectAttempts = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null;
  private reqCounter = 0;
  private pendingPingTimestamps = new Map<string, number>();
  private desiredStations = new Set<string>(['KAPA', 'KDEN', 'KLXV', 'KASE']);
  private desiredTopics = new Set<string>(['minimums', 'hazards']);

  constructor(options: WebSocketRouterOptions = {}) {
    const defaultUrl =
      typeof window !== 'undefined' && !isStaticHostingEnvironment()
        ? `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`
        : 'ws://localhost:8080/ws';

    this.url = options.url ?? defaultUrl;
    this.defaultAutoReconnect = options.autoReconnect ?? false;
    this.autoReconnect = this.defaultAutoReconnect;
    this.maxReconnectDelayMs = options.maxReconnectDelayMs ?? 10000;
    this.heartbeatIntervalMs = options.heartbeatIntervalMs ?? 15000;
    this.webSocketFactory = options.webSocketFactory ?? ((u: string) => new WebSocket(u));
  }

  public nextRequestId(prefix = 'req'): string {
    this.reqCounter += 1;
    return `${prefix}-${this.reqCounter}`;
  }

  public enterStaticSimMode(): void {
    this.autoReconnect = false;
    this.reconnectAttempts = 0;
    this.clearTimers();
    if (this.ws) {
      this.ws.onopen = null;
      this.ws.onmessage = null;
      this.ws.onerror = null;
      this.ws.onclose = null;
      this.ws.close();
      this.ws = null;
    }
    lastGatewayError.set(null);
    connectionState.set('STATIC_SIM');
  }

  public connect(customUrl?: string, connectOptions: ConnectOptions = {}): void {
    const userInitiated = connectOptions.userInitiated ?? false;
    if (isStaticHostingEnvironment() && !userInitiated) {
      this.enterStaticSimMode();
      return;
    }

    if (customUrl) {
      this.url = customUrl;
    }
    this.autoReconnect = connectOptions.autoReconnect ?? this.defaultAutoReconnect;
    this.clearTimers();

    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return;
    }

    let hasOpened = false;
    connectionState.set(this.reconnectAttempts > 0 ? 'RECONNECTING' : 'CONNECTING');
    lastGatewayError.set(null);

    try {
      this.ws = this.webSocketFactory(this.url);
    } catch (err) {
      if (!userInitiated && !this.autoReconnect) {
        this.enterStaticSimMode();
        return;
      }
      connectionState.set('ERROR');
      lastGatewayError.set(err instanceof Error ? err.message : 'WebSocket initialization failed');
      this.scheduleReconnect();
      return;
    }

    this.ws.onopen = () => {
      hasOpened = true;
      this.reconnectAttempts = 0;
      connectionState.set('CONNECTED');
      this.startHeartbeat();

      // Re-subscribe to desired stations and topics automatically upon connection
      if (this.desiredStations.size > 0 || this.desiredTopics.size > 0) {
        this.subscribe(Array.from(this.desiredStations), Array.from(this.desiredTopics));
      }
    };

    this.ws.onmessage = (event: MessageEvent) => {
      if (typeof event.data === 'string') {
        this.ingestRawFrame(event.data);
      }
    };

    this.ws.onerror = () => {
      if (!userInitiated && !hasOpened) {
        return;
      }
      lastGatewayError.set('WebSocket transport error');
    };

    this.ws.onclose = () => {
      this.stopHeartbeat();
      this.ws = null;
      if (!userInitiated && !hasOpened && !this.autoReconnect) {
        this.enterStaticSimMode();
      } else if (this.autoReconnect) {
        connectionState.set('RECONNECTING');
        this.scheduleReconnect();
      } else {
        connectionState.set(isStaticHostingEnvironment() ? 'STATIC_SIM' : 'DISCONNECTED');
      }
    };
  }

  public disconnect(): void {
    this.autoReconnect = false;
    this.clearTimers();
    if (this.ws) {
      this.ws.onclose = null;
      this.ws.close();
      this.ws = null;
    }
    connectionState.set(isStaticHostingEnvironment() ? 'STATIC_SIM' : 'DISCONNECTED');
  }

  /**
   * Demultiplexes a raw JSON string or pre-parsed `ServerEnvelope` into the
   * appropriate reactive Svelte stores.
   * Exposed publicly so both live WebSocket frames and deterministic simulation/tests
   * can drive the exact same router pipeline.
   */
  public ingestRawFrame(rawJson: string): ServerEnvelope | null {
    const trimmed = rawJson.trim();
    if (!trimmed) return null;

    if (trimmed.includes('\n')) {
      const lines = trimmed.split('\n').map((l) => l.trim()).filter(Boolean);
      let lastEnvelope: ServerEnvelope | null = null;
      for (const line of lines) {
        try {
          const env = JSON.parse(line) as ServerEnvelope;
          this.routeEnvelope(env);
          lastEnvelope = env;
        } catch {
          lastGatewayError.set('Malformed JSON frame received from gateway');
        }
      }
      return lastEnvelope;
    }

    let envelope: ServerEnvelope;
    try {
      envelope = JSON.parse(trimmed) as ServerEnvelope;
    } catch {
      lastGatewayError.set('Malformed JSON frame received from gateway');
      return null;
    }
    this.routeEnvelope(envelope);
    return envelope;
  }

  /**
   * Routes a strongly-typed `ServerEnvelope` to the corresponding store.
   */
  public routeEnvelope(envelope: ServerEnvelope): void {
    if (!envelope || typeof envelope !== 'object' || !envelope.event) {
      lastGatewayError.set('Envelope missing event discriminator');
      return;
    }

    // Track monotonic sequence numbers
    if (typeof envelope.seq === 'number' && envelope.seq > 0) {
      const prev = get(lastSeq);
      if (prev > 0 && envelope.seq > prev + 1) {
        seqGapCount.update((n) => n + (envelope.seq - prev - 1));
      }
      if (envelope.seq >= prev) {
        lastSeq.set(envelope.seq);
      }
    }

    appendLog({
      direction: 'IN',
      timestamp: envelope.timestamp_utc || new Date().toISOString(),
      eventOrAction: envelope.event,
      seq: envelope.seq,
      requestId: envelope.request_id,
      stationId: envelope.station_id,
      summary: this.summarizeEnvelope(envelope)
    });

    switch (envelope.event) {
      case ServerEvent.WEATHER_SNAPSHOT:
      case ServerEvent.WEATHER_UPDATE:
        ingestWeatherEnvelope(envelope);
        break;

      case ServerEvent.MINIMUMS_EVALUATION:
        ingestMinimumsEnvelope(envelope);
        break;

      case ServerEvent.KINEMATICS_EVALUATION:
        ingestKinematicsEnvelope(envelope);
        break;

      case ServerEvent.HAZARD_ALERT:
        ingestHazardEnvelope(envelope);
        break;

      case ServerEvent.PONG:
        this.handlePong(envelope as ServerEnvelope<PongPayload>);
        break;

      case ServerEvent.ACK:
        this.handleAck(envelope as ServerEnvelope<AckPayload>);
        break;

      case ServerEvent.ERROR:
        isEvaluating.set(false);
        lastGatewayError.set(envelope.error || 'Unspecified gateway error');
        break;

      default:
        lastGatewayError.set(`Unhandled server event: ${String(envelope.event)}`);
        break;
    }
  }

  /**
   * Sends a `ClientEnvelope` over the active WebSocket connection.
   */
  public sendEnvelope(envelope: ClientEnvelope): boolean {
    appendLog({
      direction: 'OUT',
      timestamp: new Date().toISOString(),
      eventOrAction: envelope.action,
      requestId: envelope.request_id,
      stationId: envelope.params?.station,
      summary: this.summarizeClientEnvelope(envelope)
    });

    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      return false;
    }

    try {
      this.ws.send(JSON.stringify(envelope));
      return true;
    } catch (err) {
      lastGatewayError.set(err instanceof Error ? err.message : 'Failed to send frame');
      return false;
    }
  }

  /**
   * Sends a `SUBSCRIBE` action for stations and/or topics.
   */
  public subscribe(stations: string[] = [], topics: string[] = []): string {
    const cleanStations = stations.map((s) => s.trim().toUpperCase()).filter(Boolean);
    const cleanTopics = topics.map((t) => t.trim().toLowerCase()).filter(Boolean);

    cleanStations.forEach((s) => this.desiredStations.add(s));
    cleanTopics.forEach((t) => this.desiredTopics.add(t));

    const requestId = this.nextRequestId('sub');
    this.sendEnvelope({
      action: ClientAction.SUBSCRIBE,
      request_id: requestId,
      params: {
        stations: cleanStations,
        topics: cleanTopics
      }
    });
    return requestId;
  }

  /**
   * Sends an `UNSUBSCRIBE` action for stations and/or topics.
   */
  public unsubscribe(stations: string[] = [], topics: string[] = []): string {
    const cleanStations = stations.map((s) => s.trim().toUpperCase()).filter(Boolean);
    const cleanTopics = topics.map((t) => t.trim().toLowerCase()).filter(Boolean);

    cleanStations.forEach((s) => this.desiredStations.delete(s));
    cleanTopics.forEach((t) => this.desiredTopics.delete(t));

    const requestId = this.nextRequestId('unsub');
    this.sendEnvelope({
      action: ClientAction.UNSUBSCRIBE,
      request_id: requestId,
      params: {
        stations: cleanStations,
        topics: cleanTopics
      }
    });
    return requestId;
  }

  /**
   * Sends a `GET_CONDITIONS` action for a single station ICAO code.
   */
  public requestStationConditions(stationId: string): string {
    const cleanStation = stationId.trim().toUpperCase();
    const requestId = this.nextRequestId('cond');
    this.sendEnvelope({
      action: ClientAction.GET_CONDITIONS,
      request_id: requestId,
      params: {
        station: cleanStation
      }
    });
    return requestId;
  }

  /**
   * Sends an `EVALUATE_MINIMUMS` action with corridor and FAA Part 135.L regulatory context.
   */
  public requestMinimumsEvaluation(
    corridor: CorridorParams,
    regulatoryContext: RegulatoryContextParams
  ): string {
    const requestId = this.nextRequestId('eval');
    const sent = this.sendEnvelope({
      action: ClientAction.EVALUATE_MINIMUMS,
      request_id: requestId,
      params: {
        corridor,
        regulatory_context: regulatoryContext
      }
    });
    isEvaluating.set(sent);
    return requestId;
  }

  /**
   * Sends an `EVALUATE_KINEMATICS` action to the Go/Rust C-FFI physics gateway.
   */
  public requestKinematicsEvaluation(
    kinematicsParams: import('./types/envelope').KinematicsParams,
    stationId = 'KAPA'
  ): string {
    const requestId = this.nextRequestId('kin');
    this.sendEnvelope({
      action: ClientAction.EVALUATE_KINEMATICS,
      request_id: requestId,
      params: {
        station: stationId.trim().toUpperCase(),
        kinematics: kinematicsParams
      }
    });
    return requestId;
  }

  /**
   * Sends a `PING` frame with epoch millisecond timestamp.
   */
  public sendPing(nowMs: number = Date.now()): string {
    const requestId = this.nextRequestId('ping');
    this.pendingPingTimestamps.set(requestId, nowMs);
    this.sendEnvelope({
      action: ClientAction.PING,
      request_id: requestId,
      timestamp: nowMs
    });
    return requestId;
  }

  private handlePong(envelope: ServerEnvelope<PongPayload>): void {
    const now = Date.now();
    const reqId = envelope.request_id;
    if (reqId && this.pendingPingTimestamps.has(reqId)) {
      const sentAt = this.pendingPingTimestamps.get(reqId)!;
      this.pendingPingTimestamps.delete(reqId);
      rttLatencyMs.set(Math.max(0, now - sentAt));
      return;
    }
    if (envelope.payload && typeof envelope.payload.client_timestamp === 'number') {
      rttLatencyMs.set(Math.max(0, now - envelope.payload.client_timestamp));
    }
  }

  private handleAck(envelope: ServerEnvelope<AckPayload>): void {
    const payload = envelope.payload;
    if (!payload) return;

    if (payload.status === 'SUBSCRIBED' && Array.isArray(payload.active_topics)) {
      subscribedTopics.update((existing) => {
        const merged = new Set([...existing, ...payload.active_topics!]);
        return Array.from(merged);
      });
    } else if (payload.status === 'UNSUBSCRIBED' && Array.isArray(payload.unsubscribed_topics)) {
      const removed = new Set(payload.unsubscribed_topics.map((t) => t.toLowerCase()));
      subscribedTopics.update((existing) =>
        existing.filter((t) => !removed.has(t.toLowerCase()))
      );
    }
  }

  private summarizeEnvelope(envelope: ServerEnvelope): string {
    switch (envelope.event) {
      case ServerEvent.WEATHER_SNAPSHOT:
      case ServerEvent.WEATHER_UPDATE:
        return `Station ${envelope.station_id ?? 'UNK'} weather frame`;
      case ServerEvent.MINIMUMS_EVALUATION: {
        const p = envelope.payload as { verdict?: string; limiting_factor?: string } | undefined;
        return `Verdict: ${p?.verdict ?? 'UNK'} (${p?.limiting_factor ?? 'NONE'})`;
      }
      case ServerEvent.KINEMATICS_EVALUATION: {
        const k = envelope.payload as { verdict?: string; cruise_tas_kts?: number } | undefined;
        return `Kinematics: ${k?.verdict ?? 'UNK'} TAS=${k?.cruise_tas_kts?.toFixed(0) ?? '--'}kt`;
      }
      case ServerEvent.HAZARD_ALERT: {
        const h = envelope.payload as { advisory_id?: string; hazard_type?: string } | undefined;
        return `SIGMET ${h?.advisory_id ?? ''} ${h?.hazard_type ?? ''}`.trim();
      }
      case ServerEvent.ACK: {
        const a = envelope.payload as AckPayload | undefined;
        return `ACK ${a?.status ?? ''}`;
      }
      case ServerEvent.PONG:
        return 'Heartbeat PONG';
      case ServerEvent.ERROR:
        return `ERR: ${envelope.error ?? 'Unknown'}`;
      default:
        return envelope.event;
    }
  }

  private summarizeClientEnvelope(envelope: ClientEnvelope): string {
    switch (envelope.action) {
      case ClientAction.SUBSCRIBE:
        return `Sub [${(envelope.params?.stations ?? []).join(', ')}]`;
      case ClientAction.UNSUBSCRIBE:
        return `Unsub [${(envelope.params?.stations ?? []).join(', ')}]`;
      case ClientAction.GET_CONDITIONS:
        return `Get ${envelope.params?.station ?? ''}`;
      case ClientAction.EVALUATE_MINIMUMS:
        return `Eval ${envelope.params?.corridor?.departure ?? ''} -> ${envelope.params?.corridor?.destination ?? ''}`;
      case ClientAction.EVALUATE_KINEMATICS:
        return `Kinematics airframe=${envelope.params?.kinematics?.airframe_id ?? 0} fuel=${envelope.params?.kinematics?.fuel_weight_kg ?? 0}kg`;
      case ClientAction.PING:
        return `Ping ts=${envelope.timestamp ?? 0}`;
      default:
        return envelope.action;
    }
  }

  private startHeartbeat(): void {
    this.stopHeartbeat();
    if (this.heartbeatIntervalMs > 0) {
      this.heartbeatTimer = setInterval(() => {
        this.sendPing();
      }, this.heartbeatIntervalMs);
    }
  }

  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  private scheduleReconnect(): void {
    if (!this.autoReconnect || this.reconnectTimer) return;
    this.reconnectAttempts += 1;
    const expDelay = Math.min(
      this.maxReconnectDelayMs,
      500 * Math.pow(1.6, Math.min(this.reconnectAttempts, 8))
    );
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect();
    }, expDelay);
  }

  private clearTimers(): void {
    this.stopHeartbeat();
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }
}

/**
 * Singleton router instance for the Svelte application.
 */
export const wsRouter = new TraumaTeamWebSocketRouter({
  autoReconnect: false // Managed explicitly by UI toggle or test runner
});

/**
 * Resets router diagnostic stores for isolated test runs.
 */
export function resetRouterStores(): void {
  connectionState.set('DISCONNECTED');
  lastSeq.set(0);
  seqGapCount.set(0);
  rttLatencyMs.set(null);
  subscribedTopics.set([]);
  lastGatewayError.set(null);
  envelopeStreamLog.set([]);
  kinematicsPresetStore.set(null);
}
