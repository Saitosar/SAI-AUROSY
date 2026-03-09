package openapi

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed openapi.json
var specFS embed.FS

// SpecHandler returns an HTTP handler that serves the OpenAPI spec at /openapi.json.
func SpecHandler() http.HandlerFunc {
	data, _ := fs.ReadFile(specFS, "openapi.json")
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}
}

// SwaggerUIHandler returns an HTTP handler that serves a minimal Swagger UI page.
func SwaggerUIHandler() http.HandlerFunc {
	html := `<!DOCTYPE html>
<html>
<head><title>SAI AUROSY API</title>
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"/>
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>
  var specUrl = window.location.pathname.indexOf('/api/') === 0 ? '/api/openapi.json' : '/openapi.json';
  SwaggerUIBundle({
    url: specUrl,
    dom_id: "#swagger-ui"
  });
</script>
</body>
</html>`
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}
}
