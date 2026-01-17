package analytics

import (
	"fmt"
	"time"
)

// ToLineChartData converts query results to line chart format
// xKey: field name for X-axis (labels)
// yKey: field name for Y-axis (values)
func ToLineChartData(data []map[string]interface{}, xKey, yKey string) ChartData {
	labels := make([]string, len(data))
	values := make([]interface{}, len(data))

	for i, row := range data {
		labels[i] = formatLabel(row[xKey])
		values[i] = formatValue(row[yKey])
	}

	return ChartData{
		Type:   "line",
		Labels: labels,
		Data: []ChartSeries{
			{
				Name:   yKey,
				Values: values,
			},
		},
	}
}

// ToMultiLineChartData converts query results to multi-line chart format
// xKey: field name for X-axis
// seriesConfig: map of series name to field name
func ToMultiLineChartData(data []map[string]interface{}, xKey string, seriesConfig map[string]string) ChartData {
	if len(data) == 0 {
		return ChartData{Type: "line", Labels: []string{}, Data: []ChartSeries{}}
	}

	labels := make([]string, len(data))
	for i, row := range data {
		labels[i] = formatLabel(row[xKey])
	}

	series := make([]ChartSeries, 0, len(seriesConfig))
	for seriesName, fieldName := range seriesConfig {
		values := make([]interface{}, len(data))
		for i, row := range data {
			values[i] = formatValue(row[fieldName])
		}
		series = append(series, ChartSeries{
			Name:   seriesName,
			Values: values,
		})
	}

	return ChartData{
		Type:   "line",
		Labels: labels,
		Data:   series,
	}
}

// ToBarChartData converts query results to bar chart format
func ToBarChartData(data []map[string]interface{}, xKey, yKey string) ChartData {
	labels := make([]string, len(data))
	values := make([]interface{}, len(data))

	for i, row := range data {
		labels[i] = formatLabel(row[xKey])
		values[i] = formatValue(row[yKey])
	}

	return ChartData{
		Type:   "bar",
		Labels: labels,
		Data: []ChartSeries{
			{
				Name:   yKey,
				Values: values,
			},
		},
	}
}

// ToPieChartData converts query results to pie chart format
func ToPieChartData(data []map[string]interface{}, labelKey, valueKey string) PieChartData {
	labels := make([]string, len(data))
	values := make([]float64, len(data))

	for i, row := range data {
		labels[i] = formatLabel(row[labelKey])
		values[i] = toFloat64(row[valueKey])
	}

	return PieChartData{
		Type:   "pie",
		Labels: labels,
		Values: values,
	}
}

// ToStatCards converts query results to stat cards
func ToStatCards(data map[string]interface{}, config map[string]StatCardConfig) []StatCard {
	cards := make([]StatCard, 0, len(config))

	for key, cfg := range config {
		value := data[key]
		card := StatCard{
			Title:       cfg.Title,
			Value:       formatStatValue(value, cfg.Format),
			Icon:        cfg.Icon,
			ChangeLabel: cfg.ChangeLabel,
		}

		// Calculate change if previous value provided
		if cfg.PreviousKey != "" && data[cfg.PreviousKey] != nil {
			current := toFloat64(value)
			previous := toFloat64(data[cfg.PreviousKey])

			if previous > 0 {
				change := ((current - previous) / previous) * 100
				card.Change = change

				if change > 0 {
					card.Trend = "up"
				} else if change < 0 {
					card.Trend = "down"
				} else {
					card.Trend = "neutral"
				}
			}
		}

		cards = append(cards, card)
	}

	return cards
}

// StatCardConfig represents configuration for a stat card
type StatCardConfig struct {
	Title       string
	Format      string // "number", "currency", "percentage"
	Icon        string
	ChangeLabel string
	PreviousKey string // Key for previous value to calculate change
}

// Helper functions

func formatLabel(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case time.Time:
		return v.Format("2006-01-02")
	case int, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%.2f", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func formatValue(value interface{}) interface{} {
	if value == nil {
		return 0
	}
	return value
}

func toFloat64(value interface{}) float64 {
	if value == nil {
		return 0
	}

	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

func formatStatValue(value interface{}, format string) string {
	num := toFloat64(value)

	switch format {
	case "currency":
		return fmt.Sprintf("Rp %.2f", num)
	case "percentage":
		return fmt.Sprintf("%.1f%%", num)
	case "number":
		if num >= 1000000 {
			return fmt.Sprintf("%.1fM", num/1000000)
		} else if num >= 1000 {
			return fmt.Sprintf("%.1fK", num/1000)
		}
		return fmt.Sprintf("%.0f", num)
	default:
		return fmt.Sprintf("%.2f", num)
	}
}
