package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/action/add_food"
	"personal/action/find_food"
	"personal/action/log_food"
	"personal/action/nutrition_stats"
	"personal/gateways"
)

const instructions = `Personal food tracking and nutrition logging system for comprehensive dietary monitoring.

This MCP server provides tools for managing a personal food database and logging consumption for nutrition tracking. The system is designed to help track what you eat, when you eat it, and calculate nutritional intake.

## Core Workflow:

1. **Food Management:**
   - Use 'add_food' to create new food entries with complete nutritional information
   - Foods can be basic ingredients, packaged products, or complex recipes/dishes

2. **Food Discovery:**
   - Use 'resolve_food_id_by_name' to search existing foods with multiple name variants
   - Get ranked results with exact food IDs for precise logging

3. **Consumption Logging:**
   - Use 'log_food_by_id' for precise logging when you have the exact food ID
   - Use 'log_food_by_barcode' for packaged products with barcodes
   - Use 'log_custom_food' for one-time entries without saving to database

## Best Practices:

- Search first with 'resolve_food_id_by_name' for ambiguous food names
- Use 'log_food_by_id' after search for most accurate logging
- Add foods to database with 'add_food' for frequently consumed items
- Use 'log_custom_food' for restaurant meals or temporary entries

All consumption logs include calculated nutrition values, timestamps, and optional meal categorization for comprehensive dietary tracking.`

func Server(db gateways.DB) *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{Name: "personal", Title: "Nikita personal food and activities logging", Version: "v1.0.0"},
		&mcp.ServerOptions{
			HasPrompts:        true,
			HasTools:          true,
			CompletionHandler: completionHandler,
			Instructions:      instructions,
		},
	)

	server.AddReceivingMiddleware(func(handler mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (result mcp.Result, err error) {
			// Add database to context
			ctx = gateways.WithDB(ctx, db)
			return handler(ctx, method, req)
		}
	})

	server.AddPrompt(&mcp.Prompt{
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "food_name",
				Title:       "Food Name",
				Description: "Name of the food user consumed",
				Required:    true,
			},
			{
				Name:        "amount_g",
				Title:       "Food amount",
				Description: "Consumed food amount in grams",
				Required:    true,
			}},
		Description: "Generates a message asking to save item in food consumption history",
		Name:        "add_food_log_by_name",
		Title:       "Add consumed food",
	}, promptHandler)

	mcp.AddTool(server, &add_food.MCPDefinition, add_food.AddFood)
	mcp.AddTool(server, &find_food.ResolveFoodIdByNameMCPDefinition, find_food.ResolveFoodIdByName)
	mcp.AddTool(server, &log_food.LogFoodByIdMCPDefinition, log_food.LogFoodById)
	mcp.AddTool(server, &log_food.LogFoodByNameMCPDefinition, log_food.LogFoodByName)
	mcp.AddTool(server, &log_food.LogFoodByBarcodeMCPDefinition, log_food.LogFoodByBarcode)
	mcp.AddTool(server, &log_food.LogCustomFoodMCPDefinition, log_food.LogCustomFood)
	mcp.AddTool(server, &nutrition_stats.GetNutritionStatsMCPDefinition, nutrition_stats.GetNutritionStats)

	return server
}

func promptHandler(_ context.Context, request *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := request.Params.Arguments

	return &mcp.GetPromptResult{
		Meta:        nil,
		Description: "Food consumption history adding prompt",
		Messages: []*mcp.PromptMessage{
			{
				Content: &mcp.TextContent{
					Text: fmt.Sprintf("Please add food named '%s' with amount of %s in food consumption log. Using 'log_food_by_name' tool. But if you know id of the food - use 'log_food_by_id'", args["food_name"], args["amount_g"]),
				},
				Role: "user",
			},
		},
	}, nil
}

func completionHandler(ctx context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	if req.Params.Argument.Name == "food_name" {
		// Get database from context
		db := gateways.DBFromContext(ctx)
		if db == nil {
			// Fallback to hardcoded values if no database
			return &mcp.CompleteResult{
				Completion: mcp.CompletionResultDetails{
					HasMore: false,
					Total:   2,
					Values:  []string{"банан", "яйцо"},
				},
			}, nil
		}

		// Use real database search for completion
		searchTerm := req.Params.Argument.Value
		if searchTerm == "" {
			searchTerm = "банан" // Default search term
		}

		foods, err := find_food.SearchFoodsByName(ctx, db, searchTerm)
		if err != nil {
			// Fallback to hardcoded values on error
			return &mcp.CompleteResult{
				Completion: mcp.CompletionResultDetails{
					HasMore: false,
					Total:   2,
					Values:  []string{"банан", "яйцо"},
				},
			}, nil
		}

		// Extract food names for completion
		values := make([]string, 0, len(foods))
		for _, food := range foods {
			values = append(values, food.Name)
			if len(values) >= 5 { // Limit to 5 suggestions
				break
			}
		}

		return &mcp.CompleteResult{
			Completion: mcp.CompletionResultDetails{
				HasMore: len(foods) > 5,
				Total:   len(values),
				Values:  values,
			},
		}, nil
	}

	return &mcp.CompleteResult{
		Completion: mcp.CompletionResultDetails{
			HasMore: false,
			Total:   0,
		},
	}, nil
}
