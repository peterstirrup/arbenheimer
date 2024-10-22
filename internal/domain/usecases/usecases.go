package usecases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/peterstirrup/arbenheimer/internal/domain/entities"
	arberrors "github.com/peterstirrup/arbenheimer/internal/domain/errors"
	"github.com/rs/zerolog/log"
)

type Market struct {
	store   Store
	timeNow func() time.Time // Need to be deterministic for testing
}

type MarketConfig struct {
	Store   Store
	TimeNow func() time.Time
}

func NewMarket(cfg MarketConfig) *Market {
	return &Market{
		store:   cfg.Store,
		timeNow: cfg.TimeNow,
	}
}

// GetMarkets returns the market data for the given trading pair from all exchanges.
// If the trading pair is not found on any exchange, an error is returned.
func (m *Market) GetMarkets(ctx context.Context, tradingPair string) ([]entities.Market, error) {
	var markets []entities.Market
	var found bool

	for _, exchange := range entities.Exchanges {
		m, err := m.store.GetMarket(ctx, exchange, tradingPair)
		if err != nil {
			if !errors.Is(err, arberrors.ErrMarketNotFound) {
				log.Warn().Interface("exchange", exchange).Err(err).Msgf("failed to get market data for trading pair %s", tradingPair)
			}
			continue
		}

		markets = append(markets, m)
		found = true
	}

	if !found {
		return nil, arberrors.ErrMarketNotFound
	}

	return markets, nil
}

// UpdateMarket updates the market data in the store.
// If the market data is older than the current data in the store, it will not be updated.
func (m *Market) UpdateMarket(ctx context.Context, market entities.Market) error {
	currMarket, err := m.store.GetMarket(ctx, market.Exchange, market.TradingPair)
	if err != nil && !errors.Is(err, arberrors.ErrMarketNotFound) {
		return err
	}

	if currMarket.Timestamp.After(market.Timestamp) {
		return fmt.Errorf("%w: current market data is newer than the provided data", arberrors.ErrInvalidMarketTimestamp)
	}

	return m.store.UpdateMarket(ctx, market)
}
