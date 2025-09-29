package find_food

type ResolveFoodIdByNameInput struct {
	NameVariants []string `json:"name_variants" jsonschema:"required,1-5 food name variants to search for"`
}

type ResolveFoodIdByNameOutput struct {
	Foods []FoodMatch `json:"foods" jsonschema:"found food items with ID and name"`
	Error string      `json:"error,omitempty" jsonschema:"error message if any"`
}

type FoodMatch struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	ServingName string `json:"serving_name,omitempty"`
	MatchCount  int    `json:"match_count,omitempty" jsonschema:"number of name variants that matched this food"`
}
