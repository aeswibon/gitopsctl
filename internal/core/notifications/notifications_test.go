package notifications

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestSendWebhook(t *testing.T) {
	logger := zap.NewNop()
	n := Notification{
		App:       "test-app",
		Cluster:   "test-cluster",
		Status:    "Synced",
		Message:   "Sync success",
		Commit:    "abc123",
		Timestamp: time.Now(),
	}

	t.Run("successful delivery", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("Expected POST, got %s", r.Method)
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("Expected application/json, got %s", r.Header.Get("Content-Type"))
			}

			var received Notification
			if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
				t.Fatal(err)
			}
			if received.App != n.App {
				t.Errorf("Expected app %s, got %s", n.App, received.App)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		SendWebhook(logger, server.URL, "", n)
	})

	t.Run("with signature", func(t *testing.T) {
		secret := "my-secret"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sig := r.Header.Get("X-GitOpsCTL-Signature")
			if sig == "" {
				t.Error("Expected X-GitOpsCTL-Signature header")
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		SendWebhook(logger, server.URL, secret, n)
	})

	t.Run("empty url", func(t *testing.T) {
		// Should not panic or error
		SendWebhook(logger, "", "", n)
	})
}

func TestSignPayload(t *testing.T) {
	payload := []byte("hello")
	secret := "secret"
	sig := signPayload(payload, secret)
	if sig == "" {
		t.Error("Signature should not be empty")
	}

	sig2 := signPayload(payload, secret)
	if sig != sig2 {
		t.Error("Signature should be deterministic")
	}

	sig3 := signPayload([]byte("different"), secret)
	if sig == sig3 {
		t.Error("Signature should be different for different payloads")
	}
}
