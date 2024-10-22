package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/peterstirrup/arbenheimer/internal/domain/entities"
	"github.com/peterstirrup/arbenheimer/internal/domain/errors"
	"github.com/redis/go-redis/v9"
)

type Client struct {
	marketTTL time.Duration
	rc        *redis.Client
}

type Config struct {
	Host      string
	MarketTTL time.Duration
	Port      string
}

func NewClient(cfg Config) *Client {
	client := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
	})

	if cfg.MarketTTL == 0 {
		// Default
		cfg.MarketTTL = 10 * time.Minute
	}

	return &Client{
		marketTTL: cfg.MarketTTL,
		rc:        client,
	}
}

// GetMarket retrieves market data from Redis.
func (c *Client) GetMarket(ctx context.Context, exchange entities.Exchange, tradingPair string) (entities.Market, error) {
	redisKey := "market:" + string(exchange) + ":" + tradingPair

	v, err := c.rc.Get(ctx, redisKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return entities.Market{}, fmt.Errorf("%w for %s on %s", arberrors.ErrMarketNotFound, tradingPair, exchange)
		}
		return entities.Market{}, err
	}

	var market entities.Market
	err = json.Unmarshal([]byte(v), &market)
	if err != nil {
		return entities.Market{}, err
	}

	return market, nil
}

// UpdateMarket stores market data in Redis.
func (c *Client) UpdateMarket(ctx context.Context, market entities.Market) error {
	// Serialize Market struct to JSON
	marketData, err := json.Marshal(market)
	if err != nil {
		return err
	}

	redisKey := "market:" + string(market.Exchange) + ":" + market.TradingPair

	// Price probably useless after 10 minutes
	err = c.rc.Set(ctx, redisKey, marketData, c.marketTTL).Err()
	if err != nil {
		return err
	}

	return nil
}
