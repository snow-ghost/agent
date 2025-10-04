package accounting

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Manager manages cost accounting
type Manager struct {
	aggregator CostAggregator
}

// Config holds accounting configuration
type Config struct {
	UseSQLite bool
	DBPath    string
}

// NewManager creates a new accounting manager
func NewManager(config Config) (*Manager, error) {
	var aggregator CostAggregator
	var err error

	if config.UseSQLite {
		aggregator, err = NewSQLiteAggregator(config.DBPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create SQLite aggregator: %w", err)
		}
	} else {
		aggregator = NewMemoryAggregator()
	}

	return &Manager{
		aggregator: aggregator,
	}, nil
}

// RecordCost records a cost
func (m *Manager) RecordCost(record CostRecord) error {
	return m.aggregator.RecordCost(record)
}

// GetCosts retrieves costs with filters
func (m *Manager) GetCosts(filter CostFilter) ([]CostRecord, error) {
	return m.aggregator.GetCosts(filter)
}

// GetCostSummary gets cost summary with filters
func (m *Manager) GetCostSummary(filter CostFilter) (CostSummary, error) {
	return m.aggregator.GetCostSummary(filter)
}

// GetCostReport generates a cost report
func (m *Manager) GetCostReport(filter CostFilter) (CostReport, error) {
	return m.aggregator.GetCostReport(filter)
}

// GetBudgetInfo gets budget information for a caller
func (m *Manager) GetBudgetInfo(caller string, amount float64, currency string) (BudgetInfo, error) {
	return m.aggregator.GetBudgetInfo(caller, amount, currency)
}

// ExportCosts exports costs in specified format
func (m *Manager) ExportCosts(filter CostFilter, format ExportFormat) ([]byte, error) {
	return m.aggregator.ExportCosts(filter, format)
}

// Close closes the manager
func (m *Manager) Close() error {
	return m.aggregator.Close()
}

// ParseBudgetHeader parses X-Budget-Amount header
func ParseBudgetHeader(header string) (float64, string, error) {
	if header == "" {
		return 0, "", nil
	}

	// Expected format: "amount;currency=USD"
	parts := strings.Split(header, ";")
	if len(parts) == 0 {
		return 0, "", fmt.Errorf("invalid budget header format")
	}

	amount, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid amount in budget header: %w", err)
	}

	currency := "USD" // Default currency
	if len(parts) > 1 {
		// Parse currency=USD
		currencyPart := parts[1]
		if strings.HasPrefix(currencyPart, "currency=") {
			currency = strings.TrimPrefix(currencyPart, "currency=")
		}
	}

	return amount, currency, nil
}

// CheckBudget checks if the caller has exceeded their budget
func (m *Manager) CheckBudget(caller string, budgetHeader string) (BudgetInfo, error) {
	if budgetHeader == "" {
		// No budget specified
		return BudgetInfo{}, nil
	}

	amount, currency, err := ParseBudgetHeader(budgetHeader)
	if err != nil {
		return BudgetInfo{}, err
	}

	return m.GetBudgetInfo(caller, amount, currency)
}

// RecordLLMCost records a cost from an LLM request
func (m *Manager) RecordLLMCost(caller, provider, model, requestID string, promptTokens, completionTokens int, costInput, costOutput, costTotal float64, currency string) error {
	record := CostRecord{
		Timestamp:        time.Now(),
		Caller:           caller,
		Provider:         provider,
		Model:            model,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		Currency:         currency,
		CostInput:        costInput,
		CostOutput:       costOutput,
		CostTotal:        costTotal,
		RequestID:        requestID,
	}

	return m.RecordCost(record)
}

// GetCostsByTimeRange gets costs within a time range
func (m *Manager) GetCostsByTimeRange(from, to time.Time, caller string) ([]CostRecord, error) {
	filter := CostFilter{
		From:   &from,
		To:     &to,
		Caller: caller,
	}

	return m.GetCosts(filter)
}

// GetCostsByProvider gets costs by provider
func (m *Manager) GetCostsByProvider(provider string, from, to *time.Time) ([]CostRecord, error) {
	filter := CostFilter{
		Provider: provider,
		From:     from,
		To:       to,
	}

	return m.GetCosts(filter)
}

// GetCostsByModel gets costs by model
func (m *Manager) GetCostsByModel(model string, from, to *time.Time) ([]CostRecord, error) {
	filter := CostFilter{
		Model: model,
		From:  from,
		To:    to,
	}

	return m.GetCosts(filter)
}

// GetTopCallers gets top callers by cost
func (m *Manager) GetTopCallers(limit int, from, to *time.Time) ([]CostGroup, error) {
	filter := CostFilter{
		GroupBy: "caller",
		From:    from,
		To:      to,
		Limit:   limit,
	}

	report, err := m.GetCostReport(filter)
	if err != nil {
		return nil, err
	}

	return report.Groups, nil
}

// GetTopProviders gets top providers by cost
func (m *Manager) GetTopProviders(limit int, from, to *time.Time) ([]CostGroup, error) {
	filter := CostFilter{
		GroupBy: "provider",
		From:    from,
		To:      to,
		Limit:   limit,
	}

	report, err := m.GetCostReport(filter)
	if err != nil {
		return nil, err
	}

	return report.Groups, nil
}

// GetTopModels gets top models by cost
func (m *Manager) GetTopModels(limit int, from, to *time.Time) ([]CostGroup, error) {
	filter := CostFilter{
		GroupBy: "model",
		From:    from,
		To:      to,
		Limit:   limit,
	}

	report, err := m.GetCostReport(filter)
	if err != nil {
		return nil, err
	}

	return report.Groups, nil
}
