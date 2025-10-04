package accounting

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryAggregator implements in-memory cost aggregation
type MemoryAggregator struct {
	records []CostRecord
	mu      sync.RWMutex
}

// NewMemoryAggregator creates a new in-memory aggregator
func NewMemoryAggregator() *MemoryAggregator {
	return &MemoryAggregator{
		records: make([]CostRecord, 0),
	}
}

// RecordCost records a cost
func (m *MemoryAggregator) RecordCost(record CostRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Set timestamp if not set
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}

	// Set ID if not set
	if record.ID == 0 {
		record.ID = int64(len(m.records) + 1)
	}

	m.records = append(m.records, record)
	return nil
}

// GetCosts retrieves costs with filters
func (m *MemoryAggregator) GetCosts(filter CostFilter) ([]CostRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var filtered []CostRecord

	for _, record := range m.records {
		if m.matchesFilter(record, filter) {
			filtered = append(filtered, record)
		}
	}

	// Sort by timestamp descending
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.After(filtered[j].Timestamp)
	})

	// Apply pagination
	if filter.Limit > 0 {
		start := filter.Offset
		end := start + filter.Limit
		if end > len(filtered) {
			end = len(filtered)
		}
		if start < len(filtered) {
			filtered = filtered[start:end]
		} else {
			filtered = []CostRecord{}
		}
	}

	return filtered, nil
}

// GetCostSummary gets cost summary with filters
func (m *MemoryAggregator) GetCostSummary(filter CostFilter) (CostSummary, error) {
	records, err := m.GetCosts(filter)
	if err != nil {
		return CostSummary{}, err
	}

	summary := CostSummary{
		TotalRecords: int64(len(records)),
		Currency:     "USD", // Default currency
	}

	for _, record := range records {
		summary.TotalCost += record.CostTotal
		summary.TotalInputCost += record.CostInput
		summary.TotalOutputCost += record.CostOutput
		summary.TotalPromptTokens += int64(record.PromptTokens)
		summary.TotalCompletionTokens += int64(record.CompletionTokens)

		// Use the currency from the first record
		if summary.Currency == "USD" && record.Currency != "" {
			summary.Currency = record.Currency
		}
	}

	return summary, nil
}

// GetCostReport generates a cost report
func (m *MemoryAggregator) GetCostReport(filter CostFilter) (CostReport, error) {
	records, err := m.GetCosts(filter)
	if err != nil {
		return CostReport{}, err
	}

	summary, err := m.GetCostSummary(filter)
	if err != nil {
		return CostReport{}, err
	}

	report := CostReport{
		From:    time.Now().AddDate(0, 0, -30), // Default to last 30 days
		To:      time.Now(),
		GroupBy: filter.GroupBy,
		Summary: summary,
		Records: records,
	}

	// Set time range from filter
	if filter.From != nil {
		report.From = *filter.From
	}
	if filter.To != nil {
		report.To = *filter.To
	}

	// Group records if GroupBy is specified
	if filter.GroupBy != "" {
		groups := m.groupRecords(records, filter.GroupBy)
		report.Groups = groups
		report.Records = nil // Don't include individual records when grouped
	}

	return report, nil
}

// GetBudgetInfo gets budget information for a caller
func (m *MemoryAggregator) GetBudgetInfo(caller string, amount float64, currency string) (BudgetInfo, error) {
	filter := CostFilter{
		Caller:   caller,
		Currency: currency,
	}

	summary, err := m.GetCostSummary(filter)
	if err != nil {
		return BudgetInfo{}, err
	}

	used := summary.TotalCost
	remaining := amount - used
	exceeded := used > amount

	return BudgetInfo{
		Amount:    amount,
		Currency:  currency,
		Used:      used,
		Remaining: remaining,
		Exceeded:  exceeded,
	}, nil
}

// ExportCosts exports costs in specified format
func (m *MemoryAggregator) ExportCosts(filter CostFilter, format ExportFormat) ([]byte, error) {
	records, err := m.GetCosts(filter)
	if err != nil {
		return nil, err
	}

	switch format {
	case ExportFormatJSON:
		return json.MarshalIndent(records, "", "  ")
	case ExportFormatCSV:
		return m.exportCSV(records)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// matchesFilter checks if a record matches the filter
func (m *MemoryAggregator) matchesFilter(record CostRecord, filter CostFilter) bool {
	// Time range filter
	if filter.From != nil && record.Timestamp.Before(*filter.From) {
		return false
	}
	if filter.To != nil && record.Timestamp.After(*filter.To) {
		return false
	}

	// String filters
	if filter.Caller != "" && record.Caller != filter.Caller {
		return false
	}
	if filter.Provider != "" && record.Provider != filter.Provider {
		return false
	}
	if filter.Model != "" && record.Model != filter.Model {
		return false
	}
	if filter.Currency != "" && record.Currency != filter.Currency {
		return false
	}

	return true
}

// groupRecords groups records by the specified field
func (m *MemoryAggregator) groupRecords(records []CostRecord, groupBy string) []CostGroup {
	groups := make(map[string][]CostRecord)

	for _, record := range records {
		var key string
		switch groupBy {
		case "provider":
			key = record.Provider
		case "model":
			key = record.Model
		case "caller":
			key = record.Caller
		case "currency":
			key = record.Currency
		default:
			key = "unknown"
		}

		groups[key] = append(groups[key], record)
	}

	var result []CostGroup
	for key, groupRecords := range groups {
		groupFilter := CostFilter{
			Caller:   groupRecords[0].Caller,
			Provider: groupRecords[0].Provider,
			Model:    groupRecords[0].Model,
			Currency: groupRecords[0].Currency,
		}

		// Apply group-specific filters
		switch groupBy {
		case "provider":
			groupFilter.Provider = key
		case "model":
			groupFilter.Model = key
		case "caller":
			groupFilter.Caller = key
		case "currency":
			groupFilter.Currency = key
		}

		summary, _ := m.GetCostSummary(groupFilter)

		result = append(result, CostGroup{
			GroupBy:    groupBy,
			GroupValue: key,
			Summary:    summary,
			Records:    groupRecords,
		})
	}

	// Sort groups by total cost descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].Summary.TotalCost > result[j].Summary.TotalCost
	})

	return result
}

// exportCSV exports records as CSV
func (m *MemoryAggregator) exportCSV(records []CostRecord) ([]byte, error) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{
		"ID", "Timestamp", "Caller", "Provider", "Model",
		"Prompt Tokens", "Completion Tokens", "Currency",
		"Cost Input", "Cost Output", "Cost Total", "Request ID",
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	// Write records
	for _, record := range records {
		row := []string{
			fmt.Sprintf("%d", record.ID),
			record.Timestamp.Format(time.RFC3339),
			record.Caller,
			record.Provider,
			record.Model,
			fmt.Sprintf("%d", record.PromptTokens),
			fmt.Sprintf("%d", record.CompletionTokens),
			record.Currency,
			fmt.Sprintf("%.6f", record.CostInput),
			fmt.Sprintf("%.6f", record.CostOutput),
			fmt.Sprintf("%.6f", record.CostTotal),
			record.RequestID,
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	return []byte(buf.String()), nil
}

// Close closes the aggregator
func (m *MemoryAggregator) Close() error {
	// Nothing to close for in-memory aggregator
	return nil
}
