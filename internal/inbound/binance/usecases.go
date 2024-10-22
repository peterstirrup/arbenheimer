package binance

import (
	"context"

	"github.com/peterstirrup/arbenheimer/internal/domain/entities"
)

type MarketUpdaterUseCases interface {
	UpdateMarket(ctx context.Context, market entities.Market) error
}
