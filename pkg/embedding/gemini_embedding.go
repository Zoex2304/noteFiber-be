package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type EmbeddingRequestContentPart struct {
	Text string `json:"text"`
}

type EmbeddingRequestContent struct {
	Parts []EmbeddingRequestContentPart `json:"parts"`
}

type EmbeddingRequest struct {
	Model    string                  `json:"model"`
	Content  EmbeddingRequestContent `json:"content"`
	TaskType string                  `json:"task_type,omitempty"`
}

type EmbeddingResponseEmbedding struct {
	Values []float32 `json:"values"`
}

type EmbeddingResponse struct {
	Embedding EmbeddingResponseEmbedding `json:"embedding"`
}

func GetGeminiEmbedding(
	apiKey string,
	text string,
	taskType string,
) (*EmbeddingResponse, error) {
	// CRITICAL FIX: Use text-embedding-004 with v1 endpoint for 3072 dimensions
	// v1beta endpoint returns 768 dimensions (legacy)
	// v1 endpoint returns 3072 dimensions (current)
	// Source: https://ai.google.dev/gemini-api/docs/embeddings
	modelName := "text-embedding-004"
	
	geminiReq := EmbeddingRequest{
		Model: modelName,
		Content: EmbeddingRequestContent{
			Parts: []EmbeddingRequestContentPart{
				{
					Text: text,
				},
			},
		},
		TaskType: taskType,
	}
	geminiReqJson, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, err
	}
	
	// CRITICAL FIX: Use v1 endpoint (not v1beta) for 3072 dimensions
	endpoint := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1/models/%s:embedContent",
		modelName,
	)
	log.Printf("[DEBUG] Embedding API endpoint: %s", endpoint)
	
	req, err := http.NewRequest(
		"POST",
		endpoint,
		bytes.NewBuffer(geminiReqJson),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-goog-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	resByte, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error from response, code %d, body %s", res.StatusCode, string(resByte))
	}

	var resEmbedding EmbeddingResponse
	err = json.Unmarshal(resByte, &resEmbedding)
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] Embedding generated: %d dimensions", len(resEmbedding.Embedding.Values))

	return &resEmbedding, nil
}