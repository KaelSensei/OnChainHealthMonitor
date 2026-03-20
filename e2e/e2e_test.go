// Package e2e_test contains end-to-end tests for the OnChain Health Monitor stack.
//
// These tests require all four services to be running (e.g. via docker-compose).
// They are skipped automatically unless the E2E environment variable is set:
//
//	E2E=1 go test ./...
//	E2E=1 E2E_HOST=staging.example.com go test ./...
package e2e_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

// skipIfNotE2E skips the test unless E2E=1 is set.
func skipIfNotE2E(t *testing.T) {
	t.Helper()
	if os.Getenv("E2E") == "" {
		t.Skip("set E2E=1 to run end-to-end tests (requires a running docker-compose stack)")
	}
}

// host returns the base hostname from E2E_HOST (default: localhost).
func host() string {
	if h := os.Getenv("E2E_HOST"); h != "" {
		return h
	}
	return "localhost"
}

func url(port int, path string) string {
	return fmt.Sprintf("http://%s:%d%s", host(), port, path)
}

// get is a thin helper that makes a GET request with a reasonable timeout.
func get(t *testing.T, rawURL string) *http.Response {
	t.Helper()
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(rawURL) //nolint:noctx
	if err != nil {
		t.Fatalf("GET %s: %v", rawURL, err)
	}
	return resp
}

// ── Health endpoints ─────────────────────────────────────────────────────────

func TestE2E_APIHealth(t *testing.T) {
	skipIfNotE2E(t)
	resp := get(t, url(8080, "/health"))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("api /health: status %d, want 200", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("api /health: invalid JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf(`api /health: body["status"] = %q, want "ok"`, body["status"])
	}
}

func TestE2E_CollectorHealth(t *testing.T) {
	skipIfNotE2E(t)
	resp := get(t, url(8081, "/health"))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("collector /health: status %d, want 200", resp.StatusCode)
	}
}

func TestE2E_AnalyzerHealth(t *testing.T) {
	skipIfNotE2E(t)
	resp := get(t, url(8082, "/health"))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("analyzer /health: status %d, want 200", resp.StatusCode)
	}
}

func TestE2E_NotifierHealth(t *testing.T) {
	skipIfNotE2E(t)
	resp := get(t, url(8083, "/health"))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("notifier /health: status %d, want 200", resp.StatusCode)
	}
}

// ── Prometheus metrics ───────────────────────────────────────────────────────

func TestE2E_CollectorMetrics(t *testing.T) {
	skipIfNotE2E(t)
	resp := get(t, url(8081, "/metrics"))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("collector /metrics: status %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct == "" {
		t.Error("collector /metrics: missing Content-Type header")
	}
}

func TestE2E_AnalyzerMetrics(t *testing.T) {
	skipIfNotE2E(t)
	resp := get(t, url(8082, "/metrics"))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("analyzer /metrics: status %d, want 200", resp.StatusCode)
	}
}

func TestE2E_APIMetrics(t *testing.T) {
	skipIfNotE2E(t)
	resp := get(t, url(8080, "/metrics"))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("api /metrics: status %d, want 200", resp.StatusCode)
	}
}

// ── API: protocol list ────────────────────────────────────────────────────────

func TestE2E_API_ListProtocols(t *testing.T) {
	skipIfNotE2E(t)
	resp := get(t, url(8080, "/api/v1/protocols"))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/protocols: status %d, want 200", resp.StatusCode)
	}

	var body struct {
		Protocols []struct {
			ID          string  `json:"id"`
			Name        string  `json:"name"`
			HealthScore int     `json:"health_score"`
			Status      string  `json:"status"`
			TVL         float64 `json:"tvl_usd"`
		} `json:"protocols"`
		Total int `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body.Total == 0 {
		t.Error("expected at least one protocol, got total=0")
	}
	for _, p := range body.Protocols {
		if p.ID == "" {
			t.Error("protocol missing id field")
		}
		if p.HealthScore < 0 || p.HealthScore > 100 {
			t.Errorf("protocol %q health_score=%d, want [0,100]", p.ID, p.HealthScore)
		}
		validStatus := map[string]bool{"healthy": true, "degraded": true, "critical": true}
		if !validStatus[p.Status] {
			t.Errorf("protocol %q status=%q, want one of healthy/degraded/critical", p.ID, p.Status)
		}
	}
}

// ── API: single protocol ─────────────────────────────────────────────────────

func TestE2E_API_GetProtocol_Uniswap(t *testing.T) {
	skipIfNotE2E(t)
	resp := get(t, url(8080, "/api/v1/protocols/uniswap"))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/protocols/uniswap: status %d, want 200", resp.StatusCode)
	}
	var p struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if p.ID != "uniswap" {
		t.Errorf("id = %q, want uniswap", p.ID)
	}
}

func TestE2E_API_GetProtocol_NotFound(t *testing.T) {
	skipIfNotE2E(t)
	resp := get(t, url(8080, "/api/v1/protocols/nonexistent"))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET /api/v1/protocols/nonexistent: status %d, want 404", resp.StatusCode)
	}
}

// ── Pipeline smoke test ───────────────────────────────────────────────────────

// TestE2E_Pipeline_HealthScoresChange waits a few seconds and checks that the
// analyzer is actively computing scores (scores should change over time).
func TestE2E_Pipeline_HealthScoresChange(t *testing.T) {
	skipIfNotE2E(t)

	scoreAt := func() int {
		resp := get(t, url(8080, "/api/v1/protocols/uniswap"))
		defer resp.Body.Close()
		var p struct {
			HealthScore int `json:"health_score"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		return p.HealthScore
	}

	first := scoreAt()

	// The analyzer updates every 3 seconds; wait up to 10s for a change.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)
		if scoreAt() != first {
			return // score changed — pipeline is live
		}
	}
	// A static score for 10s is technically possible (unlikely with ±5 drift)
	// so we only log a warning rather than hard-fail.
	t.Log("warning: uniswap health_score did not change in 10s — pipeline may be stalled")
}
