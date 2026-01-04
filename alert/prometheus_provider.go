package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	corealert "github.com/opsorch/opsorch-core/alert"
	"github.com/opsorch/opsorch-core/schema"
)

// ProviderName is the registry key for the Prometheus alert adapter.
const ProviderName = "prometheus"

// PrometheusAlertProvider implements alert.Provider for Prometheus Alertmanager.
type PrometheusAlertProvider struct {
	baseURL string
	client  *http.Client
}

// NewPrometheusAlertProvider creates a new Prometheus alert provider.
func NewPrometheusAlertProvider(config map[string]any) (corealert.Provider, error) {
	alertmanagerURL, ok := config["alertmanagerURL"].(string)
	if !ok || alertmanagerURL == "" {
		return nil, fmt.Errorf("missing required config field: alertmanagerURL")
	}

	return &PrometheusAlertProvider{
		baseURL: alertmanagerURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func init() {
	_ = corealert.RegisterProvider(ProviderName, NewPrometheusAlertProvider)
}

// Query fetches alerts from Prometheus Alertmanager.
func (p *PrometheusAlertProvider) Query(ctx context.Context, query schema.AlertQuery) ([]schema.Alert, error) {
	params := url.Values{}

	// Add filters
	if len(query.Statuses) > 0 {
		for _, status := range query.Statuses {
			// Prometheus Alertmanager states: active, suppressed, unprocessed
			params.Add("filter", fmt.Sprintf("state=\"%s\"", mapStatusToAlertmanager(status)))
		}
	}

	if len(query.Severities) > 0 {
		for _, severity := range query.Severities {
			params.Add("filter", fmt.Sprintf("severity=\"%s\"", severity))
		}
	}

	// Add scope filters
	if query.Scope.Service != "" {
		params.Add("filter", fmt.Sprintf("service=\"%s\"", query.Scope.Service))
	}
	if query.Scope.Team != "" {
		params.Add("filter", fmt.Sprintf("team=\"%s\"", query.Scope.Team))
	}
	if query.Scope.Environment != "" {
		params.Add("filter", fmt.Sprintf("env=\"%s\"", query.Scope.Environment))
	}

	apiURL := fmt.Sprintf("%s/api/v2/alerts?%s", p.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("alertmanager API error: %d %s", resp.StatusCode, string(bodyBytes))
	}

	var amAlerts []alertmanagerAlert
	if err := json.NewDecoder(resp.Body).Decode(&amAlerts); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	alerts := make([]schema.Alert, 0, len(amAlerts))
	for _, amAlert := range amAlerts {
		alerts = append(alerts, convertAlertmanagerAlert(amAlert))
	}

	// Apply limit if specified
	if query.Limit > 0 && query.Limit < len(alerts) {
		alerts = alerts[:query.Limit]
	}

	return alerts, nil
}

// Get fetches a single alert by fingerprint from Prometheus Alertmanager.
func (p *PrometheusAlertProvider) Get(ctx context.Context, id string) (schema.Alert, error) {
	apiURL := fmt.Sprintf("%s/api/v2/alerts", p.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return schema.Alert{}, fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return schema.Alert{}, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return schema.Alert{}, fmt.Errorf("alertmanager API error: %d %s", resp.StatusCode, string(bodyBytes))
	}

	var amAlerts []alertmanagerAlert
	if err := json.NewDecoder(resp.Body).Decode(&amAlerts); err != nil {
		return schema.Alert{}, fmt.Errorf("decode response: %w", err)
	}

	// Find alert by fingerprint (ID)
	for _, amAlert := range amAlerts {
		if amAlert.Fingerprint == id {
			return convertAlertmanagerAlert(amAlert), nil
		}
	}

	return schema.Alert{}, fmt.Errorf("alert not found: %s", id)
}

// alertmanagerAlert represents an alert from Prometheus Alertmanager API.
type alertmanagerAlert struct {
	Fingerprint string `json:"fingerprint"`
	Status      struct {
		State string `json:"state"` // firing, suppressed, etc.
	} `json:"status"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    string            `json:"startsAt"`
	EndsAt      string            `json:"endsAt"`
	UpdatedAt   string            `json:"updatedAt"`
}

func convertAlertmanagerAlert(amAlert alertmanagerAlert) schema.Alert {
	alert := schema.Alert{
		ID:          amAlert.Fingerprint,
		Title:       amAlert.Labels["alertname"],
		Description: amAlert.Annotations["description"],
		Status:      mapAlertmanagerStateToStatus(amAlert.Status.State),
		Severity:    amAlert.Labels["severity"],
		Service:     amAlert.Labels["service"],
		URL:         "/alerting/alerts#" + amAlert.Fingerprint,
		Fields: map[string]any{
			"labels":      amAlert.Labels,
			"annotations": amAlert.Annotations,
		},
		Metadata: map[string]any{
			"source":      "prometheus",
			"fingerprint": amAlert.Fingerprint,
		},
	}

	if startsAt, err := time.Parse(time.RFC3339, amAlert.StartsAt); err == nil {
		alert.CreatedAt = startsAt
	}

	if updatedAt, err := time.Parse(time.RFC3339, amAlert.UpdatedAt); err == nil {
		alert.UpdatedAt = updatedAt
	}

	return alert
}

func mapStatusToAlertmanager(status string) string {
	switch status {
	case "firing", "open", "active":
		return "active"
	case "resolved", "closed":
		return "suppressed"
	default:
		return status
	}
}

func mapAlertmanagerStateToStatus(state string) string {
	switch state {
	case "active":
		return "firing"
	case "suppressed":
		return "suppressed"
	case "unprocessed":
		return "pending"
	default:
		return state
	}
}
