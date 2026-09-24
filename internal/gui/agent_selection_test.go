package gui

import (
	"context"
	"testing"

	"keysmith/internal/core"
)

func TestCheckAgentUsesContextOperation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := checkAgent(ctx, func(context.Context) core.Result {
		return core.Result{Message: "cancelled by test"}
	})
	if result.Message != "cancelled by test" {
		t.Fatalf("result = %+v, want test message", result)
	}
}
