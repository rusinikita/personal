// Command webui-preview runs all /web/* routes (via transport/web) against a
// mock repository instead of Postgres, so new web handlers can be eyeballed
// in a browser without standing up a database or Telegram credentials the
// way the full app (main.go) requires. Auth still runs for real: set USERS
// and JWT_SECRET (e.g. in .env.local) the same way main.go requires, unless
// AUTH_DISABLED is set.
package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"personal/action/auth"
	"personal/gateways/db"
	"personal/transport/web"
)

func main() {
	if err := godotenv.Overload(".env.local"); err != nil {
		log.Println("Error loading .env.local file", err)
	}

	authDisabled := os.Getenv("AUTH_DISABLED") != ""
	if !authDisabled {
		auth.InitializeAuth()
	}

	gin.SetMode(gin.DebugMode)
	router := gin.Default()

	web.Register(router, db.NewMockRepository(), authDisabled)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("webui preview running at http://localhost:%s/web/design-system", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
