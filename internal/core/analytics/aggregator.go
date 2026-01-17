package analytics

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// Aggregator provides generic database aggregation helpers
type Aggregator struct {
	db *gorm.DB
}

// NewAggregator creates a new aggregator
func NewAggregator(db *gorm.DB) *Aggregator {
	return &Aggregator{db: db}
}

// Aggregate performs a generic aggregation query
func (a *Aggregator) Aggregate(query AggregateQuery) ([]map[string]interface{}, error) {
	// Build SELECT clause with aggregates
	selectParts := []string{}

	// Add GROUP BY columns to SELECT
	for _, col := range query.GroupBy {
		selectParts = append(selectParts, col)
	}

	// Add aggregate functions to SELECT
	for alias, agg := range query.Aggregates {
		selectParts = append(selectParts, fmt.Sprintf("%s AS %s", agg, alias))
	}

	selectClause := strings.Join(selectParts, ", ")

	// Start building query
	db := a.db.Table(query.Table).Select(selectClause)

	// Apply WHERE filters
	for condition, value := range query.Filters {
		if strings.Contains(condition, "?") {
			// Parameterized condition (e.g., "created_at BETWEEN ? AND ?")
			db = db.Where(condition, value)
		} else {
			// Simple equality (e.g., {"client_id": uuid})
			db = db.Where(fmt.Sprintf("%s = ?", condition), value)
		}
	}

	// Apply date range filter
	if query.DateRange != nil {
		db = db.Where(fmt.Sprintf("%s BETWEEN ? AND ?", query.DateRange.Field),
			query.DateRange.Start, query.DateRange.End)
	}

	// Apply GROUP BY
	if len(query.GroupBy) > 0 {
		db = db.Group(strings.Join(query.GroupBy, ", "))
	}

	// Apply ORDER BY
	if len(query.OrderBy) > 0 {
		for _, order := range query.OrderBy {
			db = db.Order(order)
		}
	}

	// Apply LIMIT
	if query.Limit > 0 {
		db = db.Limit(query.Limit)
	}

	// Execute query
	var results []map[string]interface{}
	if err := db.Find(&results).Error; err != nil {
		return nil, fmt.Errorf("aggregate query failed: %w", err)
	}

	return results, nil
}

// Count performs a simple COUNT query with filters
func (a *Aggregator) Count(table string, filters map[string]interface{}) (int64, error) {
	db := a.db.Table(table)

	// Apply filters
	for condition, value := range filters {
		if strings.Contains(condition, "?") {
			db = db.Where(condition, value)
		} else {
			db = db.Where(fmt.Sprintf("%s = ?", condition), value)
		}
	}

	var count int64
	if err := db.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count query failed: %w", err)
	}

	return count, nil
}

// Sum performs a simple SUM query
func (a *Aggregator) Sum(table, column string, filters map[string]interface{}) (float64, error) {
	query := AggregateQuery{
		Table:      table,
		Aggregates: map[string]string{"total": fmt.Sprintf("SUM(%s)", column)},
		Filters:    filters,
	}

	results, err := a.Aggregate(query)
	if err != nil {
		return 0, err
	}

	if len(results) == 0 {
		return 0, nil
	}

	// Convert to float64
	total := results[0]["total"]
	if total == nil {
		return 0, nil
	}

	switch v := total.(type) {
	case float64:
		return v, nil
	case int64:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("unexpected sum result type: %T", total)
	}
}

// Average performs a simple AVG query
func (a *Aggregator) Average(table, column string, filters map[string]interface{}) (float64, error) {
	query := AggregateQuery{
		Table:      table,
		Aggregates: map[string]string{"avg": fmt.Sprintf("AVG(%s)", column)},
		Filters:    filters,
	}

	results, err := a.Aggregate(query)
	if err != nil {
		return 0, err
	}

	if len(results) == 0 {
		return 0, nil
	}

	avg := results[0]["avg"]
	if avg == nil {
		return 0, nil
	}

	switch v := avg.(type) {
	case float64:
		return v, nil
	case int64:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("unexpected avg result type: %T", avg)
	}
}
