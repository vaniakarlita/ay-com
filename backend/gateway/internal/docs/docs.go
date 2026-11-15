package docs

import (
	_ "embed"
	"net/http"
)

//go:embed openapi.yaml
var spec []byte

const page = `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>AY.com API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({ url: '/swagger/openapi.yaml', dom_id: '#ui' });
  </script>
</body>
</html>`

func Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /swagger", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(page))
	})
	mux.HandleFunc("GET /swagger/openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(spec)
	})
}