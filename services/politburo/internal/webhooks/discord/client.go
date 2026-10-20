// Package discord posts a single Discord-ready content string to a webhook URL.
package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"infinite-experiment/politburo/internal/metrics"
)

const postTimeout = 10 * time.Second

type payload struct {
	Content string `json:"content"`
}

// Client is a thin HTTP poster. Callers pass one message per request.
type Client struct {
	http    *http.Client
	metrics *metrics.Registry
}

func NewClient(reg *metrics.Registry) *Client {
	return &Client{
		http: &http.Client{
			Timeout: postTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		metrics: reg,
	}
}

// PostContent sends one JSON body with content. webhookType is a metric label only and is not logged with the URL.
func (c *Client) PostContent(ctx context.Context, webhookType, webhookURL, content string) error {
	body, err := json.Marshal(payload{Content: content})
	if err != nil {
		return fmt.Errorf("encode discord webhook body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return errors.New("discord webhook post: invalid url")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		c.metrics.ObserveWebhookDiscordPost(webhookType, "error")
		return errors.New("discord webhook post: request failed")
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.metrics.ObserveWebhookDiscordPost(webhookType, "error")
		return fmt.Errorf("discord webhook post: status %d", resp.StatusCode)
	}
	c.metrics.ObserveWebhookDiscordPost(webhookType, "success")
	return nil
}
