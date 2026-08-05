package main

import (
	"fmt"
	"strings"
)

func join(fields []string) string {
	return strings.Join(fields, ", ")
}

func SerializeForEmbedding(obj EmbeddableObject) string {
	switch v := obj.(type) {
	case *Model:
		return serializeModel(v)
	case *Component:
		return serializeComponent(v)
	case *Relationship:
		return serializeRelationship(v)
	case *Policy:
		return serializePolicy(v)
	default:
		return ""
	}
}

func serializeModel(m *Model) string {
	return fmt.Sprintf(
		"Model: %s | Display name: %s | Category: %s | Subcategory: %s | Status: %s | Description: %s",
		m.Name, m.DisplayName, m.Category.Name, m.SubCategory, m.Status, m.Description,
	)
}

func serializeComponent(c *Component) string {
	base := fmt.Sprintf(
		"Component: %s | Kind: %s | API version: %s | Group: %s | Scope: %s | Model: %s | Status: %s | Description: %s",
		c.DisplayName, c.Component.Kind, c.Component.Version, c.Component.Group, c.Component.Scope, c.Model.Name, c.Status, c.Description,
	)
	if len(c.Component.SpecFields) > 0 {
		base += fmt.Sprintf(" | Spec fields: %s", join(c.Component.SpecFields))
	}
	return base
}

func serializeRelationship(r *Relationship) string {
	return fmt.Sprintf(
		"Relationship: %s %s %s | Model: %s | Evaluation query: %s",
		r.Kind, r.Type, r.SubType, r.Model.Name, r.EvaluationQuery,
	)
}

func serializePolicy(p *Policy) string {
	return fmt.Sprintf(
		"Policy: %s | Display name: %s | Kind: %s | Subtype: %s | Model: %s | Description: %s",
		p.Name, p.DisplayName, p.Kind, p.SubType, p.Model.Name, p.Description,
	)
}
