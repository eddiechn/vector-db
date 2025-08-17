package internal

import (
	"context"
	"fmt"
	"os"

	"github.com/sashabaranov/go-openai"
)

// OpenAIEmbedder handles text embedding using OpenAI's API
type OpenAIEmbedder struct {
	client *openai.Client
	model  openai.EmbeddingModel
}

// NewOpenAIEmbedder creates a new OpenAI embedder
func NewOpenAIEmbedder() (*OpenAIEmbedder, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is required")
	}

	client := openai.NewClient(apiKey)

	return &OpenAIEmbedder{
		client: client,
		model:  openai.AdaEmbeddingV2, // text-embedding-ada-002, 1536 dimensions
	}, nil
}

// EmbedText converts text to vector embedding using OpenAI
func (e *OpenAIEmbedder) EmbedText(text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: e.model,
	}

	resp, err := e.client.CreateEmbeddings(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	// Convert float64 to float32
	embedding := make([]float32, len(resp.Data[0].Embedding))
	for i, val := range resp.Data[0].Embedding {
		embedding[i] = float32(val)
	}

	return embedding, nil
}

// GetDimensions returns the embedding dimensions for the model
func (e *OpenAIEmbedder) GetDimensions() int {
	// text-embedding-ada-002 has 1536 dimensions
	return 1536
}
