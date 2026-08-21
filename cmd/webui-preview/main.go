// Command webui-preview runs a bare HTTP server exposing only
// /web/design-system, so the shared design system (action/webui) can be
// eyeballed in a browser without standing up Postgres or Telegram
// credentials the way the full app (main.go) requires.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"personal/action/webui"
)

func main() {
	gin.SetMode(gin.DebugMode)
	router := gin.Default()

	router.GET("/web/design-system", webui.DesignSystemHandler)
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/web/design-system")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("webui preview running at http://localhost:%s/web/design-system", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
