package gae

import (
	"errors"
	"fmt"
)

var (
	// ErrMultipleEntities is returned when a Datastore retrieval
	// finds more than 1 entity with the specified criteria.
	ErrMultipleEntities = errors.New("multiple entities retrieved when only 1 is expected")

	// ErrNilKey is returned when Key parameters are not expected to be nil.
	ErrNilKey = errors.New("key is nil")

	// ErrUnauth is returned when the request is not authenticated.
	ErrUnauth = errors.New("unauthenticated")
)

// DuplicateError is for when a duplicate value is present.
type DuplicateError struct {
	Msg  string
	Name string
}

// Error for DuplicateError returns
//
//	Duplicate value
//
// or
//
//	Duplicate value for <Name>
//
// if the `Name` field is set. The `Msg` field is appended to this message if
// set, separated by a hyphen.
func (this DuplicateError) Error() string {
	m := "Duplicate value"
	if this.Name != "" {
		m = "Duplicate value for " + this.Name
	}
	if this.Msg != "" {
		m += " - "
		m += this.Msg
	}
	return m
}

// IsDuplicateError checks if an error is the `DuplicateError` type.
func IsDuplicateError(e error) bool {
	_, ok := e.(DuplicateError)
	return ok
}

// InsufficientError is for when the quantity of an element is insufficient.
type InsufficientError struct {
	Msg  string
	Name string
}

// Error for InsufficientError returns
//
//	Insufficient value
//
// or
//
//	Insufficient value for <Name>
//
// if the `Name` field is set. The `Msg` field is appended to this message if
// set, separated by a hyphen.
func (this InsufficientError) Error() string {
	m := "Insufficient value"
	if this.Name != "" {
		m = "Insufficient value for " + this.Name
	}
	if this.Msg != "" {
		m += " - "
		m += this.Msg
	}
	return m
}

// IsInsufficientError checks if an error is the `InsufficientError` type.
func IsInsufficientError(e error) bool {
	_, ok := e.(InsufficientError)
	return ok
}

// InvalidError is a generic error for describing invalid conditions.
//
// An example is when the request parameter value is in an invalid format.
type InvalidError struct {
	Msg string
}

// Error for InvalidError returns a string in the format:
//
//	Invalid value (<msg>)
func (this InvalidError) Error() string {
	return "Invalid value (" + this.Msg + ")"
}

// IsInvalidError checks if an error is the `InvalidError` type.
func IsInvalidError(e error) bool {
	_, ok := e.(InvalidError)
	return ok
}

// JSONUnmarshalError is for unmarshalling errors when reading request JSON
// payload.
//
// The Msg field should provide an indication of where the error originated
// from. E.g. CreateSchoolAPI - request body
type JSONUnmarshalError struct {
	Msg string
	Err error
}

// Error for JSONUnmarshalError returns a string in the format:
//
//  Unable to parse JSON (<msg>) - <error string>
func (this JSONUnmarshalError) Error() string {
	m := "Unable to parse JSON"
	if this.Msg != "" {
		m += " (" + this.Msg + ")"
	}
	if this.Err != nil {
		m += " - " + this.Err.Error()
	}
	return m
}

// IsJSONUnmarshalError checks if an error is the `JSONUnmarshalError` type.
func IsJSONUnmarshalError(e error) bool {
	_, ok := e.(JSONUnmarshalError)
	return ok
}

// MismatchError is used in situations where multiple provided values do not match each other.
type MismatchError struct {
	Msg string
}

// Error for MismatchError returns a string in the format:
//
//	Mismatched values - <msg>
func (this MismatchError) Error() string {
	m := "Mismatched values"
	if this.Msg != "" {
		m += " - " + this.Msg
	}
	return m
}

// IsMismatchError checks if an error is the `MismatchError` type.
func IsMismatchError(e error) bool {
	_, ok := e.(MismatchError)
	return ok
}

// MissingError is for missing parameter values or a value is not provided
// when expected.
//
// The Msg field should provide a brief description of the parameter whose
// value is missing.
type MissingError struct {
	Msg string
}

// Error for MissingError returns:
//
//	Missing value
//
// or
//
//	Missing value - <msg>
//
// if the `Msg` field is set.
func (this MissingError) Error() string {
	m := "Missing value"
	if this.Msg != "" {
		m += " - " + this.Msg
	}
	return m
}

// IsMissingError checks if an error is the `MissingError` type.
func IsMissingError(e error) bool {
	_, ok := e.(MissingError)
	return ok
}

// NilError is for situations where variables are nil.
type NilError struct {
	Msg string
	Err error
}

// Error for NilError returns a string in the format:
//
//	Nil error (<msg>) - <error>
func (this NilError) Error() string {
	m := "Nil error"
	if this.Msg != "" {
		m += " (" + this.Msg + ")"
	}
	if this.Err != nil {
		m += " - " + this.Err.Error()
	}
	return m
}

// IsNilError checks if an error is the `NilError` type.
func IsNilError(e error) bool {
	_, ok := e.(NilError)
	return ok
}

// NotFoundError is a generic error for operations not being able to retrieve
// or find an entity.
//
// If the error is for a Datastore operation, the `Kind` field should be
// specified.
type NotFoundError struct {
	Kind string
	Err  error
}

// Error for NotFoundError returns a string in one of the following formats:
//
//	- Entity not found - <error string>
//	- '<kind>' entity not found - <error string>
func (this NotFoundError) Error() string {
	m := "entity not found"
	if this.Kind != "" {
		m = fmt.Sprintf("'%v' entity not found", this.Kind)
	}
	if this.Err != nil {
		m += " - " + this.Err.Error()
	}
	return m
}

// IsNotFoundError checks if an error is the `NotFoundError` type.
func IsNotFoundError(e error) bool {
	_, ok := e.(NotFoundError)
	return ok
}

// TypeError is for errors having to do with types and conversion.
type TypeError struct {
	Name  string
	Cause string
}

// Error returns a string in different formats depending on the properties
// specified.
//
// Basic (nothing specified): "type error"
//
// Name specified only: "type error on <name>
//
// Cause specified only: "type error - <cause>"
//
// Name and Cause specified: "type error on <name> - <cause>"
func (e TypeError) Error() string {
	m := "type error"
	if e.Name != "" {
		m += " on '%v'"
	}
	if e.Cause != "" {
		m += " - " + e.Cause
	}
	if e.Name != "" {
		return fmt.Sprintf(m, e.Name)
	}
	return m
}

// IsTypeError checks if an error is the "TypeError" type.
func IsTypeError(e error) bool {
	_, ok := e.(TypeError)
	return ok
}

// ValidityError is for errors in model validation.
type ValidityError struct {
	Msg string
}

// Error returns a string in the format:
//
//	validation error: <error string>
func (e ValidityError) Error() string {
	return "validation error - " + e.Msg
}

// IsValidityError checks if an error is the `ValidityError` type.
func IsValidityError(e error) bool {
	_, ok := e.(ValidityError)
	return ok
}
