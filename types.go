package main

type Model struct {
	ID            string   `json:"id"`
	SchemaVersion string   `json:"schemaVersion"`
	Version       string   `json:"version"`
	Name          string   `json:"name"`
	DisplayName   string   `json:"displayName"`
	Description   string   `json:"description"`
	Status        string   `json:"status"`
	Category      Category `json:"category"`
	SubCategory   string   `json:"subCategory"`
}

type Component struct {
	ID            string         `json:"id"`
	SchemaVersion string         `json:"schemaVersion"`
	Version       string         `json:"version"`
	DisplayName   string         `json:"displayName"`
	Description   string         `json:"description"`
	Status        string         `json:"status"`
	Model         ComponentModel `json:"model"`
	Component     ComponentKind  `json:"component"`
}

type ComponentModel struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ComponentKind struct {
	Kind    string `json:"kind"`
	Version string `json:"version"`
}

type Relationship struct {
	ID              string           `json:"id"`
	SchemaVersion   string           `json:"schemaVersion"`
	Version         string           `json:"version"`
	Kind            string           `json:"kind"`
	Type            string           `json:"type"`
	SubType         string           `json:"subType"`
	Status          string           `json:"status"`
	EvaluationQuery string           `json:"evaluationQuery"`
	Model           RelationshipModel `json:"model"`
	Selectors       []interface{}    `json:"selectors"`
}

type RelationshipModel struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Version     string `json:"version"`
}

type Policy struct {
	SchemaVersion string      `json:"schemaVersion"`
	Name          string      `json:"name"`
	DisplayName   string      `json:"displayName"`
	Description   string      `json:"description"`
	Kind          string      `json:"kind"`
	SubType       string      `json:"subType"`
	Model         PolicyModel `json:"model"`
	Rego          string      `json:"rego"`
}

type PolicyModel struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type EmbeddableObject interface {
	GetID() string
	GetType() string
}

func (m *Model) GetID() string        { return m.ID }
func (m *Model) GetType() string      { return "model" }

func (c *Component) GetID() string    { return c.ID }
func (c *Component) GetType() string  { return "component" }

func (r *Relationship) GetID() string  { return r.ID }
func (r *Relationship) GetType() string { return "relationship" }

func (p *Policy) GetID() string       { return p.Name } // policies have no UUID field
func (p *Policy) GetType() string     { return "policy" }
