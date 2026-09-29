package mcp

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/action/achievements"
	"personal/action/docs"
	"personal/action/food"
	"personal/action/ideas"
	"personal/action/money"
	"personal/action/progress"
	"personal/action/telegram"
	"personal/action/workout"
	"personal/gateways"
)

//go:embed instructions.md
var instructions string

func Server(db gateways.DB, tg gateways.Telegram) *mcp.Server {
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
			// Add database and Telegram gateway to context
			ctx = gateways.WithDB(ctx, db)
			ctx = gateways.WithTelegram(ctx, tg)

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

	mcp.AddTool(server, &food.MCPDefinition, food.AddFood)
	mcp.AddTool(server, &food.ResolveFoodIdByNameMCPDefinition, food.ResolveFoodIdByName)
	mcp.AddTool(server, &food.LogFoodByIdMCPDefinition, food.LogFoodById)
	mcp.AddTool(server, &food.LogFoodByBarcodeMCPDefinition, food.LogFoodByBarcode)
	mcp.AddTool(server, &food.LogCustomFoodMCPDefinition, food.LogCustomFood)
	mcp.AddTool(server, &food.GetNutritionStatsMCPDefinition, food.GetNutritionStats)
	mcp.AddTool(server, &food.GetTopProductsMCPDefinition, food.GetTopProducts)
	mcp.AddTool(server, &workout.CreateExerciseMCPDefinition, workout.CreateExercise)
	mcp.AddTool(server, &workout.ListExercisesMCPDefinition, workout.ListExercises)
	mcp.AddTool(server, &workout.SearchExercisesMCPDefinition, workout.SearchExercises)
	mcp.AddTool(server, &workout.EditExerciseMCPDefinition, workout.EditExercise)
	mcp.AddTool(server, &workout.MergeExercisesMCPDefinition, workout.MergeExercises)
	mcp.AddTool(server, &workout.LogWorkoutSetMCPDefinition, workout.LogWorkoutSet)
	mcp.AddTool(server, &workout.DeleteWorkoutSetMCPDefinition, workout.DeleteWorkoutSet)
	mcp.AddTool(server, &workout.GetExerciseHistoryMCPDefinition, workout.GetExerciseHistory)
	mcp.AddTool(server, &workout.GetPersonalRecordsMCPDefinition, workout.GetPersonalRecords)
	mcp.AddTool(server, &workout.ListWorkoutsMCPDefinition, workout.ListWorkouts)
	mcp.AddTool(server, &progress.CreateActivityMCPDefinition, progress.CreateActivity)
	mcp.AddTool(server, &progress.EditActivityMCPDefinition, progress.EditActivity)
	mcp.AddTool(server, &progress.GetActivityListMCPDefinition, progress.GetActivityList)
	mcp.AddTool(server, &progress.ListLifePartsMCPDefinition, progress.ListLifeParts)
	mcp.AddTool(server, &progress.GetProgressTypeExamplesMCPDefinition, progress.GetProgressTypeExamples)
	mcp.AddTool(server, &progress.GetActivityStatsMCPDefinition, progress.GetActivityStats)
	mcp.AddTool(server, &progress.CreateProgressPointMCPDefinition, progress.CreateProgressPoint)
	mcp.AddTool(server, &progress.EditProgressPointMCPDefinition, progress.EditProgressPoint)
	mcp.AddTool(server, &progress.DeleteProgressPointMCPDefinition, progress.DeleteProgressPoint)
	mcp.AddTool(server, &progress.DeleteActivityMCPDefinition, progress.DeleteActivity)
	mcp.AddTool(server, &progress.SearchProgressNotesMCPDefinition, progress.SearchProgressNotes)
	mcp.AddTool(server, &progress.CreateStepMCPDefinition, progress.CreateStep)
	mcp.AddTool(server, &progress.EditStepMCPDefinition, progress.EditStep)
	mcp.AddTool(server, &progress.DeleteStepMCPDefinition, progress.DeleteStep)
	mcp.AddTool(server, &progress.GetStepListMCPDefinition, progress.GetStepList)

	// Money tracking tools
	mcp.AddTool(server, &money.AddTransactionsMCPDefinition, money.AddTransactions)
	mcp.AddTool(server, &money.EditTransactionsMCPDefinition, money.EditTransactions)
	mcp.AddTool(server, &money.DeleteTransactionMCPDefinition, money.DeleteTransaction)
	mcp.AddTool(server, &money.GetTransactionsMCPDefinition, money.GetTransactions)
	mcp.AddTool(server, &money.GetSpendingByCategoryMCPDefinition, money.GetSpendingByCategory)
	mcp.AddTool(server, &money.GetTopMerchantsMCPDefinition, money.GetTopMerchants)
	mcp.AddTool(server, &money.ComparePeriodsMCPDefinition, money.ComparePeriods)
	mcp.AddTool(server, &money.GetBalanceMCPDefinition, money.GetBalance)

	// Achievements tracking tools
	mcp.AddTool(server, &achievements.CreateAchievementMCPDefinition, achievements.CreateAchievement)
	mcp.AddTool(server, &achievements.UpdateAchievementMCPDefinition, achievements.UpdateAchievement)
	mcp.AddTool(server, &achievements.RefreshAchievementsMCPDefinition, achievements.RefreshAchievements)
	mcp.AddTool(server, &achievements.GetAchievementProgressMCPDefinition, achievements.GetAchievementProgress)
	mcp.AddTool(server, &achievements.LogAchievementProgressMCPDefinition, achievements.LogAchievementProgress)

	// Ideas inbox
	mcp.AddTool(server, &ideas.CreateIdeaMCPDefinition, ideas.CreateIdea)
	mcp.AddTool(server, &ideas.ListIdeasMCPDefinition, ideas.ListIdeas)
	mcp.AddTool(server, &ideas.SearchIdeasMCPDefinition, ideas.SearchIdeas)
	mcp.AddTool(server, &ideas.UpdateIdeaMCPDefinition, ideas.UpdateIdea)
	mcp.AddTool(server, &ideas.ResolveIdeaMCPDefinition, ideas.ResolveIdea)

	// Convention docs
	mcp.AddTool(server, &docs.ListDocsMCPDefinition, docs.ListDocs)
	mcp.AddTool(server, &docs.GetDocMCPDefinition, docs.GetDoc)

	// Telegram notifications
	mcp.AddTool(server, &telegram.SendTelegramMessageMCPDefinition, telegram.SendTelegramMessage)

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

		foods, err := food.SearchFoodsByName(ctx, db, searchTerm)
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
		for _, f := range foods {
			values = append(values, f.Name)
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
