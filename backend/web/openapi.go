package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"url-shortener/web/docs"
)

var errEmptyDocument = errors.New("openapi document is empty")

// MountOpenAPI publishes the existing create API. It does not register the redirect route.
func MountOpenAPI(r chi.Router, baseURL string) {
	r.Get("/v3/api-docs", func(w http.ResponseWriter, _ *http.Request) {
		payload, err := openAPIDocument(baseURL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(raw)
	})
	r.Get("/v3/api-docs.yaml", func(w http.ResponseWriter, _ *http.Request) {
		payload, err := openAPIDocument(baseURL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(toYAML(payload)))
	})
	r.Get("/swagger-ui.html", swaggerUIPage)
	r.Get("/swagger-ui/*", httpSwagger.Handler(httpSwagger.URL("/v3/api-docs")))
}

func swaggerUIPage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>URL Shortener API 0.1.0</title>
  <link rel="stylesheet" href="/swagger-ui/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="/swagger-ui/swagger-ui-bundle.js"></script>
  <script src="/swagger-ui/swagger-ui-standalone-preset.js"></script>
  <script>
    window.onload = function () {
      window.ui = SwaggerUIBundle({
        url: "/v3/api-docs",
        dom_id: "#swagger-ui",
        deepLinking: true,
        docExpansion: "list",
        presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
        layout: "StandaloneLayout"
      });
    };
  </script>
</body>
</html>`))
}

func openAPIDocument(baseURL string) (map[string]any, error) {
	raw := docs.SwaggerInfo.ReadDoc()
	if strings.TrimSpace(raw) == "" {
		return nil, errEmptyDocument
	}
	var spec map[string]any
	if err := json.Unmarshal([]byte(raw), &spec); err != nil {
		return nil, err
	}
	definitions, _ := spec["definitions"].(map[string]any)
	rewriteRefs(definitions)
	markNullableShortCode(definitions)
	paths, _ := spec["paths"].(map[string]any)
	oasPaths := map[string]any{}
	for path, item := range paths {
		if !strings.HasPrefix(path, "/api/v1/") {
			continue
		}
		oasPaths[path] = convertPathItem(item)
	}
	info, _ := spec["info"].(map[string]any)
	if info == nil {
		info = map[string]any{}
	}
	delete(info, "contact")
	if info["title"] == nil {
		info["title"] = docs.SwaggerInfo.Title
	}
	if info["version"] == nil {
		info["version"] = docs.SwaggerInfo.Version
	}
	return map[string]any{
		"openapi": "3.0.3",
		"info":    info,
		"servers": []any{map[string]any{"url": baseURL}},
		"paths":   oasPaths,
		"components": map[string]any{
			"schemas": definitions,
		},
	}, nil
}

func convertPathItem(item any) map[string]any {
	ops, _ := item.(map[string]any)
	out := map[string]any{}
	for method, op := range ops {
		body, _ := op.(map[string]any)
		if body == nil {
			continue
		}
		out[method] = convertOperation(method, body)
	}
	return out
}

func convertOperation(method string, op map[string]any) map[string]any {
	out := map[string]any{}
	for _, key := range []string{"tags", "summary", "description", "operationId"} {
		if value, ok := op[key]; ok {
			out[key] = value
		}
	}
	if method == "post" && out["operationId"] == nil {
		out["operationId"] = "shorten"
	}
	if params, ok := op["parameters"].([]any); ok {
		for _, param := range params {
			field, _ := param.(map[string]any)
			if field == nil || field["in"] != "body" {
				continue
			}
			body := map[string]any{
				"content": map[string]any{
					"application/json": map[string]any{"schema": rewriteRefValue(field["schema"])},
				},
			}
			if field["required"] == true {
				body["required"] = true
			}
			out["requestBody"] = body
		}
	}
	if responses, ok := op["responses"].(map[string]any); ok {
		converted := map[string]any{}
		for code, response := range responses {
			field, _ := response.(map[string]any)
			if field == nil {
				continue
			}
			item := map[string]any{"description": field["description"]}
			if schema, ok := field["schema"]; ok {
				item["content"] = map[string]any{
					"application/json": map[string]any{"schema": rewriteRefValue(schema)},
				}
			}
			converted[code] = item
		}
		out["responses"] = converted
	}
	return out
}

func rewriteRefs(value any) {
	switch typed := value.(type) {
	case map[string]any:
		if ref, ok := typed["$ref"].(string); ok {
			typed["$ref"] = strings.ReplaceAll(ref, "#/definitions/", "#/components/schemas/")
		}
		for _, child := range typed {
			rewriteRefs(child)
		}
	case []any:
		for _, child := range typed {
			rewriteRefs(child)
		}
	}
}

func rewriteRefValue(value any) any {
	rewriteRefs(value)
	return value
}

func markNullableShortCode(definitions map[string]any) {
	for _, schema := range definitions {
		field, _ := schema.(map[string]any)
		if field == nil {
			continue
		}
		props, _ := field["properties"].(map[string]any)
		if props == nil {
			continue
		}
		shortCode, _ := props["shortCode"].(map[string]any)
		if shortCode != nil {
			shortCode["nullable"] = true
		}
	}
}

func toYAML(value any) string {
	var b strings.Builder
	writeYAML(&b, value, 0)
	return b.String()
}

func writeYAML(b *strings.Builder, value any, indent int) {
	switch typed := value.(type) {
	case map[string]any:
		if len(typed) == 0 {
			b.WriteString("{}\n")
			return
		}
		for _, key := range orderedKeys(typed) {
			b.WriteString(strings.Repeat("  ", indent))
			b.WriteString(yamlKey(key))
			b.WriteString(":")
			child := typed[key]
			if isScalar(child) {
				b.WriteString(" ")
				writeYAMLScalar(b, child)
				b.WriteString("\n")
				continue
			}
			b.WriteString("\n")
			writeYAML(b, child, indent+1)
		}
	case []any:
		if len(typed) == 0 {
			b.WriteString(strings.Repeat("  ", indent))
			b.WriteString("[]\n")
			return
		}
		for _, child := range typed {
			b.WriteString(strings.Repeat("  ", indent))
			b.WriteString("-")
			if isScalar(child) {
				b.WriteString(" ")
				writeYAMLScalar(b, child)
				b.WriteString("\n")
				continue
			}
			b.WriteString("\n")
			writeYAML(b, child, indent+1)
		}
	default:
		writeYAMLScalar(b, value)
		b.WriteString("\n")
	}
}

func orderedKeys(value map[string]any) []string {
	preferred := []string{"openapi", "info", "servers", "paths", "components", "schemas"}
	seen := map[string]bool{}
	var keys []string
	for _, key := range preferred {
		if _, ok := value[key]; ok {
			keys = append(keys, key)
			seen[key] = true
		}
	}
	var rest []string
	for key := range value {
		if !seen[key] {
			rest = append(rest, key)
		}
	}
	sort.Strings(rest)
	return append(keys, rest...)
}

func yamlKey(key string) string {
	if strings.ContainsAny(key, ": #{}[]&*!|>%@`'\"") || key == "" {
		return strconv.Quote(key)
	}
	return key
}

func isScalar(value any) bool {
	switch value.(type) {
	case map[string]any, []any:
		return false
	default:
		return true
	}
}

func writeYAMLScalar(b *strings.Builder, value any) {
	switch typed := value.(type) {
	case nil:
		b.WriteString("null")
	case string:
		b.WriteString(strconv.Quote(typed))
	case bool:
		if typed {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case float64:
		if typed == float64(int64(typed)) {
			b.WriteString(strconv.FormatInt(int64(typed), 10))
			return
		}
		b.WriteString(strconv.FormatFloat(typed, 'f', -1, 64))
	default:
		b.WriteString(strconv.Quote(strings.TrimSpace(strings.ReplaceAll(stringify(typed), "\n", " "))))
	}
}

func stringify(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(raw)
}
