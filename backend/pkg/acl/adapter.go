// Copyright 2025 Takhin Data, Inc.

package acl

// AuthorizerAdapter adapts the Authorizer to work with generic interfaces
type AuthorizerAdapter struct {
	*Authorizer
}

// NewAuthorizerAdapter creates a new AuthorizerAdapter
func NewAuthorizerAdapter(auth *Authorizer) *AuthorizerAdapter {
	return &AuthorizerAdapter{Authorizer: auth}
}

// Authorize checks if the principal is authorized to perform the operation
func (a *AuthorizerAdapter) Authorize(principal, host string, resourceType int8, resourceName string, operation int8) bool {
	return a.Authorizer.Authorize(principal, host, ResourceType(resourceType), resourceName, Operation(operation))
}

// AddACL adds a new ACL entry (accepts interface{})
func (a *AuthorizerAdapter) AddACL(entryInterface interface{}) error {
	entry, ok := entryInterface.(*Entry)
	if !ok {
		return nil // Silently ignore wrong types for interface compatibility
	}
	return a.Authorizer.AddACL(entry)
}

// DeleteACL removes ACL entries matching the filter (accepts interface{})
func (a *AuthorizerAdapter) DeleteACL(filterInterface interface{}) (int, error) {
	filter, ok := filterInterface.(*Filter)
	if !ok {
		return 0, nil // Silently ignore wrong types for interface compatibility
	}
	return a.Authorizer.DeleteACL(filter)
}

// ListACL returns ACL entries matching the filter (returns interface{})
func (a *AuthorizerAdapter) ListACL(filterInterface interface{}) []interface{} {
	filter, ok := filterInterface.(*Filter)
	if !ok {
		return []interface{}{} // Return empty for wrong types
	}
	entries := a.Authorizer.ListACL(filter)
	result := make([]interface{}, len(entries))
	for i, entry := range entries {
		result[i] = entry
	}
	return result
}
