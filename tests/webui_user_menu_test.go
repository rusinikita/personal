package tests

// Covers the shell's username dropdown + logout link from
// docs/functions/webui-spec.md ("User menu is a Pico dropdown, no custom
// JS") and docs/functions/auth-spec.md ("Web Session Login").

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/webui"
)

// --- webui.RenderPage: pure rendering, no routing/auth involved -------------

func TestRenderPage_UserNameSet_ShowsDropdownWithLogoutLink(t *testing.T) {
	var buf bytes.Buffer
	err := webui.RenderPage(&buf, webui.PageData{
		Title:    "Test Page",
		UserName: "alice",
		Content:  "<p>content</p>",
	})
	require.NoError(t, err)

	body := buf.String()
	assert.Contains(t, body, `<details class="dropdown">`)
	assert.Contains(t, body, "<summary", "username must be the clickable dropdown trigger")
	assert.Contains(t, body, "alice")
	assert.Contains(t, body, `href="/web/logout"`)
	assert.Contains(t, body, "Logout")
}

func TestRenderPage_UserNameEmpty_HidesDropdown(t *testing.T) {
	var buf bytes.Buffer
	err := webui.RenderPage(&buf, webui.PageData{
		Title:   "Test Page",
		Content: "<p>content</p>",
	})
	require.NoError(t, err)

	body := buf.String()
	assert.NotContains(t, body, `<details class="dropdown">`,
		"no logged-in user means no user menu")
	assert.NotContains(t, body, `href="/web/logout"`)
}

func TestRenderPage_UserNameContainingMarkup_IsEscaped(t *testing.T) {
	// The username comes from the USERS env var, not attacker input, but the
	// shell must still use html/template's contextual escaping consistently
	// (same guarantee webui_design_system_test.go already checks for other
	// fixture fields).
	var buf bytes.Buffer
	err := webui.RenderPage(&buf, webui.PageData{
		Title:    "Test Page",
		UserName: `<script>alert(1)</script>`,
		Content:  "<p>content</p>",
	})
	require.NoError(t, err)

	body := buf.String()
	assert.NotContains(t, body, "<script>alert(1)</script>")
	assert.Contains(t, body, "&lt;script&gt;")
}

// --- GET /web/design-system: real handler picks up the context username ----

// designSystemRouterWithUser mimics what main.go's WebMiddleware will do
// (set "user_name" on the gin context) without depending on action/auth,
// so this test isolates action/webui's responsibility: reading that key and
// passing it through to PageData.
func designSystemRouterWithUser(userName string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if userName != "" {
			c.Set("user_name", userName)
		}
		c.Next()
	})
	r.GET("/web/design-system", webui.DesignSystemHandler)
	return r
}

func TestDesignSystem_ShowsLoggedInUserDropdown(t *testing.T) {
	r := designSystemRouterWithUser("alice")

	req := httptest.NewRequest(http.MethodGet, "/web/design-system", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "alice")
	assert.Contains(t, body, `href="/web/logout"`)
}
