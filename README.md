# OpsOrch Prometheus Adapter

This adapter integrates OpsOrch with Prometheus, enabling metric querying, alert monitoring, and discovery through the `metric.Provider` and `alert.Provider` interfaces.

## Features

### Metrics
- **Metric Query**: Execute PromQL queries via structured expressions or raw query strings
- **Metric Discovery**: List all available metrics in your Prometheus instance
- **QueryScope Support**: Automatically map service/team/environment to Prometheus labels
- **Aggregation**: Support for aggregation functions (`sum`, `avg`, `max`, `min`, `count`)
- **Filtering**: Label-based filtering with multiple operators (`=`, `!=`, `=~`, `!~`)

### Alerts
- **Alert Query**: Fetch firing and resolved alerts from Prometheus Alertmanager
- **Alert Details**: Get individual alerts by fingerprint
- **Status Filtering**: Filter alerts by state (active, suppressed)
- **Severity Filtering**: Filter alerts by severity level

## Configuration

### Metric Provider

The metric adapter requires the following configuration:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | Yes | The base URL of the Prometheus server (e.g., `http://prometheus:9090`) |

**Example:**
```json
{
  "url": "http://localhost:9090"
}
```

### Alert Provider

The alert adapter requires the following configuration:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `alertmanagerURL` | string | Yes | The base URL of the Prometheus Alertmanager (e.g., `http://alertmanager:9093`) |

**Example:**
```json
{
  "alertmanagerURL": "http://localhost:9093"
}
```

## Usage

### In OpsOrch Core

Configure the adapter as a plugin:

```bash
export OPSORCH_METRIC_PLUGIN=/path/to/bin/metricplugin
export OPSORCH_METRIC_CONFIG='{"url":"http://prometheus:9090"}'
```

### Query Examples

#### Basic Query
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

#### Query with Filters
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

#### Query with Aggregation
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

#### Query with Scope
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
*Generates: `http_requests_total{service="api",env="prod"}`*

#### Raw PromQL Query
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

## QueryScope Mapping

The adapter automatically maps OpsOrch `QueryScope` fields to Prometheus labels:

| QueryScope Field | Prometheus Label |
|-----------------|------------------|
| `Service` | `service` |
| `Team` | `team` |
| `Environment` | `env` |

## Development

### Prerequisites

- Go 1.21 or later
- Access to a Prometheus instance (for integration tests)

### Building

```bash
# Build the plugin binary
make plugin

# Run unit tests
make test

# Run all tests and build
make build
```

### CI/CD

The repository includes GitHub Actions workflows:

- **CI** (`ci.yml`): Runs tests (including integration tests with Prometheus) and linting on every push/PR to main
- **Release** (`release.yml`): Manual workflow that:
  - Runs tests and linting
  - Creates version tags (patch/minor/major)
  - Builds multi-arch binaries (linux-amd64, linux-arm64, darwin-amd64, darwin-arm64)
  - Publishes binaries as GitHub release assets

### Pre-Built Binaries

Download pre-built plugin binaries from [GitHub Releases](https://github.com/opsorch/opsorch-prometheus-adapter/releases):

```dockerfile
# Use in custom Docker images
FROM ghcr.io/opsorch/opsorch-core:latest
WORKDIR /opt/opsorch

ADD https://github.com/opsorch/opsorch-prometheus-adapter/releases/download/v0.1.0/metricplugin-linux-amd64 ./plugins/metricplugin
RUN chmod +x ./plugins/metricplugin

ENV OPSORCH_METRIC_PLUGIN=/opt /opsorch/plugins/metricplugin
```

### Testing

**Unit Tests:**
```bash
make test
```

**Integration Tests:**

To run integration tests, you need Prometheus and Alertmanager running locally. You can use Docker:

```bash
# Start Prometheus
docker run --rm -d -p 9090:9090 --name prometheus prom/prometheus

# Start Alertmanager
docker run --rm -d -p 9093:9093 --name alertmanager prom/alertmanager

# Run integration tests
export PROMETHEUS_URL=http://localhost:9090
export ALERTMANAGER_URL=http://localhost:9093
make integ

# Clean up
docker stop prometheus alertmanager
```

Integration tests require a running Prometheus instance and cover:
- Basic queries
- Filtering
- Aggregation
- Scope mapping
- Raw query overrides

### Project Structure

```
opsorch-prometheus-adapter/
├── metric/
│   ├── prometheus_provider.go      # Core provider implementation
│   └── prometheus_provider_test.go # Unit tests
├── cmd/
│   └── metricplugin/
│       └── main.go                  # Plugin entrypoint
├── integ/
│   ├── metric/
│   │   └── main.go                  # Metric integration tests
│   └── alert/
│       └── main.go                  # Alert integration tests
├── Makefile                         # Build targets
└── README.md
```

## Plugin RPC Contract

OpsOrch Core communicates with the plugin over stdin/stdout using JSON-RPC.

**Request:**
```json
{
  "method": "metric.query",
  "config": {"url": "http://prometheus:9090"},
  "payload": { /* MetricQuery */ }
}
```

**Response:**
```json
{
  "result": [ /* MetricSeries[] */ ],
  "error": "optional error message"
}
```

**Supported Methods:**
- `metric.query`: Execute a metric query
- `metric.describe`: List available metrics
- `alert.query`: Query alerts
- `alert.get`: Get alert details

## License

See LICENSE file in the repository root.
