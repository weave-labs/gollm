package llm

import (
	"github.com/invopop/jsonschema"
)

// StructuredResponseReflector creates a jsonschema.Reflector that can handle structured responses.
func StructuredResponseReflector() *jsonschema.Reflector {
	r := &jsonschema.Reflector{
		DoNotReference:            true,
		AllowAdditionalProperties: false,
	}

	return r
}

// StripSchemaIDs Sanitize root for Gemini response_schema (no $schema/$id, defensive clears)
func StripSchemaIDs(s *jsonschema.Schema) *jsonschema.Schema {
	if s == nil {
		return s
	}
	s.Version = ""
	s.ID = ""
	s.Definitions = nil
	s.Ref = ""
	return s
}

// stripMetaDeep recursively removes all JSON-Schema meta properties from the schema.
func stripMetaDeep(s *jsonschema.Schema) {
	if s == nil {
		return
	}

	// Nuke JSON-Schema meta at this node
	s.Version = ""      // drops "$schema"
	s.ID = ""           // drops "$id"
	s.Ref = ""          // defensive
	s.Definitions = nil // no "$defs"

	// Objects: iterate ordered properties
	if s.Properties != nil {
		for p := s.Properties.Oldest(); p != nil; p = p.Next() {
			stripMetaDeep(p.Value)
		}
	}

	// Arrays
	if s.Items != nil {
		stripMetaDeep(s.Items)
	}
	for _, it := range s.PrefixItems {
		stripMetaDeep(it)
	}

	// Combinators
	for _, sub := range s.AllOf {
		stripMetaDeep(sub)
	}
	for _, sub := range s.AnyOf {
		stripMetaDeep(sub)
	}
	for _, sub := range s.OneOf {
		stripMetaDeep(sub)
	}
	if s.Not != nil {
		stripMetaDeep(s.Not)
	}

	// Conditionals
	if s.If != nil {
		stripMetaDeep(s.If)
	}
	if s.Then != nil {
		stripMetaDeep(s.Then)
	}
	if s.Else != nil {
		stripMetaDeep(s.Else)
	}

	// Other nested schemas
	if s.AdditionalProperties != nil {
		stripMetaDeep(s.AdditionalProperties)
	}
	if s.PropertyNames != nil {
		stripMetaDeep(s.PropertyNames)
	}
	if s.Contains != nil {
		stripMetaDeep(s.Contains)
	}
	for _, v := range s.DependentSchemas {
		stripMetaDeep(v)
	}
	for _, v := range s.PatternProperties {
		stripMetaDeep(v)
	}
}
