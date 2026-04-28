package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const anthropicEndpoint = "https://api.anthropic.com/v1/messages"

type QueryRequest struct {
	Question string `json:"question"`
}

type QueryResponse struct {
	Answer   string `json:"answer"`
	FactsUsed int   `json:"facts_used"`
}

func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	if s.anthropicKey == "" {
		http.Error(w, "ANTHROPIC_API_KEY not configured", http.StatusServiceUnavailable)
		return
	}

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Question == "" {
		http.Error(w, "body must be {\"question\": \"...\"}", http.StatusBadRequest)
		return
	}

	facts, _ := s.store.List(r.Context(), "")

	// Build a compact context string from all stored facts.
	var sb strings.Builder
	for _, f := range facts {
		b, _ := json.Marshal(f.Value)
		sb.WriteString(fmt.Sprintf("[%s] %s\n", f.Key, string(b)))
	}

	answer, err := callAnthropic(r.Context(), s.anthropicKey, sb.String(), req.Question)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(QueryResponse{Answer: answer, FactsUsed: len(facts)})
}

func callAnthropic(ctx context.Context, apiKey, knowledgeBase, question string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model":      "claude-sonnet-4-6",
		"max_tokens": 1024,
		// System prompt is cached so repeated queries don't re-send the full KB.
		"system": []map[string]any{
			{
				"type": "text",
				"text": "You are a company knowledge assistant. Answer questions concisely and factually using only the provided company data. If the data doesn't contain enough information, say so.",
				"cache_control": map[string]string{"type": "ephemeral"},
			},
		},
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": fmt.Sprintf("Company knowledge base:\n%s\n\nQuestion: %s", knowledgeBase, question),
			},
		},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "prompt-caching-2024-07-31")
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("anthropic: %s", result.Error.Message)
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("empty response from anthropic")
	}
	return result.Content[0].Text, nil
}
