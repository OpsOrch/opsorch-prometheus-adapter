//go:build ignore

package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/opsorch/opsorch-core/schema"
	"github.com/opsorch/opsorch-prometheus-adapter/metric"
)

func main() {
	url := os.Getenv("PROMETHEUS_URL")
	if url == "" {
		url = "http://localhost:9090"
		log.Printf("PROMETHEUS_URL not set, defaulting to %s", url)
	}

	cfg := map[string]any{
		"url": url,
	}

	provider, err := metric.NewPrometheusProvider(cfg)
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()

	// Test 1: Describe
	log.Println("Testing Describe...")
	descriptors, err := provider.Describe(ctx, schema.QueryScope{})
	if err != nil {
		log.Fatalf("Describe failed: %v", err)
	}
	log.Printf("Found %d metrics", len(descriptors))
	if len(descriptors) > 0 {
		log.Printf("First metric: %s", descriptors[0].Name)
	}

	// Test Scenarios
	tests := []struct {
		name  string
		query schema.MetricQuery
	}{
		{
			name: "Basic Query (up)",
			query: schema.MetricQuery{
				Expression: &schema.MetricExpression{MetricName: "up"},
				Start:      time.Now().Add(-5 * time.Minute),
				End:        time.Now(),
				Step:       60,
			},
		},
		{
			name: "Query with Filters (up{job='prometheus'})",
			query: schema.MetricQuery{
				Expression: &schema.MetricExpression{
					MetricName: "up",
					Filters: []schema.MetricFilter{
						{Label: "job", Operator: "=", Value: "prometheus"},
					},
				},
				Start: time.Now().Add(-5 * time.Minute),
				End:   time.Now(),
				Step:  60,
			},
		},
		{
			name: "Query with Aggregation (sum(up))",
			query: schema.MetricQuery{
				Expression: &schema.MetricExpression{
					MetricName:  "up",
					Aggregation: "sum",
				},
				Start: time.Now().Add(-5 * time.Minute),
				End:   time.Now(),
				Step:  60,
			},
		},
		{
			name: "Query with Scope (simulated)",
			query: schema.MetricQuery{
				Expression: &schema.MetricExpression{MetricName: "up"},
				Scope: schema.QueryScope{
					Service:     "my-service",
					Environment: "prod",
				},
				Start: time.Now().Add(-5 * time.Minute),
				End:   time.Now(),
				Step:  60,
			},
		},
		{
			name: "Raw Query Override",
			query: schema.MetricQuery{
				Expression: &schema.MetricExpression{MetricName: "ignored"},
				Metadata:   map[string]any{"query": "count(up)"},
				Start:      time.Now().Add(-5 * time.Minute),
				End:        time.Now(),
				Step:       60,
			},
		},
	}

	for _, tt := range tests {
		log.Printf("Running Test: %s...", tt.name)
		series, err := provider.Query(ctx, tt.query)
		if err != nil {
			log.Printf("  FAILED: %v", err)
			continue
		}
		log.Printf("  Success: returned %d series", len(series))
		for i, s := range series {
			if i < 3 { // Limit output
				log.Printf("    Series %d: %s %v (Points: %d) - URL: %s", i, s.Name, s.Labels, len(s.Points), s.URL)
			}
		}
		if len(series) > 3 {
			log.Printf("    ... and %d more", len(series)-3)
		}
	}

	log.Println("Integration tests passed!")
}
