# Meshery Embedding Spike

A working prototype that proves Meshery schema objects (Components, Relationships, Policies) can be serialized, embedded semantically, stored in SQLite, and retrieved accurately using natural language queries.

## How to Run

```bash
# Ensure Ollama is running with nomic-embed-text
ollama pull nomic-embed-text

# Run the full pipeline
rm -f embeddings.db
go run .
```

## What It Does

1. Loads real Kubernetes catalog data (Deployment, Service, their network relationship) and a KServe InferenceService parsed from a real CRD
2. Serializes each object into a clean plain-text string
3. Generates 768-dim embeddings via Ollama (`nomic-embed-text`) with a deterministic fallback if Ollama is unavailable
4. Stores everything in SQLite
5. Runs test queries and returns top-k results ranked by cosine similarity

## Test Results

| Query | Top Result | Score |
|-------|-----------|-------|
| "Create a Kubernetes deployment exposed by a service" | Service (component) | 0.726 |
| "Run a stateless application with replicated pods" | Deployment (component) | 0.678 |
| "Deploy a machine learning model for inference with a predictor" | InferenceService (CRD) | 0.752 |

See [SPIKE.md](./SPIKE.md) for the full breakdown — pipeline design, data sources, design decisions, and all results.

## Dependencies

- `modernc.org/sqlite` — pure-Go SQLite, no CGo needed
- Everything else is Go stdlib
