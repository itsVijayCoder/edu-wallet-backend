package handler

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/apidoc"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/buildinfo"
)

//go:embed api-test.html
var apiTestHTML []byte

//go:embed deployments.html
var deploymentsHTML []byte

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
	// deployStatusFile is the path to the pipeline-written status.json, bind-mounted
	// read-only into the container. Empty when unset (local dev) → deployments null.
	deployStatusFile string
}

// NewDocsHandler creates a docs handler for the configured public server URL.
//
// The deployment status file path is read from the DEPLOY_STATUS_FILE environment
// variable at construction. It is infrastructure metadata written by the deploy
// pipeline and not part of the typed application config, so os.Getenv is used here
// rather than threading it through config/DI.
func NewDocsHandler(serverURL string) *DocsHandler {
	return &DocsHandler{
		serverURL:        serverURL,
		specPath:         "/api/v1/docs/openapi.json",
		deployStatusFile: os.Getenv("DEPLOY_STATUS_FILE"),
	}
}

// SwaggerUI serves a Swagger UI page backed by the generated OpenAPI JSON.
func (h *DocsHandler) SwaggerUI(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(fmt.Sprintf(swaggerUIHTML, h.specPath)))
}

// APITestGuide serves the phase-wise API tester guide.
func (h *DocsHandler) APITestGuide(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", apiTestHTML)
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

// Deployments serves the self-contained deployment status dashboard page.
func (h *DocsHandler) Deployments(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", deploymentsHTML)
}

// DeployStatus returns build identity, uptime, and the pipeline-written deployment
// status document. It is public (no auth) and intended for the deployments dashboard.
func (h *DocsHandler) DeployStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"build": gin.H{
			"sha":        buildinfo.SHA,
			"sha_short":  buildinfo.ShortSHA(),
			"build_time": buildinfo.BuildTime,
		},
		"uptime_seconds": int64(time.Since(buildinfo.StartedAt).Seconds()),
		"server_time":    time.Now().UTC().Format(time.RFC3339),
		"deployments":    h.readDeployments(),
	})
}

// readDeployments reads and passes through the pipeline status.json verbatim. It
// returns a nil RawMessage (marshals to JSON null) when the file is unset, missing,
// unreadable, or not valid JSON — the graceful "no deployment data" state.
func (h *DocsHandler) readDeployments() json.RawMessage {
	if h.deployStatusFile == "" {
		return nil
	}
	data, err := os.ReadFile(h.deployStatusFile)
	if err != nil || !json.Valid(data) {
		return nil
	}
	return json.RawMessage(data)
}
