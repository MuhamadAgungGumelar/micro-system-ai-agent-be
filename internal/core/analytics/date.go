package analytics

import "time"

// GetDateRange returns a date range based on a period
func GetDateRange(period string) *DateRange {
	now := time.Now()
	var start, end time.Time

	switch period {
	case "today":
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())

	case "yesterday":
		yesterday := now.AddDate(0, 0, -1)
		start = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, now.Location())
		end = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 999999999, now.Location())

	case "this_week":
		// Start of week (Monday)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday = 7
		}
		start = now.AddDate(0, 0, -weekday+1)
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, now.Location())
		end = now

	case "last_week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start = now.AddDate(0, 0, -weekday-6)
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, now.Location())
		end = now.AddDate(0, 0, -weekday)
		end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 999999999, now.Location())

	case "this_month":
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = now

	case "last_month":
		start = time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
		end = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Add(-time.Second)

	case "this_year":
		start = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		end = now

	case "last_30_days":
		start = now.AddDate(0, 0, -30)
		end = now

	case "last_90_days":
		start = now.AddDate(0, 0, -90)
		end = now

	default:
		// Default to today
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
	}

	return &DateRange{
		Start: start,
		End:   end,
		Field: "created_at", // Default field
	}
}

// GetCustomDateRange creates a date range from specific dates
func GetCustomDateRange(start, end time.Time, field string) *DateRange {
	if field == "" {
		field = "created_at"
	}

	return &DateRange{
		Start: start,
		End:   end,
		Field: field,
	}
}

// GetMonthlyRanges returns date ranges for each month in a period
func GetMonthlyRanges(start, end time.Time) []DateRange {
	ranges := []DateRange{}
	current := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, start.Location())

	for current.Before(end) || current.Equal(end) {
		monthStart := current
		monthEnd := current.AddDate(0, 1, 0).Add(-time.Second)

		if monthEnd.After(end) {
			monthEnd = end
		}

		ranges = append(ranges, DateRange{
			Start: monthStart,
			End:   monthEnd,
			Field: "created_at",
		})

		current = current.AddDate(0, 1, 0)
	}

	return ranges
}

// GetWeeklyRanges returns date ranges for each week in a period
func GetWeeklyRanges(start, end time.Time) []DateRange {
	ranges := []DateRange{}

	// Adjust start to Monday
	weekday := int(start.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	current := start.AddDate(0, 0, -weekday+1)
	current = time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, start.Location())

	for current.Before(end) {
		weekStart := current
		weekEnd := current.AddDate(0, 0, 6)
		weekEnd = time.Date(weekEnd.Year(), weekEnd.Month(), weekEnd.Day(), 23, 59, 59, 999999999, weekEnd.Location())

		if weekEnd.After(end) {
			weekEnd = end
		}

		ranges = append(ranges, DateRange{
			Start: weekStart,
			End:   weekEnd,
			Field: "created_at",
		})

		current = current.AddDate(0, 0, 7)
	}

	return ranges
}

// GetDailyRanges returns date ranges for each day in a period
func GetDailyRanges(start, end time.Time) []DateRange {
	ranges := []DateRange{}
	current := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())

	for current.Before(end) || current.Equal(end) {
		dayStart := current
		dayEnd := time.Date(current.Year(), current.Month(), current.Day(), 23, 59, 59, 999999999, current.Location())

		if dayEnd.After(end) {
			dayEnd = end
		}

		ranges = append(ranges, DateRange{
			Start: dayStart,
			End:   dayEnd,
			Field: "created_at",
		})

		current = current.AddDate(0, 0, 1)
	}

	return ranges
}
