package neural

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	BaseURL string
	Model   string
}

func NewClient(url, model string) *Client {
	if url == "" {
		url = "http://localhost:11434"
	}
	if model == "" {
		model = "llama3"
	} // Default model
	return &Client{BaseURL: url, Model: model}
}

type CompletionRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type CompletionResponse struct {
	Response string `json:"response"`
}

// Low-level generate
func (c *Client) Generate(prompt string) (string, error) {
	reqBody := CompletionRequest{
		Model:  c.Model,
		Prompt: prompt,
		Stream: false,
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := http.Post(c.BaseURL+"/api/generate", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("ollama connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("ollama error: %s", resp.Status)
	}

	body, _ := io.ReadAll(resp.Body)
	var result CompletionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.Response, nil
}

// Skills

func (c *Client) Summarize(content string) (string, error) {
	prompt := fmt.Sprintf(`Task: Summarize the following note content in less than 500 chars.
Context: Zettelkasten note.
Content:
%s
Summary:`, content)
	return c.Generate(prompt)
}

func (c *Client) SuggestCues(content string) (string, error) {
	prompt := fmt.Sprintf(`Task: Generate 3 active recall questions based on the content.
Constraint: Each question MUST end with a question mark '?'. Max 120 chars each.
Content:
%s
Questions:`, content)
	return c.Generate(prompt)
}

func (c *Client) Draft(topic string) (string, error) {
	prompt := fmt.Sprintf(`Task: Generate a draft note about "%s".
Format: Strict Markdown.
Structure:
# Title
Fecha: YYYY-MM-DD
Tipo: idea

## Notas
(Content)

## Cues
- Question?

## Resumen
(Summary)

## Enlaces
- [[related]]
`, topic)
	return c.Generate(prompt)
}
