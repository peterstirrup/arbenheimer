package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/peterstirrup/arbenheimer/internal/domain/entities"
	"github.com/peterstirrup/arbenheimer/internal/domain/usecases"
	"github.com/peterstirrup/arbenheimer/internal/inbound/binance"
	"github.com/peterstirrup/arbenheimer/internal/outbound/redis"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type cliArgs struct {
	BinanceAPIKey       string        `arg:"required,env:BINANCE_API_KEY"`
	BinanceHostname     string        `arg:"required,env:BINANCE_HOSTNAME"`
	BinanceWebsocketURL string        `arg:"required,env:BINANCE_WEBSOCKET_URL"`
	HTTPClientTimeout   time.Duration `arg:"env:HTTP_CLIENT_TIMEOUT" default:"10s"`
	LogLevel            string        `arg:"--log-level,env:LOG_LEVEL" default:"debug"`
	RedisHost           string        `arg:"--redis-host,required,env:REDIS_HOST"`
	RedisPort           string        `arg:"--redis-port,required,env:REDIS_PORT"`
}

func main() {
	var args cliArgs
	arg.MustParse(&args)

	ctx := context.Background()

	logLevel, err := zerolog.ParseLevel(args.LogLevel)
	if err != nil {
		log.Warn().Msg("failed to parse log level, defaulting to debug")
		logLevel = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(logLevel)

	pairs, err := getPairsForExchange(entities.ExchangeBinance)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get trading pairs for exchange")
	}

	rc := redis.NewClient(redis.Config{Host: args.RedisHost, Port: args.RedisPort})

	u := usecases.NewMarket(usecases.MarketConfig{
		Store:   rc,
		TimeNow: time.Now,
	})

	ws, err := binance.NewWebsocket(binance.WebsocketClientConfig{
		APIKey:   args.BinanceAPIKey,
		Hostname: args.BinanceHostname,
		HTTPClient: http.Client{
			Timeout: args.HTTPClientTimeout,
		},
		TradingPairs: pairs,
		UseCases:     u,
		WebsocketURL: args.BinanceWebsocketURL,
	})

	if err := ws.Run(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to run websocket client")
	}
}

func getPairsForExchange(e entities.Exchange) ([]string, error) {
	file, err := os.Open("data/trading_pairs.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to open trading_pairs.yaml")
	}
	defer file.Close()

	var cfg config
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to decode yaml")
	}

	for _, ex := range cfg.Exchanges {
		if ex.Name == e.String() {
			return ex.Pairs, nil
		}
	}

	return nil, fmt.Errorf("exchange not specified")
}

type exchange struct {
	Name  string   `yaml:"name"`
	Pairs []string `yaml:"pairs"`
}

type config struct {
	Exchanges []exchange `yaml:"exchanges"`
}
