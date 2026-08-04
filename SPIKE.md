# Meshery Embedding Spike — Full Workflow Document

## What This Is

A proof-of-concept that answers one question:

> Can we take real Meshery schema objects (Model, Component, Relationship, Policy), embed them semantically, store them in SQLite, and retrieve the right ones for a natural language query like "Create a Kubernetes deployment exposed by a service"?

The answer is yes. This document explains everything that was built and why.

---

## File Structure

```
meshery-embedding-prototype/
├── main.go                          — orchestration: load → embed → store → query
├── types.go                         — Go structs for all four schema types + shared interface
├── serializer.go                    — converts structs to plain-text strings for embedding
├── embedder.go                      — calls Ollama API; falls back to deterministic fake
├── store.go                         — SQLite init, insert, and fetch
├── search.go                        — cosine similarity + top-k ranking
├── data/
│   ├── deployment_component.json    — real Kubernetes Deployment component (from catalog)
│   ├── service_component.json       — real Kubernetes Service component (from catalog)
│   ├── service_deployment_relationship.json  — real edge/network relationship (from catalog)
│   ├── policy_template.json         — edge_network_relationship policy
│   ├── model_template.json          — placeholder model template
│   ├── component_template.json      — placeholder (empty, not used in final run)
│   └── relationship_template.json   — placeholder (not used in final run)
└── embeddings.db                    — SQLite database (generated at runtime)
```

---

## The Full Pipeline

### Step 1 — Define the data model (`types.go`)

Four Go structs were defined, each mapping to a Meshery schema type:

- `Model` — maps to `models.meshery.io/v1beta1`
- `Component` — maps to `components.meshery.io/v1beta1`
- `Relationship` — maps to `relationships.meshery.io/v1beta1`
- `Policy` — maps to `policy.meshery.io/v1alpha1`

A shared `EmbeddableObject` interface with `GetID()` and `GetType()` methods was added so all four types can be handled uniformly in the pipeline loop.

Only semantically meaningful fields were kept. UUIDs, SVG blobs, timestamps, and registrant noise were excluded.

---

### Step 2 — Serialize to plain text (`serializer.go`)

Each struct is converted to a clean natural-language-style string before being sent to the embedder.

Examples:

```
Component: Service | Kind: Service | API version: v1 | Model: kubernetes | Status: enabled | Description: An abstract way to expose an application running on a set of Pods as a network service.

Relationship: edge non-binding network | Model: kubernetes | Evaluation query:

Policy: edge_network_relationship | Display name: Edge Network Relationship Policy | Kind: relationship | Subtype: network | Model: kubernetes | Description: Validates network relationships between Kubernetes components such as Service and Deployment.
```

The quality of these strings directly determines retrieval quality. Richer text = better embeddings = better search.

---

### Step 3 — Generate embeddings (`embedder.go`)

The `GenerateEmbedding(text string) ([]float32, error)` function:

1. **Primary path — Ollama:** POSTs to `http://localhost:11434/api/embeddings` with model `nomic-embed-text`. This produces a real 768-dimensional semantic vector.
2. **Fallback — deterministic fake:** If Ollama is unavailable, uses an FNV hash of the input text seeded through an LCG to produce a 768-float unit-normalized vector. Same input always gives same output. Lets the full pipeline run offline for development.

The timeout is set to 30 seconds to handle Ollama cold starts.

---

### Step 4 — Store in SQLite (`store.go`)

Uses `modernc.org/sqlite` (pure Go, no CGo required).

Table schema:
```sql
CREATE TABLE IF NOT EXISTS entities (
    id              TEXT PRIMARY KEY,
    type            TEXT NOT NULL,
    raw_json        TEXT NOT NULL,
    serialized_text TEXT NOT NULL,
    embedding       TEXT NOT NULL   -- JSON-encoded []float32
)
```

- `SaveEntity` marshals the `[]float32` embedding to a JSON string and upserts the row.
- `GetAllEntities` fetches all rows and unmarshals the embedding back to `[]float32`.

---

### Step 5 — Cosine similarity search (`search.go`)

`CosineSimilarity(a, b []float32) float32` computes:

```
similarity = dot(a, b) / (|a| * |b|)
```

Returns a value in [-1, 1]. Higher = more similar.

`Search(db, queryEmbedding, k)`:
1. Loads all entities from SQLite
2. Scores each against the query vector
3. Sorts descending by score
4. Returns top-k results

---

### Step 6 — Orchestration (`main.go`)

The main function wires everything together:

```
InitDB
  └─ load JSON files (os.ReadFile + json.Unmarshal)
       └─ SerializeForEmbedding
            └─ GenerateEmbedding (Ollama or fake)
                 └─ SaveEntity (SQLite upsert)

GenerateEmbedding(query)
  └─ Search (cosine similarity over all stored entities)
       └─ Print top-k results
```

Errors on individual objects use `log.Printf` + `continue` so one bad record doesn't kill the run.

---

## Real Data Extraction

The `data/` files were populated from the official Meshery Kubernetes catalog, extracted from a Docker image tar (`kubernetes.tar`) downloaded from meshery.io/catalog.

Structure inside the tar:
```
v1.32.0-alpha.3/v1.0.0/
├── model.json
├── components/
│   ├── Deployment.json
│   ├── Service.json
│   └── ... (79 components total)
└── relationships/
    └── ... (63 relationships total)
```

The Service→Deployment relationship was identified as `edge-ldz.json` — kind `edge`, type `non-binding`, subType `network`.

Fields extracted and saved match the struct shapes in `types.go`.

---

## Test Results

Query: `"Create a Kubernetes deployment exposed by a service"`

| Rank | Type | Name | Score | Why |
|------|------|------|-------|-----|
| 1 | component | Service | 0.726 | Directly matches "exposed by a service" |
| 2 | component | Deployment | 0.719 | Directly matches "kubernetes deployment" |
| 3 | policy | edge_network_relationship | 0.617 | Description mentions Service and Deployment |
| 4 | relationship | edge non-binding network | 0.542 | Connects Service to Deployment in kubernetes model |

All four relevant objects were retrieved. The ranking is semantically correct.

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `modernc.org/sqlite` | Pure-Go SQLite driver (no CGo) |
| `hash/fnv` | Stdlib — used in fake embedding fallback |
| `database/sql` | Stdlib — SQL interface |
| `encoding/json` | Stdlib — JSON marshal/unmarshal |
| `net/http` | Stdlib — Ollama HTTP client |

---

## How to Run

```bash
# First run — make sure Ollama is running with nomic-embed-text pulled
ollama pull nomic-embed-text

# Run the pipeline
go run .

# To re-ingest with fresh embeddings
rm embeddings.db && go run .
```

---

## What This Proves

1. Meshery schema objects can be serialized into meaningful text strings
2. Those strings produce useful semantic embeddings via `nomic-embed-text`
3. SQLite is sufficient for storing and querying embeddings at prototype scale
4. Cosine similarity over 768-dim vectors correctly retrieves relevant schema objects for natural language queries
5. The fake fallback keeps the pipeline runnable without Ollama, useful for CI or offline dev

---

## Design Decision: Model as Metadata, Not a Retrieval Target

The Kubernetes `Model` object was deliberately not ingested as a standalone embeddable entity. Here's why:

- Users query for things to act on — Components and Relationships. A Model record on its own is not actionable.
- The model name (`kubernetes`) is already embedded as context in every Component and Relationship via the `model.name` field in the serialized string. It influences rankings without being a result itself.
- Including Model objects in the same embedding space would pollute results for the common case (e.g. returning "Kubernetes model" when someone asks about a Deployment).

**Conclusion:** Model is a metadata/filter dimension. Every entity is tagged with its model name in the serialized text, which is sufficient. If model-level search is needed later (e.g. "what models are available?"), it should live in a separate index, not mixed with component/relationship retrieval.

---

## What's Next (Beyond the Spike)

- Load all 79 Kubernetes components and 63 relationships, not just Deployment and Service
- Add more Meshery models (Istio, AWS, etc.)
- Replace linear cosine scan with an approximate nearest-neighbour index (e.g. HNSW) for scale
- Expose as an HTTP API so other Meshery services can query it
- Evaluate retrieval quality systematically across a set of test queries
