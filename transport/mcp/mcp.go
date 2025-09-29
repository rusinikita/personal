package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/action/add_food"
	"personal/action/log_food"
	"personal/gateways"
)

func Server(db gateways.DB) *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{Name: "personal", Title: "Nikita personal food and activities logging", Version: "v1.0.0"},
		&mcp.ServerOptions{
			Instructions: "",
			HasPrompts:   true,
			// HasResources:                true,
			HasTools:          true,
			CompletionHandler: completionHandler,
		})

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
	mcp.AddTool(server, &log_food.LogFoodByIdMCPDefinition, log_food.LogFoodById)
	mcp.AddTool(server, &log_food.LogFoodByNameMCPDefinition, log_food.LogFoodByName)
	mcp.AddTool(server, &log_food.LogFoodByBarcodeMCPDefinition, log_food.LogFoodByBarcode)
	mcp.AddTool(server, &log_food.LogCustomFoodMCPDefinition, log_food.LogCustomFood)

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

func completionHandler(_ context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	if req.Params.Argument.Name == "food_name" {
		return &mcp.CompleteResult{
			Completion: mcp.CompletionResultDetails{
				HasMore: false,
				Total:   2,
				Values:  []string{"банан", "яйцо"},
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
