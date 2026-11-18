package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"traumateam/weather_ingest/pkg/gateway"
	"traumateam/weather_ingest/pkg/grpc"
	"traumateam/weather_ingest/pkg/models"
	"traumateam/weather_ingest/pkg/noaa"
	"traumateam/weather_ingest/pkg/pb"
)

const defaultBufSize = 1024 * 1024

// E2ETestHarness encapsulates the complete offline gRPC and WebSocket execution environment.
type E2ETestHarness struct {
	T              *testing.T
	NOAAClient     *noaa.WeatherFeedClient
	Hub            *gateway.Hub
	GRPCServer     *googlegrpc.Server
	GRPCListener   *bufconn.Listener
	GRPCClientConn *googlegrpc.ClientConn
	GRPCClient     pb.WeatherIngestServiceClient
	HTTPServer     *httptest.Server
	WebSocketURL   string
}

// NewE2ETestHarness initializes an in-memory gRPC and WebSocket test harness.
func NewE2ETestHarness(t *testing.T) *E2ETestHarness {
	// 1. Core NOAA Client & WebSocket Hub
	feedClient := noaa.NewWeatherFeedClient()
	hub := gateway.NewHub()
	go hub.Run()

	// 2. In-Memory gRPC Server (bufconn)
	lis := bufconn.Listen(defaultBufSize)
	grpcServer := googlegrpc.NewServer()
	weatherService := grpc.NewWeatherServer(feedClient, hub)
	pb.RegisterWeatherIngestServiceServer(grpcServer, weatherService)

	go func() {
		if err := grpcServer.Serve(lis); err != nil && err != googlegrpc.ErrServerStopped {
			t.Logf("[Harness] gRPC server stopped: %v", err)
		}
	}()

	// 3. gRPC Client dialed over in-memory pipe
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := googlegrpc.DialContext(ctx, "passthrough://bufnet",
		googlegrpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		googlegrpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("[Harness] Failed to dial bufnet: %v", err)
	}
	grpcClient := pb.NewWeatherIngestServiceClient(conn)

	// 4. In-Memory WebSocket HTTP Server (httptest)
	httpServer := httptest.NewServer(gateway.NewHandler(hub, feedClient))
	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/ws"

	return &E2ETestHarness{
		T:              t,
		NOAAClient:     feedClient,
		Hub:            hub,
		GRPCServer:     grpcServer,
		GRPCListener:   lis,
		GRPCClientConn: conn,
		GRPCClient:     grpcClient,
		HTTPServer:     httpServer,
		WebSocketURL:   wsURL,
	}
}

// DialWebSocket establishes an authenticated test WebSocket connection.
func (h *E2ETestHarness) DialWebSocket() (*websocket.Conn, error) {
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(h.WebSocketURL, nil)
	if err != nil {
		return nil, fmt.Errorf("websocket dial failed: %w", err)
	}
	return conn, nil
}

// ReadWSEvent reads and parses the next JSON event envelope from the WebSocket.
func (h *E2ETestHarness) ReadWSEvent(conn *websocket.Conn, timeout time.Duration) (*gateway.ServerEnvelope, error) {
	conn.SetReadDeadline(time.Now().Add(timeout))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	var evt gateway.ServerEnvelope
	if err := json.Unmarshal(msg, &evt); err != nil {
		return nil, fmt.Errorf("json unmarshal failed on msg %q: %w", string(msg), err)
	}
	return &evt, nil
}

// WriteClientMessage writes a JSON client message envelope to the WebSocket.
func (h *E2ETestHarness) WriteClientMessage(conn *websocket.Conn, msg gateway.ClientEnvelope) error {
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	return conn.WriteJSON(msg)
}

// SetStationObservation injects a raw METAR into the mock feed client.
func (h *E2ETestHarness) SetStationObservation(stationID, rawMETAR string) {
	_ = h.NOAAClient.SetMETAR(stationID, rawMETAR)
}

// ClearHazards removes all preloaded hazards to simulate a storm-free corridor.
func (h *E2ETestHarness) ClearHazards() {
	h.NOAAClient.ClearHazards()
}

// AddHazard registers a SIGMET hazard into the harness feed client.
func (h *E2ETestHarness) AddHazard(hazard *models.WeatherHazard) {
	h.NOAAClient.AddHazard(hazard)
}

// Teardown cleanly stops servers and closes in-memory channels.
func (h *E2ETestHarness) Teardown() {
	if h.GRPCClientConn != nil {
		h.GRPCClientConn.Close()
	}
	if h.GRPCServer != nil {
		h.GRPCServer.GracefulStop()
	}
	if h.GRPCListener != nil {
		h.GRPCListener.Close()
	}
	if h.HTTPServer != nil {
		h.HTTPServer.Close()
	}
}
