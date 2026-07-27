package ai

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestGemini points a client at a stub server, so these tests exercise the
// parsing of a reply without a network call or an API key.
func newTestGemini(t *testing.T, handler http.HandlerFunc) *Gemini {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return &Gemini{
		apiKey:  "test-key",
		model:   "test-model",
		baseURL: server.URL,
		http:    server.Client(),
	}
}

func reply(t *testing.T, body string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}
}

func TestGeminiParsesCopy(t *testing.T) {
	g := newTestGemini(t, reply(t, `{"candidates":[{"content":{"parts":[
		{"text":"{\"title\":\" Speckled Mug \",\"description\":\"Stoneware, 300ml.\",\"altText\":\"A mug on a table.\"}"}
	]}}]}`))

	got, err := g.GenerateProductCopy(context.Background(), Image{ContentType: "image/png", Data: []byte("x")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Title != "Speckled Mug" {
		t.Errorf("title = %q, want %q (surrounding space should be trimmed)", got.Title, "Speckled Mug")
	}
	if got.AltText != "A mug on a table." {
		t.Errorf("altText = %q", got.AltText)
	}
}

// Reasoning models emit their thinking as earlier parts and the answer last.
// Reading parts[0] would pick up the thinking and fail to parse.
func TestGeminiReadsTheLastPart(t *testing.T) {
	g := newTestGemini(t, reply(t, `{"candidates":[{"content":{"parts":[
		{"text":"Let me look at the photograph..."},
		{"text":"{\"title\":\"Ridged Bowl\",\"description\":\"Matte glaze.\",\"altText\":\"A bowl.\"}"}
	]}}]}`))

	got, err := g.GenerateProductCopy(context.Background(), Image{ContentType: "image/png", Data: []byte("x")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Title != "Ridged Bowl" {
		t.Errorf("title = %q, want %q", got.Title, "Ridged Bowl")
	}
}

func TestGeminiFailuresAreUnavailable(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"http error", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		}},
		{"no candidates", reply(t, `{"candidates":[]}`)},
		{"not json", reply(t, `{"candidates":[{"content":{"parts":[{"text":"sorry, I cannot"}]}}]}`)},
		{"empty title", reply(t, `{"candidates":[{"content":{"parts":[
			{"text":"{\"title\":\"\",\"description\":\"d\",\"altText\":\"a\"}"}]}}]}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newTestGemini(t, tt.handler)

			_, err := g.GenerateProductCopy(context.Background(), Image{ContentType: "image/png", Data: []byte("x")})
			// Every failure has to be ErrUnavailable, because that is what
			// callers check to decide between "write it yourself" and a real
			// error. A bare error here would be reported as a broken save.
			if !errors.Is(err, ErrUnavailable) {
				t.Errorf("error = %v, want it to wrap ErrUnavailable", err)
			}
		})
	}
}

func TestDisabledIsUnavailable(t *testing.T) {
	_, err := Disabled{}.GenerateProductCopy(context.Background(), Image{})
	if !errors.Is(err, ErrUnavailable) {
		t.Errorf("error = %v, want ErrUnavailable", err)
	}
}
