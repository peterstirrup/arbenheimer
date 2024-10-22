package entities

import (
	"time"

	"github.com/shopspring/decimal"
)

type Market struct {
	TradingPair     string // e.g. "BTC/USDT"
	Exchange        Exchange
	BestBuyPrice    decimal.Decimal
	BestSellPrice   decimal.Decimal
	LastTradedPrice decimal.Decimal
	Timestamp       time.Time // Timestamp of the market data
	Volume24hr      float64
}

type Exchange string

const (
	ExchangeBinance Exchange = "binance"
	ExchangeKuCoin  Exchange = "kucoin"
)

var Exchanges = []Exchange{
	ExchangeBinance,
	ExchangeKuCoin,
}

func (e Exchange) String() string {
	return string(e)
}
