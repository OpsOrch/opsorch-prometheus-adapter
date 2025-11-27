package metric

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/opsorch/opsorch-core/schema"
)

func TestNewPrometheusProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]any
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  map[string]any{"url": "http://localhost:9090"},
			wantErr: false,
		},
		{
			name:    "missing url",
			config:  map[string]any{},
			wantErr: true,
		},
		{
			name:    "invalid url type",
			config:  map[string]any{"url": 123},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPrometheusProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPrometheusProvider() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPrometheusProvider_Query(t *testing.T) {
	defaultStart := time.Unix(1696118400, 0)
	defaultEnd := time.Unix(1696118460, 0)

	tests := []struct {
		name           string
		query          schema.MetricQuery
		mockResponse   string
		mockStatusCode int
		expectedQuery  string
		wantSeries     int
		wantErr        bool
		validate       func(*testing.T, []schema.MetricSeries)
	}{
		{
			name: "basic query with filters",
			query: schema.MetricQuery{
				Expression: &schema.MetricExpression{
					MetricName: "http_requests_total",
					Filters: []schema.MetricFilter{
						{Label: "method", Operator: "=", Value: "POST"},
					},
				},
				Start: defaultStart,
				End:   defaultEnd,
				Step:  60,
			},
			mockResponse: `{
				"status": "success",
				"data": {
					"resultType": "matrix",
					"result": [
						{
							"metric": { "__name__": "http_requests_total", "method": "POST" },
							"values": [ [1696118400, "10"], [1696118460, "20"] ]
						}
					]
				}
			}`,
			expectedQuery: `http_requests_total{method="POST"}`,
			wantSeries:    1,
			validate: func(t *testing.T, res []schema.MetricSeries) {
				if res[0].Name != "http_requests_total" {
					t.Errorf("got name %s", res[0].Name)
				}
				if res[0].Labels["method"] != "POST" {
					t.Errorf("got label %v", res[0].Labels["method"])
				}
				if len(res[0].Points) != 2 {
					t.Errorf("got %d points", len(res[0].Points))
				}
			},
		},
		{
			name: "query with scope",
			query: schema.MetricQuery{
				Expression: &schema.MetricExpression{
					MetricName: "http_requests_total",
				},
				Scope: schema.QueryScope{
					Service:     "api",
					Environment: "prod",
				},
				Start: defaultStart,
				End:   defaultEnd,
				Step:  60,
			},
			mockResponse: `{
				"status": "success",
				"data": { "resultType": "matrix", "result": [] }
			}`,
			expectedQuery: `http_requests_total{service="api",env="prod"}`,
			wantSeries:    0,
		},
		{
			name: "aggregation and grouping",
			query: schema.MetricQuery{
				Expression: &schema.MetricExpression{
					MetricName:  "http_requests_total",
					Aggregation: "sum",
					GroupBy:     []string{"method", "status"},
				},
				Start: defaultStart,
				End:   defaultEnd,
				Step:  60,
			},
			mockResponse: `{
				"status": "success",
				"data": { "resultType": "matrix", "result": [] }
			}`,
			expectedQuery: `sum(http_requests_total) by (method,status)`,
			wantSeries:    0,
		},
		{
			name: "raw query override",
			query: schema.MetricQuery{
				Expression: &schema.MetricExpression{MetricName: "ignored"},
				Metadata:   map[string]any{"query": "up{job=\"prometheus\"}"},
				Start:      defaultStart,
				End:        defaultEnd,
				Step:       60,
			},
			mockResponse: `{
				"status": "success",
				"data": { "resultType": "matrix", "result": [] }
			}`,
			expectedQuery: `up{job="prometheus"}`,
			wantSeries:    0,
		},
		{
			name: "api error",
			query: schema.MetricQuery{
				Expression: &schema.MetricExpression{MetricName: "up"},
				Start:      defaultStart,
				End:        defaultEnd,
				Step:       60,
			},
			mockStatusCode: 400,
			mockResponse:   `{"status":"error","errorType":"bad_data","error":"bad query"}`,
			wantErr:        true,
		},
		{
			name: "malformed response",
			query: schema.MetricQuery{
				Expression: &schema.MetricExpression{MetricName: "up"},
				Start:      defaultStart,
				End:        defaultEnd,
				Step:       60,
			},
			mockResponse: `not json`,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/query_range" {
					t.Errorf("Expected path /api/v1/query_range, got %s", r.URL.Path)
				}

				var q string
				if r.Method == "POST" {
					_ = r.ParseForm()
					q = r.Form.Get("query")
				} else {
					q = r.URL.Query().Get("query")
				}

				if tt.expectedQuery != "" && q != tt.expectedQuery {
					t.Errorf("Expected query '%s', got '%s'", tt.expectedQuery, q)
				}

				w.Header().Set("Content-Type", "application/json")
				if tt.mockStatusCode != 0 {
					w.WriteHeader(tt.mockStatusCode)
				}
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			provider, err := NewPrometheusProvider(map[string]any{"url": server.URL})
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}

			result, err := provider.Query(context.Background(), tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(result) != tt.wantSeries {
					t.Errorf("Query() got %d series, want %d", len(result), tt.wantSeries)
				}
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestPrometheusProvider_Describe(t *testing.T) {
	mockResponse := `{
		"status": "success",
		"data": [
			"http_requests_total",
			"go_goroutines"
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/label/__name__/values" {
			t.Errorf("Expected path /api/v1/label/__name__/values, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	provider, err := NewPrometheusProvider(map[string]any{"url": server.URL})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()
	descriptors, err := provider.Describe(ctx, schema.QueryScope{})
	if err != nil {
		t.Fatalf("Describe failed: %v", err)
	}

	if len(descriptors) != 2 {
		t.Fatalf("Expected 2 descriptors, got %d", len(descriptors))
	}

	names := []string{descriptors[0].Name, descriptors[1].Name}
	expected := []string{"http_requests_total", "go_goroutines"}
	if !reflect.DeepEqual(names, expected) {
		t.Errorf("Expected names %v, got %v", expected, names)
	}
}
