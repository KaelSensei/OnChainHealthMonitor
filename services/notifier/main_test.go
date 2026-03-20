package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- severity ---

func TestSeverity(t *testing.T) {
	cases := []struct {
		score int
		want  string
	}{
		{0, "CRITICAL"},
		{1, "CRITICAL"},
		{19, "CRITICAL"},
		{20, "WARNING"},
		{21, "WARNING"},
		{100, "WARNING"},
	}
	for _, tc := range cases {
		got := severity(tc.score)
		if got != tc.want {
			t.Errorf("severity(%d) = %q, want %q", tc.score, got, tc.want)
		}
	}
}

// --- randomScore ---

func TestRandomScore_InRange(t *testing.T) {
	protocols := []string{"uniswap", "aave", "compound"}
	for _, p := range protocols {
		for i := 0; i < 500; i++ {
			got := randomScore(p)
			if got < 0 || got > 100 {
				t.Errorf("randomScore(%q) = %d, want [0, 100]", p, got)
			}
		}
	}
}

func TestRandomScore_UnknownProtocol(t *testing.T) {
	// Unknown protocol has base 0 → score should still be in [0, 100]
	for i := 0; i < 200; i++ {
		got := randomScore("unknown")
		if got < 0 || got > 100 {
			t.Errorf("randomScore(\"unknown\") = %d, want [0, 100]", got)
		}
	}
}

func TestRandomScore_ProducesVariation(t *testing.T) {
	seen := map[int]bool{}
	for i := 0; i < 200; i++ {
		seen[randomScore("uniswap")] = true
	}
	if len(seen) < 2 {
		t.Error("randomScore(\"uniswap\") returned the same value every time — expected variation")
	}
}

// --- criticalThreshold ---

func TestCriticalThreshold_Value(t *testing.T) {
	if criticalThreshold != 30 {
		t.Errorf("criticalThreshold = %d, want 30", criticalThreshold)
	}
}

// --- sendNotification ---

func TestSendNotification_UpdatesMetrics(t *testing.T) {
	a := &Alert{
		Protocol:  "uniswap",
		Score:     25,
		Severity:  severity(25),
		Message:   "test alert",
		FiredAt:   time.Now().UTC(),
		Resolved:  false,
	}
	// Should not panic; Prometheus metrics updated internally.
	sendNotification(a)
}

func TestSendNotification_CriticalWebhook(t *testing.T) {
	a := &Alert{
		Protocol: "compound",
		Score:    10, // < 20 triggers webhook log
		Severity: severity(10),
		Message:  "critical test",
		FiredAt:  time.Now().UTC(),
	}
	// Should not panic; webhook channel logged internally.
	sendNotification(a)
}

// --- healthHandler ---

func TestHealthHandler_StatusOK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf(`body["status"] = %q, want "ok"`, body["status"])
	}
}
