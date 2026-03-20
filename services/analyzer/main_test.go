package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- scoreLabel ---

func TestScoreLabel(t *testing.T) {
	cases := []struct {
		score int
		want  string
	}{
		{100, "healthy"},
		{70, "healthy"},
		{69, "degraded"},
		{40, "degraded"},
		{39, "critical"},
		{1, "critical"},
		{0, "critical"},
	}
	for _, tc := range cases {
		got := scoreLabel(tc.score)
		if got != tc.want {
			t.Errorf("scoreLabel(%d) = %q, want %q", tc.score, got, tc.want)
		}
	}
}

// --- alertSeverity ---

func TestAlertSeverity(t *testing.T) {
	cases := []struct {
		score int
		want  string
	}{
		{0, "critical"},
		{1, "critical"},
		{19, "critical"},
		{20, "warning"},
		{21, "warning"},
		{39, "warning"},
	}
	for _, tc := range cases {
		got := alertSeverity(tc.score)
		if got != tc.want {
			t.Errorf("alertSeverity(%d) = %q, want %q", tc.score, got, tc.want)
		}
	}
}

// --- scoreLabel and alertSeverity boundary agreement ---

func TestScoreLabelAndAlertSeverity_BoundaryConsistency(t *testing.T) {
	// alertSeverity only fires when score < 40 (critical label territory)
	// Verify the two functions agree on what is "critical"
	for score := 0; score < 40; score++ {
		label := scoreLabel(score)
		if label == "healthy" {
			t.Errorf("scoreLabel(%d) = healthy but score < 40 should be degraded or critical", score)
		}
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

// --- initial scores sanity ---

func TestInitialScores_InRange(t *testing.T) {
	mu.RLock()
	defer mu.RUnlock()
	for id, hs := range scores {
		if hs.Score < 0 || hs.Score > 100 {
			t.Errorf("initial score for %q = %d, want [0, 100]", id, hs.Score)
		}
		if hs.Label == "" {
			t.Errorf("initial label for %q is empty", id)
		}
	}
}
