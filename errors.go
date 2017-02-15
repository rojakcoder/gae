package gae

import "fmt"

// DuplicateError is for when a duplicate value is present.
type DuplicateError struct {
	Msg  string
	Name string
}

// Error for DuplicateError returns
//
//	Duplicate value for <Name>
//
// if the `Name` field is set. The `Msg` field is appended to this message if
// set, separated by a hyphen.
func (this DuplicateError) Error() string {
	m := ""
	if this.Name != "" {
		m = "Duplicate value for " + this.Name
	}
	if this.Msg != "" {
		if len(m) > 0 {
			m += " - "
		}
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
//	Insufficient value for <Name>
//
// if the `Name` field is set. The `Msg` field is appended to this message if
// set, separated by a hyphen.
func (this InsufficientError) Error() string {
	m := ""
	if this.Name != "" {
		m = "Insufficient value for " + this.Name
	}
	if this.Msg != "" {
		if len(m) > 0 {
			m += " - "
		}
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
//	Invalid value: <msg>
func (this InvalidError) Error() string {
	return "Invalid value: " + this.Msg
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
//  unable to parse JSON (<msg>): <error string>
func (this JSONUnmarshalError) Error() string {
	m := "Unable to parse JSON"
	if this.Msg != "" {
		m += " (" + this.Msg + ")"
	}
	if this.Err != nil {
		m += ": " + this.Err.Error()
	}
	return m
}

func IsJSONUnmarshalError(e error) bool {
	_, ok := e.(JSONUnmarshalError)
	return ok
}

type MismatchError struct {
	Msg string
}

func (this MismatchError) Error() string {
	m := "Mismatched values"
	if this.Msg != "" {
		m += ": " + this.Msg
	}
	return m
}

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

// Error for MissingError returns a string in the format:
//	missing value: <msg>
func (this MissingError) Error() string {
	return "Missing value: " + this.Msg
}

// IsMissingError checks if an error is the `MissingError` type.
func IsMissingError(e error) bool {
	_, ok := e.(MissingError)
	return ok
}

type NilError struct {
	Msg string
	Err error
}

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
//	- Entity not found: <error string>
//	- '<kind>' entity not found: <error string>
func (this NotFoundError) Error() string {
	m := "Entity not found"
	if this.Kind != "" {
		m = fmt.Sprintf("'%v' entity not found", this.Kind)
	}
	if this.Err != nil {
		m += ": " + this.Err.Error()
	}
	return m
}

// IsNotFoundError checks if an error is the `NotFoundError` type.
func IsNotFoundError(e error) bool {
	_, ok := e.(NotFoundError)
	return ok
}
