package server

import (
	"context"
	"strconv"

	"github.com/peterstirrup/arbenheimer/internal/domain/entities"
	"github.com/peterstirrup/arbenheimer/internal/inbound/server/pb"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MarketUseCases interface {
	GetMarkets(ctx context.Context, tradingPair string) ([]entities.Market, error)
}

type Server struct {
	pb.UnimplementedArbenheimerServiceServer
	market MarketUseCases
}

type Config struct {
	MarketUseCases MarketUseCases
}

func NewServer(cfg Config) *Server {
	return &Server{
		market: cfg.MarketUseCases,
	}
}

// GetMarket retrieves market data for the given trading pair.
// If the trading pair is not found on any exchange, an error is returned.
func (s *Server) GetMarket(ctx context.Context, req *pb.GetMarketRequest) (*pb.GetMarketResponse, error) {
	log.Info().Msg("received GetMarket request")

	markets, err := s.market.GetMarkets(ctx, req.TradingPair)
	if err != nil {
		return nil, err
	}

	resp := &pb.GetMarketResponse{
		Markets: make([]*pb.Market, 0, len(markets)),
	}

	for _, m := range markets {
		resp.Markets = append(resp.Markets, &pb.Market{
			TradingPair:     m.TradingPair,
			Exchange:        m.Exchange.String(),
			Timestamp:       timestamppb.New(m.Timestamp),
			LastTradedPrice: m.LastTradedPrice.String(),
			BestBuyPrice:    m.BestBuyPrice.String(),
			BestSellPrice:   m.BestSellPrice.String(),
			Volume_24Hr:     strconv.FormatFloat(m.Volume24hr, 'f', -1, 64),
		})
	}

	return resp, nil
}
