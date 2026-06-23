package converter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
)

const (
	MediaTypeJSON = "application/json"
	MediaTypeYAML = "application/yaml"
)

var ErrNoSpecification = errors.New("no specification supplied in either the url or request body. Try again?")

type Converter struct {
	HTTPClient *http.Client
}

func New() *Converter {
	return &Converter{
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Converter) ConvertContents(ctx context.Context, input []byte) (*openapi3.T, []string) {
	if len(strings.TrimSpace(string(input))) == 0 {
		return nil, []string{ErrNoSpecification.Error()}
	}

	doc, err := c.convertData(ctx, input, nil)
	if err != nil {
		return nil, []string{err.Error()}
	}
	return doc, nil
}

func (c *Converter) ConvertLocation(ctx context.Context, rawURL string) (*openapi3.T, []string) {
	if strings.TrimSpace(rawURL) == "" {
		return nil, []string{ErrNoSpecification.Error()}
	}

	location, err := url.Parse(rawURL)
	if err != nil {
		return nil, []string{err.Error()}
	}

	loader := c.loader(ctx)
	data, err := loader.ReadFromURIFunc(loader, location)
	if err != nil {
		return nil, []string{fmt.Sprintf("failed to process URL: %v", err)}
	}

	doc, err := c.convertData(ctx, data, location)
	if err != nil {
		return nil, []string{err.Error()}
	}
	return doc, nil
}

func (c *Converter) convertData(ctx context.Context, input []byte, location *url.URL) (*openapi3.T, error) {
	version, err := detectVersion(input)
	if err != nil {
		return nil, err
	}

	loader := c.loader(ctx)
	switch {
	case strings.HasPrefix(version, "3."):
		if location != nil {
			return loader.LoadFromDataWithPath(input, location)
		}
		return loader.LoadFromData(input)
	case version == "2.0":
		normalized, err := yamlToJSON(input)
		if err != nil {
			return nil, err
		}
		var doc2 openapi2.T
		if err := json.Unmarshal(normalized, &doc2); err != nil {
			return nil, err
		}
		return openapi2conv.ToV3WithLoader(&doc2, loader, location)
	default:
		return nil, fmt.Errorf("unsupported OpenAPI/Swagger version %q", version)
	}
}

func (c *Converter) loader(ctx context.Context) *openapi3.Loader {
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	loader := openapi3.NewLoader()
	loader.Context = ctx
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = openapi3.ReadFromURIs(openapi3.ReadFromHTTP(client), openapi3.ReadFromFile)
	return loader
}

func detectVersion(input []byte) (string, error) {
	var root map[string]any
	if err := yaml.Unmarshal(input, &root); err != nil {
		return "", err
	}

	if version, ok := root["openapi"].(string); ok {
		return version, nil
	}
	if version, ok := root["swagger"].(string); ok {
		return version, nil
	}
	return "", errors.New("input does not contain an openapi or swagger version")
}

func yamlToJSON(input []byte) ([]byte, error) {
	var decoded any
	if err := yaml.Unmarshal(input, &decoded); err != nil {
		return nil, err
	}
	return json.Marshal(decoded)
}

func Encode(doc *openapi3.T, mediaType string) ([]byte, string, error) {
	switch mediaType {
	case MediaTypeYAML:
		out, err := yaml.Marshal(doc)
		return out, MediaTypeYAML, err
	default:
		out, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return nil, "", err
		}
		out = append(out, '\n')
		return out, MediaTypeJSON, nil
	}
}

func PreferredMediaType(accept string) string {
	if strings.TrimSpace(accept) == "" {
		return MediaTypeJSON
	}

	isJSONOK := false
	isYAMLOK := false
	for _, part := range strings.Split(accept, ",") {
		mediaType := strings.ToLower(strings.TrimSpace(strings.SplitN(part, ";", 2)[0]))
		switch mediaType {
		case MediaTypeJSON:
			isJSONOK = true
		case MediaTypeYAML:
			isYAMLOK = true
		}
	}

	if isYAMLOK && !isJSONOK {
		return MediaTypeYAML
	}
	return MediaTypeJSON
}

func ReadAllLimited(r io.Reader, limit int64) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r, limit+1))
}
