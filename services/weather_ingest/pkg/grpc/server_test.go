package grpc

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"traumateam/weather_ingest/pkg/noaa"
	"traumateam/weather_ingest/pkg/pb"
)

const bufSize = 1024 * 1024

func setupTestGRPC(t *testing.T) (*grpc.ClientConn, func()) {
	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	client := noaa.NewWeatherFeedClient()
	server := NewWeatherServer(client, nil)
	pb.RegisterWeatherIngestServiceServer(s, server)

	go func() {
		if err := s.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			t.Logf("bufconn server stopped: %v", err)
		}
	}()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}

	cleanup := func() {
		conn.Close()
		s.GracefulStop()
		lis.Close()
	}

	return conn, cleanup
}

func TestGRPC_GetCurrentConditions(t *testing.T) {
	conn, cleanup := setupTestGRPC(t)
	defer cleanup()

	client := pb.NewWeatherIngestServiceClient(conn)
	ctx := context.Background()

	// 1. Valid station KAPA
	resp, err := client.GetCurrentConditions(ctx, &pb.StationConditionsRequest{
		StationId: "KAPA",
	})
	if err != nil {
		t.Fatalf("GetCurrentConditions KAPA failed: %v", err)
	}
	if resp.StationId != "KAPA" {
		t.Errorf("expected KAPA, got %s", resp.StationId)
	}
	if resp.Metar == nil || resp.Metar.StationId != "KAPA" {
		t.Errorf("expected METAR for KAPA, got %v", resp.Metar)
	}
	if resp.Metar.FlightCategory != pb.FlightRuleCategory_CATEGORY_VFR {
		t.Errorf("expected VFR, got %v", resp.Metar.FlightCategory)
	}

	// 2. Empty station ID returns InvalidArgument
	_, errEmpty := client.GetCurrentConditions(ctx, &pb.StationConditionsRequest{
		StationId: "",
	})
	if errEmpty == nil {
		t.Errorf("expected error for empty station_id")
	}
}

func TestGRPC_StreamWeatherFeeds(t *testing.T) {
	conn, cleanup := setupTestGRPC(t)
	defer cleanup()

	client := pb.NewWeatherIngestServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, err := client.StreamWeatherFeeds(ctx, &pb.StreamWeatherRequest{
		Stations:               []string{"KAPA", "KLXV"},
		RefreshIntervalSeconds: 1,
	})
	if err != nil {
		t.Fatalf("StreamWeatherFeeds failed: %v", err)
	}

	// Receive first update
	update, err := stream.Recv()
	if err != nil {
		t.Fatalf("failed to receive stream update: %v", err)
	}
	if update.StationId != "KAPA" && update.StationId != "KLXV" {
		t.Errorf("expected KAPA or KLXV, got %s", update.StationId)
	}
	if update.Metar == nil {
		t.Errorf("expected populated METAR in stream update")
	}
}

func TestGRPC_EvaluateFlightMinimums(t *testing.T) {
	conn, cleanup := setupTestGRPC(t)
	defer cleanup()

	client := pb.NewWeatherIngestServiceClient(conn)
	ctx := context.Background()

	// Case 1: Mountainous Night Cross-Country to Aspen (KASE ceiling 900 ft vs 1000 ft req)
	resp, err := client.EvaluateFlightMinimums(ctx, &pb.FlightMinimumsEvaluationRequest{
		DepartureStation:   "KAPA",
		DestinationStation: "KASE",
		IsDay:              false,
		IsMountainous:      true,
		IsLocal:            false,
		Equipment: &pb.EquipmentFlags{
			NvgEquipped:   true,
			HtawsEquipped: true,
		},
	})
	if err != nil {
		t.Fatalf("EvaluateFlightMinimums failed: %v", err)
	}

	// Active convective SIGMET or Aspen 900 ft ceiling below minimums
	if resp.IsCompliant {
		t.Errorf("expected non-compliant for KASE 900 ft at night")
	}
	if resp.WorstCeilingFt > 1000.0 {
		t.Errorf("expected worst ceiling <= 1000 ft, got %.0f", resp.WorstCeilingFt)
	}
	if len(resp.StationReports) != 2 {
		t.Errorf("expected 2 station reports (KAPA, KASE), got %d", len(resp.StationReports))
	}

	// Case 2: IFR transition authorized under § 135.611
	respIFR, err := client.EvaluateFlightMinimums(ctx, &pb.FlightMinimumsEvaluationRequest{
		DepartureStation:       "KAPA",
		DestinationStation:     "KASE",
		IsDay:                  false,
		IsMountainous:          true,
		IsLocal:                false,
		IsIfrCertified:         true,
		HasIfrAlternate:        true,
		HasSevereWeatherRadar:  true,
		Equipment: &pb.EquipmentFlags{
			NvgEquipped:   true,
			HtawsEquipped: true,
		},
	})
	if err != nil {
		t.Fatalf("EvaluateFlightMinimums IFR failed: %v", err)
	}
	// If active convective SIGMET is preloaded, it will be DISQUALIFIED; otherwise IFR_REQUIRED
	if respIFR.Verdict != pb.DispatchVerdict_VERDICT_IFR_REQUIRED && respIFR.Verdict != pb.DispatchVerdict_VERDICT_DISQUALIFIED {
		t.Errorf("expected IFR_REQUIRED or DISQUALIFIED, got %v", respIFR.Verdict)
	}
}

func TestGRPC_StreamFlightMinimums(t *testing.T) {
	conn, cleanup := setupTestGRPC(t)
	defer cleanup()

	client := pb.NewWeatherIngestServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, err := client.StreamFlightMinimums(ctx, &pb.StreamFlightMinimumsRequest{
		Parameters: &pb.FlightMinimumsEvaluationRequest{
			DepartureStation:   "KAPA",
			DestinationStation: "KLXV",
			IsDay:              true,
			IsMountainous:      false,
			IsLocal:            true,
		},
		RefreshIntervalSeconds: 1,
	})
	if err != nil {
		t.Fatalf("StreamFlightMinimums failed: %v", err)
	}

	update, err := stream.Recv()
	if err != nil && err != io.EOF {
		t.Fatalf("failed to receive flight minimums update: %v", err)
	}
	if update != nil {
		if update.Evaluation == nil {
			t.Errorf("expected populated evaluation in update")
		}
	}
}
