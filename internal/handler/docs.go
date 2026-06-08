package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/apidoc"
)

const swaggerUIHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>EduWallet API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    body { margin: 0; background: #f8fafc; }
    .topbar { display: none; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: "%s",
      dom_id: "#swagger-ui",
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis],
      layout: "BaseLayout"
    });
  </script>
</body>
</html>`

// DocsHandler serves API documentation assets.
type DocsHandler struct {
	serverURL string
	specPath  string
}

// NewDocsHandler creates a docs handler for the configured public server URL.
func NewDocsHandler(serverURL string) *DocsHandler {
	return &DocsHandler{
		serverURL: serverURL,
		specPath:  "/api/v1/docs/openapi.json",
	}
}

// SwaggerUI serves a Swagger UI page backed by the generated OpenAPI JSON.
func (h *DocsHandler) SwaggerUI(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(fmt.Sprintf(swaggerUIHTML, h.specPath)))
}

// OpenAPIJSON serves the generated OpenAPI JSON document.
func (h *DocsHandler) OpenAPIJSON(c *gin.Context) {
	payload, err := apidoc.OpenAPIJSON(h.serverURL)
	if err != nil {
		HandleError(c, err)
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", payload)
}
