package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"traumateam/weather_ingest/pkg/gateway"
	grpcpkg "traumateam/weather_ingest/pkg/grpc"
	"traumateam/weather_ingest/pkg/noaa"
	"traumateam/weather_ingest/pkg/pb"
)

func main() {
	grpcPort := flag.Int("grpc-port", 50051, "gRPC server listening port")
	httpPort := flag.Int("http-port", 8080, "HTTP & WebSocket gateway listening port")
	flag.Parse()

	log.Println("================================================================================")
	log.Println(" TRAUMATEAM NOAA AWC WEATHER INGEST & WEBSOCKET GATEWAY SERVICE (GO/GRPC)")
	log.Println("================================================================================")

	// 1. Initialize core NOAA feed client & WebSocket broadcast Hub
	weatherClient := noaa.NewWeatherFeedClient()
	wsHub := gateway.NewHub()
	go wsHub.Run()

	// 2. Start gRPC Service on configured port
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", *grpcPort))
	if err != nil {
		log.Fatalf("Failed to bind gRPC port %d: %v", *grpcPort, err)
	}

	grpcServer := grpc.NewServer()
	weatherService := grpcpkg.NewWeatherServer(weatherClient, wsHub)
	pb.RegisterWeatherIngestServiceServer(grpcServer, weatherService)
	reflection.Register(grpcServer) // Enable gRPC reflection for client discovery

	go func() {
		log.Printf("[gRPC Server] Listening on :%d", *grpcPort)
		if err := grpcServer.Serve(grpcListener); err != nil && err != grpc.ErrServerStopped {
			log.Fatalf("gRPC server encountered fatal error: %v", err)
		}
	}()

	// 3. Start HTTP & WebSocket Gateway
	httpHandler := gateway.NewHandler(wsHub, weatherClient)
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", *httpPort),
		Handler: httpHandler,
	}

	go func() {
		log.Printf("[HTTP/WebSocket Gateway] Serving at http://localhost:%d (WS at /ws)", *httpPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server encountered fatal error: %v", err)
		}
	}()

	// 4. Background Periodic Feed Poller & Broadcaster
	stopPeriodic := make(chan struct{})
	go func() {
		stations := []string{"KAPA", "KDEN", "KLXV", "KASE"}
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stopPeriodic:
				return
			case <-ticker.C:
				for _, st := range stations {
					if metar, err := weatherClient.GetMETAR(context.Background(), st); err == nil {
						wsHub.BroadcastWeatherUpdate(st, metar)
					}
				}
			}
		}
	}()

	// 5. Coordinated Graceful Shutdown via sync.WaitGroup and OS signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("Received shutdown signal: %v. Initiating graceful shutdown...", sig)
	close(stopPeriodic)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		grpcServer.GracefulStop()
		log.Println("[gRPC Server] Gracefully stopped.")
	}()

	go func() {
		defer wg.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("[HTTP Gateway] Shutdown warning: %v", err)
		}
		log.Println("[HTTP Gateway] Gracefully stopped.")
	}()

	wg.Wait()
	log.Println("Dual-serving weather_ingest service terminated cleanly.")
}
