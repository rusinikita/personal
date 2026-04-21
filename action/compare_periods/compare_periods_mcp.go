package compare_periods

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
)

var MCPDefinition = mcp.Tool{
	Name:        "compare_periods",
	Description: "Side-by-side comparison of expense spending between two time periods, broken down by top-level category. Shows diff in EUR and percentage change.",
}

// ComparePeriodsInput is the MCP tool input.
type ComparePeriodsInput struct {
	PeriodAFrom time.Time `json:"period_a_from"`
	PeriodATo   time.Time `json:"period_a_to"`
	PeriodBFrom time.Time `json:"period_b_from"`
	PeriodBTo   time.Time `json:"period_b_to"`
}

// PeriodSummary is spending data for one period.
type PeriodSummary struct {
	From       time.Time      `json:"from"`
	To         time.Time      `json:"to"`
	TotalEUR   float64        `json:"total_eur"`
	Categories []CategoryItem `json:"categories"`
}

// CategoryItem is one spending row inside a period summary.
type CategoryItem struct {
	Category string  `json:"category"`
	TotalEUR float64 `json:"total_eur"`
}

// CategoryDiff is the delta between two periods for one category.
type CategoryDiff struct {
	Category   string  `json:"category"`
	PeriodAEUR float64 `json:"period_a_eur"`
	PeriodBEUR float64 `json:"period_b_eur"`
	DiffEUR    float64 `json:"diff_eur"`
	DiffPct    float64 `json:"diff_pct"`
}

// ComparePeriodsOutput is the MCP tool output.
type ComparePeriodsOutput struct {
	PeriodA PeriodSummary  `json:"period_a"`
	PeriodB PeriodSummary  `json:"period_b"`
	Diff    []CategoryDiff `json:"diff"`
}

func ComparePeriods(ctx context.Context, _ *mcp.CallToolRequest, input ComparePeriodsInput) (*mcp.CallToolResult, ComparePeriodsOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, ComparePeriodsOutput{}, fmt.Errorf("database not available in context")
	}
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, ComparePeriodsOutput{}, fmt.Errorf("user_id not available in context")
	}

	rowsA, err := db.GetSpendingForPeriod(ctx, userID, input.PeriodAFrom, input.PeriodATo)
	if err != nil {
		return nil, ComparePeriodsOutput{}, fmt.Errorf("database error (period a): %w", err)
	}
	rowsB, err := db.GetSpendingForPeriod(ctx, userID, input.PeriodBFrom, input.PeriodBTo)
	if err != nil {
		return nil, ComparePeriodsOutput{}, fmt.Errorf("database error (period b): %w", err)
	}

	mapA := make(map[string]float64, len(rowsA))
	for _, r := range rowsA {
		mapA[r.Category] = r.TotalEUR
	}
	mapB := make(map[string]float64, len(rowsB))
	for _, r := range rowsB {
		mapB[r.Category] = r.TotalEUR
	}

	// Union of all categories.
	allCats := make(map[string]struct{})
	for k := range mapA {
		allCats[k] = struct{}{}
	}
	for k := range mapB {
		allCats[k] = struct{}{}
	}

	var diffs []CategoryDiff
	var totalA, totalB float64
	for cat := range allCats {
		a := mapA[cat]
		b := mapB[cat]
		totalA += a
		totalB += b
		diffEUR := b - a
		var diffPct float64
		if a != 0 {
			diffPct = (diffEUR / a) * 100
		}
		diffs = append(diffs, CategoryDiff{
			Category:   cat,
			PeriodAEUR: a,
			PeriodBEUR: b,
			DiffEUR:    diffEUR,
			DiffPct:    diffPct,
		})
	}

	catsA := make([]CategoryItem, len(rowsA))
	for i, r := range rowsA {
		catsA[i] = CategoryItem{Category: r.Category, TotalEUR: r.TotalEUR}
	}
	catsB := make([]CategoryItem, len(rowsB))
	for i, r := range rowsB {
		catsB[i] = CategoryItem{Category: r.Category, TotalEUR: r.TotalEUR}
	}

	return nil, ComparePeriodsOutput{
		PeriodA: PeriodSummary{From: input.PeriodAFrom, To: input.PeriodATo, TotalEUR: totalA, Categories: catsA},
		PeriodB: PeriodSummary{From: input.PeriodBFrom, To: input.PeriodBTo, TotalEUR: totalB, Categories: catsB},
		Diff:    diffs,
	}, nil
}
