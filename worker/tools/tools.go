package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/snow-ghost/agent/core"
)

// Adapter provides access to limited external tools, guarded by PolicyGuard.
type Adapter struct {
	guard core.PolicyGuard
	client *http.Client
}

func NewAdapter(guard core.PolicyGuard) *Adapter {
	return &Adapter{
		guard: guard,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// HTTPGetJSON performs an HTTP GET to an allowlisted domain and parses JSON into out.
func (a *Adapter) HTTPGetJSON(ctx context.Context, rawURL string, out any) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if u.Host == "" {
		return errors.New("missing host")
	}
	if !a.guard.AllowTool("http:" + u.Host) {
		return fmt.Errorf("tool not allowed: %s", u.Host)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

// ParseJSON parses a JSON string into out.
func (a *Adapter) ParseJSON(_ context.Context, data []byte, out any) error {
	return json.Unmarshal(data, out)
}
