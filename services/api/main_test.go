package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- statusFromScore ---

func TestStatusFromScore(t *testing.T) {
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
		got := statusFromScore(tc.score)
		if got != tc.want {
			t.Errorf("statusFromScore(%d) = %q, want %q", tc.score, got, tc.want)
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

// --- protocolsHandler: list ---

func TestProtocolsHandler_ListAll(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/protocols", nil)
	w := httptest.NewRecorder()
	protocolsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Protocols []*Protocol `json:"protocols"`
		Total     int         `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Total != len(protocols) {
		t.Errorf("Total = %d, want %d", resp.Total, len(protocols))
	}
	if len(resp.Protocols) != len(protocols) {
		t.Errorf("len(Protocols) = %d, want %d", len(resp.Protocols), len(protocols))
	}
}

func TestProtocolsHandler_ListAll_FieldsPresent(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/protocols", nil)
	w := httptest.NewRecorder()
	protocolsHandler(w, req)

	var resp struct {
		Protocols []*Protocol `json:"protocols"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	for _, p := range resp.Protocols {
		if p.ID == "" {
			t.Errorf("protocol has empty ID")
		}
		if p.Name == "" {
			t.Errorf("protocol %q has empty Name", p.ID)
		}
		if p.Status == "" {
			t.Errorf("protocol %q has empty Status", p.ID)
		}
		if p.HealthScore < 0 || p.HealthScore > 100 {
			t.Errorf("protocol %q HealthScore = %d, want [0, 100]", p.ID, p.HealthScore)
		}
	}
}

// --- protocolsHandler: by ID ---

func TestProtocolsHandler_GetByID_Known(t *testing.T) {
	knownIDs := []string{"uniswap", "aave", "compound"}
	for _, id := range knownIDs {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/protocols/"+id, nil)
		w := httptest.NewRecorder()
		protocolsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("[%s] status = %d, want %d", id, w.Code, http.StatusOK)
			continue
		}
		var p Protocol
		if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
			t.Fatalf("[%s] invalid JSON: %v", id, err)
		}
		if p.ID != id {
			t.Errorf("[%s] Protocol.ID = %q, want %q", id, p.ID, id)
		}
	}
}

func TestProtocolsHandler_GetByID_NotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/protocols/nonexistent", nil)
	w := httptest.NewRecorder()
	protocolsHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["error"] == "" {
		t.Error("expected non-empty error message in 404 response")
	}
}

// --- instrumentedHandler ---

func TestInstrumentedHandler_CallsInner(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	})
	handler := instrumentedHandler("/test", inner)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if !called {
		t.Error("inner handler was not called")
	}
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

// --- statusResponseWriter ---

func TestStatusResponseWriter_CapturesCode(t *testing.T) {
	inner := httptest.NewRecorder()
	rw := &statusResponseWriter{ResponseWriter: inner, statusCode: http.StatusOK}
	rw.WriteHeader(http.StatusTeapot)

	if rw.statusCode != http.StatusTeapot {
		t.Errorf("statusCode = %d, want %d", rw.statusCode, http.StatusTeapot)
	}
}

// --- initial protocol data sanity ---

func TestProtocols_InitialState(t *testing.T) {
	mu.RLock()
	defer mu.RUnlock()

	if len(protocols) == 0 {
		t.Fatal("protocols slice is empty")
	}
	seen := map[string]bool{}
	for _, p := range protocols {
		if p.ID == "" {
			t.Error("protocol has empty ID")
		}
		if seen[p.ID] {
			t.Errorf("duplicate protocol ID: %q", p.ID)
		}
		seen[p.ID] = true
		if p.Status != statusFromScore(p.HealthScore) {
			t.Errorf("protocol %q: Status=%q but statusFromScore(%d)=%q",
				p.ID, p.Status, p.HealthScore, statusFromScore(p.HealthScore))
		}
	}
}
