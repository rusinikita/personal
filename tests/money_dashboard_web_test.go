package tests

// Covers the "Web interface for Money" dashboard from
// docs/functions/money-spec.md (backlog: 19-08-26) — the read-only
// GET /web/money, GET /web/money/transactions, and GET /web/money/calendar
// pages, built on the shared action/webui design system (see
// docs/functions/webui-spec.md), the same way
// tests/workout_dashboard_web_test.go covers the Workouts dashboard.
// Adding/editing/deleting transactions stays out of scope — see
// tests/money_import_test.go for the existing /money/import bulk-import
// coverage, which this dashboard only links out to.

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/money"
	"personal/gateways"
)

// moneyDashboardRouter builds a minimal gin engine wired to the test
// suite's DB and user_id, serving the three money dashboard routes the same
// way transport/web does.
func (s *IntegrationTestSuite) moneyDashboardRouter(ctx context.Context) *gin.Engine {
	userID := gateways.UserIDFromContext(ctx)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		reqCtx := gateways.WithDB(c.Request.Context(), s.Repo())
		reqCtx = gateways.WithUserID(reqCtx, userID)
		c.Request = c.Request.WithContext(reqCtx)
		c.Next()
	})
	r.GET("/web/money", money.MoneyDashboardWebHandler)
	r.GET("/web/money/transactions", money.TransactionsWebHandler)
	r.GET("/web/money/calendar", money.CalendarWebHandler)
	return r
}

// withMoneyTransactionsPageSize temporarily shrinks
// money.TransactionsPageSize so pagination tests don't need to seed 100+
// rows, restoring the default afterward. Mirrors withBrowsePageSize in
// tests/progress_browse_web_test.go.
func withMoneyTransactionsPageSize(t *testing.T, size int) {
	original := money.TransactionsPageSize
	money.TransactionsPageSize = size
	t.Cleanup(func() { money.TransactionsPageSize = original })
}

// addTransaction seeds one transaction via the existing add_transactions
// action (not a raw repo insert), consistent with tests/money_analytics_test.go.
func (s *IntegrationTestSuite) addTransaction(ctx context.Context, txType, category, merchant string, amountEUR float64, at time.Time) {
	_, _, err := money.AddTransactions(ctx, nil, money.AddTransactionsInput{
		Transactions: []money.TransactionInput{
			{Type: txType, AmountOriginal: amountEUR, Currency: "EUR", AmountEUR: amountEUR, Account: "Revolut", Category: category, Merchant: merchant, TransactedAt: at},
		},
	})
	require.NoError(s.T(), err)
}

// --- GET /web/money ------------------------------------------------------

func (s *IntegrationTestSuite) TestMoneyDashboard_CategoryTableSortedByAvgMonthlySpendDesc_WithDrilldownLinks() {
	ctx := s.Context()
	now := time.Now().UTC()

	// FirstTransactionAt ~3 calendar months ago, so months_span is the same
	// divisor for every category below — average monthly spend then sorts
	// in the same order as each category's all-time total.
	threeMonthsAgo := now.AddDate(0, -3, 0)
	s.addTransaction(ctx, "expense", "rent", "Landlord", 700, threeMonthsAgo)
	s.addTransaction(ctx, "expense", "groceries", "Lidl", 900, now.AddDate(0, -2, 0))
	s.addTransaction(ctx, "expense", "groceries", "Lidl", 300, now.AddDate(0, -1, 0))
	s.addTransaction(ctx, "expense", "transport", "Bolt", 50, now.AddDate(0, -1, 0))

	r := s.moneyDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/money", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()

	// groceries avg (1200/span) > rent avg (700/span) > transport avg (50/span).
	// Matched by drill-down href, not the bare category name — "rent" alone
	// would also match inside the "Current balance" stat tile label.
	groceriesLink := `href="/web/money/transactions?category=groceries"`
	rentLink := `href="/web/money/transactions?category=rent"`
	transportLink := `href="/web/money/transactions?category=transport"`
	iGroceries := strings.Index(body, groceriesLink)
	iRent := strings.Index(body, rentLink)
	iTransport := strings.Index(body, transportLink)
	require.NotEqual(s.T(), -1, iGroceries)
	require.NotEqual(s.T(), -1, iRent)
	require.NotEqual(s.T(), -1, iTransport)
	assert.Less(s.T(), iGroceries, iRent, "higher avg monthly spend must be listed first")
	assert.Less(s.T(), iRent, iTransport, "higher avg monthly spend must be listed first")
}

func (s *IntegrationTestSuite) TestMoneyDashboard_EmptyState_NoTransactionsYet() {
	r := s.moneyDashboardRouter(s.Context())
	req := httptest.NewRequest(http.MethodGet, "/web/money", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// A fresh account (GetMoneySummary.FirstTransactionAt == nil) is an
	// expected first-visit state, not a failure — the handler must not
	// error just because there is nothing to divide by months_span yet.
	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Contains(s.T(), w.Body.String(), "webui-stat-tile")
}

func (s *IntegrationTestSuite) TestMoneyDashboard_LastSyncUsesCreatedAt_NotBackdatedTransactedAt() {
	ctx := s.Context()

	// AddTransactions always stamps CreatedAt = time.Now() regardless of
	// the caller-supplied TransactedAt (see gateways/db/repository.go
	// AddTransactions), so a backdated manual entry like this is exactly
	// the case money-spec.md's "Last sync is import freshness, not
	// transaction age" best practice is guarding against.
	longAgo := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	s.addTransaction(ctx, "expense", "misc", "Old Shop", 10, longAgo)

	r := s.moneyDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/money", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.NotContains(s.T(), body, "2020-01-01", "sync freshness must reflect created_at, not the backdated transacted_at")
	assert.Contains(s.T(), body, time.Now().UTC().Format("2006-01-02"), "sync freshness must reflect today's created_at")
}

func (s *IntegrationTestSuite) TestMoneyDashboard_LinksToTransactionsCalendarAndImport() {
	r := s.moneyDashboardRouter(s.Context())
	req := httptest.NewRequest(http.MethodGet, "/web/money", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(s.T(), body, `href="/web/money/transactions"`, "must link to the unfiltered transaction list")
	assert.Contains(s.T(), body, `href="/web/money/calendar"`, "must link to the calendar")
	assert.Contains(s.T(), body, `href="/money/import"`, "must link to the existing bulk-import page rather than duplicating it")
}

func (s *IntegrationTestSuite) TestMoneyDashboard_BalanceTrendChart_PastActualFutureProjected() {
	ctx := s.Context()
	now := time.Now().UTC()

	// First transaction 8 months ago — monthsSpan = 8, avg monthly savings
	// = 1500/8 = 187.5.
	firstAt := now.AddDate(0, -8, 0)
	s.addTransaction(ctx, "income", "salary", "Employer", 1000, firstAt)
	s.addTransaction(ctx, "income", "salary", "Employer", 500, now.AddDate(0, -4, 0))

	r := s.moneyDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/money", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()

	assert.Contains(s.T(), body, `labels: ["-12m","-6m","-3m","Now","+3m","+6m","+12m"]`)
	// -12m predates the account's first transaction (8 months ago) -> 0 (no
	// data yet, not an error). -6m/-3m are the actual cumulative balance as
	// of that past date (1000 before the second income, 1500 after).
	// Now = current balance (1500). +3m/+6m/+12m = current + avg monthly
	// savings (1500/8 = 187.5) x N — the same figures the old Projected
	// tiles used to show, now plotted as a trend instead.
	assert.Contains(s.T(), body, `data: [0,1000,1500,1500,2062.5,2625,3750]`)
	assert.NotContains(s.T(), body, "Projected", "the old separate Projected tiles must be gone")
}

// --- GET /web/money/transactions -----------------------------------------

func (s *IntegrationTestSuite) TestTransactionsList_NoFilters_MostRecentFirst() {
	ctx := s.Context()
	now := time.Now().UTC()

	s.addTransaction(ctx, "expense", "food", "Older Shop", 10, now.AddDate(0, 0, -2))
	s.addTransaction(ctx, "expense", "food", "Newer Shop", 20, now.AddDate(0, 0, -1))

	r := s.moneyDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/money/transactions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	iNewer := strings.Index(body, "Newer Shop")
	iOlder := strings.Index(body, "Older Shop")
	require.NotEqual(s.T(), -1, iNewer)
	require.NotEqual(s.T(), -1, iOlder)
	assert.Less(s.T(), iNewer, iOlder, "most recent transaction must be listed first")
}

func (s *IntegrationTestSuite) TestTransactionsList_FilterByCategory_PrefixMatch() {
	ctx := s.Context()
	now := time.Now().UTC()

	s.addTransaction(ctx, "expense", "food/cafe", "Starbucks", 5, now)
	s.addTransaction(ctx, "expense", "transport", "Bolt", 15, now)

	r := s.moneyDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/money/transactions?category=food", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(s.T(), body, "Starbucks", "category=food must prefix-match food/cafe")
	assert.NotContains(s.T(), body, "Bolt")
}

func (s *IntegrationTestSuite) TestTransactionsList_FilterByDateRange_DisplayTimezoneDayBounds() {
	ctx := s.Context()

	loc, err := time.LoadLocation("Asia/Nicosia")
	require.NoError(s.T(), err)

	// Just after Nicosia local midnight on the 15th — its UTC instant's own
	// calendar date is still the 14th, so this only lands in ?from=2026-06-15&to=2026-06-15
	// if day bounds are computed in Nicosia time, not naively in UTC.
	justAfterLocalMidnight := time.Date(2026, 6, 15, 0, 30, 0, 0, loc)
	// Just after Nicosia local midnight on the 16th — its UTC instant's own
	// calendar date is still the 15th, so a naive UTC-based filter would
	// wrongly include it; Nicosia-aware bounds exclude it.
	justAfterNextLocalMidnight := time.Date(2026, 6, 16, 0, 30, 0, 0, loc)

	s.addTransaction(ctx, "expense", "food", "In Range Shop", 5, justAfterLocalMidnight)
	s.addTransaction(ctx, "expense", "food", "Next Day Shop", 5, justAfterNextLocalMidnight)

	r := s.moneyDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/money/transactions?from=2026-06-15&to=2026-06-15", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(s.T(), body, "In Range Shop")
	assert.NotContains(s.T(), body, "Next Day Shop")
}

func (s *IntegrationTestSuite) TestTransactionsList_FilterByDateRange_FromAndToAreIndependent() {
	ctx := s.Context()

	day1 := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	s.addTransaction(ctx, "expense", "food", "Day1 Shop", 5, day1)
	s.addTransaction(ctx, "expense", "food", "Day2 Shop", 5, day2)
	s.addTransaction(ctx, "expense", "food", "Day3 Shop", 5, day3)

	r := s.moneyDashboardRouter(ctx)

	// Both bounds set: an inclusive [from, to] range.
	reqBoth := httptest.NewRequest(http.MethodGet, "/web/money/transactions?from=2026-06-10&to=2026-06-15", nil)
	wBoth := httptest.NewRecorder()
	r.ServeHTTP(wBoth, reqBoth)
	assert.Equal(s.T(), http.StatusOK, wBoth.Code)
	bodyBoth := wBoth.Body.String()
	assert.Contains(s.T(), bodyBoth, "Day1 Shop")
	assert.Contains(s.T(), bodyBoth, "Day2 Shop")
	assert.NotContains(s.T(), bodyBoth, "Day3 Shop")

	// from alone: open-ended upper bound (everything from that day onward).
	reqFrom := httptest.NewRequest(http.MethodGet, "/web/money/transactions?from=2026-06-15", nil)
	wFrom := httptest.NewRecorder()
	r.ServeHTTP(wFrom, reqFrom)
	bodyFrom := wFrom.Body.String()
	assert.NotContains(s.T(), bodyFrom, "Day1 Shop")
	assert.Contains(s.T(), bodyFrom, "Day2 Shop")
	assert.Contains(s.T(), bodyFrom, "Day3 Shop")

	// to alone: open-ended lower bound (everything up to that day).
	reqTo := httptest.NewRequest(http.MethodGet, "/web/money/transactions?to=2026-06-15", nil)
	wTo := httptest.NewRecorder()
	r.ServeHTTP(wTo, reqTo)
	bodyTo := wTo.Body.String()
	assert.Contains(s.T(), bodyTo, "Day1 Shop")
	assert.Contains(s.T(), bodyTo, "Day2 Shop")
	assert.NotContains(s.T(), bodyTo, "Day3 Shop")
}

func (s *IntegrationTestSuite) TestTransactionsList_Pagination() {
	withMoneyTransactionsPageSize(s.T(), 3)
	ctx := s.Context()
	now := time.Now().UTC()

	for i := 0; i < 5; i++ {
		s.addTransaction(ctx, "expense", "food", fmt.Sprintf("Shop %d", i), 1, now.AddDate(0, 0, -i))
	}

	r := s.moneyDashboardRouter(ctx)

	req1 := httptest.NewRequest(http.MethodGet, "/web/money/transactions", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	assert.Equal(s.T(), http.StatusOK, w1.Code)
	page1 := w1.Body.String()
	assert.Contains(s.T(), page1, "Shop 0")
	assert.Contains(s.T(), page1, "Shop 2")
	assert.NotContains(s.T(), page1, "Shop 4")

	req2 := httptest.NewRequest(http.MethodGet, "/web/money/transactions?page=2", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(s.T(), http.StatusOK, w2.Code)
	page2 := w2.Body.String()
	assert.Contains(s.T(), page2, "Shop 3")
	assert.Contains(s.T(), page2, "Shop 4")
	assert.NotContains(s.T(), page2, "Shop 0")
}

func (s *IntegrationTestSuite) TestTransactionsList_FilterFormPrefilledFromQueryParams() {
	r := s.moneyDashboardRouter(s.Context())
	req := httptest.NewRequest(http.MethodGet, "/web/money/transactions?category=groceries&from=2026-06-10&to=2026-06-20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(s.T(), body, `value="groceries"`, "category input must be pre-filled from ?category")
	assert.Contains(s.T(), body, `value="2026-06-10"`, "from input must be pre-filled from ?from")
	assert.Contains(s.T(), body, `value="2026-06-20"`, "to input must be pre-filled from ?to")
	assert.Contains(s.T(), body, `<a href="/web/money/transactions" role="button" class="outline">Clear</a>`, "Clear must sit next to Apply as an outline button, not its own line")
}

// --- GET /web/money/calendar ----------------------------------------------

func (s *IntegrationTestSuite) TestCalendar_DefaultRange_Last3CalendarMonths() {
	ctx := s.Context()
	now := time.Now().UTC()

	s.addTransaction(ctx, "expense", "food", "Recent Shop", 5, now)
	// Just before [today - 3 calendar months, today] — must not appear.
	s.addTransaction(ctx, "expense", "food", "Too Old Shop", 5, now.AddDate(0, -3, -1))

	r := s.moneyDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/money/calendar", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	// html/template HTML-escapes "&" to "&amp;" inside href attributes.
	assert.Contains(s.T(), body, fmt.Sprintf(`/web/money/transactions?from=%s&amp;to=%s`, now.Format("2006-01-02"), now.Format("2006-01-02")))
	assert.NotContains(s.T(), body, now.AddDate(0, -3, -1).Format("2006-01-02"))
}

func (s *IntegrationTestSuite) TestCalendar_CustomFromTo_SplitsIntoOneGridPerMonth_NewestFirst() {
	r := s.moneyDashboardRouter(s.Context())
	req := httptest.NewRequest(http.MethodGet, "/web/money/calendar?from=2024-06-01&to=2024-08-22", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()

	iAugust := strings.Index(body, "August 2024")
	iJuly := strings.Index(body, "July 2024")
	iJune := strings.Index(body, "June 2024")
	require.NotEqual(s.T(), -1, iAugust)
	require.NotEqual(s.T(), -1, iJuly)
	require.NotEqual(s.T(), -1, iJune)
	assert.Less(s.T(), iAugust, iJuly, "most recent month must render first")
	assert.Less(s.T(), iJuly, iJune, "most recent month must render first")
}

func (s *IntegrationTestSuite) TestCalendar_DayWithTransactions_LinksButEmptyDayDoesNot() {
	ctx := s.Context()
	day := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	s.addTransaction(ctx, "expense", "food", "Shop", 5, day)

	r := s.moneyDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/money/calendar?from=2026-06-01&to=2026-06-30", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	// html/template HTML-escapes "&" to "&amp;" inside href attributes.
	assert.Contains(s.T(), body, `href="/web/money/transactions?from=2026-06-10&amp;to=2026-06-10"`, "a day with transactions must link to its filtered list")
	assert.NotContains(s.T(), body, `href="/web/money/transactions?from=2026-06-11&amp;to=2026-06-11"`, "a day with no transactions must not be a link")
}

func (s *IntegrationTestSuite) TestCalendar_CellShowsCombinedAmountAndCount_NoTransactionsWord() {
	ctx := s.Context()
	day := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	s.addTransaction(ctx, "expense", "food", "Shop", 5, day)

	r := s.moneyDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/money/calendar?from=2026-06-01&to=2026-06-30", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(s.T(), body, "€5.00 (1)", "a day cell must show one combined amount (count) line")
	// "transactions" legitimately appears in hrefs (/web/money/transactions);
	// what must not appear is the old separate count-line wording.
	assert.NotContains(s.T(), body, "1 transactions", "a day cell must not show a separate 'N transactions' count line")
}

func (s *IntegrationTestSuite) TestCalendar_SpendIsExpenseOnly_NotNet() {
	ctx := s.Context()
	day := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	s.addTransaction(ctx, "income", "salary", "Employer", 1000, day)
	s.addTransaction(ctx, "expense", "food", "Shop", 42.10, day)

	r := s.moneyDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/money/calendar?from=2026-06-01&to=2026-06-30", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	// Day total must be the expense-only 42.10, never a net figure like
	// 1000 - 42.10 = 957.90 or 1000 + 42.10 = 1042.10.
	assert.Contains(s.T(), body, "42.10")
	assert.NotContains(s.T(), body, "957.90")
	assert.NotContains(s.T(), body, "1042.10")
}

func (s *IntegrationTestSuite) TestCalendar_SinglePageLevelPrevNext_NotOnePerMonth() {
	r := s.moneyDashboardRouter(s.Context())
	// A range clearly in the past (so Next is guaranteed visible) spanning
	// 3 calendar months — prev/next must still move by exactly one month,
	// not by the full span, and must appear exactly once on the page, not
	// once per stacked month grid.
	req := httptest.NewRequest(http.MethodGet, "/web/money/calendar?from=2020-06-01&to=2020-08-22", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(s.T(), body, "from=2020-05-01", "prev must shift the range start back by one month")
	assert.Contains(s.T(), body, "from=2020-07-01", "next must shift the range start forward by one month")
	assert.Equal(s.T(), 1, strings.Count(body, "← Prev"), "Prev must appear exactly once on the page")
	assert.Equal(s.T(), 1, strings.Count(body, "Next →"), "Next must appear exactly once on the page")
}

func (s *IntegrationTestSuite) TestCalendar_LeadingTrailingDaysFromAdjacentMonth_ShowNoStats() {
	ctx := s.Context()
	// 2026-07-31 is a Friday, so it's a leading (grayed-out) cell in
	// August 2026's grid (which starts on Monday 2026-07-27) while also
	// being an in-month cell in July 2026's own grid.
	day := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	s.addTransaction(ctx, "expense", "food", "Shop", 5, day)

	r := s.moneyDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/money/calendar?from=2026-07-01&to=2026-08-31", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	require.Contains(s.T(), body, "August 2026")
	require.Contains(s.T(), body, "July 2026")

	link := `href="/web/money/transactions?from=2026-07-31&amp;to=2026-07-31"`
	assert.Equal(s.T(), 1, strings.Count(body, link),
		"the leading cell for 2026-07-31 in August's grid must not link or show stats, only July's own in-month cell may")
	assert.Equal(s.T(), 1, strings.Count(body, "€5.00 (1)"),
		"the amount/count must only render once, on the in-month cell")
}

func (s *IntegrationTestSuite) TestCalendar_NextHiddenWhenRangeReachesCurrentMonth_ShownOtherwise() {
	r := s.moneyDashboardRouter(s.Context())

	loc, err := time.LoadLocation("Asia/Nicosia")
	require.NoError(s.T(), err)
	today := time.Now().In(loc)

	// Range ending in the current month — nothing further to show but the
	// future, so Next must be hidden.
	reqCurrent := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/web/money/calendar?from=%s&to=%s",
		today.AddDate(0, -2, 0).Format("2006-01-02"), today.Format("2006-01-02")), nil)
	wCurrent := httptest.NewRecorder()
	r.ServeHTTP(wCurrent, reqCurrent)
	assert.Equal(s.T(), http.StatusOK, wCurrent.Code)
	assert.NotContains(s.T(), wCurrent.Body.String(), "Next →")

	// Range fully in the past — there is a real next month to move to.
	pastFrom := today.AddDate(0, -6, 0)
	pastTo := today.AddDate(0, -4, 0)
	reqPast := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/web/money/calendar?from=%s&to=%s",
		pastFrom.Format("2006-01-02"), pastTo.Format("2006-01-02")), nil)
	wPast := httptest.NewRecorder()
	r.ServeHTTP(wPast, reqPast)
	assert.Equal(s.T(), http.StatusOK, wPast.Code)
	assert.Contains(s.T(), wPast.Body.String(), "Next →")
}

func (s *IntegrationTestSuite) TestCalendar_FilterFormClearIsOutlineButtonNextToApply() {
	r := s.moneyDashboardRouter(s.Context())
	req := httptest.NewRequest(http.MethodGet, "/web/money/calendar", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Contains(s.T(), w.Body.String(), `<a href="/web/money/calendar" role="button" class="outline">Clear</a>`)
}
