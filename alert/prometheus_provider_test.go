package alert

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opsorch/opsorch-core/schema"
)

func TestNewPrometheusAlertProvider(t *testing.T) {
	t.Run("requires alertmanagerURL", func(t *testing.T) {
		_, err := NewPrometheusAlertProvider(map[string]any{})
		if err == nil {
			t.Fatal("expected error when alertmanagerURL missing")
		}
	})

	t.Run("creates provider successfully", func(t *testing.T) {
		prov, err := NewPrometheusAlertProvider(map[string]any{
			"alertmanagerURL": "http://localhost:9093",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if prov == nil {
			t.Fatal("expected non-nil provider")
		}
	})
}

func TestQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/alerts" && r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]any{
				{
					"fingerprint": "abc123",
					"status": map[string]string{
						"state": "active",
					},
					"labels": map[string]string{
						"alertname": "HighCPU",
						"severity":  "critical",
						"service":   "api",
					},
					"annotations": map[string]string{
						"description": "CPU usage above 90%",
					},
					"startsAt":  "2025-12-03T10:00:00Z",
					"updatedAt": "2025-12-03T10:05:00Z",
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	prov := &PrometheusAlertProvider{
		alertmanagerURL: server.URL,
		client:          &http.Client{},
	}

	alerts, err := prov.Query(context.Background(), schema.AlertQuery{})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}

	alert := alerts[0]
	if alert.ID != "abc123" {
		t.Errorf("ID = %v, want abc123", alert.ID)
	}
	if alert.Title != "HighCPU" {
		t.Errorf("Title = %v, want HighCPU", alert.Title)
	}
	if alert.Description != "CPU usage above 90%" {
		t.Errorf("Description = %v, want description", alert.Description)
	}
	if alert.Status != "firing" {
		t.Errorf("Status = %v, want firing", alert.Status)
	}
	if alert.Severity != "critical" {
		t.Errorf("Severity = %v, want critical", alert.Severity)
	}
}

func TestGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/alerts" && r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]any{
				{
					"fingerprint": "target123",
					"status": map[string]string{
						"state": "active",
					},
					"labels": map[string]string{
						"alertname": "TargetAlert",
						"severity":  "warning",
					},
					"annotations": map[string]string{
						"description": "Target alert description",
					},
					"startsAt":  "2025-12-03T10:00:00Z",
					"updatedAt": "2025-12-03T10:05:00Z",
				},
				{
					"fingerprint": "other456",
					"status": map[string]string{
						"state": "active",
					},
					"labels": map[string]string{
						"alertname": "OtherAlert",
					},
					"annotations": map[string]string{},
					"startsAt":    "2025-12-03T11:00:00Z",
					"updatedAt":   "2025-12-03T11:05:00Z",
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	prov := &PrometheusAlertProvider{
		alertmanagerURL: server.URL,
		client:          &http.Client{},
	}

	alert, err := prov.Get(context.Background(), "target123")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if alert.ID != "target123" {
		t.Errorf("ID = %v, want target123", alert.ID)
	}
	if alert.Title != "TargetAlert" {
		t.Errorf("Title = %v, want TargetAlert", alert.Title)
	}
}

func TestGetNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/alerts" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]any{})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	prov := &PrometheusAlertProvider{
		alertmanagerURL: server.URL,
		client:          &http.Client{},
	}

	_, err := prov.Get(context.Background(), "notfound")
	if err == nil {
		t.Fatal("expected error when alert not found")
	}
}
