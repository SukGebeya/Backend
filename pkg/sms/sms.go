// Package sms sends text messages to Ethiopian mobile numbers.
package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/SukGebeya/Backend/pkg/phone"
)

type Sender interface {
	Send(ctx context.Context, to, message string) error
}

// LogSender writes messages to the log instead of delivering them (development only).
type LogSender struct{}

func (LogSender) Send(ctx context.Context, to, message string) error {
	slog.InfoContext(ctx, "[SMS]", "to", to, "message", message)
	return nil
}

type AfroMessageConfig struct {
	BaseURL      string
	Token        string
	Sender       string
	IdentifierID string
}

// AfroMessageSender delivers SMS through AfroMessage, an Ethiopian SMS gateway.
type AfroMessageSender struct {
	cfg        AfroMessageConfig
	httpClient *http.Client
}

func NewAfroMessageSender(cfg AfroMessageConfig) *AfroMessageSender {
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return &AfroMessageSender{cfg: cfg, httpClient: &http.Client{Timeout: 15 * time.Second}}
}

func (a *AfroMessageSender) Send(ctx context.Context, to, message string) error {
	normalized, err := phone.Normalize(to)
	if err != nil {
		return fmt.Errorf("sms: %w", err)
	}

	query := url.Values{}
	query.Set("to", strings.TrimPrefix(normalized, "+"))
	query.Set("message", message)
	if a.cfg.Sender != "" {
		query.Set("sender", a.cfg.Sender)
	}
	if a.cfg.IdentifierID != "" {
		query.Set("from", a.cfg.IdentifierID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.cfg.BaseURL+"/send?"+query.Encode(), nil)
	if err != nil {
		return fmt.Errorf("sms: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.cfg.Token)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sms: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sms: provider returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Acknowledge string `json:"acknowledge"`
	}
	if err := json.Unmarshal(body, &result); err == nil && result.Acknowledge != "" && result.Acknowledge != "success" {
		return fmt.Errorf("sms: provider rejected message: %s", string(body))
	}
	return nil
}
