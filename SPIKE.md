# Meshery Embedding Spike

## What This Is

A proof-of-concept that answers one question:

> Can we take real Meshery schema objects, embed them semantically, store them in SQLite, and retrieve the right ones for a natural language query?

The answer is yes. This document covers everything built, every decision made, and all test results.

---

## The Problem

Meshery has a large catalog of schema objects — Models, Components, Relationships, Policies. When a user says "Create a Kubernetes deployment exposed by a service", the system needs to know which specific objects to surface. Today that's done by keyword search or manual browsing. The goal of this spike is to prove that semantic embedding retrieval can do it better.

---

## File Structure

```
meshery-embedding-prototype/
├── main.go                                    — orchestration
├── types.go                                   — Go structs + shared interface
├── serializer.go                              — struct → plain text for embedding
├── embedder.go                                — Ollama client + fake fallback
├── store.go                                   — SQLite: init, insert, fetch
├── search.go                                  — cosine similarity + top-k
├── data/
│   ├── deployment_component.json              — real K8s Deployment (from catalog)
│   ├── service_component.json                 — real K8s Service (from catalog)
│   ├── service_deployment_relationship.json   — real edge/network relationship
│   ├── policy_template.json                   — edge_network_relationship policy
│   ├── inferenceservice_component.json        — dynamic: parsed from KServe CRD
│   ├── inferenceservice_crd.yaml              — raw KServe InferenceService CRD
│   ├── model_template.json                    — placeholder (not ingested)
│   ├── component_template.json                — placeholder (not ingested)
│   └── relationship_template.json             — placeholder (not ingested)
└── embeddings.db                              — SQLite database (generated at runtime)
```

---

## The Pipeline

```
JSON file (catalog or CRD-derived)
  └─ json.Unmarshal → Go struct
       └─ SerializeForEmbedding() → plain text string
            └─ GenerateEmbedding() → []float32 (768-dim)
                 └─ SaveEntity() → SQLite row
                                          ↓
                              query string
                                └─ GenerateEmbedding()
                                     └─ Search() → cosine similarity over all rows
                                          └─ top-k results printed
```

---

## Step-by-Step Breakdown

### 1. Data model — `types.go`

Four Go structs matching Meshery schema versions:

- `Model` — `models.meshery.io/v1beta1`
- `Component` — `components.meshery.io/v1beta1`
- `Relationship` — `relationships.meshery.io/v1beta1`
- `Policy` — `policy.meshery.io/v1alpha1`

A shared `EmbeddableObject` interface (`GetID()`, `GetType()`) lets all four types flow through the pipeline loop without type-switching in main.

`Component` has a `source` field (`"catalog"` or `"crd"`) to distinguish static catalog objects from dynamically parsed CRD objects. `ComponentKind` was extended with `group`, `scope`, and `specFields` to carry CRD-specific metadata.

---

### 2. Serialization — `serializer.go`

Each struct is converted to a clean natural-language string. This is the most important step — the quality of this string determines retrieval quality.

Examples of what gets serialized:

```
Component: Service | Kind: Service | API version: v1 | Group:  | Scope:  | Model: kubernetes | Status: enabled | Description: An abstract way to expose an application running on a set of Pods as a network service.

Component: InferenceService | Kind: InferenceService | API version: v1beta1 | Group: serving.kserve.io | Scope: Namespaced | Model: kserve | Status: enabled | Description: Deploys a machine learning inference service with optional predictor, transformer, and explainer components. | Spec fields: predictor, transformer, explainer, canary, canaryTrafficPercent

Relationship: edge non-binding network | Model: kubernetes | Evaluation query:

Policy: edge_network_relationship | Display name: Edge Network Relationship Policy | Kind: relationship | Subtype: network | Model: kubernetes | Description: Validates network relationships between Kubernetes components such as Service and Deployment.
```

SVG blobs, timestamps, and UUIDs are excluded — they add noise without semantic value.

---

### 3. Embedding — `embedder.go`

`GenerateEmbedding(text string) ([]float32, error)`:

- **Primary:** POST to `http://localhost:11434/api/embeddings` with model `nomic-embed-text`. Returns a real 768-dimensional semantic vector. Timeout: 30s.
- **Fallback:** if Ollama is unavailable, uses FNV hash of the input text as an LCG seed to produce a deterministic 768-float unit-normalized vector. Same input always returns the same vector. Keeps the pipeline runnable offline.

---

### 4. Storage — `store.go`

Uses `modernc.org/sqlite` — pure Go, no CGo required.

```sql
CREATE TABLE IF NOT EXISTS entities (
    id              TEXT PRIMARY KEY,
    type            TEXT NOT NULL,
    raw_json        TEXT NOT NULL,
    serialized_text TEXT NOT NULL,
    embedding       TEXT NOT NULL   -- JSON-encoded []float32
)
```

`SaveEntity` marshals `[]float32` to JSON string and upserts. `GetAllEntities` scans all rows and unmarshals embeddings back to `[]float32`.

---

### 5. Search — `search.go`

`CosineSimilarity(a, b []float32) float32` — standard dot product over magnitudes.

`Search(db, queryEmbedding, k)`:
1. Loads all entities
2. Scores each against the query vector
3. Sorts descending
4. Returns top-k

---

### 6. Orchestration — `main.go`

Loads five objects: Deployment, Service, Service→Deployment relationship, edge_network_relationship policy, InferenceService (CRD-derived). For each: serialize → embed → store. Then runs three test queries and prints results.

---

## Real Data Sources

### Static — Kubernetes catalog
Extracted from official Meshery Kubernetes catalog (`v1.32.0-alpha.3`), downloaded as a Docker image tar from meshery.io/catalog. Unpacked the image layer to get:

```
v1.32.0-alpha.3/v1.0.0/
├── components/   (79 components — used Deployment.json, Service.json)
└── relationships/ (63 relationships — used edge-ldz.json: Service→Deployment network edge)
```

### Dynamic — KServe CRD
Downloaded the real `InferenceService` CRD from the KServe GitHub repo (`serving.kserve.io_inferenceservices.yaml`). Parsed key fields:

| Field | Value |
|-------|-------|
| Kind | InferenceService |
| Group | serving.kserve.io |
| Version | v1beta1 |
| Scope | Namespaced |
| Spec fields | predictor, transformer, explainer, canary, canaryTrafficPercent |

These were serialized into a Component JSON (`inferenceservice_component.json`) using the same struct shape as catalog components, but with `source: "crd"`.

---

## Design Decisions

### Model as metadata, not a retrieval target

The Kubernetes Model object was not ingested as a standalone embeddable entity. Reasons:
- Users query for actionable things — Components and Relationships, not model records
- The model name (`kubernetes`) is already embedded as context in every Component and Relationship via `model.name` in the serialized string
- Including Model objects would pollute results with non-actionable entries

**Decision:** Model is a metadata/filter dimension. Every entity carries its model name in the serialized text. If model-level search is needed later, it belongs in a separate index.

### CRD fields to include in serialization

For CRD-sourced components, `specFields` (top-level keys from the CRD's `spec.properties`) are appended to the serialized string. This is what makes `InferenceService` match queries about "predictor" or "inference" — without it, the component would only match on its name and description.

### One table, not four

A single `entities` table with a `type` column covers all four schema types. Simpler schema, simpler queries. At prototype scale this is the right call. At production scale you might split or add indices.

---

## Test Results

All three queries ran against five ingested objects using real Ollama embeddings (`nomic-embed-text`).

---

### Query 1: "Create a Kubernetes deployment exposed by a service"

| Rank | Type | Name | Score |
|------|------|------|-------|
| 1 | component | Deployment | 0.711 |
| 2 | component | Service | 0.710 |
| 3 | policy | edge_network_relationship | 0.617 |
| 4 | component | InferenceService | 0.602 |

Deployment and Service nearly tied — both directly match the query. Policy correctly follows. InferenceService present but lower — not the target.

---

### Query 2: "Run a stateless application with replicated pods"

| Rank | Type | Name | Score |
|------|------|------|-------|
| 1 | component | Deployment | 0.678 |
| 2 | component | Service | 0.602 |
| 3 | component | InferenceService | 0.451 |
| 4 | relationship | edge non-binding network | 0.430 |

Deployment leads — "replicated application" in its description directly matches "replicated pods". Networking objects drop significantly. InferenceService stays out of the way.

---

### Query 3: "Deploy a machine learning model for inference with a predictor"

| Rank | Type | Name | Score |
|------|------|------|-------|
| 1 | component | InferenceService (CRD) | **0.752** |
| 2 | component | Deployment | 0.547 |
| 3 | relationship | edge non-binding network | 0.519 |
| 4 | policy | edge_network_relationship | 0.491 |

InferenceService ranked #1 by a clear margin — `predictor` appears both in its description and spec fields, and "machine learning inference" is a direct semantic match. Deployment is a reasonable #2 since it's still a deployment concept.

---

## What This Proves

1. Meshery schema objects can be serialized into semantically meaningful text
2. Those strings produce useful embeddings via `nomic-embed-text` (768-dim)
3. SQLite is sufficient for storing and scanning embeddings at prototype scale
4. Cosine similarity correctly retrieves relevant objects for natural language queries
5. Static catalog objects and dynamic CRD-derived objects coexist in the same pipeline without interference
6. Ranking shifts correctly based on query intent — networking queries surface Service/Deployment, ML queries surface InferenceService

---

## How to Run

```bash
# Make sure Ollama is running (it probably already is)
curl http://localhost:11434/api/tags

# Pull the embedding model if not already there
ollama pull nomic-embed-text

# Run the full pipeline
cd meshery-embedding-prototype
rm -f embeddings.db
go run .
```

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `modernc.org/sqlite` | Pure-Go SQLite driver |
| `hash/fnv` | Stdlib — fake embedding fallback |
| `database/sql` | Stdlib — SQL interface |
| `encoding/json` | Stdlib — marshal/unmarshal |
| `net/http` | Stdlib — Ollama HTTP client |

---

## What's Next

- Ingest all 79 Kubernetes components and 63 relationships, not just Deployment and Service
- Write a real CRD parser that reads `spec.properties` from YAML directly instead of hand-authoring the JSON
- Add more Meshery models (Istio, AWS, Prometheus)
- Replace linear cosine scan with approximate nearest-neighbour (HNSW) for scale
- Expose as an HTTP API for other Meshery services to query
- Evaluate retrieval quality systematically across a broader test query set
