# khinkali

Khinkali reads an OpenAPI 2.0 or 3.x document from bytes or a URL, converts
Swagger/OpenAPI 2.0 to OpenAPI 3.0.3, and can serialize the result as JSON or
YAML.

## Features

- Convert Swagger/OpenAPI 2.0 documents to OpenAPI 3.0.3.
- Accept JSON or YAML input.
- Load documents directly from bytes or from a URL.
- Use as a normal Go package without running an HTTP server.
- Optionally run the bundled `GET /convert` and `POST /convert` HTTP API.
- Return JSON by default, or YAML when requested.

## Install

From another Go module:

```sh
go get github.com/myyrakle/khinkali
```

Inside this repository:

```sh
go mod tidy
go test ./...
```

## Use As A Library

Convert a local Swagger/OpenAPI file and write YAML output:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/myyrakle/khinkali/converter"
)

func main() {
	input, err := os.ReadFile("swagger.yaml")
	if err != nil {
		panic(err)
	}

	c := converter.New()
	doc, messages := c.ConvertContents(context.Background(), input)
	if len(messages) != 0 {
		panic(messages)
	}

	out, mediaType, err := converter.Encode(doc, converter.MediaTypeYAML)
	if err != nil {
		panic(err)
	}

	fmt.Println(mediaType)
	fmt.Print(string(out))
}
```

Convert a document from a URL:

```go
package main

import (
	"context"
	"fmt"

	"github.com/myyrakle/khinkali/converter"
)

func main() {
	doc, messages := converter.New().ConvertLocation(
		context.Background(),
		"https://example.com/swagger.yaml",
	)
	if len(messages) != 0 {
		panic(messages)
	}

	out, _, err := converter.Encode(doc, converter.MediaTypeJSON)
	if err != nil {
		panic(err)
	}

	fmt.Print(string(out))
}
```

## Library API

`converter.New() *converter.Converter`

Creates a converter with a default HTTP client timeout of 30 seconds.

`(*Converter).ConvertContents(ctx context.Context, input []byte) (*openapi3.T, []string)`

Converts a JSON or YAML document already loaded in memory. The returned
`messages` slice is empty on success. On failure, `doc` is `nil` and `messages`
contains parse or conversion errors.

`(*Converter).ConvertLocation(ctx context.Context, rawURL string) (*openapi3.T, []string)`

Loads a document from `rawURL`, then converts it. Relative external references
are resolved against the source URL when possible.

`converter.Encode(doc *openapi3.T, mediaType string) ([]byte, string, error)`

Serializes a converted document. Use `converter.MediaTypeJSON` for pretty JSON
or `converter.MediaTypeYAML` for YAML.

`converter.PreferredMediaType(accept string) string`

Chooses an output media type using the same rule as the original converter:
JSON is the default, and YAML is selected only when `application/yaml` is
acceptable and `application/json` is not.

## Custom HTTP Client

You can override the HTTP client used by `ConvertLocation`:

```go
c := converter.New()
c.HTTPClient = &http.Client{
	Timeout: 10 * time.Second,
}
```

## Run The HTTP Server

```sh
go run ./cmd/khinkali
```

The server listens on `:8080` by default. Set `ADDR` to override it:

```sh
ADDR=:8081 go run ./cmd/khinkali
```

## HTTP API

Convert a request body:

```sh
curl -X POST http://localhost:8080/convert \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/yaml' \
  --data-binary @swagger.json
```

Convert a remote document:

```sh
curl 'http://localhost:8080/convert?url=https://example.com/swagger.yaml'
```

Request JSON output:

```sh
curl -X POST http://localhost:8080/convert \
  -H 'Accept: application/json' \
  --data-binary @swagger.yaml
```

Request YAML output:

```sh
curl -X POST http://localhost:8080/convert \
  -H 'Accept: application/yaml' \
  --data-binary @swagger.yaml
```

## Error Handling

The library returns conversion errors as `[]string`, matching the shape of the
original Swagger Converter controller. The HTTP API returns `400 Bad Request`
for invalid, missing, or unsupported specifications.

The `POST /convert` endpoint limits request bodies to 16 MiB.
