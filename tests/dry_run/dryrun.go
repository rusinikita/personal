//go:build ignore

package main

import (
	"fmt"
	"math"
	"os"
	"sort"

	"personal/action/money_import"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: go run tests/dry_run/dryrun.go <account> <file.csv>")
		os.Exit(1)
	}
	account := os.Args[1]
	path := os.Args[2]

	parser := money_import.ParserFor(account)
	if parser == nil {
		fmt.Printf("unknown account %q — supported: Revolut, Bank of Cyprus\n", account)
		os.Exit(1)
	}

	f, err := os.Open(path)
	if err != nil {
		fmt.Println("open error:", err)
		os.Exit(1)
	}
	defer f.Close()

	raws, err := parser.Parse(f)
	if err != nil {
		fmt.Println("parse error:", err)
		os.Exit(1)
	}

	skipped, imported := 0, 0
	categoryTotals := map[string]float64{}
	uncategorizedByMerchant := map[string]float64{}
	uncategorizedCountByMerchant := map[string]int{}

	fmt.Printf("%-12s %-10s %-8s %8s %-20s %-30s\n", "date", "type", "currency", "amount", "merchant", "category")
	fmt.Println("----------------------------------------------------------------------------------------------------")

	for _, raw := range raws {
		if raw.Amount == 0 {
			skipped++
			continue
		}
		if raw.Currency != "EUR" {
			skipped++
			fmt.Printf("%-12s %-10s %-8s %8.2f %-20s %-30s  [SKIP: non-EUR]\n",
				raw.Date.Format("2006-01-02"),
				"?",
				raw.Currency,
				math.Abs(raw.Amount),
				"",
				"",
			)
			continue
		}

		txType := "expense"
		amt := math.Abs(raw.Amount)
		if override := money_import.InferTypeOverride(raw.Description); override != "" {
			txType = override
		} else if raw.Amount > 0 {
			txType = "income"
		}

		merchant := money_import.RecognizeMerchant(raw.Description)
		category := money_import.InferCategory(merchant, raw.Description)
		if category == "" {
			category = "(uncategorized)"
		}

		fmt.Printf("%-12s %-10s %-8s %8.2f %-20s %-30s\n",
			raw.Date.Format("2006-01-02"),
			txType,
			raw.Currency,
			amt,
			merchant,
			category,
		)
		imported++
		categoryTotals[category] += amt
		if category == "(uncategorized)" {
			uncategorizedByMerchant[merchant] += amt
			uncategorizedCountByMerchant[merchant]++
		}
	}

	fmt.Println("----------------------------------------------------------------------------------------------------")
	fmt.Printf("would import: %d  |  skipped: %d\n\n", imported, skipped)

	// Sort categories alphabetically for stable output
	categories := make([]string, 0, len(categoryTotals))
	for cat := range categoryTotals {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	fmt.Printf("%-35s %10s\n", "category", "total EUR")
	fmt.Println("-----------------------------------------------")
	for _, cat := range categories {
		fmt.Printf("%-35s %10.2f\n", cat, categoryTotals[cat])
	}

	// Uncategorized breakdown by merchant
	if len(uncategorizedByMerchant) > 0 {
		type merchantTotal struct {
			name  string
			total float64
			count int
		}
		ranked := make([]merchantTotal, 0, len(uncategorizedByMerchant))
		for m, t := range uncategorizedByMerchant {
			ranked = append(ranked, merchantTotal{m, t, uncategorizedCountByMerchant[m]})
		}
		sort.Slice(ranked, func(i, j int) bool {
			return ranked[i].total > ranked[j].total
		})

		fmt.Printf("\n%-35s %10s %6s\n", "uncategorized merchant", "total EUR", "count")
		fmt.Println("---------------------------------------------------")
		for _, mt := range ranked {
			fmt.Printf("%-35s %10.2f %6d\n", mt.name, mt.total, mt.count)
		}
	}
}
