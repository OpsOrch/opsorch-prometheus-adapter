# OpsOrch Prometheus Adapter

[![Version](https://img.shields.io/github/v/release/opsorch/opsorch-prometheus-adapter)](https://github.com/opsorch/opsorch-prometheus-adapter/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/opsorch/opsorch-prometheus-adapter)](https://github.com/opsorch/opsorch-prometheus-adapter/blob/main/go.mod)
[![License](https://img.shields.io/github/license/opsorch/opsorch-prometheus-adapter)](https://github.com/opsorch/opsorch-prometheus-adapter/blob/main/LICENSE)
[![CI](https://github.com/opsorch/opsorch-prometheus-adapter/workflows/CI/badge.svg)](https://github.com/opsorch/opsorch-prometheus-adapter/actions)

This adapter integrates OpsOrch with Prometheus, enabling metric querying, alert monitoring, and discovery through the Prometheus HTTP API.

## Capabilities

This adapter provides two capabilities:

1. **Metric Provider**: Query time-series metrics and discover available metrics
2. **Alert Provider**: Query alerts from Prometheus Alertmanager

## Features

### Metrics
- **Metric Query**: Execute PromQL queries via structured expressions or raw query strings
- **Metric Discovery**: List all available metrics in your Prometheus instance
- **QueryScope Support**: Automatically map service/team/environment to Prometheus labels
- **Aggregation**: Support for aggregation functions (sum, avg, max, min, count)
- **Filtering**: Label-based filtering with multiple operators (=, !=, =~, !~)
- **Range Queries**: Query metrics over time ranges with configurable step sizes

### Alerts
- **Alert Query**: Fetch firing, suppressed, and pending alerts from Prometheus Alertmanager
- **Alert Details**: Get individual alerts by fingerprint
- **Status Filtering**: Map OpsOrch statuses (firing, resolved, open, closed) to Alertmanager states for filtering
- **Severity Filtering**: Filter alerts by severity level
- **Scope Filtering**: Filter alerts by service/team/environment label hints

### Version Compatibility

- **Adapter Version**: 0.1.0
- **Requires OpsOrch Core**: >=0.1.0
- **Prometheus**: 2.x
- **Alertmanager**: 0.20+
- **Go Version**: 1.21+

## Configuration

### Metric Provider Configuration

The metric adapter requires the following configuration:

| Field | Type | Required | Description | Default |
|-------|------|----------|-------------|---------|
| `url` | string | Yes | The base URL of the Prometheus server (e.g., `http://prometheus:9090`) | - |

### Alert Provider Configuration

The alert adapter requires the following configuration:

| Field | Type | Required | Description | Default |
|-------|------|----------|-------------|---------|
| `alertmanagerURL` | string | Yes | The base URL of the Prometheus Alertmanager (e.g., `http://alertmanager:9093`) | - |

### Example Configuration

**Metric Adapter - JSON format:**
```json
{
  "url": "http://localhost:9090"
}
```

**Alert Adapter - JSON format:**
```json
{
  "alertmanagerURL": "http://localhost:9093"
}
```

**Environment variables (Metric):**
```bash
export OPSORCH_METRIC_PLUGIN=/path/to/bin/metricplugin
export OPSORCH_METRIC_CONFIG='{"url":"http://prometheus:9090"}'
```

**Environment variables (Alert):**
```bash
export OPSORCH_ALERT_PLUGIN=/path/to/bin/alertplugin
export OPSORCH_ALERT_CONFIG='{"alertmanagerURL":"http://alertmanager:9093"}'
```

## Field Mapping

### Metric Adapter

#### Query Mapping

| OpsOrch Field | PromQL Mapping | Notes |
|---------------|----------------|-------|
| `MetricQuery.Expression.MetricName` | Metric selector | Base metric name (e.g., `http_requests_total`) |
| `MetricQuery.Expression.Filters` | Label matchers | Converted to `{label="value"}` syntax |
| `MetricQuery.Scope.Service` | Label filter | Adds `service="<name>"` matcher |
| `MetricQuery.Scope.Team` | Label filter | Adds `team="<name>"` matcher |
| `MetricQuery.Scope.Environment` | Label filter | Adds `env="<name>"` matcher |
| `MetricQuery.Expression.Aggregation` | Aggregation function | Wraps query (e.g., `sum(...)`) |
| `MetricQuery.Expression.GroupBy` | `by` clause | Adds `by (label1, label2)` |
| `MetricQuery.Step` | Query step parameter | Time resolution for range queries |
| `query.Metadata["query"]` | Raw PromQL | Bypasses query builder if provided |

**Filter Operators:**
- `=`: Exact match
- `!=`: Not equal
- `=~`: Regex match
- `!~`: Negative regex match

#### QueryScope Mapping

The adapter automatically maps OpsOrch `QueryScope` fields to Prometheus labels:

| QueryScope Field | Prometheus Label |
|-----------------|------------------|
| `Service` | `service` |
| `Team` | `team` |
| `Environment` | `env` |

#### Response Normalization

| Prometheus Field | OpsOrch Field | Notes |
|------------------|---------------|-------|
| `metric` | `Labels` | All metric labels |
| `values` | `Points` | Time-series data points |
| `__name__` | Extracted to metric name | Metric name from labels |

### Alert Adapter

#### Query Mapping

| OpsOrch Field | Alertmanager API Parameter | Notes |
|---------------|---------------------------|-------|
| `Statuses` | `filter` parameter with state matcher | Maps OpsOrch statuses (firing/resolved/open/closed) to Alertmanager states (active/suppressed) |
| `Severities` | `filter` parameter with severity label | Filters by `severity` label |
| `Scope` fields | `filter` parameter with label matchers | Adds label filters for service/team/env |

#### Response Normalization

| Alertmanager Field | OpsOrch Field | Notes |
|-------------------|---------------|-------|
| `labels.alertname` | `Title` | Alert name |
| `labels.severity` | `Severity` | Alert severity level |
| `labels.service` | `Service` | Service label mapped directly |
| `annotations.description` | `Description` | Alert description text |
| `status.state` | `Status` | Maps `active→firing`, `suppressed→suppressed`, `unprocessed→pending` |
| `startsAt` | `CreatedAt` | When alert started firing |
| `updatedAt` | `UpdatedAt` | Last Alertmanager update time (`endsAt` is not currently used) |
| `annotations` | `Fields["annotations"]` | Raw annotations preserved under `Fields` |
| `labels` | `Fields["labels"]` | All alert labels preserved under `Fields` |
| `fingerprint` | `ID` | Unique alert identifier (also stored in `Metadata["fingerprint"]` along with `Metadata["source"] = "prometheus"`) |

## Usage

### In-Process Mode

Import the alert adapter for side effects and register the metric provider explicitly with OpsOrch Core:

```go
import (
    coremetric "github.com/opsorch/opsorch-core/metric"
    adaptermetric "github.com/opsorch/opsorch-prometheus-adapter/metric"
    _ "github.com/opsorch/opsorch-prometheus-adapter/alert"
)

func init() {
    if err := coremetric.RegisterProvider("prometheus", func(cfg map[string]any) (coremetric.Provider, error) {
        return adaptermetric.NewPrometheusProvider(cfg)
    }); err != nil {
        panic(err)
    }
}
```

Configure via environment variables:

```bash
export OPSORCH_METRIC_PROVIDER=prometheus
export OPSORCH_METRIC_CONFIG='{"url":"http://prometheus:9090"}'

export OPSORCH_ALERT_PROVIDER=prometheus
export OPSORCH_ALERT_CONFIG='{"alertmanagerURL":"http://alertmanager:9093"}'
```

### Plugin Mode

Build the plugin binaries:

```bash
make plugin
```

This builds two plugin binaries in `./bin/`:
- `metricplugin`
- `alertplugin`

Configure OpsOrch Core to use the plugins:

```bash
# Metric Plugin
export OPSORCH_METRIC_PLUGIN=/path/to/bin/metricplugin
export OPSORCH_METRIC_CONFIG='{"url":"http://prometheus:9090"}'

# Alert Plugin
export OPSORCH_ALERT_PLUGIN=/path/to/bin/alertplugin
export OPSORCH_ALERT_CONFIG='{"alertmanagerURL":"http://alertmanager:9093"}'
```

### Docker Deployment

Download pre-built plugin binaries from [GitHub Releases](https://github.com/opsorch/opsorch-prometheus-adapter/releases):

```dockerfile
FROM ghcr.io/opsorch/opsorch-core:latest
WORKDIR /opt/opsorch

# Download plugin binaries
ADD https://github.com/opsorch/opsorch-prometheus-adapter/releases/download/v0.1.0/metricplugin-linux-amd64 ./plugins/metricplugin
ADD https://github.com/opsorch/opsorch-prometheus-adapter/releases/download/v0.1.0/alertplugin-linux-amd64 ./plugins/alertplugin
RUN chmod +x ./plugins/*

# Configure plugins
ENV OPSORCH_METRIC_PLUGIN=/opt/opsorch/plugins/metricplugin \
    OPSORCH_ALERT_PLUGIN=/opt/opsorch/plugins/alertplugin
```

## Query Examples

### Basic Metric Query

```json
{
  "expression": {
    "metricName": "http_requests_total"
  },
  "start": "2024-01-01T00:00:00Z",
  "end": "2024-01-01T01:00:00Z",
  "step": 60
}
```

Generates PromQL: `http_requests_total`

### Query with Filters

```json
{
  "expression": {
    "metricName": "http_requests_total",
    "filters": [
      {"label": "method", "operator": "=", "value": "POST"},
      {"label": "status", "operator": "=~", "value": "2.."}
    ]
  },
  "start": "2024-01-01T00:00:00Z",
  "end": "2024-01-01T01:00:00Z",
  "step": 60
}
```

Generates PromQL: `http_requests_total{method="POST",status=~"2.."}`

### Query with Aggregation

```json
{
  "expression": {
    "metricName": "http_requests_total",
    "aggregation": "sum",
    "groupBy": ["method", "status"]
  },
  "start": "2024-01-01T00:00:00Z",
  "end": "2024-01-01T01:00:00Z",
  "step": 60
}
```

Generates PromQL: `sum(http_requests_total) by (method, status)`

### Query with Scope

```json
{
  "expression": {
    "metricName": "http_requests_total"
  },
  "scope": {
    "service": "api",
    "environment": "prod"
  },
  "start": "2024-01-01T00:00:00Z",
  "end": "2024-01-01T01:00:00Z",
  "step": 60
}
```

Generates PromQL: `http_requests_total{service="api",env="prod"}`

### Raw PromQL Query

```json
{
  "expression": {"metricName": "ignored"},
  "metadata": {
    "query": "rate(http_requests_total[5m])"
  },
  "start": "2024-01-01T00:00:00Z",
  "end": "2024-01-01T01:00:00Z",
  "step": 60
}
```

Uses raw PromQL: `rate(http_requests_total[5m])`

## Development

### Prerequisites

- Go 1.21 or later
- Access to a Prometheus instance (for integration tests)
- Access to an Alertmanager instance (for alert integration tests)

### Building

```bash
# Download dependencies
go mod download

# Run unit tests
make test

# Build all packages
make build

# Build plugin binaries
make plugin

# Run integration tests (requires Prometheus and Alertmanager)
make integ
```

### Testing

**Unit Tests:**
```bash
make test
```

**Integration Tests:**

Integration tests require running Prometheus and Alertmanager instances. You can use Docker:

**Prerequisites:**
- Docker installed
- Prometheus running on localhost:9090
- Alertmanager running on localhost:9093

**Setup with Docker:**
```bash
# Start Prometheus
docker run --rm -d -p 9090:9090 --name prometheus prom/prometheus

# Start Alertmanager
docker run --rm -d -p 9093:9093 --name alertmanager prom/alertmanager

# Set environment variables
export PROMETHEUS_URL=http://localhost:9090
export ALERTMANAGER_URL=http://localhost:9093

# Run integration tests
make integ

# Or run specific capability tests
make integ-metric
make integ-alert

# Clean up
docker stop prometheus alertmanager
```

**What the tests do:**
- **Metric tests**: Query Prometheus metrics, test filtering, aggregation, and scope mapping
- **Alert tests**: Query Alertmanager alerts, test status and severity filtering
- **Discovery tests**: List available metrics from Prometheus

**Expected behavior:**
- Tests verify basic queries work
- Tests verify filtering and aggregation
- Tests verify scope mapping to labels
- Tests verify raw query overrides
- Tests may return empty results if no metrics/alerts exist

### Project Structure

```
opsorch-prometheus-adapter/
├── metric/                      # Metric provider implementation
│   ├── prometheus_provider.go  # Core provider logic
│   └── prometheus_provider_test.go
├── alert/                       # Alert provider implementation
│   ├── alertmanager_provider.go
│   └── alertmanager_provider_test.go
├── cmd/
│   ├── metricplugin/           # Metric plugin entrypoint
│   │   └── main.go
│   └── alertplugin/            # Alert plugin entrypoint
│       └── main.go
├── integ/                       # Integration tests
│   ├── metric/
│   │   └── main.go
│   └── alert/
│       └── main.go
├── Makefile
└── README.md
```

**Key Components:**

- **metric/prometheus_provider.go**: Implements metric.Provider interface, builds PromQL queries and executes range queries
- **alert/alertmanager_provider.go**: Implements alert.Provider interface, queries Alertmanager API
- **cmd/metricplugin**: JSON-RPC plugin wrapper for metric provider
- **cmd/alertplugin**: JSON-RPC plugin wrapper for alert provider

## CI/CD & Pre-Built Binaries

The repository includes GitHub Actions workflows:

- **CI** (`ci.yml`): Runs tests (including integration tests with Prometheus) and linting on every push/PR to main
- **Release** (`release.yml`): Manual workflow that:
  - Runs tests and linting
  - Creates version tags (patch/minor/major)
  - Builds multi-arch binaries for both plugins (linux-amd64, linux-arm64, darwin-amd64, darwin-arm64)
  - Publishes binaries as GitHub release assets

### Downloading Pre-Built Binaries

Pre-built plugin binaries are available from [GitHub Releases](https://github.com/opsorch/opsorch-prometheus-adapter/releases).

**Supported platforms:**
- Linux (amd64, arm64)
- macOS (amd64, arm64)

**Available binaries:**
- `metricplugin-{platform}-{arch}`
- `alertplugin-{platform}-{arch}`

## Plugin RPC Contract

OpsOrch Core communicates with the plugins over stdin/stdout using JSON-RPC.

### Message Format

**Request:**
```json
{
  "method": "{capability}.{operation}",
  "config": { /* decrypted configuration */ },
  "payload": { /* method-specific request body */ }
}
```

**Response:**
```json
{
  "result": { /* method-specific result */ },
  "error": "optional error message"
}
```

### Configuration Injection

The `config` field contains the decrypted configuration map from `OPSORCH_{CAPABILITY}_CONFIG`. The plugin receives this on every request, so it never stores secrets on disk.

### Supported Methods

#### Metric Plugin

- `metric.query`: Execute a metric query
- `metric.describe`: List available metrics

**Example - metric.query:**
```json
{
  "method": "metric.query",
  "config": {"url": "http://prometheus:9090"},
  "payload": {
    "expression": {"metricName": "http_requests_total"},
    "start": "2024-01-01T00:00:00Z",
    "end": "2024-01-01T01:00:00Z",
    "step": 60
  }
}
```

**Response:**
```json
{
  "result": [
    {
      "labels": {"method": "GET", "status": "200"},
      "points": [[1704067200, 100], [1704067260, 105]]
    }
  ]
}
```

**Example - metric.describe:**
```json
{
  "method": "metric.describe",
  "config": {"url": "http://prometheus:9090"},
  "payload": {}
}
```

**Response:**
```json
{
  "result": [
    {"name": "http_requests_total", "type": "unknown"},
    {"name": "http_request_duration_seconds", "type": "unknown"}
  ]
}
```

The `type` field is currently returned as `"unknown"` until the provider begins querying the Prometheus metadata API.

#### Alert Plugin

- `alert.query`: Query alerts
- `alert.get`: Get alert details

**Example - alert.query:**
```json
{
  "method": "alert.query",
  "config": {"alertmanagerURL": "http://alertmanager:9093"},
  "payload": {
    "statuses": ["firing"],
    "severities": ["critical", "warning"]
  }
}
```

**Response:**
```json
{
  "result": [
    {
      "id": "abc123",
      "title": "HighErrorRate",
      "severity": "critical",
      "status": "firing",
      "createdAt": "2024-01-01T10:00:00Z"
    }
  ]
}
```

## Security Considerations

1. **Network access**: Ensure Prometheus and Alertmanager are accessible from OpsOrch Core
2. **Authentication**: If Prometheus/Alertmanager require authentication, configure it appropriately
3. **TLS**: Use HTTPS URLs for production deployments
4. **Firewall rules**: Restrict access to Prometheus/Alertmanager to authorized systems only
5. **Query limits**: Be mindful of query complexity and time ranges to avoid overloading Prometheus

## License

Apache 2.0

See LICENSE file in the repository root.
