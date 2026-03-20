package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- drift ---

func TestDrift_AlwaysInBounds(t *testing.T) {
	baseline := 100.0
	lo := baseline * 0.80
	hi := baseline * 1.20

	currents := []float64{0, baseline * 0.5, baseline * 0.8, baseline, baseline * 1.2, baseline * 2}
	for _, current := range currents {
		for i := 0; i < 200; i++ {
			got := drift(current, baseline, 0.10)
			if got < lo || got > hi {
				t.Errorf("drift(%f, %f, 0.10) = %f, want [%f, %f]", current, baseline, got, lo, hi)
			}
		}
	}
}

func TestDrift_ProducesVariation(t *testing.T) {
	baseline := 100.0
	seen := map[float64]bool{}
	for i := 0; i < 200; i++ {
		seen[drift(baseline, baseline, 0.05)] = true
	}
	if len(seen) < 2 {
		t.Error("drift() returned the same value every time — expected random variation")
	}
}

// --- normaliseEventType ---

func TestNormaliseEventType(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"price_update", "price_update"},
		{"tvl_change", "tvl_change"},
		{"swap", "protocol_event"},
		{"liquidation", "protocol_event"},
		{"deposit", "protocol_event"},
		{"unknown", "protocol_event"},
		{"", "protocol_event"},
	}
	for _, tc := range cases {
		got := normaliseEventType(tc.input)
		if got != tc.want {
			t.Errorf("normaliseEventType(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// --- pickEventType ---

func TestPickEventType_ValidValues(t *testing.T) {
	valid := map[string]bool{
		"price_update": true,
		"tvl_change":   true,
		"swap":         true,
		"liquidation":  true,
		"deposit":      true,
	}
	for i := 0; i < 200; i++ {
		got := pickEventType()
		if !valid[got] {
			t.Errorf("pickEventType() = %q, not in valid set", got)
		}
	}
}

func TestPickEventType_CoversAllValues(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 5000; i++ {
		seen[pickEventType()] = true
	}
	for _, want := range []string{"price_update", "tvl_change", "swap", "liquidation", "deposit"} {
		if !seen[want] {
			t.Errorf("pickEventType() never returned %q in 5000 calls", want)
		}
	}
}

// --- generateEvent ---

func TestGenerateEvent_FieldsPopulated(t *testing.T) {
	p := Protocol{ID: "uniswap", Name: "Uniswap"}
	ev := generateEvent(context.Background(), p)

	if ev.ProtocolID != "uniswap" {
		t.Errorf("ProtocolID = %q, want %q", ev.ProtocolID, "uniswap")
	}
	if ev.ProtocolName != "Uniswap" {
		t.Errorf("ProtocolName = %q, want %q", ev.ProtocolName, "Uniswap")
	}
	if ev.Price <= 0 {
		t.Errorf("Price = %f, want > 0", ev.Price)
	}
	if ev.TVL <= 0 {
		t.Errorf("TVL = %f, want > 0", ev.TVL)
	}
	if ev.Volume24h <= 0 {
		t.Errorf("Volume24h = %f, want > 0", ev.Volume24h)
	}
	if ev.Timestamp.IsZero() {
		t.Error("Timestamp is zero")
	}
	valid := map[string]bool{
		"price_update": true, "tvl_change": true,
		"swap": true, "liquidation": true, "deposit": true,
	}
	if !valid[ev.EventType] {
		t.Errorf("EventType = %q, not in valid set", ev.EventType)
	}
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

// --- getEnv ---

func TestGetEnv_Fallback(t *testing.T) {
	got := getEnv("__NONEXISTENT_VAR_XYZ__", "default")
	if got != "default" {
		t.Errorf("getEnv fallback = %q, want %q", got, "default")
	}
}

func TestGetEnv_ReadsEnv(t *testing.T) {
	t.Setenv("__TEST_VAR_XYZ__", "hello")
	got := getEnv("__TEST_VAR_XYZ__", "default")
	if got != "hello" {
		t.Errorf("getEnv = %q, want %q", got, "hello")
	}
}
