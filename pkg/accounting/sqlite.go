package accounting

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteAggregator implements SQLite-based cost aggregation
type SQLiteAggregator struct {
	db *sql.DB
}

// NewSQLiteAggregator creates a new SQLite aggregator
func NewSQLiteAggregator(dbPath string) (*SQLiteAggregator, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	aggregator := &SQLiteAggregator{db: db}

	// Create table if not exists
	if err := aggregator.createTable(); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return aggregator, nil
}

// createTable creates the costs table
func (s *SQLiteAggregator) createTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS costs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		caller TEXT NOT NULL,
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		prompt_tokens INTEGER NOT NULL,
		completion_tokens INTEGER NOT NULL,
		currency TEXT NOT NULL,
		cost_input REAL NOT NULL,
		cost_output REAL NOT NULL,
		cost_total REAL NOT NULL,
		request_id TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_costs_timestamp ON costs(timestamp);
	CREATE INDEX IF NOT EXISTS idx_costs_caller ON costs(caller);
	CREATE INDEX IF NOT EXISTS idx_costs_provider ON costs(provider);
	CREATE INDEX IF NOT EXISTS idx_costs_model ON costs(model);
	CREATE INDEX IF NOT EXISTS idx_costs_currency ON costs(currency);
	`

	_, err := s.db.Exec(query)
	return err
}

// RecordCost records a cost
func (s *SQLiteAggregator) RecordCost(record CostRecord) error {
	query := `
	INSERT INTO costs (
		timestamp, caller, provider, model, prompt_tokens, completion_tokens,
		currency, cost_input, cost_output, cost_total, request_id
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		record.Timestamp,
		record.Caller,
		record.Provider,
		record.Model,
		record.PromptTokens,
		record.CompletionTokens,
		record.Currency,
		record.CostInput,
		record.CostOutput,
		record.CostTotal,
		record.RequestID,
	)

	return err
}

// GetCosts retrieves costs with filters
func (s *SQLiteAggregator) GetCosts(filter CostFilter) ([]CostRecord, error) {
	query, args := s.buildQuery(filter)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []CostRecord
	for rows.Next() {
		var record CostRecord
		err := rows.Scan(
			&record.ID,
			&record.Timestamp,
			&record.Caller,
			&record.Provider,
			&record.Model,
			&record.PromptTokens,
			&record.CompletionTokens,
			&record.Currency,
			&record.CostInput,
			&record.CostOutput,
			&record.CostTotal,
			&record.RequestID,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}

// GetCostSummary gets cost summary with filters
func (s *SQLiteAggregator) GetCostSummary(filter CostFilter) (CostSummary, error) {
	whereClause, args := s.buildWhereClause(filter)

	query := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total_records,
			COALESCE(SUM(cost_total), 0) as total_cost,
			COALESCE(SUM(cost_input), 0) as total_input_cost,
			COALESCE(SUM(cost_output), 0) as total_output_cost,
			COALESCE(SUM(prompt_tokens), 0) as total_prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) as total_completion_tokens,
			COALESCE(currency, 'USD') as currency
		FROM costs
		%s
	`, whereClause)

	var summary CostSummary
	err := s.db.QueryRow(query, args...).Scan(
		&summary.TotalRecords,
		&summary.TotalCost,
		&summary.TotalInputCost,
		&summary.TotalOutputCost,
		&summary.TotalPromptTokens,
		&summary.TotalCompletionTokens,
		&summary.Currency,
	)

	return summary, err
}

// GetCostReport generates a cost report
func (s *SQLiteAggregator) GetCostReport(filter CostFilter) (CostReport, error) {
	records, err := s.GetCosts(filter)
	if err != nil {
		return CostReport{}, err
	}

	summary, err := s.GetCostSummary(filter)
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
		groups, err := s.getGroupedCosts(filter)
		if err != nil {
			return CostReport{}, err
		}
		report.Groups = groups
		report.Records = nil // Don't include individual records when grouped
	}

	return report, nil
}

// GetBudgetInfo gets budget information for a caller
func (s *SQLiteAggregator) GetBudgetInfo(caller string, amount float64, currency string) (BudgetInfo, error) {
	filter := CostFilter{
		Caller:   caller,
		Currency: currency,
	}

	summary, err := s.GetCostSummary(filter)
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
func (s *SQLiteAggregator) ExportCosts(filter CostFilter, format ExportFormat) ([]byte, error) {
	records, err := s.GetCosts(filter)
	if err != nil {
		return nil, err
	}

	switch format {
	case ExportFormatJSON:
		return json.MarshalIndent(records, "", "  ")
	case ExportFormatCSV:
		return s.exportCSV(records)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// buildQuery builds a SQL query with filters
func (s *SQLiteAggregator) buildQuery(filter CostFilter) (string, []interface{}) {
	whereClause, args := s.buildWhereClause(filter)

	query := fmt.Sprintf(`
		SELECT 
			id, timestamp, caller, provider, model, prompt_tokens, completion_tokens,
			currency, cost_input, cost_output, cost_total, request_id
		FROM costs
		%s
		ORDER BY timestamp DESC
	`, whereClause)

	// Add pagination
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", filter.Offset)
		}
	}

	return query, args
}

// buildWhereClause builds WHERE clause with filters
func (s *SQLiteAggregator) buildWhereClause(filter CostFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if filter.From != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, *filter.From)
	}
	if filter.To != nil {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, *filter.To)
	}
	if filter.Caller != "" {
		conditions = append(conditions, "caller = ?")
		args = append(args, filter.Caller)
	}
	if filter.Provider != "" {
		conditions = append(conditions, "provider = ?")
		args = append(args, filter.Provider)
	}
	if filter.Model != "" {
		conditions = append(conditions, "model = ?")
		args = append(args, filter.Model)
	}
	if filter.Currency != "" {
		conditions = append(conditions, "currency = ?")
		args = append(args, filter.Currency)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	return whereClause, args
}

// getGroupedCosts gets grouped cost data
func (s *SQLiteAggregator) getGroupedCosts(filter CostFilter) ([]CostGroup, error) {
	whereClause, args := s.buildWhereClause(filter)

	query := fmt.Sprintf(`
		SELECT 
			%s as group_value,
			COUNT(*) as total_records,
			COALESCE(SUM(cost_total), 0) as total_cost,
			COALESCE(SUM(cost_input), 0) as total_input_cost,
			COALESCE(SUM(cost_output), 0) as total_output_cost,
			COALESCE(SUM(prompt_tokens), 0) as total_prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) as total_completion_tokens,
			COALESCE(currency, 'USD') as currency
		FROM costs
		%s
		GROUP BY %s
		ORDER BY total_cost DESC
	`, filter.GroupBy, whereClause, filter.GroupBy)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []CostGroup
	for rows.Next() {
		var group CostGroup
		var summary CostSummary

		err := rows.Scan(
			&group.GroupValue,
			&summary.TotalRecords,
			&summary.TotalCost,
			&summary.TotalInputCost,
			&summary.TotalOutputCost,
			&summary.TotalPromptTokens,
			&summary.TotalCompletionTokens,
			&summary.Currency,
		)
		if err != nil {
			return nil, err
		}

		group.GroupBy = filter.GroupBy
		group.Summary = summary
		groups = append(groups, group)
	}

	return groups, nil
}

// exportCSV exports records as CSV
func (s *SQLiteAggregator) exportCSV(records []CostRecord) ([]byte, error) {
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
func (s *SQLiteAggregator) Close() error {
	return s.db.Close()
}
