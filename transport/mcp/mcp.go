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
	"personal/action/progress"
	"personal/action/top_products"
	"personal/gateways"
)

const instructions = `Personal tracking system for nutrition, workout, and life progress tracking.

This MCP server provides tools for managing food database, nutrition tracking, workout logging, and life progress reflection. Track what you eat, your workout sessions, and your progress across different life areas with daily/weekly reflection check-ins.

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

## Progress Tracking Workflow:

**IMPORTANT**: Progress tracking is for daily/weekly reflection on life areas (mood, habits, projects, promises). Activities are pre-created by the user (NOT via MCP tools).

1. **Starting a Reflection Session:**
   - ALWAYS call 'get_progress_type_examples' FIRST to load natural language mappings
   - Then call 'get_activity_list' to see what needs check-in today
   - Activities are ordered by frequency (daily first, weekly next)

2. **For Each Activity - Gather Context:**
   - Call 'get_activity_stats' with activity_id to see historical trends
   - Present context: "Last time you logged +1 (bright). This week averaging +1.8"
   - This helps user reflect on their current state

3. **Ask Progress Question Based on Type:**
   - **Mood activities**: "How are you feeling today?" or suggest metaphors: "Like weather - sunny to stormy?"
   - **Habit activities**: "How's your [habit name] going?" or "Is it blooming or wilting?"
   - **Project activities**: "Any progress on [project name]?" or "Sprinting or stuck?"
   - **Promise activities**: "Did you remember to [promise]?" or "Is the flame still burning?"

4. **Interpret User Response:**
   - Map natural language to numeric values using get_progress_type_examples data
   - Examples: "sunny" → +2, "trying" → 0, "stuck" → 0, "forgot" → -1
   - If ambiguous, offer metaphor choices or ask clarifying questions

5. **Log Progress:**
   - Call 'create_progress_point' with activity_id, mapped value, and optional note
   - Include user's explanation in the note field
   - For projects, ask about hours_left if relevant
   - Confirm with emoji: "Logged your mood as sunny ☀️ (+2)!"

6. **Handle Completion:**
   - If user says "I finished it", "It's done", "Completed" → use 'finish_activity'
   - This is PERMANENT - activity won't appear in future lists
   - Celebrate completion: "Congratulations! 🎉"

## Progress Tracking Best Practices:

**Before Every Reflection Session:**
1. Call 'get_progress_type_examples' to load mappings (CRITICAL - do this first!)
2. Call 'get_activity_list' to see active activities
3. For each activity, call 'get_activity_stats' before asking question

**During Conversation:**
- Use emojis from the mappings to make it engaging
- Show trends: "You're averaging +1.5 this week - trending up!"
- Reference last values: "Last time you were at 0 (neutral)"
- Offer metaphor choices when user is uncertain
- Save detailed notes in the progress point

**When to Finish Activities:**
- Projects completed: "Deployed to production", "Launch finished"
- Goals achieved: "30-day challenge done"
- User wants to stop tracking a habit
- DO NOT finish for regular check-ins - use create_progress_point instead

**Key Concepts:**
- Progress values: -2 (worst) to +2 (best), with 0 as neutral
- Four types: mood, habit_progress, project_progress, promise_state
- Each type has multiple natural language metaphors with emojis
- Activities have frequency_days (1=daily, 7=weekly, etc.)
- Historical stats show overall, last month, and last week trends

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
	mcp.AddTool(server, &progress.GetActivityListMCPDefinition, progress.GetActivityList)
	mcp.AddTool(server, &progress.GetProgressTypeExamplesMCPDefinition, progress.GetProgressTypeExamples)
	mcp.AddTool(server, &progress.GetActivityStatsMCPDefinition, progress.GetActivityStats)
	mcp.AddTool(server, &progress.CreateProgressPointMCPDefinition, progress.CreateProgressPoint)
	mcp.AddTool(server, &progress.FinishActivityMCPDefinition, progress.FinishActivity)

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
