package kashier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client wraps Kashier's payment-session API.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

func NewClient(apiKey, baseURL string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

type CreateSessionInput struct {
	Amount          string
	Currency        string
	MerchantOrderID string // our consultation ID, as string
	ReturnURL       string
	CancelURL       string
	Description     string
}

type CreateSessionOutput struct {
	SessionID  string
	SessionURL string
}

func (c *Client) CreateSession(ctx context.Context, in CreateSessionInput) (*CreateSessionOutput, error) {
	body := map[string]any{
		"amount":          in.Amount,
		"currency":        in.Currency,
		"merchantOrderId": in.MerchantOrderID,
		"returnUrl":       in.ReturnURL,
		"cancelUrl":       in.CancelURL,
		"description":     in.Description,
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/v3/payment/sessions", bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("kashier status %d: %s", resp.StatusCode, string(raw))
	}

	var parsed struct {
		SessionID  string `json:"sessionId"`
		SessionURL string `json:"sessionUrl"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	if parsed.SessionURL == "" {
		return nil, fmt.Errorf("kashier returned empty sessionUrl")
	}
	return &CreateSessionOutput{
		SessionID:  parsed.SessionID,
		SessionURL: parsed.SessionURL,
	}, nil
}
