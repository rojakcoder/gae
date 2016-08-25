// Package gae is a library for interacting with Google App Engine and Datastore.
package gae

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/memcache"
)

const (
	// HEADER_CURSOR is the header for holding the pagination cursor.
	HEADER_CURSOR = "x-cursor"
	// HEADER_ERROR is the header for holding error description. This is in all
	// lower-case because it is following the specifications and App Engine
	// changes it to all lowercase no matter the original casing.
	HEADER_ERROR = "x-error"
)

var (
	// ErrMismatch is returned when a PUT request specifies different values for
	// the ID in the path parameter and the payload model.
	ErrMismatch = errors.New("mismatched values")

	// ErrMultipleEntities is returned when a Datastore retrieval
	// finds more than 1 entity with the specified criteria.
	ErrMultipleEntities = errors.New("multiple entities retrieved when only 1 is expected")

	// ErrNilKey is returned by SetKey methods when the parameter is nil.
	ErrNilKey = errors.New("key is nil")

	// ErrMissingID is returned when a request does not provide an ID.
	ErrMissingID = errors.New("expected ID not specified")

	// ErrUnexpectedID is returned when a POST request includes the ID property in
	// the payload model when it is not supposed to.
	ErrUnexpectedID = errors.New("ID is specified when it is not expected")

	// ErrWrongType is returned when the provided function argument is
	// incompatible from what is expected.
	ErrWrongType = errors.New("provided type is different from expected")
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

// String for DateTime returns the time in this format
// "YYYY-MM-DDTHH:mm:ss+HH:mm"
//
//	e.g. 2006-01-02T15:04:05+07:00
//
// In other words, the output is formatted using `time.RFC3339`
func (this *DateTime) String() string {
	return this.Time.Format(time.RFC3339)
}

// UnmarshalJSON expects the input to a string like
//  "2006-01-02T15:04:05+07:00"
// to convert into a time.Time struct wrapped inside DateTime. It is able to
// understand an empty string ("") and convert it to a zeroed `time.Time`
// instance.
func (this *DateTime) UnmarshalJSON(input []byte) error {
	if bytes.Equal([]byte(`""`), input) { //i.e. ""
		this.Time = time.Time{}
		return nil
	}
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

// NewDateTime creates a new DateTime instance from a string. The parameter
// `tstamp` is a string in the format "YYYY-MM-DDTHH:mm:ss+HH:mm"
func NewDateTime(tstamp string) (DateTime, error) {
	t, err := time.Parse(time.RFC3339, tstamp)
	if err != nil {
		return DateTime{}, err
	} else {
		return DateTime{t}, nil
	}
}

// NewDateTimeNow creates a new DateTime instance representing the moment in
// time the function was called. This is basically shorthand for:
//
//	DateTime{time.Now()}
func NewDateTimeNow() DateTime {
	return DateTime{time.Now()}
}

// EntityNotFoundError is for Datastore retrieval not finding the entity.
type EntityNotFoundError struct {
	Kind string
	Err  error
}

// Error for EntityNotFoundError returns a string in the format:
//  <kind> entity not found: <error string>
func (this EntityNotFoundError) Error() string {
	e := "entity not found"
	if this.Kind != "" {
		e = this.Kind + " entity not found"
	}
	if this.Err != nil {
		e += ": " + this.Err.Error()
	}
	return e
}

// InvalidError is a generic error for describing invalid conditions.
//
// An example is when the request parameter value is in an invalid format.
type InvalidError struct {
	Msg string
}

// Error for InvalidError returns a string in the format:
//	invalid value: <msg>
func (this InvalidError) Error() string {
	return "invalid value: " + this.Msg
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
	e := "unable to parse JSON"
	if this.Msg != "" {
		e += " (" + this.Msg + ")"
	}
	if this.Err != nil {
		e += ": " + this.Err.Error()
	}
	return e
}

// MissingError is for missing parameter values.
//
// The Msg field should provide a brief description of the parameter whose
// value is missing.
type MissingError struct {
	Msg string
}

// Error for MissingError returns a string in the format:
//	missing value: <msg>
func (this MissingError) Error() string {
	return "missing value: " + this.Msg
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
	Msg string
}

// Error for ValidityError returns a string in the format:
//	Validation error: <error string>
func (this ValidityError) Error() string {
	return "validation error: " + this.Msg
}

// Model is an interface that all application models must implement
// in order to be able to save to and load from the Datastore
//
// The ID method is for converting a *datastore.Key field into a string.
//
// The MakeKey method is for getting the Key of the entity (if present) or
// make a new one for saving (if absent).
//
// ValidationError returns a slice of string with the fields that do not meet
// the validation rules. This is used by IsValid to determine the validity of
// the model.
type Model interface {
	Key() *datastore.Key
	MakeKey(context.Context) *datastore.Key
	SetKey(*datastore.Key) error
	Update(Model) error
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
	return datastore.Delete(ctx, k)
}

// IsValid checks if a model has satisfied its validation rules.
func IsValid(m Model) bool {
	if len(m.ValidationError()) > 0 {
		return false
	}
	return true
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
	return datastore.Get(ctx, k, m)
}

// PrepPageParams parses the query parameters to get the pagination cursor and
// count.
//
// The cursor should be specified as "cursor". If not specified, an empty
// string is returned.
//
// The count should be specified as "ipp". Default value is 50.
func PrepPageParams(params url.Values) (limit int, cursor string) {
	ipp := params.Get("ipp")
	cursor = params.Get("cursor")
	limit = 50
	if ipp != "" {
		limit, _ = strconv.Atoi(ipp)
	}
	return
}

// ReadID reads the model's Key and returns the Key in a base 64
// representation.
//
// If the Key is nil, an empty string is returned.
func ReadID(m Model) string {
	if m.Key() == nil {
		return ""
	}
	return m.Key().Encode()
}

// RetrieveEntityByID attempts to retrieve the entity from Memcache before
// retrieving from the Datastore.
//
// If the entity is retrieved from the Datastore, it is placed into Memcache.
func RetrieveEntityByID(ctx context.Context, id string, m Model) error {
	_m, err := memcache.Get(ctx, id) //read from cache
	if err == nil {                  //i.e. a hit
		e := json.Unmarshal(_m.Value, m)
		err = e
	}
	if err != nil { //i.e. a miss or error
		err = LoadByID(ctx, id, m) //load from DB
		if err != nil {
			return err
		} //else update the cache
		if mj, err := json.Marshal(m); err == nil {
			item := &memcache.Item{
				Key:   id,
				Value: mj,
			}
			memcache.Set(ctx, item) //ignore any error
		} //else marshalling error - cannot cache
	}
	return nil
}

// RetrieveEntityByKey does the same thing as `RetrieveEntityByID`.
func RetrieveEntityByKey(ctx context.Context, key *datastore.Key, m Model) error {
	return RetrieveEntityByID(ctx, key.Encode(), m)
}

// Save checks for validity of model m prior to saving to the Datastore.
//
// Save also invokes the Presave method of m if it is set to perform any
// pre-saving actions prior to updating the entity in the Datastore.
//
// After saving, the key is assigned to m.
func Save(ctx context.Context, m Model) error {
	if !IsValid(m) {
		return ValidityError{strings.Join(m.ValidationError(), ", ")}
	}
	if presaver, ok := m.(Presaver); ok {
		presaver.Presave()
	}
	key, err := datastore.Put(ctx, m.MakeKey(ctx), m)
	if err != nil {
		return err
	}
	m.SetKey(key)
	return nil
}

// SaveCacheEntity saves and caches the entity.
//
// The operation to save the entity to the Datastore is performed first. If
// that fails, this function returns with the error.
//
// After saving the entity, it is then put into Memcache. Any error from
// Memcache is ignored.
func SaveCacheEntity(ctx context.Context, m Model) error {
	if err := Save(ctx, m); err != nil {
		return err
	}
	if _m, err := json.Marshal(m); err == nil {
		item := &memcache.Item{
			Key:   m.Key().Encode(),
			Value: _m,
		}
		memcache.Set(ctx, item) //ignore any error
	}
	return nil
}

// WriteJSON writes an instance of Model as a JSON string into the response
// body and sets the status code as specified.
//
// Due to the nature of the language, the slice of the implementing structs
// cannot be passed to this function as-is - it needs to be changed into a
// slice of Model explicity. E.g.
//
// If there is any error writing the JSON, a 500 Internal Server error is
// returned.
func WriteJSON(w http.ResponseWriter, m Model, status int) {
	if err := json.NewEncoder(w).Encode(m); err != nil {
		WriteRespErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(status)
}

// WriteJSONColl writes a slice of Model instances as JSON string into the
// response body and sets the status code as specified.
//
//	coll := make([]gae.Model, len(users))
//	for k, v := range users {
//		coll[k] = &v
//	}
//
// If there is any error writing the JSON, a 500 Internal Server error is
// returned.
func WriteJSONColl(w http.ResponseWriter, m []Model, status int, cursor string) {
	if err := json.NewEncoder(w).Encode(m); err != nil {
		WriteRespErr(w, http.StatusInternalServerError, err)
	} else {
		w.Header().Add(HEADER_CURSOR, cursor)
		w.WriteHeader(status)
	}
}

// WriteLogRespErr logs the error string and then writes it to the response
// header (HEADER_ERROR) before setting the response code.
func WriteLogRespErr(c context.Context, w http.ResponseWriter, code int, e error) {
	log.Errorf(c, e.Error())
	w.Header().Add(HEADER_ERROR, e.Error())
	w.WriteHeader(code)
}

// WriteRespErr writes the error string to the response header (HEADER_ERROR)
// before setting the response code.
func WriteRespErr(w http.ResponseWriter, code int, e error) {
	w.Header().Set(http.CanonicalHeaderKey(HEADER_ERROR), e.Error())
	w.WriteHeader(code)
}
