package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const geminiEndpoint = "https://generativelanguage.googleapis.com/v1beta/models"

// generationTimeout bounds one call. A seller waiting on a form should get an
// answer or a clear failure, not an open connection.
const generationTimeout = 30 * time.Second

// Gemini generates copy through the Google AI Studio API.
//
// This talks to the REST endpoint directly rather than pulling in a client
// library. The request is one JSON document and the response is another; a
// dependency would not carry its weight.
type Gemini struct {
	apiKey string
	model  string
	// baseURL is a field rather than a constant so tests can point it at a
	// local server and exercise the parsing without a network call.
	baseURL string
	http    *http.Client
}

func NewGemini(apiKey, model string) *Gemini {
	return &Gemini{
		apiKey:  apiKey,
		model:   model,
		baseURL: geminiEndpoint,
		http:    &http.Client{Timeout: generationTimeout},
	}
}

const copyPrompt = `You are writing a product listing for a small independent shop.

Look at the photograph and write:

- title: what the product is, 2 to 5 words, in title case. No brand names you
  cannot see, no marketing adjectives like "premium" or "luxury".
- description: 2 or 3 sentences on what it is, what it is made of and what it
  is for, based only on what the photograph actually shows. Plain, concrete,
  no superlatives. Do not invent dimensions, materials, prices or origin.
- altText: one sentence describing the photograph for someone who cannot see
  it. Describe the image, not the product's selling points.

If the photograph does not show a sellable product, say so plainly in the
description and leave the title generic.`

// responseSchema forces the model to answer with exactly these fields.
// Without it the reply is prose that has to be parsed, which fails the moment
// the model decides to add a preamble.
var responseSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"title":       map[string]any{"type": "STRING"},
		"description": map[string]any{"type": "STRING"},
		"altText":     map[string]any{"type": "STRING"},
	},
	"required": []string{"title", "description", "altText"},
}

func (g *Gemini) GenerateProductCopy(ctx context.Context, image Image) (ProductCopy, error) {
	body, err := json.Marshal(map[string]any{
		"contents": []any{map[string]any{
			"parts": []any{
				map[string]any{"inline_data": map[string]any{
					"mime_type": image.ContentType,
					"data":      base64.StdEncoding.EncodeToString(image.Data),
				}},
				map[string]any{"text": copyPrompt},
			},
		}},
		"generationConfig": map[string]any{
			"responseMimeType": "application/json",
			"responseSchema":   responseSchema,
		},
	})
	if err != nil {
		return ProductCopy{}, fmt.Errorf("build request: %w", err)
	}

	url := fmt.Sprintf("%s/%s:generateContent", g.baseURL, g.model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return ProductCopy{}, fmt.Errorf("build request: %w", err)
	}
	// The key goes in a header, not the query string: query strings end up in
	// proxy logs and browser history.
	req.Header.Set("x-goog-api-key", g.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.http.Do(req)
	if err != nil {
		return ProductCopy{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ProductCopy{}, fmt.Errorf("%w: gemini returned %s", ErrUnavailable, resp.Status)
	}

	var reply generateReply
	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		return ProductCopy{}, fmt.Errorf("%w: decoding reply: %v", ErrUnavailable, err)
	}

	// A model that answered with no candidates, or with a thinking block and
	// no text, failed to generate rather than failed to connect.
	text := reply.answer()
	if text == "" {
		return ProductCopy{}, fmt.Errorf("%w: the model returned no text", ErrUnavailable)
	}

	var generated copyPayload
	if err := json.Unmarshal([]byte(text), &generated); err != nil {
		return ProductCopy{}, fmt.Errorf("%w: the model did not follow the schema: %v", ErrUnavailable, err)
	}

	result := ProductCopy{
		Title:       strings.TrimSpace(generated.Title),
		Description: strings.TrimSpace(generated.Description),
		AltText:     strings.TrimSpace(generated.AltText),
	}
	if result.Title == "" || result.Description == "" {
		return ProductCopy{}, fmt.Errorf("%w: the model returned empty copy", ErrUnavailable)
	}
	return result, nil
}

// copyPayload is the JSON the model is constrained to produce by
// responseSchema.
type copyPayload struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	AltText     string `json:"altText"`
}

type generateReply struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// answer returns the last non-empty text part of the first candidate.
// Reasoning models put their thinking in earlier parts and the answer last.
func (r generateReply) answer() string {
	if len(r.Candidates) == 0 {
		return ""
	}
	parts := r.Candidates[0].Content.Parts
	for i := len(parts) - 1; i >= 0; i-- {
		if strings.TrimSpace(parts[i].Text) != "" {
			return parts[i].Text
		}
	}
	return ""
}
