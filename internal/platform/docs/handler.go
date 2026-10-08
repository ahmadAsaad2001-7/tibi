package docs

import (
	_ "embed"
	"net/http"
)

//go:embed openapi.yaml
var openAPISpec []byte

const scalarHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Tibi API Reference</title>
  <style>
    html, body { margin: 0; padding: 0; height: 100%; }
  </style>
</head>
<body>
  <div id="app"></div>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  <script>
    Scalar.createApiReference('#app', {
      url: '/openapi.yaml',
      theme: 'default',
      layout: 'modern',
      darkMode: false,
      hideModels: false,
      authentication: {
        preferredSecurityScheme: 'bearerAuth'
      }
    })
  </script>
</body>
</html>
`

// Mount registers Scalar UI at /docs and the OpenAPI document at /openapi.yaml.
func Mount(mux interface {
	Get(pattern string, handlerFn http.HandlerFunc)
}) {
	mux.Get("/openapi.yaml", Spec)
	mux.Get("/docs", UI)
	mux.Get("/docs/", UI)
}

// Spec serves the embedded OpenAPI 3.1 document.
func Spec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(openAPISpec)
}

// UI serves the Scalar API Reference page.
func UI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write([]byte(scalarHTML))
}
