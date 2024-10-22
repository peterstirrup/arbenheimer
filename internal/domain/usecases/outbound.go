package usecases

import (
	"context"

	"github.com/peterstirrup/arbenheimer/internal/domain/entities"
)

type Store interface {
	GetMarket(ctx context.Context, exchange entities.Exchange, tradingPair string) (entities.Market, error)
	UpdateMarket(ctx context.Context, market entities.Market) error
}
