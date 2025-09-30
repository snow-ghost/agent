package local

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/snow-ghost/agent/core"
)

// Guard is a simple implementation of core.PolicyGuard
// - Wrap: enforces timeout from Budget (Timeout or CPUMillis), and approximates CPU ticks by wall time
// - AllowTool: hostname/name allowlist
// Note: Memory limits are advisory in this layer; actual enforcement occurs in WASM runtime config.
type Guard struct {
	allow map[string]bool
}

func NewGuard(allowlist []string) *Guard {
	m := make(map[string]bool, len(allowlist))
	for _, n := range allowlist {
		m[strings.ToLower(n)] = true
	}
	return &Guard{allow: m}
}

// Wrap applies a timeout based on Budget and runs the function.
// Order of precedence: Task.Timeout > CPUMillis > default 30s.
func (g *Guard) Wrap(ctx context.Context, b core.Budget, run func(ctx context.Context) error) error {
	var timeout time.Duration
	switch {
	case b.Timeout > 0:
		timeout = b.Timeout
	case b.CPUMillis > 0:
		timeout = time.Duration(b.CPUMillis) * time.Millisecond
	default:
		timeout = 30 * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- run(execCtx)
	}()

	select {
	case <-execCtx.Done():
		// return context error to signal timeout/cancel
		if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
			return context.DeadlineExceeded
		}
		return execCtx.Err()
	case err := <-done:
		return err
	}
}

// AllowTool returns true if the tool name/hostname is allowlisted.
func (g *Guard) AllowTool(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return false
	}
	// allow exact matches
	if g.allow[name] {
		return true
	}
	// allow host-based entries like http:example.com
	// try to match by suffix after ':'
	if i := strings.LastIndexByte(name, ':'); i >= 0 && i+1 < len(name) {
		if g.allow[name[i+1:]] {
			return true
		}
	}
	return false
}
