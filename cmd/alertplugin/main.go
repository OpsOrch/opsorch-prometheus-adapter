package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	corealert "github.com/opsorch/opsorch-core/alert"
	"github.com/opsorch/opsorch-core/schema"
	_ "github.com/opsorch/opsorch-prometheus-adapter/alert"
)

type rpcRequest struct {
	Method  string         `json:"method"`
	Config  map[string]any `json:"config"`
	Payload any            `json:"payload"`
}

type rpcResponse struct {
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func main() {
	dec := json.NewDecoder(os.Stdin)
	enc := json.NewEncoder(os.Stdout)

	for {
		var req rpcRequest
		if err := dec.Decode(&req); err != nil {
			if err == io.EOF {
				break
			}
			_ = enc.Encode(rpcResponse{Error: fmt.Sprintf("decode request: %v", err)})
			continue
		}

		resp := handle(req)
		if err := enc.Encode(resp); err != nil {
			fmt.Fprintf(os.Stderr, "encode response: %v\n", err)
			break
		}
	}
}

func handle(req rpcRequest) rpcResponse {
	ctor, ok := corealert.LookupProvider("prometheus")
	if !ok {
		return rpcResponse{Error: "prometheus alert provider not registered"}
	}

	prov, err := ctor(req.Config)
	if err != nil {
		return rpcResponse{Error: fmt.Sprintf("create provider: %v", err)}
	}

	ctx := context.Background()

	switch req.Method {
	case "alert.query":
		var query schema.AlertQuery
		if err := remarshal(req.Payload, &query); err != nil {
			return rpcResponse{Error: fmt.Sprintf("decode query: %v", err)}
		}
		alerts, err := prov.Query(ctx, query)
		if err != nil {
			return rpcResponse{Error: fmt.Sprintf("query alerts: %v", err)}
		}
		return rpcResponse{Result: alerts}

	case "alert.get":
		payload, ok := req.Payload.(map[string]any)
		if !ok {
			return rpcResponse{Error: "invalid payload format"}
		}
		id, ok := payload["id"].(string)
		if !ok {
			return rpcResponse{Error: "missing or invalid id"}
		}
		alert, err := prov.Get(ctx, id)
		if err != nil {
			return rpcResponse{Error: fmt.Sprintf("get alert: %v", err)}
		}
		return rpcResponse{Result: alert}

	default:
		return rpcResponse{Error: fmt.Sprintf("unknown method: %s", req.Method)}
	}
}

func remarshal(from, to any) error {
	data, err := json.Marshal(from)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, to)
}
