package tests

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/get_transactions"
	money_import "personal/action/money_import"
	"personal/gateways"
)

// importRouter builds a minimal gin engine wired to the test suite's DB
// and user_id from ctx, so the import handler stores data under the test's
// isolated user. ctx must be the same context obtained at the test body level.
func (s *IntegrationTestSuite) importRouter(ctx context.Context) *gin.Engine {
	userID := gateways.UserIDFromContext(ctx)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		reqCtx := gateways.WithDB(c.Request.Context(), s.Repo())
		reqCtx = gateways.WithUserID(reqCtx, userID)
		c.Request = c.Request.WithContext(reqCtx)
		c.Next()
	})
	r.GET("/money/import", money_import.ImportGETHandler)
	r.POST("/money/import", money_import.ImportPOSTHandler)
	return r
}

// multipartCSV builds a multipart/form-data body with account + CSV file.
func multipartCSV(account, csvContent string) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("account", account)
	part, _ := writer.CreateFormFile("file", "export.csv")
	_, _ = fmt.Fprint(part, csvContent)
	writer.Close()
	return body, writer.FormDataContentType()
}

// --- GET /money/import -------------------------------------------------------

func (s *IntegrationTestSuite) TestImport_GET_RendersForm() {
	ctx := s.Context()
	r := s.importRouter(ctx)

	req := httptest.NewRequest(http.MethodGet, "/money/import", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Contains(s.T(), w.Body.String(), "<form")
	assert.Contains(s.T(), w.Body.String(), "account")
}

// --- POST /money/import — Revolut --------------------------------------------

const revolutCSV = `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CARD_PAYMENT,Current,2026-04-01 09:00:00,2026-04-01 09:05:00,Starbucks Coffee,-5.00,0.00,EUR,COMPLETED,995.00
CARD_PAYMENT,Current,2026-04-02 12:00:00,2026-04-02 12:01:00,LIDL CYPRUS 0042 NICOSIA,-32.50,0.00,EUR,COMPLETED,962.50
TOPUP,Current,2026-04-01 08:00:00,2026-04-01 08:00:00,Salary from Employer,3500.00,0.00,EUR,COMPLETED,3500.00
CARD_PAYMENT,Current,2026-04-03 10:00:00,2026-04-03 10:00:00,PENDING TRANSACTION,-10.00,0.00,EUR,PENDING,952.50
`

func (s *IntegrationTestSuite) TestImport_POST_Revolut_Success() {
	ctx := s.Context()
	r := s.importRouter(ctx)

	body, ct := multipartCSV("Revolut", revolutCSV)
	req := httptest.NewRequest(http.MethodPost, "/money/import", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Contains(s.T(), w.Body.String(), "imported")
	// PENDING row skipped → 3 completed rows imported
	assert.Contains(s.T(), w.Body.String(), "imported 3")

	// Verify via get_transactions
	_, listOut, err := get_transactions.GetTransactions(ctx, nil,
		get_transactions.GetTransactionsInput{Limit: 50})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 3, listOut.Total)
}

func (s *IntegrationTestSuite) TestImport_POST_Revolut_ExpenseIncomeSplit() {
	ctx := s.Context()
	r := s.importRouter(ctx)

	body, ct := multipartCSV("Revolut", revolutCSV)
	req := httptest.NewRequest(http.MethodPost, "/money/import", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(s.T(), http.StatusOK, w.Code)

	expenseType := "expense"
	incomeType := "income"

	_, expOut, err := get_transactions.GetTransactions(ctx, nil,
		get_transactions.GetTransactionsInput{Type: &expenseType, Limit: 50})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 2, expOut.Total)

	_, incOut, err := get_transactions.GetTransactions(ctx, nil,
		get_transactions.GetTransactionsInput{Type: &incomeType, Limit: 50})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 1, incOut.Total)
	assert.Equal(s.T(), 3500.00, incOut.Transactions[0].AmountEUR)
}

func (s *IntegrationTestSuite) TestImport_POST_Revolut_MerchantRecognized() {
	ctx := s.Context()
	r := s.importRouter(ctx)

	body, ct := multipartCSV("Revolut", revolutCSV)
	req := httptest.NewRequest(http.MethodPost, "/money/import", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(s.T(), http.StatusOK, w.Code)

	_, listOut, err := get_transactions.GetTransactions(ctx, nil,
		get_transactions.GetTransactionsInput{Limit: 50})
	require.NoError(s.T(), err)

	merchants := map[string]bool{}
	for _, tx := range listOut.Transactions {
		merchants[tx.Merchant] = true
	}
	assert.True(s.T(), merchants["Starbucks"], "Starbucks should be recognized")
	assert.True(s.T(), merchants["Lidl"], "Lidl should be recognized")
}

func (s *IntegrationTestSuite) TestImport_POST_Revolut_CategoryInferred() {
	ctx := s.Context()
	r := s.importRouter(ctx)

	body, ct := multipartCSV("Revolut", revolutCSV)
	req := httptest.NewRequest(http.MethodPost, "/money/import", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(s.T(), http.StatusOK, w.Code)

	starbucksMerchant := "Starbucks"
	_, listOut, err := get_transactions.GetTransactions(ctx, nil,
		get_transactions.GetTransactionsInput{Merchant: &starbucksMerchant, Limit: 10})
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, listOut.Total)
	assert.Equal(s.T(), "food/cafe", listOut.Transactions[0].Category)
}

func (s *IntegrationTestSuite) TestImport_POST_Revolut_OriginalDescriptionPreserved() {
	ctx := s.Context()
	r := s.importRouter(ctx)

	body, ct := multipartCSV("Revolut", revolutCSV)
	req := httptest.NewRequest(http.MethodPost, "/money/import", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(s.T(), http.StatusOK, w.Code)

	lidlMerchant := "Lidl"
	_, listOut, err := get_transactions.GetTransactions(ctx, nil,
		get_transactions.GetTransactionsInput{Merchant: &lidlMerchant, Limit: 10})
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, listOut.Total)
	require.NotNil(s.T(), listOut.Transactions[0].OriginalDescription)
	assert.Contains(s.T(), *listOut.Transactions[0].OriginalDescription, "LIDL CYPRUS")
}

// --- POST /money/import — Bank of Cyprus -------------------------------------

const bocCSV = `Date,Description,Debit,Credit,Currency,Balance
02/04/2026,WOLT ORDER #12345,18.50,,EUR,481.50
03/04/2026,ATM WITHDRAWAL,50.00,,EUR,431.50
01/04/2026,SALARY TRANSFER,,2000.00,EUR,2431.50
`

func (s *IntegrationTestSuite) TestImport_POST_BankOfCyprus_Success() {
	ctx := s.Context()
	r := s.importRouter(ctx)

	body, ct := multipartCSV("Bank of Cyprus", bocCSV)
	req := httptest.NewRequest(http.MethodPost, "/money/import", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Contains(s.T(), w.Body.String(), "imported 3")

	_, listOut, err := get_transactions.GetTransactions(ctx, nil,
		get_transactions.GetTransactionsInput{Limit: 50})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 3, listOut.Total)
}

func (s *IntegrationTestSuite) TestImport_POST_BankOfCyprus_DebitIsExpense() {
	ctx := s.Context()
	r := s.importRouter(ctx)

	body, ct := multipartCSV("Bank of Cyprus", bocCSV)
	req := httptest.NewRequest(http.MethodPost, "/money/import", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(s.T(), http.StatusOK, w.Code)

	expenseType := "expense"
	_, out, err := get_transactions.GetTransactions(ctx, nil,
		get_transactions.GetTransactionsInput{Type: &expenseType, Limit: 50})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 2, out.Total)
}

func (s *IntegrationTestSuite) TestImport_POST_BankOfCyprus_WoltRecognized() {
	ctx := s.Context()
	r := s.importRouter(ctx)

	body, ct := multipartCSV("Bank of Cyprus", bocCSV)
	req := httptest.NewRequest(http.MethodPost, "/money/import", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(s.T(), http.StatusOK, w.Code)

	woltMerchant := "Wolt"
	_, out, err := get_transactions.GetTransactions(ctx, nil,
		get_transactions.GetTransactionsInput{Merchant: &woltMerchant, Limit: 10})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 1, out.Total)
	assert.Equal(s.T(), "food/delivery", out.Transactions[0].Category)
}

// --- POST /money/import — error cases ----------------------------------------

func (s *IntegrationTestSuite) TestImport_POST_UnknownAccount() {
	ctx := s.Context()
	r := s.importRouter(ctx)

	body, ct := multipartCSV("MyBank", "date,amount\n2026-01-01,10\n")
	req := httptest.NewRequest(http.MethodPost, "/money/import", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Contains(s.T(), strings.ToLower(w.Body.String()), "unknown account")
}

func (s *IntegrationTestSuite) TestImport_POST_EmptyCSV() {
	ctx := s.Context()
	r := s.importRouter(ctx)

	body, ct := multipartCSV("Revolut", "Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance\n")
	req := httptest.NewRequest(http.MethodPost, "/money/import", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Contains(s.T(), strings.ToLower(w.Body.String()), "no transactions")
}

func (s *IntegrationTestSuite) TestImport_POST_MissingAccountField() {
	ctx := s.Context()
	r := s.importRouter(ctx)

	// Send without account field
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "export.csv")
	_, _ = fmt.Fprint(part, revolutCSV)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/money/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Contains(s.T(), strings.ToLower(w.Body.String()), "account name is required")
}
