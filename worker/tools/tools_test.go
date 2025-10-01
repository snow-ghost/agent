package tools

import (
	"context"
	"testing"
	"time"

	"github.com/snow-ghost/agent/policy/local"
	"github.com/stretchr/testify/assert"
)

func TestAdapter_HTTPGetJSON_Denied(t *testing.T) {
	guard := local.NewGuard([]string{"example.com"})
	adapter := NewAdapter(guard)
	ctx := context.Background()
	var out map[string]any
	err := adapter.HTTPGetJSON(ctx, "https://not-allowed.com/data.json", &out)
	assert.Error(t, err)
}

func TestAdapter_HTTPGetJSON_Timeout(t *testing.T) {
	guard := local.NewGuard([]string{"example.com"})
	adapter := NewAdapter(guard)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	var out map[string]any
	// even allowed host but impossible to complete before timeout
	err := adapter.HTTPGetJSON(ctx, "https://example.com/data.json", &out)
	assert.Error(t, err)
}

func TestAdapter_ParseJSON(t *testing.T) {
	guard := local.NewGuard(nil)
	adapter := NewAdapter(guard)
	ctx := context.Background()
	var out struct {
		Name string `json:"name"`
	}
	err := adapter.ParseJSON(ctx, []byte(`{"name":"test"}`), &out)
	assert.NoError(t, err)
	assert.Equal(t, "test", out.Name)
}
