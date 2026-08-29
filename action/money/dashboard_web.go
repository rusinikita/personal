// Package money's web dashboard: read-only /web/money pages built on the
// action/webui design system, for reviewing the overall financial picture
// (balance, income/spend trends, category weight, sync freshness, a
// day-by-day calendar) in a browser. Adding/editing/deleting transactions
// stays out of scope — this file links out to the existing /money/import
// bulk-import page (import_web.go) instead of duplicating it.
package money

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"personal/action/webui"
	"personal/domain"
	"personal/gateways"
)

// displayTimezone is the timezone day boundaries (the transactions list's
// ?from/?to filter and the calendar's day grouping) are computed in,
// matching GetDailyTransactionSummary's own use of Asia/Nicosia.
const displayTimezone = "Asia/Nicosia"

// TransactionsPageSize is the fixed page size for GET /web/money/transactions.
// It's a var, not a const, so tests can shrink it instead of seeding 100+ rows.
var TransactionsPageSize = 100

// calendarDefaultMonths is how far back the calendar page defaults when
// ?from/?to are absent.
const calendarDefaultMonths = 3

var moneyNav = webui.BuildNav(webui.NavMoney)

// moneyDashboardLinks is the small nav line under the dashboard's stat
// tiles. The URLs are fixed route constants, not user data, so embedding
// them as template.HTML directly (no templating) is safe.
const moneyDashboardLinks template.HTML = `<p><a href="/web/money/transactions">View all transactions</a> · <a href="/web/money/calendar">Calendar</a> · <a href="/web/money/export">Export</a> · <a href="/money/import">Import transactions</a></p>`

// formatEUR renders a EUR amount the way CalendarDay.Total's own doc
// example does, e.g. "€42.10".
func formatEUR(v float64) string {
	return fmt.Sprintf("€%.2f", v)
}

// formatSignedEUR prefixes the amount with "-" for an expense or "+" for
// income, so the transactions list reads as a ledger rather than requiring
// the reader to cross-reference the Type column.
func formatSignedEUR(t domain.TransactionType, v float64) string {
	switch t {
	case domain.TransactionTypeExpense:
		return "-" + formatEUR(v)
	case domain.TransactionTypeIncome:
		return "+" + formatEUR(v)
	default:
		return formatEUR(v)
	}
}

// formatDateOrDash renders a nullable date as "2026-08-22" or "—".
func formatDateOrDash(t *time.Time) string {
	if t == nil {
		return "—"
	}
	return t.Format("2006-01-02")
}

// monthsSince counts full elapsed calendar months between from and to,
// floored at 1 — money-spec.md's "months-span floors at 1" rule, avoiding a
// divide-by-zero (and a meaningless huge average) for a brand-new account.
func monthsSince(from, to time.Time) int {
	months := (to.Year()-from.Year())*12 + int(to.Month()) - int(from.Month())
	if to.Day() < from.Day() {
		months--
	}
	if months < 1 {
		months = 1
	}
	return months
}

// categoryLinkURL builds the drill-down link from a dashboard category row
// (or a calendar day) to the transactions list, pre-filtered.
func categoryLinkURL(category string) string {
	v := url.Values{}
	v.Set("category", category)
	return "/web/money/transactions?" + v.Encode()
}

// dayLinkURL builds the drill-down link from a calendar day cell to the
// transactions list, filtered to just that one day (from == to).
func dayLinkURL(date string) string {
	v := url.Values{}
	v.Set("from", date)
	v.Set("to", date)
	return "/web/money/transactions?" + v.Encode()
}

// buildDashboardStats builds the dashboard's stat tile row: sync freshness,
// current balance, all-time income, last calendar month's net, and average
// monthly savings. The 3/6/12-month outlook these savings drive is shown as
// the balance trend chart instead (see buildBalanceTrendPoints), not as
// separate tiles. See money-spec.md's "Web Dashboard" sequence diagram and
// Best Practices for the derivation of each figure.
func buildDashboardStats(summary domain.MoneySummary, allTime, lastMonth domain.BalanceResult, monthsSpan int) []webui.StatTileData {
	avgMonthlySavings := allTime.BalanceEUR / float64(monthsSpan)
	return []webui.StatTileData{
		{Label: "Last sync", Value: formatDateOrDash(summary.LastSyncedAt)},
		{Label: "Current balance", Value: formatEUR(allTime.BalanceEUR), Emphasis: true},
		{Label: "Total income (all time)", Value: formatEUR(allTime.IncomeEUR)},
		{Label: "Net (last month)", Value: formatEUR(lastMonth.BalanceEUR)},
		{Label: "Avg monthly savings", Value: formatEUR(avgMonthlySavings)},
	}
}

// balanceTrendOffsets are the months-from-now offsets the balance trend
// chart plots: four actual past points, the current balance, and four
// projected future points, all spaced 3 months apart so the line renders
// on an evenly-spaced axis instead of looking bent at the -6/-3 and 6/12
// gaps.
var balanceTrendOffsets = []int{-12, -9, -6, -3, 0, 3, 6, 9, 12}

// balanceTrendLabel renders a months-from-now offset as "-12m", "Now", or
// "+3m".
func balanceTrendLabel(months int) string {
	switch {
	case months == 0:
		return "Now"
	case months < 0:
		return fmt.Sprintf("%dm", months)
	default:
		return fmt.Sprintf("+%dm", months)
	}
}

// buildBalanceTrendPoints builds the dashboard's balance trend line: actual
// cumulative balance (GetBalance(firstAt, pointDate), the same all-time
// query the "Current balance" tile itself uses, just with an earlier upper
// bound) at 12/6/3 months ago, the current balance, and avgMonthlySavings-
// driven projections 3/6/12 months out — one continuous past-to-future
// trend instead of separate "Projected" tiles. A past point older than the
// account's first transaction has no data yet and reads as 0, the same
// empty-state convention used elsewhere on this page.
func buildBalanceTrendPoints(ctx context.Context, db gateways.DB, userID int64, firstAt, now time.Time, currentBalance, avgMonthlySavings float64) ([]webui.LineChartPoint, error) {
	points := make([]webui.LineChartPoint, 0, len(balanceTrendOffsets))
	for _, months := range balanceTrendOffsets {
		var value float64
		switch {
		case months == 0:
			value = currentBalance
		case months < 0:
			pointDate := now.AddDate(0, months, 0)
			if pointDate.Before(firstAt) {
				value = 0
			} else {
				balance, err := db.GetBalance(ctx, userID, firstAt, pointDate)
				if err != nil {
					return nil, err
				}
				value = balance.BalanceEUR
			}
		default:
			value = currentBalance + avgMonthlySavings*float64(months)
		}
		points = append(points, webui.LineChartPoint{Label: balanceTrendLabel(months), Value: value})
	}
	return points, nil
}

// buildCategoryRows merges all-time and last-month per-category totals,
// computes each category's average monthly spend (all-time total divided
// by the account-wide monthsSpan, never the category's own age — see
// money-spec.md's "Months-span is global, not per-category"), and sorts
// descending by that average.
func buildCategoryRows(allTime, lastMonth []domain.SpendingByCategory, monthsSpan int) []webui.TableRow {
	lastMonthByCategory := make(map[string]float64, len(lastMonth))
	for _, c := range lastMonth {
		lastMonthByCategory[c.Category] = c.TotalEUR
	}

	type row struct {
		category      string
		totalEUR      float64
		lastMonthEUR  float64
		avgMonthlyEUR float64
	}
	rows := make([]row, 0, len(allTime))
	for _, c := range allTime {
		rows = append(rows, row{
			category:      c.Category,
			totalEUR:      c.TotalEUR,
			lastMonthEUR:  lastMonthByCategory[c.Category],
			avgMonthlyEUR: c.TotalEUR / float64(monthsSpan),
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].avgMonthlyEUR > rows[j].avgMonthlyEUR })

	tableRows := make([]webui.TableRow, 0, len(rows))
	for _, r := range rows {
		tableRows = append(tableRows, webui.TableRow{
			Cells:   []string{r.category, formatEUR(r.avgMonthlyEUR), formatEUR(r.totalEUR), formatEUR(r.lastMonthEUR)},
			LinkURL: categoryLinkURL(r.category),
		})
	}
	return tableRows
}

// MoneyDashboardWebHandler renders GET /web/money: stat tiles for balance,
// income, last-month net and savings, a balance trend line chart (actual
// balance 12/6/3 months ago through the current balance to a 3/6/12-month
// projection), and a category table sorted by average monthly spend
// descending. A fresh account with no transactions yet renders
// zero/placeholder values instead of erroring.
func MoneyDashboardWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	userID := webui.CurrentUserID(c)

	summary, err := db.GetMoneySummary(ctx, userID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load money summary: %v", err)
		return
	}

	now := time.Now().UTC()
	firstAt := now
	if summary.FirstTransactionAt != nil {
		firstAt = *summary.FirstTransactionAt
	}
	monthsSpan := monthsSince(firstAt, now)

	startOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	startOfLastMonth := startOfThisMonth.AddDate(0, -1, 0)
	endOfLastMonth := startOfThisMonth.Add(-time.Nanosecond)

	allTimeBalance, err := db.GetBalance(ctx, userID, firstAt, now)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load balance: %v", err)
		return
	}
	lastMonthBalance, err := db.GetBalance(ctx, userID, startOfLastMonth, endOfLastMonth)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load balance: %v", err)
		return
	}

	allTimeCategories, err := db.GetSpendingByCategory(ctx, userID, firstAt, now, 1)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load spending by category: %v", err)
		return
	}
	lastMonthCategories, err := db.GetSpendingByCategory(ctx, userID, startOfLastMonth, endOfLastMonth, 1)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load spending by category: %v", err)
		return
	}

	stats := buildDashboardStats(summary, allTimeBalance, lastMonthBalance, monthsSpan)
	table := webui.TableData{
		Columns: []webui.TableColumn{
			{Label: "Category"},
			{Label: "Avg monthly spend", Align: "right"},
			{Label: "Total (all time)", Align: "right"},
			{Label: "Last month", Align: "right"},
		},
		Rows: buildCategoryRows(allTimeCategories, lastMonthCategories, monthsSpan),
	}

	avgMonthlySavings := allTimeBalance.BalanceEUR / float64(monthsSpan)
	trendPoints, err := buildBalanceTrendPoints(ctx, db, userID, firstAt, now, allTimeBalance.BalanceEUR, avgMonthlySavings)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load balance trend: %v", err)
		return
	}
	trendChart := webui.LineChartData{
		ID:         "chart-balance-trend",
		Title:      "Balance trend",
		SeriesName: "Balance (EUR)",
		Points:     trendPoints,
	}

	content := webui.RenderStatTiles(stats) + webui.RenderLineChart(trendChart) + moneyDashboardLinks + webui.RenderTable(table)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	if err := webui.RenderPage(c.Writer, webui.PageData{
		Title:    "Money — Dashboard",
		Nav:      moneyNav,
		UserName: c.GetString("user_name"),
		Content:  content,
	}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
	}
}

// transactionsFilterFormSrc is embedded into the transactions list page — a
// GET <form> so category/from/to filter state always lives in the URL,
// never server-side session state (money-spec.md's "Filters live in the
// URL"). Clear sits right of Apply as an outline-styled link-button, in the
// same fieldset, rather than as its own line.
const transactionsFilterFormSrc = `<form method="GET">
<fieldset role="group">
<input type="text" name="category" value="{{.Category}}" placeholder="Category, e.g. food">
<input type="date" name="from" value="{{.From}}" placeholder="From">
<input type="date" name="to" value="{{.To}}" placeholder="To">
<button type="submit">Apply</button>
<a href="/web/money/transactions" role="button" class="outline">Clear</a>
</fieldset>
</form>`

var transactionsFilterFormTemplate = template.Must(template.New("transactionsFilterForm").Parse(transactionsFilterFormSrc))

type transactionsFilterFormData struct {
	Category string
	From     string
	To       string
}

func renderTransactionsFilterForm(category, from, to string) (template.HTML, error) {
	var b strings.Builder
	if err := transactionsFilterFormTemplate.Execute(&b, transactionsFilterFormData{Category: category, From: from, To: to}); err != nil {
		return "", err
	}
	return template.HTML(b.String()), nil
}

// buildTransactionsPagination mirrors action/progress's buildPagination,
// but also carries the category/from/to filters on every prev/next link so
// paginating doesn't drop the active filter.
func buildTransactionsPagination(page, totalCount int, category, from, to string) *webui.PaginationData {
	return webui.BuildPagination(page, totalCount, TransactionsPageSize, func(p int) string {
		v := url.Values{}
		if category != "" {
			v.Set("category", category)
		}
		if from != "" {
			v.Set("from", from)
		}
		if to != "" {
			v.Set("to", to)
		}
		v.Set("page", strconv.Itoa(p))
		return "/web/money/transactions?" + v.Encode()
	})
}

// parseDayBound parses a "2006-01-02" query param in the display timezone.
// atEndOfDay shifts the result to the last instant of that day (23:59:59.999999999)
// so it can be used as an inclusive upper bound.
func parseDayBound(loc *time.Location, value string, atEndOfDay bool) (time.Time, error) {
	day, err := time.ParseInLocation("2006-01-02", value, loc)
	if err != nil {
		return time.Time{}, err
	}
	if atEndOfDay {
		return day.AddDate(0, 0, 1).Add(-time.Nanosecond), nil
	}
	return day, nil
}

// TransactionsWebHandler renders GET /web/money/transactions: the one
// paginated transaction list page in the dashboard, shared by every entry
// point (dashboard "view all", a category row, a calendar day) — they
// differ only in which ?category/?from/?to query params are pre-filled.
// from and to are independent — either, both, or neither may be set, giving
// an open-ended range when only one bound is provided.
func TransactionsWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	userID := webui.CurrentUserID(c)

	category := strings.TrimSpace(c.Query("category"))
	fromParam := strings.TrimSpace(c.Query("from"))
	toParam := strings.TrimSpace(c.Query("to"))

	filter := domain.TransactionFilter{UserID: userID}
	if category != "" {
		filter.Category = &category
	}
	if fromParam != "" || toParam != "" {
		loc, err := time.LoadLocation(displayTimezone)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to load timezone: %v", err)
			return
		}
		if fromParam != "" {
			from, err := parseDayBound(loc, fromParam, false)
			if err != nil {
				c.String(http.StatusBadRequest, "invalid from date")
				return
			}
			filter.From = &from
		}
		if toParam != "" {
			to, err := parseDayBound(loc, toParam, true)
			if err != nil {
				c.String(http.StatusBadRequest, "invalid to date")
				return
			}
			filter.To = &to
		}
	}

	page := webui.ParsePage(c)
	filter.Limit = TransactionsPageSize
	filter.Offset = (page - 1) * TransactionsPageSize

	txs, total, err := db.GetTransactions(ctx, filter)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load transactions: %v", err)
		return
	}

	rows := make([]webui.TableRow, 0, len(txs))
	for _, tx := range txs {
		note := ""
		if tx.Note != nil {
			note = *tx.Note
		}
		rows = append(rows, webui.TableRow{
			Cells: []string{
				tx.TransactedAt.Format("2006-01-02 15:04"),
				tx.Category,
				tx.Merchant,
				formatSignedEUR(tx.Type, tx.AmountEUR),
				note,
			},
		})
	}

	table := webui.TableData{
		Columns: []webui.TableColumn{
			{Label: "Date"},
			{Label: "Category"},
			{Label: "Merchant"},
			{Label: "Amount (EUR)", Align: "right"},
			{Label: "Note"},
		},
		Rows:       rows,
		Pagination: buildTransactionsPagination(page, total, category, fromParam, toParam),
	}

	formHTML, err := renderTransactionsFilterForm(category, fromParam, toParam)
	if err != nil {
		c.String(http.StatusInternalServerError, "template error: %v", err)
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	if err := webui.RenderPage(c.Writer, webui.PageData{
		Title:    "Money — Transactions",
		Nav:      moneyNav,
		UserName: c.GetString("user_name"),
		Content:  formHTML + webui.RenderTable(table),
	}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
	}
}

// calendarFilterFormSrc is the calendar page's from/to range form, same
// GET-form-drives-the-URL convention as the transactions list filter. Clear
// sits right of Apply as an outline-styled link-button, in the same
// fieldset, rather than as its own line.
const calendarFilterFormSrc = `<form method="GET">
<fieldset role="group">
<input type="date" name="from" value="{{.From}}">
<input type="date" name="to" value="{{.To}}">
<button type="submit">Apply</button>
<a href="/web/money/calendar" role="button" class="outline">Clear</a>
</fieldset>
</form>`

var calendarFilterFormTemplate = template.Must(template.New("calendarFilterForm").Parse(calendarFilterFormSrc))

type calendarFilterFormData struct {
	From string
	To   string
}

func renderCalendarFilterForm(from, to string) (template.HTML, error) {
	var b strings.Builder
	if err := calendarFilterFormTemplate.Execute(&b, calendarFilterFormData{From: from, To: to}); err != nil {
		return "", err
	}
	return template.HTML(b.String()), nil
}

// calendarNavSrc is the single page-level Prev/Next control shown once
// above every stacked month grid — not repeated per grid. NextURL is empty
// (hiding the link) when shifting forward would move entirely into the
// future; PrevURL is always present since browsing further into the past is
// always valid.
const calendarNavSrc = `<nav class="webui-calendar-nav">
<ul><li><a role="button" href="{{.PrevURL}}">← Prev</a></li></ul>
{{if .NextURL}}<ul><li><a role="button" href="{{.NextURL}}">Next →</a></li></ul>{{end}}
</nav>`

var calendarNavTemplate = template.Must(template.New("calendarNav").Parse(calendarNavSrc))

type calendarNavData struct {
	PrevURL string
	NextURL string
}

func renderCalendarNav(prevURL, nextURL string) (template.HTML, error) {
	var b strings.Builder
	if err := calendarNavTemplate.Execute(&b, calendarNavData{PrevURL: prevURL, NextURL: nextURL}); err != nil {
		return "", err
	}
	return template.HTML(b.String()), nil
}

// calendarLinkURL builds a prev/next link that carries both from and to,
// shifted by whatever the caller already applied to each.
func calendarLinkURL(from, to time.Time) string {
	v := url.Values{}
	v.Set("from", from.Format("2006-01-02"))
	v.Set("to", to.Format("2006-01-02"))
	return "/web/money/calendar?" + v.Encode()
}

// buildMonthGrid builds a 7-column (Mon..Sun) week grid for one calendar
// month, filling in leading/trailing days from adjacent months (webui-spec.md's
// "Calendar is Go-computed weeks") and zero-filling days absent from dayData
// (money-spec.md's "Calendar renders one grid per calendar month, zero-fills
// client-side"). Each day with activity shows one combined "€42.10 (3)"
// line (spend, then transaction count in parentheses) rather than a
// separate count line.
func buildMonthGrid(year int, month time.Month, dayData map[string]domain.DailySummary) [][]webui.CalendarDay {
	firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	mondayFirstStart := (int(firstOfMonth.Weekday()) + 6) % 7
	gridStart := firstOfMonth.AddDate(0, 0, -mondayFirstStart)

	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	mondayFirstEnd := (int(lastOfMonth.Weekday()) + 6) % 7
	gridEnd := lastOfMonth.AddDate(0, 0, 6-mondayFirstEnd)

	var weeks [][]webui.CalendarDay
	var week []webui.CalendarDay
	for d := gridStart; !d.After(gridEnd); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		day := webui.CalendarDay{Day: d.Day(), InMonth: d.Month() == month}
		if summary, ok := dayData[key]; ok && summary.Count > 0 && day.InMonth {
			day.Total = fmt.Sprintf("%s (%d)", formatEUR(summary.SpendEUR), summary.Count)
			day.LinkURL = dayLinkURL(key)
		}
		week = append(week, day)
		if len(week) == 7 {
			weeks = append(weeks, week)
			week = nil
		}
	}
	return weeks
}

// CalendarWebHandler renders GET /web/money/calendar: one or more
// month-grid calendars stacked on one page, newest month first, defaulting
// to [today - 3 calendar months, today] when ?from/?to are absent. A single
// Prev/Next control at the top of the page (not one per grid) shifts the
// whole range by exactly one calendar month; Next is hidden once the range
// already reaches the current month, since there is nothing further but
// the future to show.
func CalendarWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	userID := webui.CurrentUserID(c)

	loc, err := time.LoadLocation(displayTimezone)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load timezone: %v", err)
		return
	}

	today := time.Now().In(loc)
	fromDate := today.AddDate(0, -calendarDefaultMonths, 0)
	toDate := today

	if v := strings.TrimSpace(c.Query("from")); v != "" {
		parsed, err := time.ParseInLocation("2006-01-02", v, loc)
		if err != nil {
			c.String(http.StatusBadRequest, "invalid from date")
			return
		}
		fromDate = parsed
	}
	if v := strings.TrimSpace(c.Query("to")); v != "" {
		parsed, err := time.ParseInLocation("2006-01-02", v, loc)
		if err != nil {
			c.String(http.StatusBadRequest, "invalid to date")
			return
		}
		toDate = parsed
	}

	fromBound := time.Date(fromDate.Year(), fromDate.Month(), fromDate.Day(), 0, 0, 0, 0, loc)
	toBound := time.Date(toDate.Year(), toDate.Month(), toDate.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1).Add(-time.Nanosecond)

	daily, err := db.GetDailyTransactionSummary(ctx, userID, fromBound, toBound)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load daily summary: %v", err)
		return
	}
	dayData := make(map[string]domain.DailySummary, len(daily))
	for _, d := range daily {
		dayData[d.Date.Format("2006-01-02")] = d
	}

	prevURL := calendarLinkURL(fromDate.AddDate(0, -1, 0), toDate.AddDate(0, -1, 0))

	// Next is hidden once the range already reaches the current month —
	// shifting forward from there would only move into the future.
	var nextURL string
	toMonthStart := time.Date(toDate.Year(), toDate.Month(), 1, 0, 0, 0, 0, loc)
	todayMonthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, loc)
	if toMonthStart.Before(todayMonthStart) {
		nextURL = calendarLinkURL(fromDate.AddDate(0, 1, 0), toDate.AddDate(0, 1, 0))
	}

	navHTML, err := renderCalendarNav(prevURL, nextURL)
	if err != nil {
		c.String(http.StatusInternalServerError, "template error: %v", err)
		return
	}

	// Newest month first, oldest last.
	var grids template.HTML
	monthCursor := time.Date(toDate.Year(), toDate.Month(), 1, 0, 0, 0, 0, loc)
	firstMonth := time.Date(fromDate.Year(), fromDate.Month(), 1, 0, 0, 0, 0, loc)
	for !monthCursor.Before(firstMonth) {
		grids += webui.RenderCalendar(webui.CalendarData{
			Title:    monthCursor.Format("January 2006"),
			Weekdays: []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
			Weeks:    buildMonthGrid(monthCursor.Year(), monthCursor.Month(), dayData),
		})
		monthCursor = monthCursor.AddDate(0, -1, 0)
	}

	formHTML, err := renderCalendarFilterForm(fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"))
	if err != nil {
		c.String(http.StatusInternalServerError, "template error: %v", err)
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	if err := webui.RenderPage(c.Writer, webui.PageData{
		Title:    "Money — Calendar",
		Nav:      moneyNav,
		UserName: c.GetString("user_name"),
		Content:  navHTML + formHTML + grids,
	}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
	}
}
