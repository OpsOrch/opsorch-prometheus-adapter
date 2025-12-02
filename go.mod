module github.com/opsorch/opsorch-prometheus-adapter

go 1.24.0

replace github.com/opsorch/opsorch-core => ../opsorch-core

require (
	github.com/opsorch/opsorch-core v0.0.2
	github.com/prometheus/client_golang v1.23.2
	github.com/prometheus/common v0.67.4
)

require (
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)
