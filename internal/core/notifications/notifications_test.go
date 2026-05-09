package notifications

import (
	"encoding/json"
	"io"
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
		Status:    "Success",
		Timestamp: time.Now(),
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		var received Notification
		_ = json.Unmarshal(body, &received)

		if received.App != n.App {
			t.Errorf("Expected app %s, got %s", n.App, received.App)
		}

		signature := r.Header.Get("X-GitOpsCTL-Signature")
		if signature == "" {
			t.Error("Expected X-GitOpsCTL-Signature header")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	SendWebhook(logger, ts.URL, "secret", n)
}

func TestSendWebhook_EmptyURL_NoOp(t *testing.T) {
	SendWebhook(zap.NewNop(), "", "secret", Notification{App: "x"})
}

func TestSendWebhook_ErrorStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()
	SendWebhook(zap.NewNop(), ts.URL, "", Notification{App: "x"})
}

func TestSendWebhook_InvalidURL(t *testing.T) {
	SendWebhook(zap.NewNop(), "http://127.0.0.1:0/nope", "", Notification{App: "x"})
}

func TestSignPayload(t *testing.T) {
	payload := []byte("hello")
	secret := "world"
	sig := signPayload(payload, secret)
	if sig == "" {
		t.Error("Signature should not be empty")
	}

	// Verify it's consistent
	sig2 := signPayload(payload, secret)
	if sig != sig2 {
		t.Error("Signatures should be consistent")
	}
}
