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
	LinkURL string        // empty = not clickable
	Tags    []TableRowTag // this row's tags, rendered in the table's trailing tags column (see TableData.TagsColumnLabel); nil = none, only meaningful when TagsColumnLabel is set
	Heading string        // when non-empty, row renders as a full-width section-heading row instead of Cells/LinkURL/Tags (all ignored); empty = normal row
}

// TableRowTag is one small tag shown next to a table row's first cell (e.g.
// a life_part tag on the Progress browse table), styled as a Pico CSS
// contrast button (role="button" class="contrast") — no custom color CSS of
// ours. Tooltip renders via Pico's own data-tooltip attribute (pure CSS, no
// JS) instead of the native HTML title attribute — Chrome's native title
// tooltip proved unreliable in practice.
type TableRowTag struct {
	Label   string
	Tooltip string // shown on hover via Pico's data-tooltip attribute; empty = no tooltip
}

// TableData is a full table component.
type TableData struct {
	Columns         []TableColumn
	Rows            []TableRow
	Pagination      *PaginationData // nil = no pagination controls rendered
	TagsColumnLabel string          // header for a trailing column rendering each row's Tags; empty = no tags column at all
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
	Title       string
	Description template.HTML // optional subtitle under Title, e.g. an activity's free-text description; empty renders nothing
	BackURL     string
	BackText    string
	Stats       []StatTileData // optional summary tiles at top
	Table       TableData
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

// ComboChartPoint is one (x, y) sample plotted as both a bar and a line —
// the same value drawn two ways, not two different series.
type ComboChartPoint struct {
	Label string  // x-axis label, e.g. "2026-08"
	Value float64 // e.g. balance at this point in the trend
}

// ComboChartData is a single-series chart rendered as both a bar and an
// overlaid line for the same values (e.g. Money's balance trend) — for when
// a plain LineChartData or BarChartData reads less clearly alone than the
// two drawing styles combined on one series.
type ComboChartData struct {
	ID         string // unique DOM id for this chart's <canvas>, e.g. "chart-balance-trend"
	Title      string
	SeriesName string // e.g. "Balance (EUR)" — shown in the tooltip, no legend (both datasets are the same series)
	Points     []ComboChartPoint
}

// DualAxisChartPoint is one x position with up to two values, one per Y axis.
// nil = no value for that series at this point (gap in that line only).
type DualAxisChartPoint struct {
	Label string   // x-axis label, e.g. "2026-08-01"
	Left  *float64 // left Y axis value, e.g. weight in kg
	Right *float64 // right Y axis value, e.g. rep count
}

// DualAxisChartData is a two-series line chart, each series on its own Y
// axis (e.g. Workouts exercise drill-down: weight left, reps right).
type DualAxisChartData struct {
	ID              string // unique DOM id for this chart's <canvas>, caller-supplied
	Title           string
	LeftSeriesName  string // legend + left axis label, e.g. "Weight (kg)"
	RightSeriesName string // legend + right axis label, e.g. "Reps"
	Points          []DualAxisChartPoint
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

// AchievementTileData is one achievement rendered as a card with a progress bar (see
// docs/functions/achievements-spec.md).
type AchievementTileData struct {
	Name            string
	ProgressLabel   string  // e.g. "€420 / €1,000 (42%)", "82kg / 100kg", "14 / 30 day streak"
	PercentComplete float64 // 0-100, clamped for the bar width (ProgressLabel still shows the real, unclamped numbers)
	Deadline        string  // formatted, e.g. "by Dec 31, 2026" — empty hides the deadline line (no deadline set)
	OverTarget      bool    // current exceeds target — tile renders with a warning tint. Only meaningful for achievement
	// types where exceeding is bad (e.g. money_spend over budget); left false otherwise
	LinkURL string // drill-down link to the underlying category/exercise/activity page; empty = not clickable
	// (same convention as TableRow.LinkURL/CalendarDay.LinkURL)
}

// AchievementTilesData is a grid of achievement tiles — either every active achievement (the
// dedicated /web/achievements page) or a subset pre-filtered to one domain's achievement
// types (embedded in the Money/Progress-browse/Workouts pages).
type AchievementTilesData struct {
	Tiles        []AchievementTileData
	EmptyMessage string // shown instead of the grid when Tiles is empty; empty string means render nothing (see achievements-spec.md's embedded-vs-dedicated empty-state convention)
}
