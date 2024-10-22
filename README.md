# Arbenheimer

## Overview

A service that provides real-time market data for various cryptocurrency trading pairs across multiple exchanges. Provides:
- Latest traded price
- Best buy/sell price
- 24-hour trading volumes

### Naming

[Arbitrage trading](https://www.coinbase.com/en-ca/learn/advanced-trading/what-is-crypto-arbitrage-trading) + [Barbenheimer phenomenon](https://en.wikipedia.org/wiki/Barbenheimer) (Made the repo in 2023)

### Go Pattern

I developed this pattern based on Hexagonal/Clean Architecture, described here: [Keep Your Logic Agnostic: Hexagonal Architecture and Go](https://medium.com/@peterstirrup/keep-your-logic-agnostic-hexagonal-architecture-and-go-451af21b77e9).

I've implemented it early on in several startups and it's worked quite well. The separation of concerns is perfect for the chaos of web3 - swapping out one exchange for another has saved my skin more than once!

## Features

- **Real-time Market Data**: Provides up-to-date information on trading pairs across exchanges.
- **Support for Multiple Exchanges**: Currently supports Binance and KuCoin with the ability to extend to other exchanges.
- **gRPC Interface**: Clients can interact with the service via gRPC.
- **WebSocket Integration**: Leverages WebSockets to keep market data up to date.

## Installation

### Prerequisites

- Go 1.21 or later
- Docker
- Redis
- A Binance API key

### Clone and install dependencies

```bash
git clone https://github.com/peterstirrup/arbenheimer.git
cd arbenheimer
go mod download
```

### Generate Protobuf files

```bash
make proto
```

This generates Go code from the .proto definitions located in internal/proto.

### Use Binance API key

In docker-compose.yaml:

```yaml
environment:
  - BINANCE_API_KEY={keyhere}
```

## Run

### Docker

Build and run the service with Docker:

```bash
docker-compose build --no-cache
docker-compose up
```

This will start the server, binanceupdater, kucoinupdater and redis.

### Calling the service

I use BloomRPC, but you can use any gRPC client. An example of a request is:

```json
{
  "trading_pair": "BTC/USDT"
}
```

Returns:

```json
{
  "markets": [
    {
      "trading_pair": "BTC/USDT",
      "exchange": "binance",
      "timestamp": {
        "seconds": "1729572014",
        "nanos": 960000000
      },
      "last_traded_price": "67383.91",
      "best_buy_price": "67383.9",
      "best_sell_price": "67383.91",
      "volume_24hr": "2155122152.5424314"
    },
    {
      "trading_pair": "BTC/USDT",
      "exchange": "kucoin",
      "timestamp": {
        "seconds": "1729572014",
        "nanos": 136000000
      },
      "last_traded_price": "67378.5",
      "best_buy_price": "67385.2",
      "best_sell_price": "67385.3",
      "volume_24hr": "140391160.91314483"
    }
  ]
}
```

### Application

~~ChatGPT~~ I created a simple application that uses the service to display real-time market data.

```bash
cd testapp
go run grpc_proxy.go # Converts gRPC to REST
```

```bash
cd testapp
python3 -m http.server
```

Go to `localhost:8000` in your browser.

## Todo

- Use `go generate ./...` to generate Protobuf Go code.
- Endpoint to get trading pairs with the highest variance between exchanges.
- Trading bot to perform the trades - another microservice.
- Integration tests.
- Unit tests at server and Redis level.
- Add more exchanges!
- Add more trading pairs!