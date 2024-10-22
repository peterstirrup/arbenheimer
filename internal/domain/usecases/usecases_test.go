package usecases_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/peterstirrup/arbenheimer/internal/domain/entities"
	arberrors "github.com/peterstirrup/arbenheimer/internal/domain/errors"
	"github.com/peterstirrup/arbenheimer/internal/domain/usecases"
	"github.com/peterstirrup/arbenheimer/internal/domain/usecases/mocks"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

var (
	ctx      = context.Background()
	testTime = time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	marketBinance = entities.Market{
		Exchange:        entities.ExchangeBinance,
		TradingPair:     "BTC/USDT",
		BestBuyPrice:    decimal.NewFromFloat(69000),
		BestSellPrice:   decimal.NewFromFloat(70000),
		LastTradedPrice: decimal.NewFromFloat(69500),
		Timestamp:       testTime,
		Volume24hr:      1000000,
	}
	marketKuCoin = entities.Market{
		Exchange:        entities.ExchangeBinance,
		TradingPair:     "BTC/USDT",
		BestBuyPrice:    decimal.NewFromFloat(69500),
		BestSellPrice:   decimal.NewFromFloat(71000),
		LastTradedPrice: decimal.NewFromFloat(69600),
		Timestamp:       testTime,
		Volume24hr:      2000000,
	}
)

type setupMarketTestConfig struct {
	mockCtrl *gomock.Controller
	store    *mocks.MockStore

	market *usecases.Market
}

func setupTest(t *testing.T) *setupMarketTestConfig {
	ctrl := gomock.NewController(t)
	store := mocks.NewMockStore(ctrl)

	return &setupMarketTestConfig{
		mockCtrl: ctrl,
		store:    store,
		market: usecases.NewMarket(usecases.MarketConfig{
			Store:   store,
			TimeNow: func() time.Time { return testTime },
		}),
	}
}

func TestMarket_GetMarkets(t *testing.T) {
	t.Run("gets markets successfully", func(t *testing.T) {
		cfg := setupTest(t)

		markets := []entities.Market{marketBinance, marketKuCoin}

		cfg.store.EXPECT().GetMarket(ctx, entities.ExchangeBinance, "BTC/USDT").Return(markets[0], nil)
		cfg.store.EXPECT().GetMarket(ctx, entities.ExchangeKuCoin, "BTC/USDT").Return(markets[1], nil)

		resp, err := cfg.market.GetMarkets(ctx, "BTC/USDT")
		require.NoError(t, err)
		require.Len(t, resp, 2)
		require.Equal(t, markets, resp)
	})

	t.Run("fails to get markets for one exchange, returns found markets", func(t *testing.T) {
		cfg := setupTest(t)

		markets := []entities.Market{marketKuCoin}

		cfg.store.EXPECT().GetMarket(ctx, entities.ExchangeBinance, "BTC/USDT").Return(entities.Market{}, arberrors.ErrMarketNotFound)
		cfg.store.EXPECT().GetMarket(ctx, entities.ExchangeKuCoin, "BTC/USDT").Return(markets[0], nil)

		resp, err := cfg.market.GetMarkets(ctx, "BTC/USDT")
		require.NoError(t, err)
		require.Len(t, resp, 1)
		require.Equal(t, markets, resp)
	})

	t.Run("fails to get markets for all exchanges, returns error", func(t *testing.T) {
		cfg := setupTest(t)

		cfg.store.EXPECT().GetMarket(ctx, entities.ExchangeBinance, "BTC/USDT").Return(entities.Market{}, arberrors.ErrMarketNotFound)
		cfg.store.EXPECT().GetMarket(ctx, entities.ExchangeKuCoin, "BTC/USDT").Return(entities.Market{}, arberrors.ErrMarketNotFound)

		_, err := cfg.market.GetMarkets(ctx, "BTC/USDT")
		require.ErrorIs(t, err, arberrors.ErrMarketNotFound)
	})
}

func TestMarket_UpdateMarket(t *testing.T) {
	t.Run("updates market successfully when doesn't exist in store", func(t *testing.T) {
		cfg := setupTest(t)

		cfg.store.EXPECT().GetMarket(ctx, marketBinance.Exchange, marketBinance.TradingPair).Return(entities.Market{}, arberrors.ErrMarketNotFound)
		cfg.store.EXPECT().UpdateMarket(ctx, marketBinance).Return(nil)

		err := cfg.market.UpdateMarket(ctx, marketBinance)
		require.NoError(t, err)
	})

	t.Run("updates market successfully, old market exists in store", func(t *testing.T) {
		cfg := setupTest(t)

		oldMarket := marketBinance
		oldMarket.Timestamp = testTime.Add(-time.Second)

		cfg.store.EXPECT().GetMarket(ctx, marketBinance.Exchange, marketBinance.TradingPair).Return(oldMarket, nil)
		cfg.store.EXPECT().UpdateMarket(ctx, marketBinance).Return(nil)

		err := cfg.market.UpdateMarket(ctx, marketBinance)
		require.NoError(t, err)
	})

	t.Run("fails to update market, current market has later timestamp than new", func(t *testing.T) {
		cfg := setupTest(t)

		oldMarket := marketBinance
		oldMarket.Timestamp = testTime.Add(time.Second)

		cfg.store.EXPECT().GetMarket(ctx, marketBinance.Exchange, marketBinance.TradingPair).Return(oldMarket, nil)

		err := cfg.market.UpdateMarket(ctx, marketBinance)
		require.Error(t, err)
	})
}
