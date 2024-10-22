package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const (
	ListenKeyRoute = "/api/v3/userDataStream"
	APIKeyHeader   = "X-MBX-APIKEY"
)

// init initializes WebSocketClient by creating a listenKey, establishing a WebSocket connection,
// and subscribing to the desired trading pairs.
func (c *WebsocketClient) init(ctx context.Context) error {
	listenKey, err := c.getListenKey(ctx)
	if err != nil {
		return err
	}
	c.listenKey = listenKey

	go c.keepAliveListenKey(ctx)

	ws, resp, err := websocket.DefaultDialer.Dial(c.websocketURL+c.listenKey, nil)
	if err != nil {
		return fmt.Errorf("failed to dial websocket: %w", err)
	}
	resp.Body.Close()

	c.ws = ws

	return c.subscribe()
}

// subscribe subscribes to trading pairs via WebSocket.
func (c *WebsocketClient) subscribe() error {
	var params []string
	for pair := range c.binanceSymbolToPair {
		// Binance needs the pair formatted in lower case (e.g: btcbusd@ticker)
		params = append(params, fmt.Sprintf("%s@ticker", strings.ToLower(pair)))
	}

	err := c.ws.WriteJSON(&subscriptionRequest{
		Method: "SUBSCRIBE",
		Params: params,
		ID:     1,
	})

	return err
}

type subscriptionRequest struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	ID     int      `json:"id"`
}

// getListenKey creates and retrieves a listenKey from Binance.
func (c *WebsocketClient) getListenKey(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.hostname+ListenKeyRoute, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("X-MBX-APIKEY", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("non-ok status: %s", resp.Status)
	}

	listenKeyResp := createListenKeyResponse{}
	if err = json.Unmarshal(body, &listenKeyResp); err != nil {
		return "", err
	}

	return listenKeyResp.ListenKey, nil
}

// createListenKeyResponse represents the JSON returned by Binance when requesting a listenKey.
type createListenKeyResponse struct {
	ListenKey string `json:"listenKey"`
}

// keepAliveListenKey sends a PUT request to Binance to keep the listenKey alive.
// Binance requires periodic pings (every 60m) to keep the listenKey from expiring.
// If the request is not successful, returns an error.
func (c *WebsocketClient) keepAliveListenKey(ctx context.Context) {
	pt := time.NewTicker(c.pingInterval)
	defer pt.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Context canceled, stopping ping")
			return
		case <-pt.C:
			if err := c.pingListenKey(ctx); err != nil {
				log.Err(err).Msg("Failed to ping Binance listenKey")
			}
		}
	}
}

// pingListenKeyRequest refreshes the Binance listenKey by sending a PUT request.
func (c *WebsocketClient) pingListenKey(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.hostname+ListenKeyRoute, nil)
	if err != nil {
		return fmt.Errorf("error creating ping request: %w", err)
	}

	req.Header.Set(APIKeyHeader, c.apiKey)
	q := req.URL.Query()
	q.Add("listenKey", c.listenKey)
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending ping request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("binance ping listenKey returned status %s", resp.Status)
	}

	return nil
}
