//go:build ignore

package main

import (
	"context"
	"log"
	"os"

	"github.com/opsorch/opsorch-core/schema"
	"github.com/opsorch/opsorch-prometheus-adapter/alert"
)

func main() {
	url := os.Getenv("ALERTMANAGER_URL")
	if url == "" {
		url = "http://localhost:9093"
		log.Printf("ALERTMANAGER_URL not set, defaulting to %s", url)
	}

	cfg := map[string]any{
		"alertmanagerURL": url,
	}

	provider, err := alert.NewPrometheusAlertProvider(cfg)
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()

	// Test Scenarios
	tests := []struct {
		name  string
		query schema.AlertQuery
	}{
		{
			name:  "List All Alerts",
			query: schema.AlertQuery{},
		},
		{
			name: "List Active Alerts",
			query: schema.AlertQuery{
				Statuses: []string{"firing"},
			},
		},
		{
			name: "List Critical Alerts",
			query: schema.AlertQuery{
				Severities: []string{"critical"},
			},
		},
		{
			name: "List with Scope (Service=api)",
			query: schema.AlertQuery{
				Scope: schema.QueryScope{
					Service: "api",
				},
			},
		},
	}

	var lastAlertID string

	for _, tt := range tests {
		log.Printf("Running Test: %s...", tt.name)
		alerts, err := provider.Query(ctx, tt.query)
		if err != nil {
			log.Printf("  FAILED: %v", err)
			continue
		}
		log.Printf("  Success: returned %d alerts", len(alerts))
		for i, a := range alerts {
			if i < 3 { // Limit output
				log.Printf("    Alert %d: [%s] %s (%s)", i, a.Status, a.Title, a.ID)
			}
			lastAlertID = a.ID
		}
		if len(alerts) > 3 {
			log.Printf("    ... and %d more", len(alerts)-3)
		}
	}

	// Test Get Alert
	if lastAlertID != "" {
		log.Printf("Testing Get Alert (%s)...", lastAlertID)
		a, err := provider.Get(ctx, lastAlertID)
		if err != nil {
			log.Fatalf("Get failed: %v", err)
		}
		log.Printf("  Success: Found alert %s", a.Title)
	} else {
		log.Println("Skipping Get Alert test (no alerts found)")
	}

	log.Println("Integration tests passed!")
}
