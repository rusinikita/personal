package transport

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/action/add_food"
	"personal/action/log_food"
	"personal/gateways"
)

func MCPServer(db gateways.DB) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "greeter", Version: "v1.0.0"}, nil)

	server.AddReceivingMiddleware(func(handler mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (result mcp.Result, err error) {
			// Add database to context
			ctx = gateways.WithDB(ctx, db)
			return handler(ctx, method, req)
		}
	})

	mcp.AddTool(server, &add_food.MCPDefinition, add_food.AddFood)
	mcp.AddTool(server, &log_food.LogFoodByIdMCPDefinition, log_food.LogFoodById)
	mcp.AddTool(server, &log_food.LogFoodByNameMCPDefinition, log_food.LogFoodByName)
	mcp.AddTool(server, &log_food.LogFoodByBarcodeMCPDefinition, log_food.LogFoodByBarcode)
	mcp.AddTool(server, &log_food.LogCustomFoodMCPDefinition, log_food.LogCustomFood)

	return server
}
