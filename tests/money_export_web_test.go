package tests

// Covers the "Spending Export" screen from docs/functions/money-spec.md
// (backlog: 22-08-26 — Spending export & category view for Money):
// GET /web/money/export (filter + editable preview) and
// GET /web/money/export/download (the CSV itself). Uses
// moneyDashboardRouter and addTransaction from money_dashboard_web_test.go.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- GET /web/money/export -------------------------------------------------

func (s *IntegrationTestSuite) TestExportPreview_DefaultRange_CurrentCalendarMonth() {
	ctx := s.Context()
	now := time.Now().UTC()

	s.addTransaction(ctx, "expense", "groceries", "Lidl", 120, now)
	// Clearly outside the current calendar month.
	s.addTransaction(ctx, "expense", "rent", "Landlord", 700, now.AddDate(0, -2, 0))

	r := s.moneyDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/money/export", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(s.T(), body, `<td style="vertical-align:middle">groceries</td>`)
	assert.NotContains(s.T(), body, `<td style="vertical-align:middle">rent</td>`, "a transaction outside the current calendar month must not appear in the default-range preview")
}

func (s *IntegrationTestSuite) TestExportPreview_CustomRange_AllCategoriesIncludedByDefault_RealTotalsAndCount() {
	ctx := s.Context()
	day := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	s.addTransaction(ctx, "expense", "groceries", "Lidl", 100, day)
	s.addTransaction(ctx, "expense", "groceries", "Lidl", 20, day)
	s.addTransaction(ctx, "expense", "transport", "Bolt", 15, day)

	r := s.moneyDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/money/export?from=2026-06-01&to=2026-06-30", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()

	assert.Contains(s.T(), body, `<input type="checkbox" name="incl[groceries]" checked style="margin:0">`, "every category must default to included on first visit")
	assert.Contains(s.T(), body, `<input type="checkbox" name="incl[transport]" checked style="margin:0">`)
	assert.Contains(s.T(), body, `name="amt[groceries]" value="120.00"`, "amount input must default to the real total")
	assert.Contains(s.T(), body, "€120.00", "real total column")
	assert.Contains(s.T(), body, `value="2026-06-01"`)
	assert.Contains(s.T(), body, `value="2026-06-30"`)
}

func (s *IntegrationTestSuite) TestExportPreview_ExcludedCategory_DropsOutOfRunningTotal() {
	ctx := s.Context()
	day := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	s.addTransaction(ctx, "expense", "groceries", "Lidl", 100, day)
	s.addTransaction(ctx, "expense", "transport", "Bolt", 15, day)

	r := s.moneyDashboardRouter(ctx)
	// groceries checkbox omitted entirely = unchecked = excluded.
	req := httptest.NewRequest(http.MethodGet, "/web/money/export?from=2026-06-01&to=2026-06-30&incl%5Btransport%5D=on&amt%5Bgroceries%5D=100.00&amt%5Btransport%5D=15.00", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(s.T(), body, `<input type="checkbox" name="incl[groceries]"  style="margin:0">`, "an omitted incl[category] must render unchecked")
	assert.Contains(s.T(), body, `<input type="checkbox" name="incl[transport]" checked style="margin:0">`)
	assert.Contains(s.T(), body, "€15.00", "running total must only sum included categories (transport, not groceries)")
	assert.NotContains(s.T(), body, "€115.00", "excluded groceries must not be added into the running total")
}

func (s *IntegrationTestSuite) TestExportPreview_OverrideAmount_ReflectedInInputAndRunningTotal() {
	ctx := s.Context()
	day := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	s.addTransaction(ctx, "expense", "groceries", "Lidl", 100, day)

	r := s.moneyDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/money/export?from=2026-06-01&to=2026-06-30&incl%5Bgroceries%5D=on&amt%5Bgroceries%5D=42.50", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(s.T(), body, `name="amt[groceries]" value="42.50"`, "amount input must reflect the submitted override, not the real total")
	assert.Contains(s.T(), body, "€100.00", "real total column must still show the unmodified real total")
	assert.Contains(s.T(), body, "€42.50", "running total must use the overridden amount")
}

// --- GET /web/money/export/download ----------------------------------------

func (s *IntegrationTestSuite) TestExportDownload_NoSelection_AllCategoriesAtRealTotals() {
	ctx := s.Context()
	day := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	s.addTransaction(ctx, "expense", "groceries", "Lidl", 100, day)
	s.addTransaction(ctx, "expense", "groceries", "Lidl", 20, day)
	s.addTransaction(ctx, "expense", "transport", "Bolt", 15, day)

	r := s.moneyDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/money/export/download?from=2026-06-01&to=2026-06-30", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Equal(s.T(), "text/csv", w.Header().Get("Content-Type"))
	assert.Equal(s.T(), `attachment; filename="spending-export-2026-06-01-2026-06-30.csv"`, w.Header().Get("Content-Disposition"))

	body := w.Body.String()
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	require.Equal(s.T(), []string{
		"Category,Amount (EUR),Count",
		"groceries,120.00,2",
		"transport,15.00,1",
	}, lines)
}

func (s *IntegrationTestSuite) TestExportDownload_ExcludedCategory_AbsentFromCSV() {
	ctx := s.Context()
	day := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	s.addTransaction(ctx, "expense", "groceries", "Lidl", 100, day)
	s.addTransaction(ctx, "expense", "transport", "Bolt", 15, day)

	r := s.moneyDashboardRouter(ctx)
	// Only transport checked -> groceries excluded from the export.
	req := httptest.NewRequest(http.MethodGet, "/web/money/export/download?from=2026-06-01&to=2026-06-30&incl%5Btransport%5D=on", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.NotContains(s.T(), body, "groceries", "excluded category must not appear in the export")
	assert.Contains(s.T(), body, "transport,15.00,1")
}

func (s *IntegrationTestSuite) TestExportDownload_OverriddenAmount_UsedInsteadOfRealTotal() {
	ctx := s.Context()
	day := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	s.addTransaction(ctx, "expense", "groceries", "Lidl", 100, day)

	r := s.moneyDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/money/export/download?from=2026-06-01&to=2026-06-30&incl%5Bgroceries%5D=on&amt%5Bgroceries%5D=1.00", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(s.T(), body, "groceries,1.00,1", "the redacted/overridden amount must be exported, not the real 100.00 total")
	assert.NotContains(s.T(), body, "100.00")
}

func (s *IntegrationTestSuite) TestExportDownload_CategoryWithComma_QuotedPerRFC4180() {
	ctx := s.Context()
	day := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	s.addTransaction(ctx, "expense", "gifts, misc", "Shop", 10, day)

	r := s.moneyDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/money/export/download?from=2026-06-01&to=2026-06-30", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Contains(s.T(), w.Body.String(), `"gifts, misc",10.00,1`)
}
