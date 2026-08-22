package webui

import (
	"github.com/gin-gonic/gin"

	"personal/gateways"
)

// DefaultUserID is the fallback user id for web dashboard pages when no
// session is present (AUTH_DISABLED / webui-preview use).
const DefaultUserID int64 = 1

// CurrentUserID reads the session user id set by auth.WebMiddleware, falling
// back to DefaultUserID for AUTH_DISABLED / webui-preview use without a real
// session.
func CurrentUserID(c *gin.Context) int64 {
	userID := gateways.UserIDFromContext(c.Request.Context())
	if userID == 0 {
		userID = DefaultUserID
	}
	return userID
}
