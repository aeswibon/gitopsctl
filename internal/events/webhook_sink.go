package events

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WebhookSink POSTs each envelope as JSON with optional retries and HMAC signatures.
type WebhookSink struct {
	url     string
	client  *http.Client
	token   string
	secret  string
	retries int
	backoff time.Duration
}

// WebhookOptions configures webhook delivery behavior.
type WebhookOptions struct {
	Timeout       time.Duration
	Retries       int
	RetryBackoff  time.Duration
	SigningSecret string
}

// NewWebhookSink creates a webhook sink with sensible defaults.
// If bearerToken is non-empty, sets Authorization header.
func NewWebhookSink(url, bearerToken string) *WebhookSink {
	return NewWebhookSinkWithOptions(url, bearerToken, WebhookOptions{})
}

// NewWebhookSinkWithOptions creates a webhook sink with explicit options.
func NewWebhookSinkWithOptions(url, bearerToken string, opts WebhookOptions) *WebhookSink {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 12 * time.Second
	}
	backoff := opts.RetryBackoff
	if backoff <= 0 {
		backoff = 750 * time.Millisecond
	}
	retries := opts.Retries
	if retries < 0 {
		retries = 0
	}
	return &WebhookSink{
		url: url,
		client: &http.Client{
			Timeout: timeout,
		},
		token:   bearerToken,
		secret:  opts.SigningSecret,
		retries: retries,
		backoff: backoff,
	}
}

// Write POSTs the envelope as application/json.
func (s *WebhookSink) Write(ctx context.Context, e *Envelope) error {
	body, err := json.Marshal(e)
	if err != nil {
		return err
	}

	timestamp := time.Now().UTC().Format(time.RFC3339Nano)
	signature := ""
	if s.secret != "" {
		// Sign "<timestamp>.<json body>" so receivers can prevent replay and tampering.
		mac := hmac.New(sha256.New, []byte(s.secret))
		mac.Write([]byte(timestamp))
		mac.Write([]byte("."))
		mac.Write(body)
		signature = "sha256=" + hex.EncodeToString(mac.Sum(nil))
	}

	var lastErr error
	for attempt := 0; attempt <= s.retries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "gitopsctl-events/1.0")
		req.Header.Set("X-GitOpsctl-Event-ID", e.ID)
		req.Header.Set("X-GitOpsctl-Timestamp", timestamp)
		if signature != "" {
			req.Header.Set("X-GitOpsctl-Signature", signature)
		}
		if s.token != "" {
			req.Header.Set("Authorization", "Bearer "+s.token)
		}

		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = err
		} else {
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode < 300 {
				return nil
			}
			lastErr = fmt.Errorf("webhook returned status %s", resp.Status)
			if resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
				return lastErr // non-retryable by default
			}
		}

		if attempt == s.retries {
			break
		}
		wait := s.backoff * time.Duration(1<<attempt)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	return lastErr
}
