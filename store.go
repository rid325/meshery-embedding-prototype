package main

import (
	"database/sql"
	"encoding/json"

	_ "modernc.org/sqlite"
)

// Entity is the flat representation stored in and retrieved from SQLite.
type Entity struct {
	ID             string
	Type           string
	RawJSON        []byte
	SerializedText string
	Embedding      []float32
}

// InitDB opens the SQLite file and creates the entities table if it doesn't exist.
func InitDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS entities (
		id             TEXT PRIMARY KEY,
		type           TEXT NOT NULL,
		raw_json       TEXT NOT NULL,
		serialized_text TEXT NOT NULL,
		embedding      TEXT NOT NULL
	)`)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// SaveEntity JSON-encodes the embedding and upserts the row.
func SaveEntity(db *sql.DB, id, entityType string, rawJSON []byte, serializedText string, embedding []float32) error {
	embJSON, err := json.Marshal(embedding)
	if err != nil {
		return err
	}
	_, err = db.Exec(
		`INSERT OR REPLACE INTO entities (id, type, raw_json, serialized_text, embedding) VALUES (?, ?, ?, ?, ?)`,
		id, entityType, string(rawJSON), serializedText, string(embJSON),
	)
	return err
}

// GetAllEntities fetches every row and deserializes the embedding back to []float32.
func GetAllEntities(db *sql.DB) ([]Entity, error) {
	rows, err := db.Query(`SELECT id, type, raw_json, serialized_text, embedding FROM entities`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []Entity
	for rows.Next() {
		var e Entity
		var embJSON string
		if err := rows.Scan(&e.ID, &e.Type, &e.RawJSON, &e.SerializedText, &embJSON); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(embJSON), &e.Embedding); err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}
