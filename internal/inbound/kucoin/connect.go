package kucoin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// init gets the websocket connection info, starts the connection, and subscribes to the market topics.
func (c *WebsocketClient) init(ctx context.Context) error {
	url, token, pingInterval, err := c.getWebsocketData(ctx)
	if err != nil {
		return err
	}

	ws, resp, err := websocket.DefaultDialer.Dial(fmt.Sprintf("%s?token=%s", url, token), nil)
	if err != nil {
		return fmt.Errorf("failed to dial websocket: %w", err)
	}
	resp.Body.Close()

	c.ws = ws
	c.pingInterval = pingInterval

	go c.keepAlive(ctx)

	return c.subscribe()
}

// getWebsocketData calls KuCoin BulletPublic API to get a valid endpoint and token to connect to their websocket
// It returns the endpoint, the token, the ping interval in milliseconds and an error, if any
func (c *WebsocketClient) getWebsocketData(ctx context.Context) (string, string, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.hostname+BulletPublicRoute, nil)
	if err != nil {
		return "", "", 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to get websocket connection data: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", 0, err
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", 0, fmt.Errorf("non-ok status: %d", resp.StatusCode)
	}

	bulletResp := bulletPublicResponse{}
	if err = json.Unmarshal(body, &bulletResp); err != nil {
		return "", "", 0, err
	}

	if len(bulletResp.Data.InstanceServers) == 0 {
		return "", "", 0, fmt.Errorf("no instance servers found in response")
	}

	pingInterval := time.Duration(bulletResp.Data.InstanceServers[0].PingInterval) * time.Millisecond

	return bulletResp.Data.InstanceServers[0].Endpoint, bulletResp.Data.Token, pingInterval, nil
}

const (
	BulletPublicRoute = "/api/v1/bullet-public"
)

type bulletPublicResponse struct {
	Data struct {
		Token           string `json:"token"`
		InstanceServers []struct {
			Endpoint     string `json:"endpoint"`
			PingInterval int    `json:"pingInterval"`
		} `json:"instanceServers"`
	} `json:"data"`
}

// subscribe sends a message to the websocket connection subscribing to each trading pair's market topic.
func (c *WebsocketClient) subscribe() error {
	// Iterate over each trading pair and create a subscription message
	for symbol, pair := range c.kucoinSymbolToPair {
		topic := fmt.Sprintf("/market/snapshot:%s", symbol)

		msg := subscriptionRequest{
			wsRequest: wsRequest{
				ID:   strconv.FormatInt(c.timeNow().UnixNano(), 10),
				Type: "subscribe",
			},
			Topic:          topic,
			PrivateChannel: false,
			Response:       false,
		}

		m, err := json.Marshal(msg)
		if err != nil {
			log.Err(err).Msgf("Failed to marshall subscription message for trading pair: %s", pair)
			return err
		}

		if err = c.ws.WriteMessage(websocket.TextMessage, m); err != nil {
			log.Err(err).Msgf("Failed to subscribe to trading pair: %s", pair)
			return err
		}

		log.Info().Msgf("Subscribed to topic: %s", topic)
	}

	return nil
}

// ping sends a ping message to the websocket every pingInterval
func (c *WebsocketClient) keepAlive(ctx context.Context) {
	pt := time.NewTicker(c.pingInterval - 500*time.Millisecond)
	defer pt.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Context canceled, stopping ping")
			return
		case <-pt.C:
			msg, err := json.Marshal(wsRequest{
				ID:   strconv.FormatInt(c.timeNow().UnixNano(), 10),
				Type: "ping",
			})
			if err != nil {
				log.Err(err).Msg("Failed to marshall ping message")
			}

			if err = c.ws.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Err(err).Msg("Failed to send ping message")
			}
		}
	}
}

type subscriptionRequest struct {
	wsRequest
	Topic          string `json:"topic"`
	PrivateChannel bool   `json:"privateChannel"`
	Response       bool   `json:"response"`
}
