package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func main() {

	db, err := InitDB("embeddings.db")
	if err != nil {
		log.Fatal("failed to init db:", err)
	}
	defer db.Close()


	type entry struct {
		obj     EmbeddableObject
		rawJSON []byte
	}

	// helper to load and unmarshal a JSON file into an EmbeddableObject
	loadComponent := func(path string) (Component, []byte) {
		b, err := os.ReadFile(path)
		if err != nil {
			log.Fatal(err)
		}
		var c Component
		if err := json.Unmarshal(b, &c); err != nil {
			log.Fatal(err)
		}
		return c, b
	}
	loadRelationship := func(path string) (Relationship, []byte) {
		b, err := os.ReadFile(path)
		if err != nil {
			log.Fatal(err)
		}
		var r Relationship
		if err := json.Unmarshal(b, &r); err != nil {
			log.Fatal(err)
		}
		return r, b
	}

	var policy Policy
	policyBytes, err := os.ReadFile("data/policy_template.json")
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(policyBytes, &policy); err != nil {
		log.Fatal(err)
	}

	deployment, deploymentBytes := loadComponent("data/deployment_component.json")
	service, serviceBytes := loadComponent("data/service_component.json")
	rel, relBytes := loadRelationship("data/service_deployment_relationship.json")

	entries := []entry{
		{&deployment, deploymentBytes},
		{&service, serviceBytes},
		{&rel, relBytes},
		{&policy, policyBytes},
	}

	for _, e := range entries {

		text := SerializeForEmbedding(e.obj)

		embedding, err := GenerateEmbedding(text)
		if err != nil {
			log.Printf("embedding failed for %s %s: %v", e.obj.GetType(), e.obj.GetID(), err)
			continue
		}

		if err := SaveEntity(db, e.obj.GetID(), e.obj.GetType(), e.rawJSON, text, embedding); err != nil {
			log.Printf("save failed for %s %s: %v", e.obj.GetType(), e.obj.GetID(), err)
		}
	}

	fmt.Println("Ingestion complete.")

	queries := []string{
		"Create a Kubernetes deployment exposed by a service",
		"Run a stateless application with replicated pods",
	}

	for _, query := range queries {
		fmt.Printf("\n=== Query: %q ===\n", query)
		queryEmbedding, err := GenerateEmbedding(query)
		if err != nil {
			log.Printf("query embedding failed: %v", err)
			continue
		}
		results, err := Search(db, queryEmbedding, 4)
		if err != nil {
			log.Printf("search failed: %v", err)
			continue
		}
		for i, r := range results {
			fmt.Printf("%d. [%s] %s (score: %.4f)\n   %s\n", i+1, r.Entity.Type, r.Entity.ID, r.Score, r.Entity.SerializedText)
		}
	}
}
