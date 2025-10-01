package top_products

import "personal/domain"

const DEFAULT_USER_ID = int64(1)

// GetTopProductsOutput is the output structure for the get_top_products MCP tool
type GetTopProductsOutput struct {
	Products []domain.FoodStats `json:"products" jsonschema:"List of top 30 most frequently logged products in the last 3 months, sorted by log count (descending)"`
}
