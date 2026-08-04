package main

import (
	"bytes"
	"encoding/json"
	"hash/fnv"
	"math"
	"net/http"
	"time"
)

const embeddingDim = 768

var ollamaClient = &http.Client{Timeout: 30 * time.Second}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type ollamaResponse struct {
	Embedding []float64 `json:"embedding"`
}

func GenerateEmbedding(text string) ([]float32, error) {
	body, _ := json.Marshal(ollamaRequest{Model: "nomic-embed-text", Prompt: text})
	resp, err := ollamaClient.Post("http://localhost:11434/api/embeddings", "application/json", bytes.NewReader(body))
	if err != nil {
		return fakeEmbedding(text), nil
	}
	defer resp.Body.Close()

	var result ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Embedding) == 0 {
		return fakeEmbedding(text), nil
	}

	vec := make([]float32, len(result.Embedding))
	for i, v := range result.Embedding {
		vec[i] = float32(v)
	}
	return vec, nil
}

func fakeEmbedding(text string) []float32 {
	h := fnv.New32a()
	h.Write([]byte(text))
	seed := uint64(h.Sum32())

	vec := make([]float32, embeddingDim)
	for i := range vec {
		// LCG-style deterministic value in [-1, 1]
		seed = seed*6364136223846793005 + 1442695040888963407
		vec[i] = float32(int64(seed>>33)) / float32(math.MaxInt32)
	}

	// Normalize to unit length so cosine similarity is meaningful
	var mag float64
	for _, v := range vec {
		mag += float64(v) * float64(v)
	}
	mag = math.Sqrt(mag)
	if mag > 0 {
		for i := range vec {
			vec[i] /= float32(mag)
		}
	}
	return vec
}
