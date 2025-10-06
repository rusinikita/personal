package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/action/add_food"
	"personal/action/create_exercise"
	"personal/action/find_food"
	"personal/action/list_exercises"
	"personal/action/list_workouts"
	"personal/action/log_food"
	"personal/action/log_workout_set"
	"personal/action/nutrition_stats"
	"personal/action/top_products"
	"personal/gateways"
)

const instructions = `Personal tracking system for nutrition and workout logging.

This MCP server provides tools for managing food database, nutrition tracking, and workout logging. The system helps track what you eat, when you eat it, and your workout sessions with exercises and sets.

## Food Tracking Workflow:

1. **Food Management:**
   - Use 'add_food' to create new food entries with complete nutritional information
   - Foods can be basic ingredients, packaged products, or complex recipes/dishes

2. **Food Discovery:**
   - Use 'resolve_food_id_by_name' to search existing foods with multiple name variants
   - Get ranked results with exact food IDs for precise logging
   - Use 'get_top_products' to see your 30 most frequently logged products from last 3 months

3. **Consumption Logging:**
   - Use 'log_food_by_id' for precise logging when you have the exact food ID
   - Use 'log_food_by_barcode' for packaged products with barcodes
   - Use 'log_custom_food' for one-time entries without saving to database

4. **Nutrition Analytics:**
   - Use 'get_nutrition_stats' to view nutrition summary for last meal and last 4 days
   - Use 'get_top_products' to identify your most frequently logged foods

## Workout Tracking Workflow:

1. **Exercise Management:**
   - Use 'create_exercise' to add new exercises with equipment type (machine, barbell, dumbbells, bodyweight)
   - Use 'list_exercises' to see available exercises sorted by last usage

2. **Workout Logging:**
   - Use 'log_workout_set' to log exercise sets with reps/duration and weight
   - Automatically creates or reuses active workouts (active for 2 hours)
   - Track reps-based exercises (bench press, squats) or time-based (plank, running)

3. **Workout History:**
   - Use 'list_workouts' to see recent workouts (last 30 days) with all exercises and sets
   - View active and completed workouts with detailed set information

## Optimal User Experience:

**For Quick Food Logging:**
1. Start with 'get_top_products' to see frequently logged items
2. Use 'log_food_by_id' with IDs from top products for instant logging
3. Check 'get_nutrition_stats' after meals to track daily intake

**For Workout Sessions:**
1. Use 'list_exercises' to see available exercises
2. Log sets with 'log_workout_set' - workouts are created automatically
3. Use 'list_workouts' to review recent training sessions

## Best Practices:

**Food Tracking:**
- Start sessions by calling 'get_top_products' to see frequently logged foods
- Use 'log_food_by_id' for fastest and most accurate logging
- Add foods to database with 'add_food' for frequently consumed items
- After logging, ALWAYS ask user if they want to see nutrition statistics

**Workout Tracking:**
- Create exercises once, reuse them across workouts
- Log sets as you complete them - workouts auto-close after 2 hours
- Review 'list_workouts' to track progress over time

All logs include timestamps and comprehensive details for accurate tracking.`

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
			ctx = gateways.WithUserID(ctx, 1)

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
	mcp.AddTool(server, &log_food.LogFoodByBarcodeMCPDefinition, log_food.LogFoodByBarcode)
	mcp.AddTool(server, &log_food.LogCustomFoodMCPDefinition, log_food.LogCustomFood)
	mcp.AddTool(server, &nutrition_stats.GetNutritionStatsMCPDefinition, nutrition_stats.GetNutritionStats)
	mcp.AddTool(server, &top_products.GetTopProductsMCPDefinition, top_products.GetTopProducts)
	mcp.AddTool(server, &create_exercise.MCPDefinition, create_exercise.CreateExercise)
	mcp.AddTool(server, &list_exercises.MCPDefinition, list_exercises.ListExercises)
	mcp.AddTool(server, &log_workout_set.MCPDefinition, log_workout_set.LogWorkoutSet)
	mcp.AddTool(server, &list_workouts.MCPDefinition, list_workouts.ListWorkouts)

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
