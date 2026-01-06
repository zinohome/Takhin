// Copyright 2025 Takhin Data, Inc.

package schema

import (
	"encoding/json"
	"fmt"
)

// CompatibilityChecker checks schema compatibility
type CompatibilityChecker interface {
	CheckCompatibility(newSchema string, existingSchemas []string, mode CompatibilityMode) error
}

// SchemaValidator validates schema syntax
type SchemaValidator interface {
	Validate(schema string, schemaType SchemaType) error
}

// DefaultCompatibilityChecker implements basic compatibility checking
type DefaultCompatibilityChecker struct {
	validator SchemaValidator
}

// NewDefaultCompatibilityChecker creates a new compatibility checker
func NewDefaultCompatibilityChecker() *DefaultCompatibilityChecker {
	return &DefaultCompatibilityChecker{
		validator: &DefaultSchemaValidator{},
	}
}

// CheckCompatibility checks if a new schema is compatible with existing schemas
func (c *DefaultCompatibilityChecker) CheckCompatibility(newSchema string, existingSchemas []string, mode CompatibilityMode) error {
	if mode == CompatibilityNone {
		return nil
	}

	if len(existingSchemas) == 0 {
		return nil
	}

	switch mode {
	case CompatibilityBackward, CompatibilityBackwardTransit:
		return c.checkBackwardCompatibility(newSchema, existingSchemas, mode == CompatibilityBackwardTransit)
	case CompatibilityForward, CompatibilityForwardTransit:
		return c.checkForwardCompatibility(newSchema, existingSchemas, mode == CompatibilityForwardTransit)
	case CompatibilityFull, CompatibilityFullTransit:
		if err := c.checkBackwardCompatibility(newSchema, existingSchemas, mode == CompatibilityFullTransit); err != nil {
			return err
		}
		return c.checkForwardCompatibility(newSchema, existingSchemas, mode == CompatibilityFullTransit)
	default:
		return fmt.Errorf("unsupported compatibility mode: %s", mode)
	}
}

// checkBackwardCompatibility ensures new schema can read data written with old schemas
func (c *DefaultCompatibilityChecker) checkBackwardCompatibility(newSchema string, existingSchemas []string, transitive bool) error {
	schemasToCheck := existingSchemas
	if !transitive && len(existingSchemas) > 0 {
		schemasToCheck = []string{existingSchemas[len(existingSchemas)-1]}
	}

	var newFields map[string]interface{}
	if err := json.Unmarshal([]byte(newSchema), &newFields); err != nil {
		return fmt.Errorf("invalid new schema JSON: %w", err)
	}

	for _, oldSchema := range schemasToCheck {
		var oldFields map[string]interface{}
		if err := json.Unmarshal([]byte(oldSchema), &oldFields); err != nil {
			continue
		}

		if err := c.validateBackward(newFields, oldFields); err != nil {
			return NewSchemaError(ErrCodeIncompatibleSchema, fmt.Sprintf("backward incompatible: %v", err))
		}
	}

	return nil
}

// checkForwardCompatibility ensures old schemas can read data written with new schema
func (c *DefaultCompatibilityChecker) checkForwardCompatibility(newSchema string, existingSchemas []string, transitive bool) error {
	schemasToCheck := existingSchemas
	if !transitive && len(existingSchemas) > 0 {
		schemasToCheck = []string{existingSchemas[len(existingSchemas)-1]}
	}

	var newFields map[string]interface{}
	if err := json.Unmarshal([]byte(newSchema), &newFields); err != nil {
		return fmt.Errorf("invalid new schema JSON: %w", err)
	}

	for _, oldSchema := range schemasToCheck {
		var oldFields map[string]interface{}
		if err := json.Unmarshal([]byte(oldSchema), &oldFields); err != nil {
			continue
		}

		if err := c.validateForward(newFields, oldFields); err != nil {
			return NewSchemaError(ErrCodeIncompatibleSchema, fmt.Sprintf("forward incompatible: %v", err))
		}
	}

	return nil
}

// validateBackward checks if new schema can read old data
func (c *DefaultCompatibilityChecker) validateBackward(newFields, oldFields map[string]interface{}) error {
	oldFieldsMap := extractFields(oldFields)
	newFieldsMap := extractFields(newFields)

	for oldField := range oldFieldsMap {
		if _, exists := newFieldsMap[oldField]; !exists {
			return fmt.Errorf("field %s removed from schema", oldField)
		}
	}

	return nil
}

// validateForward checks if old schema can read new data
func (c *DefaultCompatibilityChecker) validateForward(newFields, oldFields map[string]interface{}) error {
	oldFieldsMap := extractFields(oldFields)
	newFieldsMap := extractFields(newFields)

	for newField, newMeta := range newFieldsMap {
		if _, exists := oldFieldsMap[newField]; !exists {
			if !hasDefault(newMeta) {
				return fmt.Errorf("new field %s without default value", newField)
			}
		}
	}

	return nil
}

// extractFields extracts field names from schema
func extractFields(schema map[string]interface{}) map[string]interface{} {
	fields := make(map[string]interface{})

	if fieldsArray, ok := schema["fields"].([]interface{}); ok {
		for _, field := range fieldsArray {
			if fieldMap, ok := field.(map[string]interface{}); ok {
				if name, ok := fieldMap["name"].(string); ok {
					fields[name] = fieldMap
				}
			}
		}
	}

	if props, ok := schema["properties"].(map[string]interface{}); ok {
		for name, prop := range props {
			fields[name] = prop
		}
	}

	return fields
}

// hasDefault checks if a field has a default value
func hasDefault(field interface{}) bool {
	if fieldMap, ok := field.(map[string]interface{}); ok {
		_, hasDefault := fieldMap["default"]
		return hasDefault
	}
	return false
}

// DefaultSchemaValidator implements basic schema validation
type DefaultSchemaValidator struct{}

// Validate validates schema syntax
func (v *DefaultSchemaValidator) Validate(schema string, schemaType SchemaType) error {
	switch schemaType {
	case SchemaTypeJSON:
		return v.validateJSON(schema)
	case SchemaTypeAvro:
		return v.validateAvro(schema)
	case SchemaTypeProtobuf:
		return v.validateProtobuf(schema)
	default:
		return NewSchemaError(ErrCodeInvalidSchema, fmt.Sprintf("unsupported schema type: %s", schemaType))
	}
}

// validateJSON validates JSON schema
func (v *DefaultSchemaValidator) validateJSON(schema string) error {
	var data interface{}
	if err := json.Unmarshal([]byte(schema), &data); err != nil {
		return NewSchemaError(ErrCodeInvalidSchema, fmt.Sprintf("invalid JSON schema: %v", err))
	}
	return nil
}

// validateAvro validates Avro schema
func (v *DefaultSchemaValidator) validateAvro(schema string) error {
	var avroSchema map[string]interface{}
	if err := json.Unmarshal([]byte(schema), &avroSchema); err != nil {
		return NewSchemaError(ErrCodeInvalidSchema, fmt.Sprintf("invalid Avro schema: %v", err))
	}

	if _, ok := avroSchema["type"]; !ok {
		return NewSchemaError(ErrCodeInvalidSchema, "Avro schema must have 'type' field")
	}

	return nil
}

// validateProtobuf validates Protobuf schema
func (v *DefaultSchemaValidator) validateProtobuf(schema string) error {
	if len(schema) == 0 {
		return NewSchemaError(ErrCodeInvalidSchema, "empty Protobuf schema")
	}
	return nil
}
