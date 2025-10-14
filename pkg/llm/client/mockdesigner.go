package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/snow-ghost/agent/design"
)

// MockDesigner implements Designer by reading a fixture JSON from testdata
type MockDesigner struct {
	// Path to fixture file (defaults to testdata/design_sort.json)
	FixturePath string
}

func (m *MockDesigner) Design(ctx context.Context, taskJSON string) (design.HypothesisDesign, []byte, error) {
	path := m.FixturePath
	if path == "" {
		path = "testdata/design_sort.json"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return design.HypothesisDesign{}, nil, fmt.Errorf("failed to read fixture: %w", err)
	}

	var hd design.HypothesisDesign
	if err := json.Unmarshal(data, &hd); err != nil {
		return design.HypothesisDesign{}, data, fmt.Errorf("failed to parse fixture: %w", err)
	}

	return hd, data, nil
}

var _ Designer = (*MockDesigner)(nil)
