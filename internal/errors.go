package internal

import "fmt"

// Custom error types for the vector database

// DimensionMismatchError is returned when vector dimensions don't match
type DimensionMismatchError struct {
	Expected int
	Actual   int
}

func (e *DimensionMismatchError) Error() string {
	return fmt.Sprintf("dimension mismatch: expected %d, got %d", e.Expected, e.Actual)
}

// VectorNotFoundError is returned when a vector with the given ID is not found
type VectorNotFoundError struct {
	ID string
}

func (e *VectorNotFoundError) Error() string {
	return fmt.Sprintf("vector not found: %s", e.ID)
}

// InvalidConfigError is returned when configuration is invalid
type InvalidConfigError struct {
	Field  string
	Value  interface{}
	Reason string
}

func (e *InvalidConfigError) Error() string {
	return fmt.Sprintf("invalid config for field %s (value: %v): %s", e.Field, e.Value, e.Reason)
}

// DatabaseError is a general database error
type DatabaseError struct {
	Operation string
	Cause     error
}

func (e *DatabaseError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("database error during %s: %v", e.Operation, e.Cause)
	}
	return fmt.Sprintf("database error during %s", e.Operation)
}

func (e *DatabaseError) Unwrap() error {
	return e.Cause
}

// IndexError is returned when there's an issue with the index
type IndexError struct {
	Type    string
	Message string
}

func (e *IndexError) Error() string {
	return fmt.Sprintf("index error (%s): %s", e.Type, e.Message)
}

// PersistenceError is returned when there's an issue with data persistence
type PersistenceError struct {
	Operation string
	Path      string
	Cause     error
}

func (e *PersistenceError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("persistence error during %s (path: %s): %v", e.Operation, e.Path, e.Cause)
	}
	return fmt.Sprintf("persistence error during %s (path: %s)", e.Operation, e.Path)
}

func (e *PersistenceError) Unwrap() error {
	return e.Cause
}
