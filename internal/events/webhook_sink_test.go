package events

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWebhookSink_SignsAndSendsHeaders(t *testing.T) {
	secret := "top-secret"
	token := "abc123"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+token {
			t.Fatalf("expected auth header, got %q", got)
		}
		if got := r.Header.Get("X-GitOpsctl-Event-ID"); got == "" {
			t.Fatal("missing X-GitOpsctl-Event-ID")
		}
		ts := r.Header.Get("X-GitOpsctl-Timestamp")
		if ts == "" {
			t.Fatal("missing X-GitOpsctl-Timestamp")
		}
		if _, err := time.Parse(time.RFC3339Nano, ts); err != nil {
			t.Fatalf("invalid timestamp header: %v", err)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		var env Envelope
		if err := json.Unmarshal(body, &env); err != nil {
			t.Fatalf("invalid json body: %v", err)
		}
		if env.Type != string(TypeAppSyncRequested) {
			t.Fatalf("unexpected type: %q", env.Type)
		}

		sig := r.Header.Get("X-GitOpsctl-Signature")
		if !strings.HasPrefix(sig, "sha256=") {
			t.Fatalf("unexpected signature header: %q", sig)
		}
		gotHex := strings.TrimPrefix(sig, "sha256=")
		got, err := hex.DecodeString(gotHex)
		if err != nil {
			t.Fatalf("invalid signature hex: %v", err)
		}

		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(ts))
		mac.Write([]byte("."))
		mac.Write(body)
		want := mac.Sum(nil)
		if !hmac.Equal(got, want) {
			t.Fatal("signature mismatch")
		}

		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	sink := NewWebhookSinkWithOptions(srv.URL, token, WebhookOptions{
		SigningSecret: secret,
		Timeout:       2 * time.Second,
		Retries:       0,
	})
	err := sink.Write(context.Background(), NewEnvelope("test", TypeAppSyncRequested, map[string]any{"app": "a"}))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}
}

func TestWebhookSink_RetriesOnTransient(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	sink := NewWebhookSinkWithOptions(srv.URL, "", WebhookOptions{
		Retries:      2,
		RetryBackoff: 1 * time.Millisecond,
		Timeout:      2 * time.Second,
	})
	err := sink.Write(context.Background(), NewEnvelope("test", TypeControllerStarted, nil))
	if err != nil {
		t.Fatalf("expected eventual success, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestWebhookSink_DoesNotRetryOnNonTransient4xx(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	sink := NewWebhookSinkWithOptions(srv.URL, "", WebhookOptions{
		Retries:      3,
		RetryBackoff: 1 * time.Millisecond,
		Timeout:      2 * time.Second,
	})
	err := sink.Write(context.Background(), NewEnvelope("test", TypeControllerStarted, nil))
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}
