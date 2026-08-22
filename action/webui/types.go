package webui

import "html/template"

// NavItem is one link in the shared page nav.
type NavItem struct {
	Label  string
	URL    string
	Active bool // current page, for highlighting
}

// PageData is the top-level data passed to the shell template.
type PageData struct {
	Title    string
	Nav      []NavItem
	UserName string        // logged-in user's name, shown in the header dropdown; empty hides it
	Content  template.HTML // pre-rendered content template output
}

// StatTileData is one summary number (e.g. "Current balance: €4,231").
type StatTileData struct {
	Label    string
	Value    string
	SubLabel string // optional, e.g. "vs last month: +€120"
	Emphasis bool   // visually highlighted tile (e.g. the primary metric)
}

// TableColumn describes one column header.
type TableColumn struct {
	Label string
	Align string // "left" | "right", default left
}

// TableRow is one row's cells plus an optional drill-down link.
type TableRow struct {
	Cells   []string
	LinkURL string // empty = not clickable
}

// TableData is a full table component.
type TableData struct {
	Columns    []TableColumn
	Rows       []TableRow
	Pagination *PaginationData // nil = no pagination controls rendered
}

// PaginationData drives the prev/next + "page X of Y" controls RenderTable
// renders below the rows when set. Page is 1-indexed. PrevURL/NextURL are
// pre-built by the caller (so the component stays agnostic of each page's
// own query params) and are empty on the first/last page respectively,
// which hides that link.
type PaginationData struct {
	Page       int
	TotalPages int
	PrevURL    string
	NextURL    string
}

// DetailViewData is a drill-down page: a header plus a table of related records.
type DetailViewData struct {
	Title    string
	BackURL  string
	BackText string
	Stats    []StatTileData // optional summary tiles at top
	Table    TableData
}

// LineChartPoint is one (x, y) sample in a line chart.
type LineChartPoint struct {
	Label string  // x-axis label, e.g. "2026-08-01"
	Value float64 // y-axis value, e.g. weight in kg or rep count
}

// LineChartData is a single-series line chart (e.g. exercise weight/reps over
// time, or a Progress activity's value-over-time drill-down).
type LineChartData struct {
	ID         string // unique DOM id for this chart's <canvas>, e.g. "chart-bench-press"
	Title      string
	SeriesName string // legend label, e.g. "Weight (kg)"
	Points     []LineChartPoint
}

// BarChartBar is one labeled bar in a bar chart.
type BarChartBar struct {
	Label string  // e.g. category name "groceries"
	Value float64 // e.g. average monthly spend
}

// BarChartData is a single-series categorical bar chart (e.g. Money
// spend-by-category).
type BarChartData struct {
	ID         string // unique DOM id for this chart's <canvas>, e.g. "chart-spend-by-category"
	Title      string
	SeriesName string // legend label, e.g. "Avg monthly spend (EUR)"
	Bars       []BarChartBar
}

// CalendarDay is one cell in a month-grid calendar.
type CalendarDay struct {
	Day     int    // day-of-month number shown in the cell, e.g. 5
	InMonth bool   // false for the leading/trailing days of adjacent months
	Count   string // e.g. "3 transactions" — empty hides the count line
	Total   string // e.g. "€42.10" — empty hides the total line
	LinkURL string // empty = not clickable, same convention as TableRow.LinkURL
}

// CalendarData is a month-grid calendar (e.g. Money's transaction calendar).
type CalendarData struct {
	Title    string          // e.g. "August 2026"
	PrevURL  string          // previous month link; empty hides it (and the whole nav row when NextURL is also empty)
	NextURL  string          // next month link; empty hides it (and the whole nav row when PrevURL is also empty)
	Weekdays []string        // 7 column headers, e.g. ["Mon", ..., "Sun"]
	Weeks    [][]CalendarDay // each inner slice has exactly 7 CalendarDay entries
}
