# Meshery RAG Embedding Spike

This directory contains the spike (prototype) to prove that real Meshery registry objects can be serialized, embedded, stored locally in SQLite, searched semantically, and returned as a useful context bundle for the Meshery AI Adapter.

## Project Structure

```text
meshery-rag-spike/
  ├── data/
  │    ├── model_template.json           # Template JSON file for Model
  │    ├── component_template.json       # Template JSON file for Component
  │    ├── relationship_template.json    # Template JSON file for Relationship
  │    └── policy_template.json          # Template JSON file for Policy
  ├── main.go                            # Orchestrator and entrypoint
  ├── serializer.go                      # Serialization definitions & canonical string logic
  ├── store.go                           # SQLite database connection and operations
  ├── search.go                          # Cosine similarity logic and query execution
  └── README.md                          # This instruction and architecture document
```

---

## 🛠️ Step-by-Step Implementation Guide

Follow these steps to complete the spike programmatically.

### Step 1: Populate Sample Data
Copy the real sample JSON/YAML schemas from the parent `meshery/schemas` repository into the `data/` directory. Ensure they represent:
1. A **Model** (e.g., Kubernetes)
2. A **Component** (e.g., Deployment or Service)
3. A **Relationship** (e.g., Edge-binding or Service-to-Deployment linking)
4. A **Policy** (e.g., Registry evaluation policy)

---

### Step 2: Write the Serializer (`serializer.go`)
Define how nested objects are flattened into a single text representation (canonical representation) suitable for embedding model ingestion.

#### Recommendations:
* **Model Serialization**: Concatenate `name`, `version`, `category`, and nested descriptive attributes from `metadata`.
* **Component Serialization**: Concatenate the component's `name`, its parent `model` metadata, and keys/types inside the `configuration` schema.
* **Relationship Serialization**: Capture the `kind`, `type`, `subType`, and structural selectors (e.g., "From component X to To component Y").
* **Policy Serialization**: Include the policy name, rules, pattern definitions, and dynamic triggers.

---

### Step 3: Implement SQLite Storage (`store.go`)
Initialize a simple SQLite database schema.

#### Suggested Table Schema:
```sql
CREATE TABLE IF NOT EXISTS registry_embeddings (
    id TEXT PRIMARY KEY,
    entity_type TEXT NOT NULL,       -- 'model', 'component', 'relationship', 'policy'
    name TEXT NOT NULL,
    raw_json TEXT NOT NULL,          -- Full registry JSON structure
    serialized_text TEXT NOT NULL,   -- Canonical string representation
    embedding TEXT NOT NULL          -- Serialized array (e.g., JSON array of floats)
);
```

* **Driver**: Use a pure-Go SQLite driver like `modernc.org/sqlite` or the CGO-based `github.com/mattn/go-sqlite3`.

---

### Step 4: Implement Semantic Search (`search.go`)
Write a simple vector similarity search mechanism:
1. Parse the stored embedding JSON string back into a slice of `float32`.
2. Compute the **Cosine Similarity** between the query vector $A$ and each entity vector $B$:
   $$\text{Similarity}(A, B) = \frac{A \cdot B}{\|A\| \|B\|}$$
3. Order the registry entries in descending order of similarity score.
4. Return the top-k results.

---

### Step 5: Implement Main Pipeline (`main.go`)
Orchestrate the flow:
1. Parse JSON templates from `data/`.
2. Call `Serialize` for each entity type.
3. Generate embeddings:
   * **Ollama API**: Send a POST request to `http://localhost:11434/api/embeddings` using a model like `nomic-embed-text` or `all-minilm`.
   * **Fallback**: Write a deterministic mock embedding function (e.g., hashing or word count) to return a pseudo-vector if Ollama is not installed locally.
4. Save everything to SQLite.
5. Execute the test query: *"Create a Kubernetes deployment exposed by a service"*
6. Print the top results with their similarity scores.

---

## 🎯 What to Learn From the Spike

As you run the tests and examine query retrieval, document answers to the following questions to construct the sections of your LFX proposal:

1. **Which fields matter for Model serialization?**
2. **Which fields matter for Component serialization?**
3. **Which fields matter for Relationship serialization?**
4. **Which fields matter for Policy serialization?**
5. **Does relationship data need to be indexed separately?**
6. **What metadata filters are essential?**
7. **How should dynamic registry updates invalidate embeddings?**
8. **What should the context API return?**
