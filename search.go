package main

import (
	"database/sql"
	"math"
	"sort"
)

// SearchResult pairs a retrieved entity with its similarity score.
type SearchResult struct {
	Entity Entity
	Score  float32
}

// CosineSimilarity returns the cosine similarity between two vectors.
// Returns 0 if either vector has zero magnitude.
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}
	var dot, magA, magB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		magA += float64(a[i]) * float64(a[i])
		magB += float64(b[i]) * float64(b[i])
	}
	magA = math.Sqrt(magA)
	magB = math.Sqrt(magB)
	if magA == 0 || magB == 0 {
		return 0
	}
	return float32(dot / (magA * magB))
}

// Search loads all entities, scores each against queryEmbedding, and returns the top-k by score.
func Search(db *sql.DB, queryEmbedding []float32, k int) ([]SearchResult, error) {
	entities, err := GetAllEntities(db)
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(entities))
	for _, e := range entities {
		score := CosineSimilarity(queryEmbedding, e.Embedding)
		results = append(results, SearchResult{Entity: e, Score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if k > len(results) {
		k = len(results)
	}
	return results[:k], nil
}
