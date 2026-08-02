package money_import

import "testing"

func TestInferCategory_MCCFallback(t *testing.T) {
	cases := []struct {
		description string
		want        string
	}{
		{"ME 5411 VOLI 83", "groceries"},
		{"ME 5812 KAFANA BAR", "food/restaurant"},
		{"ME 5813 COFFEE 60", "food/bar"},
		{"ME 5912 APOTEKA", "health/pharmacy"},
		{"ME 9999 UNKNOWN MERCHANT", "travel/montenegro"}, // unmapped MCC falls back to country catch-all
		{"XX 9999 UNKNOWN MERCHANT", ""},                  // unmapped MCC, unmapped country: no guess
	}
	for _, c := range cases {
		got := InferCategory("", c.description)
		if got != c.want {
			t.Errorf("InferCategory(%q) = %q, want %q", c.description, got, c.want)
		}
	}
}

func TestInferCategory_ExplicitRuleBeatsMCC(t *testing.T) {
	// "anthropic" must win over any MCC in the same description.
	got := InferCategory("", "US 5734 ANTHROPIC CLAUDE")
	if got != "services/ai" {
		t.Errorf("InferCategory() = %q, want services/ai", got)
	}
}

func TestInferCategory_NoFalseSubstringMatch(t *testing.T) {
	// Regression: "astry" (Sri Lankan hotel) used to match inside "COOKASTRY".
	got := InferCategory("", "COOKASTRY LIMITED")
	if got == "travel/hotel" {
		t.Errorf("InferCategory(%q) = %q, want anything but travel/hotel", "COOKASTRY LIMITED", got)
	}
}

func TestRecognizeMerchant_StripsCountryMCCPrefix(t *testing.T) {
	got := RecognizeMerchant("ME 5411 VOLI 83")
	if got != "VOLI 83" {
		t.Errorf("RecognizeMerchant() = %q, want %q", got, "VOLI 83")
	}
}

func TestInferCategoryByCountry_AnchoredNotSubstring(t *testing.T) {
	// A substring-based "ge " country match would misfire on ordinary words
	// like "SAUSAGE FEE" or "COLLEGE FEE" — the country code must only be
	// recognized as a genuine leading prefix.
	cases := []string{"SAUSAGE FEE", "COLLEGE FEE", "BAGGAGE FEE"}
	for _, desc := range cases {
		if got := InferCategory("", desc); got == "travel/georgia" {
			t.Errorf("InferCategory(%q) = %q, want anything but travel/georgia", desc, got)
		}
	}
}

func TestInferCategoryByCountry_KnownPrefix(t *testing.T) {
	got := InferCategory("", "JP 9999 UNKNOWN SHOP")
	if got != "travel/japan" {
		t.Errorf("InferCategory() = %q, want travel/japan", got)
	}
}
