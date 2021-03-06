package gae

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/aetest"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/memcache"
)

type Dummy struct {
	Codes map[int]int
}

func (this Dummy) Key() *datastore.Key { return nil }

func (this Dummy) MakeKey(ctx context.Context) *datastore.Key {
	return datastore.NewIncompleteKey(ctx, "Dummy", nil)
}

func (this Dummy) SetKey(key *datastore.Key) error { return nil }

func (this Dummy) Update(m Datastorer) error { return nil }

func (this Dummy) ValidationError() []string { return make([]string, 0) }

type Ointment struct {
	KeyID  *datastore.Key `json:"id"`
	Batch  int            `json:"batch"`
	Expiry DateTime
	Name   string
}

func (this *Ointment) Key() *datastore.Key {
	return this.KeyID
}

func (this *Ointment) MakeKey(ctx context.Context) *datastore.Key {
	if this.KeyID == nil {
		this.KeyID = datastore.NewIncompleteKey(ctx, "Ointment", nil)
	}
	return this.KeyID
}

func (this *Ointment) Presave() {
	if !this.Expiry.IsZero() {
		this.Expiry = DateTime{this.Expiry.AddDate(0, -1, 0)}
	}
}

func (this *Ointment) SetKey(key *datastore.Key) error {
	if key == nil {
		return NilError{
			Msg: "key is nil",
		}
	}
	this.KeyID = key
	return nil
}

func (this *Ointment) Update(m Datastorer) error {
	that, ok := m.(*Ointment)
	if !ok {
		return MismatchError{
			Msg: "provided parameter is not of type *Ointment",
		}
	}
	this.Batch = that.Batch
	this.Expiry = that.Expiry
	this.Name = that.Name
	return nil
}

func (this *Ointment) ValidationError() []string {
	msg := make([]string, 0, 1)
	if this.Name == "" {
		msg = append(msg, "Name is required")
	}
	return msg
}

func TestJSON(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	//conversion of empty instance
	m1 := Ointment{}
	j, err := json.Marshal(&m1)
	if err != nil {
		t.Error("Error converting to JSON")
	}
	js := string(j)

	exp := `"id":null`
	re := regexp.MustCompile(exp)
	if !re.MatchString(js) {
		t.Errorf("JSON id is not null. Expected %v, got %v\n", exp, js)
	}

	exp = `"KeyID":`
	re = regexp.MustCompile(exp)
	if re.MatchString(js) {
		t.Error("KeyID present in JSON:", js)
	}

	exp = `"batch":0`
	re = regexp.MustCompile(exp)
	if !re.MatchString(js) {
		t.Errorf("JSON batch is not 0. Expected %v, got %v\n", exp, js)
	}

	exp = `"Expiry":""`
	re = regexp.MustCompile(exp)
	if !re.MatchString(js) {
		t.Errorf("JSON Expiry is not empty. Expected %v, got %v\n", exp, js)
	}

	exp = `"Name":""`
	re = regexp.MustCompile(exp)
	if !re.MatchString(js) {
		t.Errorf("JSON Name is not empty. Expected %v, got %v\n", exp, js)
	}

	//parsing of empty JSON
	var o1 Ointment
	err = json.Unmarshal(j, &o1)

	if err != nil {
		t.Errorf("Error parsing JSON: %v\nJSON: %v", err, string(j))
	}

	if o1.KeyID != nil {
		t.Error("Object KeyID is not empty:", o1.KeyID)
	}
	if o1.Batch != 0 {
		t.Error("Object.Batch is not 0:", o1.Batch)
	}
	if !o1.Expiry.IsZero() {
		t.Error("Object.Expiry is not zero:", o1.Expiry)
	}
	if o1.Name != "" {
		t.Error("Object.Name is not empty:", o1.Name)
	}

	//conversion of non-empty instance
	sgt, _ := time.LoadLocation("Asia/Singapore")
	t2 := DateTime{time.Date(2016, 07, 06, 14, 39, 0, 0, sgt)}
	m2 := &Ointment{
		nil,
		43,
		t2,
		"Lion",
	}
	m2.MakeKey(ctx)
	j, err = json.Marshal(m2)
	if err != nil {
		t.Error("Failed to convert Ointment JSON", err)
	}
	js = string(j)

	exp = `"id":"[A-Za-z0-9-]{34,}"`
	re = regexp.MustCompile(exp)
	if !re.MatchString(js) {
		t.Errorf("JSON ID is not set. Expected %v, got %v\n", exp, js)
	}

	re = regexp.MustCompile(exp)
	if !re.MatchString(js) {
		t.Errorf("JSON batch is not set. Expected %v, got %v\n", exp, js)
	}

	exp = `"Expiry":"2016-07-06T14:39:00\+08:00"`
	re = regexp.MustCompile(exp)
	if !re.MatchString(js) {
		t.Errorf("JSON Expiry is not set. Expected %v, got %v\n", exp, js)
	}

	re = regexp.MustCompile(exp)
	if !re.MatchString(js) {
		t.Errorf("JSON Name is not set. Expected %v, got %v\n", exp, js)
	}

	//parsing of non-empty JSON
	o2 := &Ointment{}
	err = json.Unmarshal(j, o2)

	if err != nil {
		t.Errorf("Failed to convert JSON to Ointment: %v\nJSON: %v", err, string(j))
	}

	if o2.Batch != m2.Batch {
		t.Errorf("Expected object.Batch %v; got %v", m2.Batch, o2.Batch)
	}
	if !o2.Expiry.Equal(m2.Expiry) {
		t.Errorf("Expected object.Expiry %v; got %v", m2.Expiry, o2.Expiry)
	}
	if o2.Name != m2.Name {
		t.Errorf("Expected object.Name %v; got %v", m2.Name, o2.Name)
	}
}

func TestSaveLoadDelete(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	m1 := Ointment{
		Batch:  0,
		Expiry: DateTime{},
		Name:   "",
	}

	if m1.Key() != nil {
		t.Error("key should be nil by default")
	}
	m1.MakeKey(ctx)
	if m1.Key() == nil {
		t.Error("key should be set by MakeKey")
	}

	t2, _ := time.Parse(time.RFC3339, "2007-06-05T16:03:02+08:00")
	m2 := &Ointment{
		nil,
		22,
		DateTime{t2},
		"",
	}

	if err := Save(ctx, m2); err == nil {
		t.Error("Expected validation error but none was encountered")
	}

	m2.Name = "Tiger"

	expiry := m2.Expiry
	if err := Save(ctx, m2); err != nil {
		t.Error("Error saving to Datastore", err.Error())
	}

	//test Presave
	if expiry.Equal(m2.Expiry) {
		t.Error("Expiry field should be modified by Presave")
	}

	if m2.Key() == nil {
		t.Error("Ointment key not set after saving")
	}

	o2 := new(Ointment)
	if err := LoadByKey(ctx, m2.Key(), o2); err != nil {
		t.Error("Error loading Ointment", err)
	}
	if o2.Batch != m2.Batch {
		t.Error("Retrieved Ointment.Batch is different from saved")
	}
	if !o2.Expiry.Equal(m2.Expiry) {
		//Presave reduces Expiry by 1 day
		t.Error("Retrieved Ointment.Expiry is different from saved")
	}
	if o2.Name != m2.Name {
		t.Error("Retrieved Ointment.Name is different from saved")
	}

	DeleteByID(ctx, m2.Key().Encode())

	o3 := Ointment{}
	if err := LoadByID(ctx, m2.Key().Encode(), &o3); err == nil {
		t.Error("Expected error from not finding entity. Should be deleted already")
	}
}

type Package struct {
	Weight float64
	Type   *Ointment
}

func TestEquality(t *testing.T) {
	t1, _ := time.Parse(time.RFC3339, "2007-06-05T16:03:02+08:00")
	t2, _ := time.Parse(time.RFC3339, "2008-06-05T16:03:02+08:00")
	a1 := Ointment{nil, 100, DateTime{t1}, "ml"}
	a2 := Ointment{nil, 100, DateTime{t1}, "ml"}
	b1 := Ointment{nil, 100, DateTime{t1}, "cc"}
	b2 := Ointment{nil, 100, DateTime{t2}, "ml"}

	if a1 != a2 {
		t.Error("a1 should be equal to a2 because all values are identical")
	}
	if a1 == b1 {
		t.Errorf("a1 should not be equal to b1 because of different Name values")
	}
	if a1 == b2 {
		t.Error("a1 should not be equal to b2 because of different DateTime values")
	}

	p1 := Package{12, &a1}
	p2 := Package{12, &a2}

	if p1 == p2 {
		t.Error("p1 should not be equal to p2 because the memory locations of Type are different even though values are the same")
	}
}

func ExampleDateTime_Equal_dateTime() {
	t1 := time.Date(2017, time.July, 3, 17, 59, 59, 1, time.Local)
	t2 := time.Date(2017, time.July, 3, 9, 59, 59, 59, time.UTC)
	ta := DateTime{t1}
	tb := DateTime{t2}
	if ta.Equal(tb) {
		fmt.Println("Equal")
	} else {
		fmt.Println("Not equal")
	}
	// Output: Equal
}

func ExampleDateTime_Equal_time() {
	t1 := time.Date(2017, time.July, 3, 17, 59, 59, 1, time.Local)
	t2 := time.Date(2017, time.July, 3, 9, 59, 59, 59, time.UTC)
	if t1.Equal(t2) {
		fmt.Println("Equal")
	} else {
		fmt.Println("Not equal")
	}
	// Output: Not equal
}

func ExampleDateTime_MarshalJSON_dateTime() {
	t1 := DateTime{}
	js, _ := t1.MarshalJSON()
	fmt.Println(string(js))
	// Output: ""
}

func ExampleDateTime_MarshalJSON_time() {
	//Compare to the output of time.Time
	t1 := time.Time{}
	js, _ := json.Marshal(t1.Format(time.RFC3339))
	fmt.Println(string(js))
	// Output: "0001-01-01T00:00:00Z"
}

func ExampleDateTime_String_dateTime() {
	//To create a new DateTime instance
	t1 := DateTime{}
	fmt.Println(t1.String())
	// Output: 0001-01-01T00:00:00Z
}

func ExampleDateTime_String_time() {
	//Compare to the output of time.Time
	t1 := time.Time{}
	fmt.Println(t1.String())
	// Output: 0001-01-01 00:00:00 +0000 UTC
}

func ExampleNewDateTime() {
	t, _ := NewDateTime("2017-07-03T09:44:00+08:00")
	fmt.Println(t.String())
	// Output: 2017-07-03T09:44:00+08:00
}

func TestDateTime(t *testing.T) {
	test := func(s string, exp, act int) {
		if exp != act {
			t.Errorf(s, exp, act)
		}
	}

	t1 := DateTime{}
	j1, _ := t1.MarshalJSON()

	if string(j1) != `""` {
		t.Errorf("expected empty string for zeroed time; got %v", string(j1))
	}

	t1a := DateTime{}
	err := t1a.UnmarshalJSON(([]byte)(`""`))
	if err != nil {
		t.Errorf("error unmarshalling time from empty quotes \"\": %v", err)
	}
	if !t1a.IsZero() {
		t.Errorf("expect time to be zeroed; got %v", t1a)
	}

	t2 := DateTime{time.Now()}
	if t1.Equal(t2) {
		t.Errorf("t1 (%v) should not be equal to t2 (%v)", t1, t2)
	}

	sgt, _ := time.LoadLocation("Asia/Singapore")
	t3 := DateTime{time.Date(2007, 06, 05, 16, 03, 02, 12345678, sgt)}
	j3, _ := t3.MarshalJSON()
	ts3 := `"2007-06-05T16:03:02\+08:00"`
	re := regexp.MustCompile(ts3)
	if !re.MatchString(string(j3)) {
		t.Errorf("expected JSON time to be `%v` (with quotes); got %v", ts3, string(j3))
	}

	//test converting invalid JSON
	t1b := DateTime{time.Time{}}
	err = t1b.UnmarshalJSON([]byte("invalid"))
	if err == nil {
		t.Errorf("expect unmarshalling to return error for empty string")
	}

	//test converting partly invalid JSON
	t1c := DateTime{}
	err = t1c.UnmarshalJSON([]byte(`"2016-07-32T10:33:00+08:00"`))
	if err == nil {
		t.Errorf("expect unmarshalling to return error for invalid timestamp")
	}

	t3, err = NewDateTime("2016-05-04T13:22:31+08:00")
	if err != nil {
		t.Error("expect NewDateTime not to return error; got", err)
	}
	test("expect year %d; got %d", 2016, t3.Year())
	test("expect month %d; got %d", 5, int(t3.Month()))
	test("expect day %d; got %d", 4, t3.Day())
	test("expect hour %d; got %d", 13, t3.Hour())
	test("expect minute %d; got %d", 22, t3.Minute())
	test("expect second %d; got %d", 31, t3.Second())

	_, err = NewDateTime("2016-05-04T13:22:31")
	if err == nil {
		t.Error("expect NewDateTime to return error without timezone")
	}

	_, err = NewDateTime(`"2016-05-04T13:22:31+08:00"`)
	if err == nil {
		t.Error("expect NewDateTime to return error due to extraneous quotes")
	}

	_, err = NewDateTime("2016-07-32T10:33:00+08:00")
	if err == nil {
		t.Errorf("expect NewDateTime to return error due to invalid date")
	}

	now1 := DateTime{time.Now()}
	now2 := NewDateTimeNow()
	if !now1.Equal(now2) {
		t.Errorf("expect both timestamps to be the same since they should occur within the same second; got %v and %v", now1, now2)
	}
}

func TestCoverage(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	//cover DateTime
	jsTime := "ABC"
	dt1 := new(DateTime)
	if err := dt1.UnmarshalJSON(([]byte)(jsTime)); err == nil {
		t.Error("failed to cover DateTime.UnmarshalJSON")
	}

	//cover DeleteByID
	if err := DeleteByID(ctx, "invalid-key"); err == nil {
		t.Error("expected DeleteByID to fail with invalid ID:", "invalid-key")
	}

	//cover LoadByID
	if err := LoadByID(ctx, "invalid-key", &Ointment{}); err == nil {
		t.Error("expected LoadByKey to fail with invalid ID:", "inavlid-key")
	}

	//cover Save
	if err := Save(ctx, Dummy{}); err == nil {
		t.Error("expected error from saving Dummy")
	}

	//cover String
	now := NewDateTimeNow()
	s := now.String()
	ts := "[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}"
	re := regexp.MustCompile(ts)
	if !re.MatchString(s) {
		t.Errorf("expect DateTime.String to be in the format %v; got %v", ts, s)
	}
}

func TestCoverageCounter(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}

	mkey := counterMemcacheKey("c1")
	err = memcache.JSON.Set(ctx, &memcache.Item{
		Key:    mkey,
		Object: 33,
	})
	if err != nil {
		t.Fatal(err)
	}
	n, err := CounterCount(ctx, "c1")
	if err != nil {
		t.Error("failed to cover CounterCount; got error %v", err)
	}
	if n != 33 {
		t.Error("expect memcache to hold value 33; got %d", n)
	}

	done()

	n, err = CounterCount(ctx, "c1")
	if err == nil {
		t.Error("failed to cover CounterCount")
	}
	if n != 0 {
		t.Errorf("expect 0 for error; got %d", n)
	}

	err = CounterIncrement(ctx, "c1")
	if err == nil {
		t.Error("failed to cover CounterIncrement")
	}

	err = CounterIncreaseShards(ctx, "c1", 10)
	if err == nil {
		t.Error("failed to cover CounterIncreaseShards")
	}
}

func TestErrors(t *testing.T) {
	//cover EntityNotFoundError
	enfeTests := []struct {
		e    error
		want string
	}{
		{NotFoundError{}, "entity not found"},
		{NotFoundError{Kind: "Assignment"}, "'Assignment' entity not found"},
		{NotFoundError{"Deadline", errors.New("overdue")}, "'Deadline' entity not found - overdue"},
	}
	for _, tt := range enfeTests {
		if tt.e.Error() != tt.want {
			t.Errorf("Error string for EntityNotFoundError is different.\n - Expected: %v\n -      Got: %v\n", tt.want, tt.e.Error())
		}
	}

	//cover InvalidError
	ieTests := []struct {
		e    error
		want string
	}{
		{InvalidError{}, "Invalid value ()"},
		{InvalidError{"Currency expected"}, "Invalid value (Currency expected)"},
	}
	for _, tt := range ieTests {
		if tt.e.Error() != tt.want {
			t.Errorf("Error string for InvalidError is different.\n - Expected: %v\n -      Got: %v\n", tt.want, tt.e.Error())
		}
	}

	//cover JSONMarshalError
	jmeTests := []struct {
		e    error
		want string
	}{
		{JSONUnmarshalError{}, "Unable to parse JSON"},
		{JSONUnmarshalError{Msg: "empty string"}, "Unable to parse JSON (empty string)"},
		{JSONUnmarshalError{"numbers only", errors.New("Numbers only")}, "Unable to parse JSON (numbers only) - Numbers only"},
	}
	for _, tt := range jmeTests {
		if tt.e.Error() != tt.want {
			t.Errorf("Error string for JSONMarshalError is different.\n - Expected: %v\n -      Got: %v\n", tt.want, tt.e.Error())
		}
	}
	if !IsJSONUnmarshalError(jmeTests[0].e) {
		t.Errorf("expect IsJSONUnmarshalError to return true; got false")
	}

	//cover MissingError
	meTests := []struct {
		e    error
		want string
	}{
		{MissingError{}, "Missing value"},
		{MissingError{"key"}, "Missing value - key"},
	}
	for _, tt := range meTests {
		if tt.e.Error() != tt.want {
			t.Errorf("Error string for MissingError is different.\n - Expected: %v\n -      Got: %v\n", tt.want, tt.e.Error())
		}
	}

	//cover ValidityError
	veTests := []struct {
		e    error
		want string
	}{
		{ValidityError{}, "validation error - "},
		{ValidityError{"Name is required"}, "validation error - Name is required"},
	}
	for _, tt := range veTests {
		if tt.e.Error() != tt.want {
			t.Errorf("Error string for ValidityError is different.\n - Expected: %v\n -      Got: %v\n", tt.want, tt.e.Error())
		}
	}
}

func TestSaveRetrieveEntity(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	//defer done() - manual invocation
	test := func(s string, exp, act interface{}) {
		if exp != act {
			t.Errorf(s, exp, act)
		}
	}

	m1 := &Ointment{}
	k1 := datastore.NewKey(ctx, "go", "go", 0, nil)
	if err := RetrieveEntityByKey(ctx, k1, m1); err == nil {
		t.Error("expect EntityNotFound error; got none")
	}

	m0 := &Ointment{}
	m0.KeyID = k1
	m0.Batch = 12
	m0.Name = "M One"
	if err := Save(ctx, m0); err != nil {
		t.Fatal("error saving to DB", err)
	}

	if err := RetrieveEntityByKey(ctx, k1, m1); err != nil {
		t.Errorf("expect RetrieveEntityByKey to get a cache miss, DB hit; got error %v", err.Error())
	}
	test("expect Batch value %v; got %v", m0.Batch, m1.Batch)
	test("expect Name value %v; got %v", m0.Name, m1.Name)

	m2 := &Ointment{}
	if err := RetrieveEntityByKey(ctx, k1, m2); err != nil {
		t.Error("expect RetrieveEntityByKey to get a cache hit; got error", err)
	}
	test("expect Batch value %v; got %v", m0.Batch, m2.Batch)
	test("expect Name value %v; got %v", m0.Name, m2.Name)

	//delete the entity from DB to test cache miss
	if err := DeleteByKey(ctx, k1); err != nil {
		t.Fatal("error deleting from DB", err)
	}
	m3 := &Ointment{}
	if err := RetrieveEntityByKey(ctx, k1, m3); err == nil {
		t.Error("expect RetrieveEntityByKey to get a cache miss when entity is deleted; got nil error")
	}
	test("expect Batch value %v; got %v", 0, m3.Batch)
	test("expect Name value %v; got %v", "", m3.Name)

	if err := memcache.Delete(ctx, k1.Encode()); err != memcache.ErrCacheMiss {
		t.Error("expect memcache.Delete to return ErrCacheMiss as a result of DeleteByKey")
	}
	//retrieval should now give error
	m4 := &Ointment{}
	if err := RetrieveEntityByKey(ctx, k1, m4); err != datastore.ErrNoSuchEntity {
		t.Error("expect RetrieveEntityByKey to return ErrNoSuchEntity; got", err)
	}

	if err := SaveCacheEntity(ctx, m1); err != nil {
		t.Errorf("expect SaveCacheEntity to complete with no errors; got %v", err.Error())
	}
	item, err := memcache.Get(ctx, k1.Encode()) //read from cache
	if err != nil {
		t.Error("expect SaveCacheEntity to cache entity; got error:", err)
	}
	m5 := &Ointment{}
	if json.Unmarshal(item.Value, m5) != nil {
		t.Fatal("json.Unmarshal returned error")
	}
	test("expect Batch value %v; got %v", m0.Batch, m5.Batch)
	test("expect Name value %v; got %v", m0.Name, m5.Name)

	done()
	if SaveCacheEntity(ctx, m1) == nil {
		t.Error("expect SaveCacheEntity to return error after done(); got none")
	}
}

func TestServerFuncs(t *testing.T) {
	inst, err := aetest.NewInstance(nil)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}
	defer inst.Close()

	path := "/"
	r1, err := inst.NewRequest("GET", path, nil)
	if err != nil {
		t.Fatalf("Failed to create request for %v: %v", path, err)
	}

	//test Page struct - especially Dictionary
	p1 := Page{
		Title:       "Page 1",
		Description: "Placeholder for page 1",
	}
	d1 := p1.ToDictionary()
	if len(d1) != 2 {
		t.Errorf("expected Dictionary to contain %d items; got %d", 2, len(d1))
	}
	title := d1["Title"]
	if p1.Title != (title) {
		t.Errorf("expected Title in dictionary to be %s; got %s", p1.Title, title)
	}
	desc := d1["Description"]
	if p1.Description != (desc) {
		t.Errorf("expected Description in dictionary to be %s; got %s", p1.Description, desc)
	}
	p2 := Page{
		Title:       "Page 2",
		Description: "Placeholder for page 2",
	}
	//cannot assign value because Dictionary is not initialized
	//p2.Dictionary["name"] = "Name 2"
	p2.AddVar("name", "Name 2")
	p2.AddVar("number", "Two")
	d2 := p2.ToDictionary()
	if len(d2) != 4 {
		t.Errorf("expected Dictionary to contain %d items; got %d", 4, len(d2))
	}
	title = d2["Title"]
	if p2.Title != (title) {
		t.Errorf("expected Title in dictionary to be %s; got %s", p2.Title, title)
	}
	desc = d2["Description"]
	if p2.Description != (desc) {
		t.Errorf("expected Description in dictionary to be %s; got %s", p2.Description, desc)
	}
	name := d2["name"]
	if p2.Dictionary["name"] != (name) {
		t.Errorf("expected name in dictionary to be %s; got %s", p2.Dictionary["name"], name)
	}

	//test PrepPageParams
	limit, cursor := PrepPageParams(r1.URL.Query())
	if limit != 50 {
		t.Errorf("expected default limit value 50; got %v", limit)
	}
	if cursor != "" {
		t.Errorf("expected cursor to be empty; got %v", cursor)
	}

	path = "/?ipp=300&cursor=abc"
	r2, err := inst.NewRequest("GET", path, nil)
	if err != nil {
		t.Fatalf("Failed to create request for %v: %v", path, err)
	}

	limit, cursor = PrepPageParams(r2.URL.Query())
	if limit != 300 {
		t.Errorf("expected specified limit value 300; got %v", limit)
	}
	if cursor != "abc" {
		t.Errorf("expected cursor to be 'abc'; got %v", cursor)
	}

	//test WriteJSON
	w := httptest.NewRecorder()
	WriteJSON(w, &Ointment{}, http.StatusOK)
	if w.Code != 200 {
		t.Errorf("expected response code %v; got %v", 200, w.Code)
	}
	json := "{\"id\":null,\"batch\":0,\"Expiry\":\"\",\"Name\":\"\"}"
	if string(w.Body.Bytes()) != json {
		t.Errorf("expected JSON output:\n - %v(%d)\ngot:\n - %v(%d)", json, len(json), string(w.Body.Bytes()), len(string(w.Body.Bytes())))
	}

	w = httptest.NewRecorder()
	WriteJSON(w, &Dummy{}, http.StatusOK)
	if w.Code != 500 {
		t.Errorf("expected response code %v; got %v", 500, w.Code)
	}
	if len(w.Body.Bytes()) != 0 {
		t.Errorf("expected error response body to be empty")
	}
	_, hasHeader := w.HeaderMap[http.CanonicalHeaderKey(HeaderError)]
	if !hasHeader {
		t.Errorf("expected error response to contain header %v", HeaderError)
	}

	//test WriteJSONColl
	w = httptest.NewRecorder()
	oints := []Ointment{
		Ointment{},
	}
	coll := make([]Datastorer, len(oints))
	for k, v := range oints {
		coll[k] = &v
	}
	cursor = "cursorabc"
	WriteJSONColl(w, coll, http.StatusOK, cursor)
	if w.Code != 200 {
		t.Errorf("expected response code %v; got %v", 200, w.Code)
	}
	json = "[{\"id\":null,\"batch\":0,\"Expiry\":\"\",\"Name\":\"\"}]"
	if string(w.Body.Bytes()) != json {
		t.Errorf("expected JSON output:\n - %v(%d)\ngot:\n - %v(%d)", json, len(json), string(w.Body.Bytes()), len(string(w.Body.Bytes())))
	}
	header, hasHeader := w.HeaderMap[http.CanonicalHeaderKey(HeaderCursor)]
	if !hasHeader {
		t.Errorf("expected response to contain header %v", HeaderCursor)
	}
	if len(header) != 1 {
		t.Errorf("expected response header %v to contain only %v value; got %v", HeaderCursor, 1, len(header))
	}
	if header[0] != cursor {
		t.Errorf("expected response header value %v; got %v", cursor, header)
	}

	w = httptest.NewRecorder()
	dums := []Dummy{Dummy{}}
	coll = make([]Datastorer, len(dums))
	for k, v := range dums {
		coll[k] = &v
	}
	WriteJSONColl(w, coll, http.StatusOK, cursor)
	if w.Code != 500 {
		t.Errorf("expected response code %v; got %v", 500, w.Code)
	}
	if len(w.Body.Bytes()) != 0 {
		t.Errorf("expected error response body to be empty")
	}
	_, hasHeader = w.HeaderMap[http.CanonicalHeaderKey(HeaderError)]
	if !hasHeader {
		t.Errorf("expected error response to contain header %v", HeaderError)
	}
	_, hasHeader = w.HeaderMap[http.CanonicalHeaderKey(HeaderCursor)]
	if hasHeader {
		t.Errorf("expected error response to NOT contain header %v", HeaderCursor)
	}
	//test WriteLogRespErr
	c1 := appengine.NewContext(r1)
	w = httptest.NewRecorder()
	WriteLogRespErr(c1, w, http.StatusBadRequest, InvalidError{"Invalid request - this output is expected in TestServerFuncs"})
	if w.Code != 400 {
		t.Errorf("expected response code %v; got %v", 400, w.Code)
	}
	if len(w.Body.Bytes()) != 0 {
		t.Errorf("expected error response body to be empty")
	}
	_, hasHeader = w.HeaderMap[http.CanonicalHeaderKey(HeaderError)]
	if !hasHeader {
		t.Errorf("expected error response to contain header %v", HeaderError)
	}
}

func TestSession(t *testing.T) {
	inst, err := aetest.NewInstance(&aetest.Options{
		StronglyConsistentDatastore: true,
	})
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}
	//defer inst.Close() - manual invocation for code coverge
	r, err := inst.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := appengine.NewContext(r)

	s1 := &Session{}
	if s1.Valid() {
		t.Error("Session is valid when it should not be")
	}

	dur := time.Duration(1) * time.Hour
	exp := time.Now().Add(dur)
	s2 := &Session{
		Name:       "session",
		Value:      "two",
		Expiration: exp,
	}
	if !s2.Valid() {
		t.Error("Session is invalid when it should be")
	}

	dur = time.Duration(-1) * time.Hour
	exp = time.Now().Add(dur)
	s3 := &Session{
		Name:       "session",
		Value:      "three",
		Expiration: exp,
	}
	if s3.Valid() {
		t.Error("Session is valid when it should not be")
	}

	n4 := "session"
	v4 := "four"
	s4, err := MakeSessionCookie(ctx, n4, v4, 60)
	if n4 != s4.Name {
		t.Errorf("expect name of cookie to be %v; got %v", n4, s4.Name)
	}
	if "" == s4.Value {
		t.Error("expect value of cookie to be non-empty; got empty string")
	}

	testCheckSession := func(name string, exp, act bool) {
		if exp != act {
			t.Errorf("expect %v to return %v; got %v", name, exp, act)
		}
	}

	verified := CheckSession(ctx, s4.Value)
	testCheckSession("CheckSession (valid session from cache)", true, verified)
	memcache.Delete(ctx, s4.Value)
	verified = CheckSession(ctx, s4.Value)
	testCheckSession("CheckSession (valid session from store)", true, verified)

	k5 := datastore.NewKey(ctx, KindSession, "", 12, nil)
	verified = CheckSession(ctx, k5.Encode())
	testCheckSession("CheckSession (non-existing session)", false, verified)
	item := &memcache.Item{
		Key:   k5.Encode(),
		Value: []byte("123"),
	}
	memcache.Set(ctx, item)
	verified = CheckSession(ctx, k5.Encode())
	testCheckSession("CheckSession (invalid cache item)", false, verified)

	s6, err := MakeSessionCookie(ctx, "session", "six", -60)
	verified = CheckSession(ctx, s6.Value)
	testCheckSession("CheckSession (expired session)", false, verified)

	verified = CheckSession(ctx, "invalid-ID")
	testCheckSession("CheckSession (invalid ID)", false, verified)

	inst.Close()
	_, e := MakeSessionCookie(ctx, n4, v4, 60)
	if e == nil {
		t.Error("expect MakeSessionCookie to return error after Close(); got none")
	}
}

func TestErrorResponseEqual(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	e1 := ErrorResponse{}
	e2 := ErrorResponse{}

	type tc struct {
		title  string
		result bool
		setup  func(context.Context)
	}

	cases := []tc{
		{
			title:  "Both empty",
			result: true,
		},
		{
			title:  "Only ErrorCode 1",
			result: false,
			setup: func(ctx context.Context) {
				e1.ErrorCode = "EC1"
			},
		},
		{
			title:  "Only ErrorCode 2",
			result: true,
			setup: func(ctx context.Context) {
				e2.ErrorCode = "EC1"
			},
		},
		{
			title:  "Only 1 all",
			result: false,
			setup: func(ctx context.Context) {
				e1.Field = "email"
				e1.HelpURL = "/help"
				e1.Message = "Email format"
				e1.OriginalValue = "email"
			},
		},
		{
			title:  "2 mismatch 3",
			result: false,
			setup: func(ctx context.Context) {
				e2.Field = "email"
			},
		},
		{
			title:  "2 mismatch 2",
			result: false,
			setup: func(ctx context.Context) {
				e2.HelpURL = "/help"
			},
		},
		{
			title:  "2 mismatch 1",
			result: false,
			setup: func(ctx context.Context) {
				e2.Message = "Email format"
			},
		},
		{
			title:  "Match",
			result: true,
			setup: func(ctx context.Context) {
				e2.OriginalValue = "email"
			},
		},
	}

	for _, c := range cases {
		if c.setup != nil {
			c.setup(ctx)
		}
		if c.result != e1.Equal(e2) {
			if c.result {
				t.Errorf("%v: expect instances to be equal", c.title)
			} else {
				t.Errorf("%v: expect instances to be NOT equal", c.title)
			}
		}
	}
	rec := httptest.NewRecorder()
	WriteErrorResponse(rec, 400, e2)
	if rec.Code != 400 {
		t.Errorf("expect error code %d; got %d", 400, rec.Code)
	}
	resp := ErrorResponse{}
	if e := json.Unmarshal(rec.Body.Bytes(), &resp); e != nil {
		t.Fatal(e)
	}
	if !e2.Equal(resp) {
		t.Errorf("expect WriteErrorResponse to return\n\n%+v; got\n\n%+v", e2, resp)
	}
}

func TestErrorResponseError(t *testing.T) {
	ec := "SESSION_ERROR"
	field := "userID"
	msg := "user not authorized"
	value := "guest"

	cases := []struct {
		output string
		er     ErrorResponse
	}{
		{
			output: "user not authorized (SESSION_ERROR) - userID(guest)",
			er: ErrorResponse{
				ErrorCode:     ec,
				Field:         field,
				Message:       msg,
				OriginalValue: value,
			},
		},
		{
			output: "user not authorized - userID(guest)",
			er: ErrorResponse{
				Field:         field,
				Message:       msg,
				OriginalValue: value,
			},
		},
		{
			output: "user not authorized (SESSION_ERROR) - (guest)",
			er: ErrorResponse{
				ErrorCode:     ec,
				Message:       msg,
				OriginalValue: value,
			},
		},
		{
			output: "(SESSION_ERROR) - userID(guest)",
			er: ErrorResponse{
				ErrorCode:     ec,
				Field:         field,
				OriginalValue: value,
			},
		},
		{
			output: "user not authorized (SESSION_ERROR) - userID",
			er: ErrorResponse{
				ErrorCode: ec,
				Field:     field,
				Message:   msg,
			},
		},
		{
			output: "user not authorized - (guest)",
			er: ErrorResponse{
				Message:       msg,
				OriginalValue: value,
			},
		},
		{
			output: "userID(guest)",
			er: ErrorResponse{
				Field:         field,
				OriginalValue: value,
			},
		},
		{
			output: "user not authorized - userID",
			er: ErrorResponse{
				Field:   field,
				Message: msg,
			},
		},
		{
			output: "(SESSION_ERROR) - (guest)",
			er: ErrorResponse{
				ErrorCode:     ec,
				OriginalValue: value,
			},
		},
		{
			output: "user not authorized (SESSION_ERROR)",
			er: ErrorResponse{
				ErrorCode: ec,
				Message:   msg,
			},
		},
		{
			output: "(SESSION_ERROR) - userID",
			er: ErrorResponse{
				ErrorCode: ec,
				Field:     field,
			},
		},
		{
			output: "(SESSION_ERROR)",
			er: ErrorResponse{
				ErrorCode: ec,
			},
		},
		{
			output: "userID",
			er: ErrorResponse{
				Field: field,
			},
		},
		{
			output: "user not authorized",
			er: ErrorResponse{
				Message: msg,
			},
		},
		{
			output: "(guest)",
			er: ErrorResponse{
				OriginalValue: value,
			},
		},
	}

	for _, c := range cases {
		if c.output != c.er.Error() {
			t.Errorf("expect error string\n\t%v; got\n\t%v",
				c.output, c.er.Error())
		}
	}
}

func TestCounterCount(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	cases := []struct {
		title     string
		name      string
		increase  int
		delay     int
		wantCount int
		wantErr   bool
	}{
		{
			title:     "Starting 0",
			name:      "instances",
			wantCount: 0,
		},
		{
			title:     "Adding 1 - still 0 because of eventual consistency",
			name:      "instances",
			increase:  1,
			wantCount: 0,
		},
		{
			title:     "Check for 1 again",
			name:      "instances",
			delay:     2,
			wantCount: 1,
		},
		{
			title:     "Adding 11",
			name:      "instances",
			increase:  11,
			delay:     6,
			wantCount: 12,
		},
	}

	for _, c := range cases {
		if c.increase > 0 {
			for i := 0; i < c.increase; i++ {
				go func() {
					if e := CounterIncrement(ctx, c.name); e != nil {
						t.Fatal(e)
					}
				}()
			}
		}
		if c.delay > 0 {
			time.Sleep(time.Second * time.Duration(c.delay))
		}
		num, err := CounterCount(ctx, c.name)
		if c.wantErr && err == nil {
			t.Errorf("%v: expect error; got nil", c.title)
		}
		if !c.wantErr && err != nil {
			t.Fatal(err)
		}
		if c.wantCount != num {
			t.Errorf("%v: expect counter to be %d; got %d",
				c.title, c.wantCount, num)
		}
	}
}

func TestCounterShard(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	cases := []struct {
		title     string
		name      string
		num       int
		wantCount int
	}{
		{
			title:     "Start with 20",
			name:      "kounter",
			num:       20,
			wantCount: 20,
		},
		{
			title:     "No increase when numbers same",
			name:      "kounter",
			num:       20,
			wantCount: 20,
		},
		{
			title:     "Increase to 25",
			name:      "kounter",
			num:       25,
			wantCount: 25,
		},
		{
			title:     "Start with 1",
			name:      "kounter1",
			num:       1,
			wantCount: 5,
		},
	}

	for _, c := range cases {
		if c.num > 0 {
			if e := CounterIncreaseShards(ctx, c.name, c.num); e != nil {
				t.Fatal(e)
			}
		}
		var config counterConfig
		ckey := datastore.NewKey(ctx, KindCounterConfig, c.name, 0, nil)
		if e := datastore.Get(ctx, ckey, &config); e != nil {
			t.Fatal(e)
		}
		if c.wantCount != config.Shards {
			t.Errorf("%v: expect number of shards to be %d; got %d",
				c.title, c.wantCount, config.Shards)
		}
	}
}
