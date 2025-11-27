package metric

import (
	"context"
	"fmt"
	"time"

	"github.com/opsorch/opsorch-core/schema"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// PrometheusProvider implements the metric.Provider interface for Prometheus.
type PrometheusProvider struct {
	api v1.API
}

// NewPrometheusProvider creates a new Prometheus provider.
func NewPrometheusProvider(config map[string]any) (*PrometheusProvider, error) {
	url, ok := config["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("missing required config field: url")
	}

	client, err := api.NewClient(api.Config{
		Address: url,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus client: %w", err)
	}

	return &PrometheusProvider{
		api: v1.NewAPI(client),
	}, nil
}

// Query executes a metric query against Prometheus.
func (p *PrometheusProvider) Query(ctx context.Context, query schema.MetricQuery) ([]schema.MetricSeries, error) {
	promQL, err := buildPromQL(query)
	if err != nil {
		return nil, err
	}

	r := v1.Range{
		Start: query.Start,
		End:   query.End,
		Step:  time.Duration(query.Step) * time.Second,
	}

	result, warnings, err := p.api.QueryRange(ctx, promQL, r)
	if err != nil {
		return nil, fmt.Errorf("prometheus query failed: %w", err)
	}
	if len(warnings) > 0 {
		// Log warnings? For now just proceed.
	}

	return convertResult(result)
}

// Describe lists available metrics from Prometheus.
func (p *PrometheusProvider) Describe(ctx context.Context, scope schema.QueryScope) ([]schema.MetricDescriptor, error) {
	// Use the label values API to get all metric names
	// This corresponds to querying label values for "__name__"
	values, warnings, err := p.api.LabelValues(ctx, "__name__", nil, time.Time{}, time.Time{})
	if err != nil {
		return nil, fmt.Errorf("failed to list metrics: %w", err)
	}
	if len(warnings) > 0 {
		// Log warnings
	}

	descriptors := make([]schema.MetricDescriptor, 0, len(values))
	for _, v := range values {
		descriptors = append(descriptors, schema.MetricDescriptor{
			Name: string(v),
			Type: "unknown", // Prometheus metadata API is needed for type, keeping simple for now
		})
	}

	return descriptors, nil
}

func buildPromQL(query schema.MetricQuery) (string, error) {
	// If raw query is provided in metadata, use it
	if raw, ok := query.Metadata["query"].(string); ok && raw != "" {
		return raw, nil
	}

	if query.Expression == nil {
		return "", fmt.Errorf("missing query expression")
	}

	expr := query.Expression.MetricName

	// Add filters
	var filters []string
	for _, f := range query.Expression.Filters {
		filters = append(filters, fmt.Sprintf("%s%s%q", f.Label, f.Operator, f.Value))
	}

	// Add scope filters
	if query.Scope.Service != "" {
		filters = append(filters, fmt.Sprintf("service=%q", query.Scope.Service))
	}
	if query.Scope.Team != "" {
		filters = append(filters, fmt.Sprintf("team=%q", query.Scope.Team))
	}
	if query.Scope.Environment != "" {
		filters = append(filters, fmt.Sprintf("env=%q", query.Scope.Environment))
	}

	if len(filters) > 0 {
		expr += "{"
		for i, f := range filters {
			if i > 0 {
				expr += ","
			}
			expr += f
		}
		expr += "}"
	}

	// Add aggregation
	if query.Expression.Aggregation != "" {
		expr = fmt.Sprintf("%s(%s)", query.Expression.Aggregation, expr)
		if len(query.Expression.GroupBy) > 0 {
			expr += " by ("
			for i, g := range query.Expression.GroupBy {
				if i > 0 {
					expr += ","
				}
				expr += g
			}
			expr += ")"
		}
	}

	return expr, nil
}

func convertResult(val model.Value) ([]schema.MetricSeries, error) {
	matrix, ok := val.(model.Matrix)
	if !ok {
		return nil, fmt.Errorf("expected matrix result, got %T", val)
	}

	series := make([]schema.MetricSeries, 0, len(matrix))
	for _, stream := range matrix {
		s := schema.MetricSeries{
			Name:   string(stream.Metric["__name__"]),
			Labels: make(map[string]any),
			Points: make([]schema.MetricPoint, 0, len(stream.Values)),
		}

		for k, v := range stream.Metric {
			if k != "__name__" {
				s.Labels[string(k)] = string(v)
			}
		}

		for _, p := range stream.Values {
			s.Points = append(s.Points, schema.MetricPoint{
				Timestamp: p.Timestamp.Time(),
				Value:     float64(p.Value),
			})
		}
		series = append(series, s)
	}

	return series, nil
}
