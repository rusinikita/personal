package webui

import (
	"html/template"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// designSystemPageData is what templates/pages/design_system.html renders —
// each field is an already-rendered component fragment (template.HTML, so
// it's inserted verbatim rather than re-escaped), so the page template just
// lays sections around them declaratively.
type designSystemPageData struct {
	StatsHTML       template.HTML
	TableHTML       template.HTML
	WeightChartHTML template.HTML
	MoodChartHTML   template.HTML
	SpendChartHTML  template.HTML
	CalendarHTML    template.HTML
	DetailHTML      template.HTML
	GoalTilesHTML   template.HTML
}

// fixtureCalendarWeeks builds a real, correctly-computed August 2026 month
// grid (Go-computed weeks, not template date math — see the Best Practices
// in webui-spec.md) with a few sample linked/unlinked days, so the demo
// page shows the same leading/trailing-day padding a real caller would
// produce. This is fixture-only logic local to the demo page — real
// callers (e.g. action/money) build their own grid from live data.
func fixtureCalendarWeeks() [][]CalendarDay {
	first := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	mondayOffset := (int(first.Weekday()) + 6) % 7
	start := first.AddDate(0, 0, -mondayOffset)

	last := first.AddDate(0, 1, -1)
	endOffset := 6 - (int(last.Weekday())+6)%7
	end := last.AddDate(0, 0, endOffset)

	sample := map[string]CalendarDay{
		"2026-08-05": {Total: "€42.10 (3)", LinkURL: "/web/design-system"},
		"2026-08-14": {Total: "€128.90 (5)", LinkURL: "/web/design-system"},
		"2026-08-22": {Total: "€18.00 (1)", LinkURL: "/web/design-system"},
	}

	var weeks [][]CalendarDay
	var week []CalendarDay
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		day := CalendarDay{Day: d.Day(), InMonth: d.Month() == time.August}
		if s, ok := sample[d.Format("2006-01-02")]; ok {
			day.Total, day.LinkURL = s.Total, s.LinkURL
		}
		week = append(week, day)
		if len(week) == 7 {
			weeks = append(weeks, week)
			week = nil
		}
	}
	return weeks
}

// DesignSystemHandler renders the /web/design-system demo/style-guide page:
// every shared component (nav, stat tiles, table, line charts, bar chart,
// calendar, drill-down/detail view) against fixture data, so the visual
// language can be reviewed before any real dashboard is wired up. It
// touches no database — every value below is a fixture.
func DesignSystemHandler(c *gin.Context) {
	nav := BuildNav(NavDesignSystem)

	stats := []StatTileData{
		{Label: "Current balance", Value: "€4,231", SubLabel: "vs last month: +€120"},
		{Label: "Current streak", Value: "12 days", SubLabel: "last check-in updated <3 days> ago", Emphasis: true},
		{Label: "Personal record", Value: "120 kg", SubLabel: "Bench press"},
	}

	table := TableData{
		Columns: []TableColumn{
			{Label: "Category", Align: "left"},
			{Label: "Avg / month", Align: "right"},
		},
		Rows: []TableRow{
			{Cells: []string{"Groceries", "€312.40"}, LinkURL: "/web/design-system"},
			{Cells: []string{"Transport", "€88.10"}, LinkURL: "/web/design-system"},
			{Cells: []string{"Rent & Utilities", "€900.00"}},
		},
	}

	weightChart := LineChartData{
		ID:         "chart-weight-demo",
		Title:      "Bench press — weight over time",
		SeriesName: "Weight (kg)",
		Points: []LineChartPoint{
			{Label: "2026-07-01", Value: 100},
			{Label: "2026-07-15", Value: 105},
			{Label: "2026-08-01", Value: 110},
			{Label: "2026-08-15", Value: 120},
		},
	}

	moodChart := LineChartData{
		ID:         "chart-mood-demo",
		Title:      "Mood — value over time",
		SeriesName: "Mood value",
		Points: []LineChartPoint{
			{Label: "2026-08-16", Value: 1},
			{Label: "2026-08-17", Value: 0},
			{Label: "2026-08-18", Value: 2},
			{Label: "2026-08-19", Value: 1},
			{Label: "2026-08-20", Value: -1},
		},
	}

	spendChart := BarChartData{
		ID:         "chart-spend-demo",
		Title:      "Spend by category (avg/month)",
		SeriesName: "EUR",
		Bars: []BarChartBar{
			{Label: "Groceries", Value: 312.40},
			{Label: "Transport", Value: 88.10},
			{Label: "Rent & Utilities", Value: 900.00},
			{Label: "Dining out", Value: 145.20},
		},
	}

	calendar := CalendarData{
		Title:    "August 2026",
		PrevURL:  "/web/design-system",
		NextURL:  "/web/design-system",
		Weekdays: []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
		Weeks:    fixtureCalendarWeeks(),
	}

	detail := DetailViewData{
		Title:    "Groceries — transaction history",
		BackURL:  "/web/design-system",
		BackText: "Back to overview",
		Stats: []StatTileData{
			{Label: "Total this month", Value: "€312.40"},
		},
		Table: TableData{
			Columns: []TableColumn{
				{Label: "Date", Align: "left"},
				{Label: "Merchant", Align: "left"},
				{Label: "Amount", Align: "right"},
			},
			Rows: []TableRow{
				{Cells: []string{"2026-08-20", "Lidl", "€42.10"}},
				{Cells: []string{"2026-08-15", "Metro", "€68.90"}},
			},
		},
	}

	goalTiles := GoalTilesData{
		Tiles: []GoalTileData{
			{Name: "Emergency Fund", ProgressLabel: "€2,100.00 / €5,000.00 (42%)", PercentComplete: 42, Deadline: "by Dec 31, 2026"},
			{Name: "Food Budget", ProgressLabel: "€320.00 / €300.00 (107%)", PercentComplete: 100, OverTarget: true},
			{Name: "Bench Press 100kg", ProgressLabel: "82.5kg / 100kg", PercentComplete: 82.5},
		},
	}

	content := execToHTML("pages/design_system", designSystemPageData{
		StatsHTML:       RenderStatTiles(stats),
		TableHTML:       RenderTable(table),
		WeightChartHTML: RenderLineChart(weightChart),
		MoodChartHTML:   RenderLineChart(moodChart),
		SpendChartHTML:  RenderBarChart(spendChart),
		CalendarHTML:    RenderCalendar(calendar),
		DetailHTML:      RenderDetailView(detail),
		GoalTilesHTML:   RenderGoalTiles(goalTiles),
	})

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	if err := RenderPage(c.Writer, PageData{
		Title:    "Design System",
		Nav:      nav,
		UserName: c.GetString("user_name"),
		Content:  content,
	}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
		return
	}
}
