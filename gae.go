// Package gae is a library for interacting with Google App Engine and Datastore.
package gae

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	// KIND_SESSION is the kind of the entity stored in the datastore for
	// maintaining session.
	KIND_SESSION = "GAESession"
)

var (
	// ErrMismatch is returned when a PUT request specifies different values for
	// the ID in the path parameter and the payload model.
	//
	// Deprecated - replaced by MismatchError
	ErrMismatch = errors.New("mismatched values")

	// ErrMultipleEntities is returned when a Datastore retrieval
	// finds more than 1 entity with the specified criteria.
	//
	// Deprecated - replaced by DuplicateError
	ErrMultipleEntities = errors.New("multiple entities retrieved when only 1 is expected")

	// ErrNilKey is returned by SetKey methods when the parameter is nil.
	//
	// Deprecated - replaced by NilError
	ErrNilKey = errors.New("key is nil")

	// ErrMissingID is returned when a request does not provide an ID.
	//
	// Deprecated - replaced by MissingError
	ErrMissingID = errors.New("expected ID not specified")

	// ErrUnexpectedID is returned when a POST request includes the ID property in
	// the payload model when it is not supposed to.
	//
	// Deprecated - replaced by InvalidError
	ErrUnexpectedID = errors.New("ID is specified when it is not expected")

	// ErrWrongType is returned when the provided function argument is
	// incompatible from what is expected.
	//
	// Deprecated - replaced by MismatchError
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
//
// Deprecated - replaced by NotFoundError
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

// Page describes the contents for a page. It is to be used with templates.
type Page struct {
	Title       string
	Description string
	Dictionary  map[string]string
	Path        string
	Param       map[string]string
	Handler     func(http.ResponseWriter, *http.Request)
	Template    string
}

// AddVar is a convenient method to adding values into the Dictionary map.
//
// This method performs the additional check for initialization of the
// Dictionary map so that the calling code has the option of not initializing
// the map.
func (this *Page) AddVar(word, meaning string) {
	if this.Dictionary == nil {
		this.Dictionary = make(map[string]string)
	}
	this.Dictionary[word] = meaning
}

// ToDictionary creates a map with the existing values in the `Dictionary`
// field combined with the `Title` and `Description` fields.
//
// This is for use with templates where additional variables are needed.
//
// Note that if dictionary also contains the same keys ("Title" and
// "Dictionary"), they will be overridden.
func (this *Page) ToDictionary() map[string]string {
	var dict = make(map[string]string)
	//copy all data over
	for k, v := range this.Dictionary {
		dict[k] = v
	}
	//copy title and description over
	dict["Title"] = this.Title
	dict["Description"] = this.Description
	return dict
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

// Session keeps track of a user's session information.
//
// Any value that it needs to store should be jsonified and stored as a string
// in the Value field.
type Session struct {
	KeyID      *datastore.Key `datastore:"-"`
	Name       string         `datastore:",noindex"`
	Value      string         `datastore:",noindex"`
	Expiration time.Time      `datastore:",noindex"`
}

// Valid for Session returns true if the Expiration field is after the current
// time.
//
// If the value is not set (i.e. `IsZero`) then the session is also not valid.
func (this *Session) Valid() bool {
	if this.Expiration.IsZero() {
		return false
	}
	if this.Expiration.Before(time.Now()) {
		return false
	}
	return true
}

// CheckSession checks for a valid session based on its ID.
//
// If the session does not exist, false is returned. If the expiration time of
// the session is after the current time, returns true. Returns false otherwise.
func CheckSession(ctx context.Context, sessID string) bool {
	s := &Session{}
	item, err := memcache.Get(ctx, sessID) //read from cache
	if err == nil {                        //i.e. a hit
		err = json.Unmarshal(item.Value, s)
	}
	if err == nil { //i.e. a valid hit
		return s.Valid()
	} //else miss or error

	k, err := datastore.DecodeKey(sessID)
	if err != nil {
		return false
	}
	err = datastore.Get(ctx, k, s)
	if err != nil {
		return false
	} //else update the cache
	if _s, err := json.Marshal(s); err == nil {
		item := &memcache.Item{
			Key:   sessID,
			Value: _s,
		}
		memcache.Add(ctx, item) //ignore any error
	} //else marshalling error - cannot cache
	return s.Valid() //even if cache error, store success
}

// MakeSessionCookie creates a session and a cookie based on the database Key
// encoded value.
//
// The session is also placed in Memcache in addition to the Datastore.
//
// The `obj` parameter is the value to be stored in the cookie. It is JSONified
// before storing as a string. The `duration` parameter is the number of
// seconds for which the cookie is to be valid.
func MakeSessionCookie(ctx context.Context, name string, obj interface{},
	duration int64) (*http.Cookie, error) {
	dur := time.Duration(duration) * time.Second
	exp := time.Now().Add(dur)
	s := &Session{
		Name:       name,
		Expiration: exp,
	}
	if obj != nil {
		if js, e := json.Marshal(obj); e == nil {
			s.Value = string(js)
		}
	}
	key, err := datastore.Put(ctx, datastore.NewIncompleteKey(ctx, KIND_SESSION, nil), s)
	if err != nil {
		return nil, err
	}
	if _s, err := json.Marshal(s); err == nil {
		item := &memcache.Item{
			Key:   key.Encode(),
			Value: _s,
		}
		memcache.Set(ctx, item)
	}
	return &http.Cookie{
		Name:    name,
		Value:   key.Encode(),
		Expires: exp,
	}, nil
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
	j, e := json.Marshal(m)
	if e != nil {
		WriteRespErr(w, http.StatusInternalServerError, e)
		return
	}
	w.WriteHeader(status)
	fmt.Fprintf(w, string(j))
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
	j, e := json.Marshal(m)
	if e != nil {
		WriteRespErr(w, http.StatusInternalServerError, e)
		return
	}
	w.Header().Add(HEADER_CURSOR, cursor)
	w.WriteHeader(status)
	fmt.Fprintf(w, string(j))
}

// WriteLogRespErr logs the error string and then writes it to the response
// header (HEADER_ERROR) before setting the response code.
func WriteLogRespErr(c context.Context, w http.ResponseWriter, code int, e error) {
	if e != nil {
		log.Errorf(c, e.Error())
		w.Header().Add(HEADER_ERROR, e.Error())
	}
	w.WriteHeader(code)
}

// WriteRespErr writes the error string to the response header (HEADER_ERROR)
// before setting the response code.
func WriteRespErr(w http.ResponseWriter, code int, e error) {
	if e != nil {
		w.Header().Set(http.CanonicalHeaderKey(HEADER_ERROR), e.Error())
	}
	w.WriteHeader(code)
}
