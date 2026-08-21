package webui

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// DesignSystemHandler renders the /web/design-system demo/style-guide page:
// every shared component (nav, stat tiles, table, line charts, bar chart,
// drill-down/detail view) against fixture data, so the visual language can
// be reviewed before any real dashboard is wired up. It touches no
// database — every value below is a fixture.
func DesignSystemHandler(c *gin.Context) {
	nav := []NavItem{
		{Label: "Home", URL: "/web"},
		{Label: "Money", URL: "/web/money"},
		{Label: "Progress", URL: "/web/progress/browse"},
		{Label: "Workouts", URL: "/web/workouts"},
		{Label: "Design System", URL: "/web/design-system", Active: true},
	}

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
		Title:      "Spend by category (avg/month)",
		SeriesName: "EUR",
		Bars: []BarChartBar{
			{Label: "Groceries", Value: 312.40},
			{Label: "Transport", Value: 88.10},
			{Label: "Rent & Utilities", Value: 900.00},
			{Label: "Dining out", Value: 145.20},
		},
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

	var body strings.Builder
	body.WriteString("<section><h2>Stat tiles</h2>")
	body.WriteString(string(RenderStatTiles(stats)))
	body.WriteString("</section>")

	body.WriteString("<section><h2>Table</h2>")
	body.WriteString(string(RenderTable(table)))
	body.WriteString("</section>")

	body.WriteString("<section><h2>Line charts</h2>")
	body.WriteString(string(RenderLineChart(weightChart)))
	body.WriteString(string(RenderLineChart(moodChart)))
	body.WriteString("</section>")

	body.WriteString("<section><h2>Bar chart</h2>")
	body.WriteString(string(RenderBarChart(spendChart)))
	body.WriteString("</section>")

	body.WriteString("<section><h2>Drill-down / detail view</h2>")
	body.WriteString(string(RenderDetailView(detail)))
	body.WriteString("</section>")

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	if err := RenderPage(c.Writer, PageData{
		Title:   "Design System",
		Nav:     nav,
		Content: template.HTML(body.String()),
	}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
		return
	}
}
