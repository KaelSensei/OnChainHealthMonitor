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

// --- criticalThreshold ---

func TestCriticalThreshold_Value(t *testing.T) {
	if criticalThreshold != 30 {
		t.Errorf("criticalThreshold = %d, want 30", criticalThreshold)
	}
}

// --- sendSystemAlert ---

func TestSendSystemAlert_UpdatesMetrics(t *testing.T) {
	a := &Alert{
		Protocol: "uniswap",
		Score:    25,
		Severity: severity(25),
		Message:  "test alert",
		FiredAt:  time.Now().UTC(),
	}
	// Should not panic; Prometheus metrics updated internally.
	sendSystemAlert(a)
}

func TestSendSystemAlert_CriticalWebhook(t *testing.T) {
	a := &Alert{
		Protocol: "compound",
		Score:    10, // < 20 triggers webhook log
		Severity: severity(10),
		Message:  "critical test",
		FiredAt:  time.Now().UTC(),
	}
	// Should not panic; webhook channel logged internally.
	sendSystemAlert(a)
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
