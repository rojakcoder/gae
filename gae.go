// Package gae is a library for interacting with Google App Engine and Datastore.
package gae

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

// HEADER_ERROR is the header for holding error description. This is in all
// lower-case because it is following the specifications and App Engine
// changes it to all lowercase no matter the original casing.
const (
	HEADER_ERROR = "x-error"
)

var (
	// ErrMisMatch is returned when a PUT request specifies different values for
	// the ID in the path parameter and the payload model.
	ErrMisMatch = errors.New("Mismatched values")

	// ErrMultipleEntities is returned when a Datastore retrieval
	// finds more than 1 entity with the specified criteria.
	ErrMultipleEntities = errors.New("Multiple entities retrieved when only 1 is expected")

	// ErrNilKey is returned by SetKey methods when the parameter is nil.
	ErrNilKey = errors.New("Key is nil")

	// ErrUnexpectedID is returned when a POST request includes the ID property in
	// the payload model when it is not supposed to.
	ErrUnexpectedID = errors.New("ID is specified when it is not expected")
)

// DateTime is an auxillary struct for time.Time specifically for the purpose
// of converting to RFC3339 time format in JSON.
//
// DateTime handles time up to the seconds, ignoring the microseconds.
type DateTime struct {
	time.Time
}

// Equal checks whether the two timestamps are referring to the same moment,
// taking into account timezone differences.
func (this *DateTime) Equal(that DateTime) bool {
	return this.Time.Truncate(time.Second).Equal(that.Time.Truncate(time.Second))
}

// MarshalJSON converts the time into a format like
//  "2006-01-02T15:04:05+07:00"
// or an empty string if `time.Time.IsZero()`
func (this *DateTime) MarshalJSON() ([]byte, error) {
	if this.Time.IsZero() {
		return json.Marshal("")
	}
	return json.Marshal(this.Time.Format(time.RFC3339))
}

// UnmarshalJSON expects the input to a string like
//  "2006-01-02T15:04:05+07:00"
// to convert into a time.Time struct wrapped inside DateTime. It is able to
// understand an empty string ("") and convert it to a zeroed `time.Time`
// instance.
func (this *DateTime) UnmarshalJSON(input []byte) error {
	var s string
	if err := json.Unmarshal(input, &s); err != nil {
		return err
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	this.Time = t
	return nil
}

// EntityNotFoundError is for Datastore retrieval not finding the entity.
type EntityNotFoundError struct {
	kind string
	err  error
}

// Error for EntityNotFoundError returns a string in the format:
//  <kind> entity not found: <error string>
func (this EntityNotFoundError) Error() string {
	return this.kind + " entity not found: " + this.err.Error()
}

// JSONUnmarshalError is for unmarshalling errors when reading request JSON
// payload.
type JSONUnmarshalError struct {
	error
}

// Error for JSONUnmarshalError returns a string in the format:
//  Unable to parse JSON: <error string>
func (this JSONUnmarshalError) Error() string {
	return "Unable to parse JSON: " + this.Error()
}

// Page describes the contents for a page. It is to be used with templates.
type Page struct {
	Title       string
	Description string
	Path        string
	Handler     func(http.ResponseWriter, *http.Request)
}

// ValidityError is for errors in model validation.
type ValidityError struct {
	message string
}

// Error for ValidityError returns a string in the format:
//	Validation error: <error string>
func (this ValidityError) Error() string {
	return "Validation error: " + this.message
}

// Model is an interface that all application models must implement
// in order to be able to save to and load from the Datastore
type Model interface {
	IsValid() bool
	MakeKey(context.Context) *datastore.Key
	SetKey(*datastore.Key) error
	ValidationError() []string
}

// Presaver specifies a method Presave with no return values.
//
// Data models that require some "cleanup" before saving into the Datastore
// should implement this method to do the cleanup.
//
// Presave is called after IsValid.
type Presaver interface {
	Presave()
}

// DeleteByID removes an entity from the Datastore using the opaque
// representation of the key.
func DeleteByID(ctx context.Context, id string) error {
	key, err := datastore.DecodeKey(id)
	if err != nil {
		return err
	}
	return DeleteByKey(ctx, key)
}

// DeleteByKey removes an entity from the Datastore.
func DeleteByKey(ctx context.Context, k *datastore.Key) error {
	if err := datastore.Delete(ctx, k); err != nil {
		return err
	}
	return nil
}

// LoadByID retrieves a model from the Datastore using the opaque
// representation of the key.
func LoadByID(ctx context.Context, id string, m Model) error {
	key, err := datastore.DecodeKey(id)
	if err != nil {
		return err
	}
	return LoadByKey(ctx, key, m)
}

// LoadByKey retrieves a model from the Datastore.
func LoadByKey(ctx context.Context, k *datastore.Key, m Model) error {
	m.SetKey(k)
	if err := datastore.Get(ctx, k, m); err != nil {
		return err
	}
	return nil
}

// ReadJSON reads a HTTP request body into an instance of Model. It assumes that
// the body is a JSON string.
//
// structs that implement the Model interface should be done on the pointer
func ReadJSON(r *http.Request, m Model) error {
	dec := json.NewDecoder(r.Body)
	//TODO verify if & can be omitted from below
	if err := dec.Decode(&m); err != nil {
		return err
	}
	return nil
}

// Save checks for validity of model m prior to saving to the Datastore.
//
// Save also invokes the Presave method of m if it is set to perform any
// pre-saving actions prior to updating the entity in the Datastore.
//
// After saving, the key is assigned to m.
func Save(ctx context.Context, m Model) error {
	if !m.IsValid() {
		return ValidityError{strings.Join(m.ValidationError(), ", ")}
	}
	if presaver, ok := m.(Presaver); ok {
		presaver.Presave()
	}
	key, err := datastore.Put(ctx, m.MakeKey(ctx), m)
	if err != nil {
		return err
	}
	if err = m.SetKey(key); err != nil {
		return err
	}

	return nil
}

// WriteJSON writes an instance of Model as a JSON string into the response
// body and sets the status code as specified.
//
// If there is any error writing the JSON, a 500 Internal Server error is
// returned.
func WriteJSON(w http.ResponseWriter, m Model, status int) {
	if err := json.NewEncoder(w).Encode(m); err != nil {
		WriteResponse(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(status)
}

// WriteLogResponse logs the error string and then writes it to the response
// header (HEADER_ERROR) before setting the response code.
func WriteLogResponse(c context.Context, w http.ResponseWriter, code int, e error) {
	log.Errorf(c, e.Error())
	w.Header().Add(HEADER_ERROR, e.Error())
	w.WriteHeader(code)
}

// WriteResponse writes the error string to the response header (HEADER_ERROR)
// before setting the response code.
func WriteResponse(w http.ResponseWriter, code int, e error) {
	w.Header().Set(http.CanonicalHeaderKey(HEADER_ERROR), e.Error())
	w.WriteHeader(code)
}
