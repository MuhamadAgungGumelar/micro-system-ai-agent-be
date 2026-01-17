package analytics

import "time"

// AggregateQuery represents a generic database aggregation query
type AggregateQuery struct {
	Table      string                 // Table or JOIN clause
	GroupBy    []string               // GROUP BY columns
	Aggregates map[string]string      // Aggregate functions: {"total": "SUM(amount)", "count": "COUNT(*)"}
	Filters    map[string]interface{} // WHERE conditions
	DateRange  *DateRange             // Date range filter
	OrderBy    []string               // ORDER BY clauses
	Limit      int                    // LIMIT (0 = no limit)
}

// DateRange represents a time period for filtering
type DateRange struct {
	Start time.Time
	End   time.Time
	Field string // Date field to filter on (e.g., "created_at")
}

// ChartData represents generic chart data format
type ChartData struct {
	Type   string      `json:"type"`   // "line", "bar", "pie", "donut"
	Labels []string    `json:"labels"` // X-axis labels or pie segments
	Data   []ChartSeries `json:"data"`   // Y-axis data series
}

// ChartSeries represents a data series in a chart
type ChartSeries struct {
	Name   string        `json:"name"`   // Series name (e.g., "Sales", "Revenue")
	Values []interface{} `json:"values"` // Data values
	Color  string        `json:"color,omitempty"`  // Optional color
}

// PieChartData represents pie chart specific data
type PieChartData struct {
	Type   string         `json:"type"` // "pie" or "donut"
	Labels []string       `json:"labels"`
	Values []float64      `json:"values"`
	Colors []string       `json:"colors,omitempty"`
}

// StatCard represents a summary statistic card
type StatCard struct {
	Title       string  `json:"title"`
	Value       string  `json:"value"`
	Change      float64 `json:"change"`       // Percentage change
	ChangeLabel string  `json:"change_label"` // "vs last month", "vs yesterday"
	Trend       string  `json:"trend"`        // "up", "down", "neutral"
	Icon        string  `json:"icon,omitempty"`
}
