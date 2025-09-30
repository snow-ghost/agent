package local

import (
	"context"
	"testing"
	"time"

	"github.com/snow-ghost/agent/core"
	"github.com/stretchr/testify/assert"
)

func TestGuard_AllowTool(t *testing.T) {
	g := NewGuard([]string{"example.com", "jsonplaceholder.typicode.com"})
	assert.True(t, g.AllowTool("http:example.com"))
	assert.True(t, g.AllowTool("example.com"))
	assert.False(t, g.AllowTool("http:evil.com"))
	assert.False(t, g.AllowTool(""))
}

func TestGuard_WrapTimeout(t *testing.T) {
	g := NewGuard(nil)
	ctx := context.Background()
	budget := core.Budget{Timeout: 10 * time.Millisecond}

	start := time.Now()
	err := g.Wrap(ctx, budget, func(ctx context.Context) error {
		// Simulate long work
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
			return nil
		}
	})
	elapsed := time.Since(start)
	assert.Error(t, err)
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(10))
}
