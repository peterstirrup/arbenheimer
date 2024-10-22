package kucoin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/peterstirrup/arbenheimer/internal/domain/entities"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

type timeNow func() time.Time

type WebSocket interface {
	Close() error
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
}

type WebsocketClientConfig struct {
	Hostname     string
	HTTPClient   http.Client
	TradingPairs []string // e.g. ["BTC/BUSD", "ETH/BUSD"]
	UseCases     MarketUpdaterUseCases
	TimeNow      timeNow
}

type WebsocketClient struct {
	kucoinSymbolToPair map[string]string // BASE-QUOTE --> BASE/QUOTE
	hostname           string
	httpClient         http.Client
	timeNow            timeNow
	useCases           MarketUpdaterUseCases

	// Set when the websocket starts.
	pingInterval time.Duration
	ws           WebSocket
}

// NewWebsocket creates a new KuCoin websocket client.
func NewWebsocket(cfg WebsocketClientConfig) (*WebsocketClient, error) {
	c := &WebsocketClient{
		hostname:           cfg.Hostname,
		httpClient:         cfg.HTTPClient,
		kucoinSymbolToPair: make(map[string]string),
		useCases:           cfg.UseCases,
		timeNow:            cfg.TimeNow,
	}

	for _, pair := range cfg.TradingPairs {
		s := strings.Split(pair, "/")
		if len(s) != 2 {
			return nil, fmt.Errorf("invalid pair %s", pair)
		}

		c.kucoinSymbolToPair[strings.ToUpper(fmt.Sprintf("%s-%s", s[0], s[1]))] = pair
	}

	return c, nil
}

// Run starts the websocket client and blocks until the context is cancelled.
// Attempts to reconnect on any error.
func (c *WebsocketClient) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("context canceled, stopping websocket run")
			return ctx.Err()
		default:
		}

		if err := c.init(ctx); err != nil {
			return fmt.Errorf("failed to init websocket: %w", err)
		}

		if err := c.listen(ctx); err != nil {
			log.Err(err).Msg("Error while listening to websocket")
		}

		if err := c.ws.Close(); err != nil {
			log.Err(err).Msg("Error closing WebSocket")
		}
	}
}

// listen to the websocket, updating markets when a message is received.
// Returns an error if the context is cancelled.
func (c *WebsocketClient) listen(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Context canceled, stopping websocket listener")
			return ctx.Err()
		default:
			messageType, p, err := c.ws.ReadMessage()
			if err != nil {
				log.Err(err).Msg("Failed to read message")
				return err
			}

			if messageType == websocket.PongMessage {
				continue
			}

			var msg wsResponse
			if err = json.Unmarshal(p, &msg); err != nil {
				log.Err(err).Interface("msg", string(p)).Msg("Failed to unmarshal response")
				continue
			}

			// Filter based on the "/market/snapshot" topic
			if !strings.HasPrefix(msg.Topic, "/market/snapshot:") {
				continue
			}

			// Extract the symbol part from the topic
			symbol := strings.TrimPrefix(msg.Topic, "/market/snapshot:")

			pair, ok := c.kucoinSymbolToPair[symbol]
			if !ok {
				log.Warn().Msgf("Received data for unknown symbol: %s", symbol)
				continue
			}

			lastTradedPrice := decimal.NewFromFloat(msg.Data.Market.LastTradedPrice)
			buyPrice := decimal.NewFromFloat(msg.Data.Market.Buy)
			sellPrice := decimal.NewFromFloat(msg.Data.Market.Sell)

			err = c.useCases.UpdateMarket(ctx, entities.Market{
				TradingPair:     pair,
				Exchange:        entities.ExchangeKuCoin,
				BestBuyPrice:    buyPrice,
				BestSellPrice:   sellPrice,
				LastTradedPrice: lastTradedPrice,
				Timestamp:       time.UnixMilli(msg.Data.Market.Datetime),
				Volume24hr:      msg.Data.Market.VolValue,
			})
			if err != nil {
				log.Err(err).Interface("msg", msg).Msg("Failed to update market")
			}
		}
	}
}

type wsRequest struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type wsResponse struct {
	Type    string   `json:"type"`
	Topic   string   `json:"topic"`
	Subject string   `json:"subject"`
	Data    snapshot `json:"data"`
}

type snapshot struct {
	Sequence string `json:"sequence"`
	Market   market `json:"data"`
}

type market struct {
	Buy             float64 `json:"buy"`
	Datetime        int64   `json:"datetime"`
	LastTradedPrice float64 `json:"lastTradedPrice"`
	Open            float64 `json:"open"` // Todo check this
	Sell            float64 `json:"sell"`
	VolValue        float64 `json:"volValue"`
}
