package converter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const swagger2Petstore = `{
  "swagger": "2.0",
  "info": {"title": "Petstore", "version": "1.0.0"},
  "host": "example.com",
  "basePath": "/api",
  "schemes": ["https"],
  "paths": {
    "/pets": {
      "get": {
        "operationId": "listPets",
        "produces": ["application/json"],
        "responses": {
          "200": {
            "description": "ok",
            "schema": {
              "type": "array",
              "items": {"$ref": "#/definitions/Pet"}
            }
          }
        }
      }
    }
  },
  "definitions": {
    "Pet": {
      "type": "object",
      "required": ["id"],
      "properties": {
        "id": {"type": "integer", "format": "int64"},
        "name": {"type": "string"}
      }
    }
  }
}`

const openapi3YAML = `
openapi: 3.0.3
info:
  title: Already v3
  version: 1.0.0
paths:
  /ping:
    get:
      responses:
        "204":
          description: no content
`

func TestConvertContentsConvertsSwagger2ToOpenAPI3(t *testing.T) {
	doc, messages := New().ConvertContents(context.Background(), []byte(swagger2Petstore))
	if len(messages) != 0 {
		t.Fatalf("unexpected messages: %v", messages)
	}
	if doc.OpenAPI != "3.0.3" {
		t.Fatalf("OpenAPI version = %q, want 3.0.3", doc.OpenAPI)
	}
	if doc.Components.Schemas["Pet"] == nil {
		t.Fatal("converted document is missing Pet schema")
	}
	if got := doc.Servers[0].URL; got != "https://example.com/api" {
		t.Fatalf("server URL = %q", got)
	}
}

func TestConvertContentsKeepsOpenAPI3(t *testing.T) {
	doc, messages := New().ConvertContents(context.Background(), []byte(openapi3YAML))
	if len(messages) != 0 {
		t.Fatalf("unexpected messages: %v", messages)
	}
	if doc.Info.Title != "Already v3" {
		t.Fatalf("title = %q", doc.Info.Title)
	}
}

func TestPreferredMediaTypeMatchesJavaController(t *testing.T) {
	tests := map[string]string{
		"":                                   MediaTypeJSON,
		"application/json":                   MediaTypeJSON,
		"application/yaml":                   MediaTypeYAML,
		"application/yaml, application/json": MediaTypeJSON,
		"text/yaml, */*":                     MediaTypeJSON,
		"application/yaml; q=0.9":            MediaTypeYAML,
	}

	for accept, want := range tests {
		if got := PreferredMediaType(accept); got != want {
			t.Fatalf("PreferredMediaType(%q) = %q, want %q", accept, got, want)
		}
	}
}

func TestHandlerConvertByContent(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/convert", strings.NewReader(swagger2Petstore))
	req.Header.Set("Accept", MediaTypeYAML)
	rec := httptest.NewRecorder()

	NewHandler(New()).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != MediaTypeYAML {
		t.Fatalf("Content-Type = %q", got)
	}
	if !strings.Contains(rec.Body.String(), "openapi: 3.0.3") {
		t.Fatalf("response does not look like OpenAPI YAML: %s", rec.Body.String())
	}
}

func TestHandlerMissingSpecification(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/convert", strings.NewReader(""))
	rec := httptest.NewRecorder()

	NewHandler(New()).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != ErrNoSpecification.Error() {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestHandlerConvertByURL(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", MediaTypeJSON)
		_, _ = w.Write([]byte(swagger2Petstore))
	}))
	defer upstream.Close()

	req := httptest.NewRequest(http.MethodGet, "/convert?url="+upstream.URL, nil)
	rec := httptest.NewRecorder()

	NewHandler(New()).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if body["openapi"] != "3.0.3" {
		t.Fatalf("openapi = %v", body["openapi"])
	}
}
