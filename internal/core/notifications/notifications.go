package notifications

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Notification represents a sync event notification.
type Notification struct {
	App         string    `json:"app"`
	Cluster     string    `json:"cluster"`
	Status      string    `json:"status"`
	Message     string    `json:"message"`
	Commit      string    `json:"commit"`
	Timestamp   time.Time `json:"timestamp"`
}

// SendWebhook sends a notification to the specified webhook URL.
func SendWebhook(logger *zap.Logger, url, secret string, n Notification) {
	if url == "" {
		return
	}

	data, err := json.Marshal(n)
	if err != nil {
		logger.Error("Failed to marshal notification", zap.Error(err))
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		logger.Error("Failed to create webhook request", zap.Error(err))
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GitOpsCTL-Controller")

	if secret != "" {
		signature := signPayload(data, secret)
		req.Header.Set("X-GitOpsCTL-Signature", signature)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Warn("Failed to send webhook notification", zap.String("url", url), zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		logger.Warn("Webhook returned error status", zap.String("url", url), zap.Int("status", resp.StatusCode))
	} else {
		logger.Debug("Webhook notification sent successfully", zap.String("url", url))
	}
}

func signPayload(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}
