package accounting

import (
	"time"
)

// CostRecord represents a single cost record
type CostRecord struct {
	ID               int64     `json:"id" db:"id"`
	Timestamp        time.Time `json:"timestamp" db:"timestamp"`
	Caller           string    `json:"caller" db:"caller"`
	Provider         string    `json:"provider" db:"provider"`
	Model            string    `json:"model" db:"model"`
	PromptTokens     int       `json:"prompt_tokens" db:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens" db:"completion_tokens"`
	Currency         string    `json:"currency" db:"currency"`
	CostInput        float64   `json:"cost_input" db:"cost_input"`
	CostOutput       float64   `json:"cost_output" db:"cost_output"`
	CostTotal        float64   `json:"cost_total" db:"cost_total"`
	RequestID        string    `json:"request_id" db:"request_id"`
}

// CostSummary represents aggregated cost data
type CostSummary struct {
	TotalRecords          int64   `json:"total_records"`
	TotalCost             float64 `json:"total_cost"`
	TotalInputCost        float64 `json:"total_input_cost"`
	TotalOutputCost       float64 `json:"total_output_cost"`
	TotalPromptTokens     int64   `json:"total_prompt_tokens"`
	TotalCompletionTokens int64   `json:"total_completion_tokens"`
	Currency              string  `json:"currency"`
}

// CostGroup represents cost data grouped by a field
type CostGroup struct {
	GroupBy    string       `json:"group_by"`
	GroupValue string       `json:"group_value"`
	Summary    CostSummary  `json:"summary"`
	Records    []CostRecord `json:"records,omitempty"`
}

// CostReport represents a cost report with optional grouping
type CostReport struct {
	From    time.Time    `json:"from"`
	To      time.Time    `json:"to"`
	GroupBy string       `json:"group_by,omitempty"` // provider, model, caller, currency
	Summary CostSummary  `json:"summary"`
	Groups  []CostGroup  `json:"groups,omitempty"`
	Records []CostRecord `json:"records,omitempty"`
}

// BudgetInfo represents budget information
type BudgetInfo struct {
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Used      float64 `json:"used"`
	Remaining float64 `json:"remaining"`
	Exceeded  bool    `json:"exceeded"`
}

// CostFilter represents filters for cost queries
type CostFilter struct {
	From     *time.Time `json:"from,omitempty"`
	To       *time.Time `json:"to,omitempty"`
	Caller   string     `json:"caller,omitempty"`
	Provider string     `json:"provider,omitempty"`
	Model    string     `json:"model,omitempty"`
	Currency string     `json:"currency,omitempty"`
	GroupBy  string     `json:"group_by,omitempty"`
	Limit    int        `json:"limit,omitempty"`
	Offset   int        `json:"offset,omitempty"`
}

// ExportFormat represents supported export formats
type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json"
	ExportFormatCSV  ExportFormat = "csv"
)

// CostAggregator interface for cost aggregation
type CostAggregator interface {
	// RecordCost records a cost
	RecordCost(record CostRecord) error

	// GetCosts retrieves costs with filters
	GetCosts(filter CostFilter) ([]CostRecord, error)

	// GetCostSummary gets cost summary with filters
	GetCostSummary(filter CostFilter) (CostSummary, error)

	// GetCostReport generates a cost report
	GetCostReport(filter CostFilter) (CostReport, error)

	// GetBudgetInfo gets budget information for a caller
	GetBudgetInfo(caller string, amount float64, currency string) (BudgetInfo, error)

	// ExportCosts exports costs in specified format
	ExportCosts(filter CostFilter, format ExportFormat) ([]byte, error)

	// Close closes the aggregator
	Close() error
}
