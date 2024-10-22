package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/peterstirrup/arbenheimer/internal/inbound/server/pb"
	"google.golang.org/grpc"
)

const grpcServerAddress = "localhost:9000" // The address of your gRPC server

// MarketResponse is a struct that mirrors the gRPC response to return as JSON
type MarketResponse struct {
	Markets []*pb.Market `json:"markets"`
}

func main() {
	http.HandleFunc("/market", handleMarketRequest)

	log.Println("HTTP server listening on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("failed to start HTTP server: %v", err)
	}
}

func handleMarketRequest(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight requests
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8000")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Add CORS headers for the actual request
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8000")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Parse the request body to get the trading pair
	var req struct {
		TradingPair string `json:"trading_pair"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Connect to the gRPC server
	conn, err := grpc.Dial(grpcServerAddress, grpc.WithInsecure())
	if err != nil {
		http.Error(w, "Failed to connect to gRPC server", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	client := pb.NewArbenheimerServiceClient(conn)

	// Create a gRPC context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// Send the gRPC request
	grpcReq := &pb.GetMarketRequest{TradingPair: req.TradingPair}
	resp, err := client.GetMarket(ctx, grpcReq)
	if err != nil {
		http.Error(w, "Failed to get market data from gRPC server", http.StatusInternalServerError)
		return
	}

	// Respond with the gRPC data as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MarketResponse{Markets: resp.Markets})
}
