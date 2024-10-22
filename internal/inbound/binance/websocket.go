package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/peterstirrup/arbenheimer/internal/domain/entities"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

type WebSocket interface {
	Close() error
	ReadMessage() (messageType int, p []byte, err error)
	WriteJSON(v interface{}) error
}

type WebsocketClientConfig struct {
	APIKey       string
	Hostname     string
	HTTPClient   http.Client
	PingInterval time.Duration
	TradingPairs []string // e.g. ["BTC/BUSD", "ETH/BUSD"]
	UseCases     MarketUpdaterUseCases
	WebsocketURL string
}

type WebsocketClient struct {
	apiKey              string
	binanceSymbolToPair map[string]string // BASEQUOTE --> BASE/QUOTE
	hostname            string
	httpClient          http.Client
	listenKey           string
	pingInterval        time.Duration
	useCases            MarketUpdaterUseCases
	websocketURL        string
	ws                  WebSocket
}

func NewWebsocket(cfg WebsocketClientConfig) (*WebsocketClient, error) {
	if cfg.PingInterval == 0 {
		// Default to 20 minutes, Binance requires a ping every 60 minutes
		cfg.PingInterval = 20 * time.Minute
	}

	c := &WebsocketClient{
		apiKey:              cfg.APIKey,
		binanceSymbolToPair: make(map[string]string),
		httpClient:          cfg.HTTPClient,
		hostname:            cfg.Hostname,
		pingInterval:        cfg.PingInterval,
		useCases:            cfg.UseCases,
		websocketURL:        cfg.WebsocketURL,
	}

	for _, pair := range cfg.TradingPairs {
		s := strings.Split(pair, "/")
		if len(s) != 2 {
			return nil, fmt.Errorf("invalid pair %s", pair)
		}

		c.binanceSymbolToPair[s[0]+s[1]] = pair
	}

	return c, nil
}

// Run listens to an opened connection to a websocket provided by Binance, subscribed to markets.
// It also reconnects if the websocket is closed by Binance.
// Every 60 minutes at most, we must ping Binance with a "ping" connection, so it's kept alive.
func (c *WebsocketClient) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Context canceled, stopping websocket run")
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
			log.Err(err).Msg("Error when closing WebSocket")
		}
	}
}

// listen for "24hrTicker" messages on Binance WebSocket and updates the price for the corresponding trading pair.
// If message is not a "24hrTicker", ignore. Listens until an error occurs or the context is cancelled.
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

			var msg ticker24hrEvent
			if err = json.Unmarshal(p, &msg); err != nil {
				log.Err(err).Interface("msg", string(p)).Msg("Failed to unmarshal msg to JSON")
				continue
			}

			if msg.Type != "24hrTicker" {
				continue
			}

			lastPrice, err := decimal.NewFromString(msg.LastPrice)
			if err != nil {
				log.Err(err).Interface("msg", msg).Msg("Failed to parse msg.LatestPrice")
				continue
			}

			bestBuyPrice, err := decimal.NewFromString(msg.BestBuyPrice)
			if err != nil {
				log.Err(err).Interface("msg", msg).Msg("Failed to parse msg.BestBuyPrice")
				continue
			}

			bestSellPrice, err := decimal.NewFromString(msg.BestSellPrice)
			if err != nil {
				log.Err(err).Interface("msg", msg).Msg("Failed to parse msg.BestSellPrice")
				continue
			}

			tradeAmount, err := strconv.ParseFloat(msg.TradeAmountInQuoteAsset, 64)
			if err != nil {
				log.Err(err).Interface("msg", msg).Msg("Failed to parse msg.TradeAmountInQuoteAsset")
				continue
			}

			pair, ok := c.binanceSymbolToPair[msg.Symbol]
			if !ok {
				log.Warn().Str("symbol", msg.Symbol).Interface("message", msg).Msg("Received data for unknown symbol")
				continue
			}

			err = c.useCases.UpdateMarket(ctx, entities.Market{
				TradingPair:     pair,
				Exchange:        entities.ExchangeBinance,
				BestBuyPrice:    bestBuyPrice,
				BestSellPrice:   bestSellPrice,
				LastTradedPrice: lastPrice,
				Timestamp:       time.UnixMilli(msg.Timestamp),
				Volume24hr:      tradeAmount,
			})
			if err != nil {
				log.Err(err).Interface("msg", msg).Msg("Failed to update market")
			}
		}
	}
}

type ticker24hrEvent struct {
	Type                    string      `json:"e"`
	Timestamp               int64       `json:"E"`
	Symbol                  string      `json:"s"`
	LastPrice               string      `json:"c"`
	BestBuyPrice            string      `json:"b"`
	BestSellPrice           string      `json:"a"`
	TradeAmountInQuoteAsset string      `json:"q"`
	UpperA                  interface{} `json:"A"` // Don't remove
	UpperB                  interface{} `json:"B"` // Don't remove
	UpperC                  interface{} `json:"C"` // Don't remove
	UpperQ                  interface{} `json:"Q"` // Don't remove
	UpperS                  string      `json:"S"` // Don't remove
}
