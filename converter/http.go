package converter

import (
	"encoding/json"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
)

const MaxRequestBodyBytes int64 = 16 << 20

type Handler struct {
	Converter *Converter
}

func NewHandler(c *Converter) http.Handler {
	if c == nil {
		c = New()
	}
	return Handler{Converter: c}
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.convertByURL(w, r)
	case http.MethodPost:
		h.convertByContent(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func (h Handler) convertByContent(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	body, err := ReadAllLimited(r.Body, MaxRequestBodyBytes)
	if err != nil {
		writeError(w, http.StatusBadRequest, []string{err.Error()})
		return
	}
	if int64(len(body)) > MaxRequestBodyBytes {
		writeError(w, http.StatusRequestEntityTooLarge, []string{"request body is too large"})
		return
	}

	doc, messages := h.Converter.ConvertContents(r.Context(), body)
	h.writeResult(w, r, doc, messages)
}

func (h Handler) convertByURL(w http.ResponseWriter, r *http.Request) {
	doc, messages := h.Converter.ConvertLocation(r.Context(), r.URL.Query().Get("url"))
	h.writeResult(w, r, doc, messages)
}

func (h Handler) writeResult(w http.ResponseWriter, r *http.Request, doc *openapi3.T, messages []string) {
	if len(messages) != 0 {
		writeError(w, http.StatusBadRequest, messages)
		return
	}

	out, mediaType, err := Encode(doc, PreferredMediaType(r.Header.Get("Accept")))
	if err != nil {
		writeError(w, http.StatusInternalServerError, []string{err.Error()})
		return
	}
	w.Header().Set("Content-Type", mediaType)
	_, _ = w.Write(out)
}

func writeError(w http.ResponseWriter, status int, messages []string) {
	w.Header().Set("Content-Type", MediaTypeJSON)
	w.WriteHeader(status)
	if len(messages) == 1 && messages[0] == ErrNoSpecification.Error() {
		_, _ = w.Write([]byte(messages[0]))
		return
	}
	_ = json.NewEncoder(w).Encode(messages)
}
