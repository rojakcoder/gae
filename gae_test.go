package gae

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/appengine/aetest"
	"google.golang.org/appengine/datastore"
)

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

func (this *Ointment) SetKey(key *datastore.Key) error {
	if key == nil {
		return ErrNilKey
	}
	this.KeyID = key
	return nil
}

func (this *Ointment) Update(m Model) error {
	that, ok := m.(*Ointment)
	if !ok {
		return ErrWrongType
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

	//paring of empty JSON
	var o1 Ointment
	err = json.Unmarshal(j, &o1)

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

	exp = fmt.Sprintf(`"batch":%v`, m2.Batch)
	re = regexp.MustCompile(exp)
	if !re.MatchString(js) {
		t.Errorf("JSON batch is not set. Expected %v, got %v\n", exp, js)
	}

	exp = `"Expiry":"2016-07-06T14:39:00\+08:00"`
	re = regexp.MustCompile(exp)
	if !re.MatchString(js) {
		t.Errorf("JSON Expiry is not set. Expected %v, got %v\n", exp, js)
	}

	exp = fmt.Sprintf(`"Name":"%v"`, m2.Name)
	re = regexp.MustCompile(exp)
	if !re.MatchString(js) {
		t.Errorf("JSON Name is not set. Expected %v, got %v\n", exp, js)
	}

	//parsing of non-empty JSON
	o2 := &Ointment{}
	err = json.Unmarshal(j, o2)
	if err != nil {
		t.Error("Failed to convert JSON to Ointment", err)
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
		"Tiger",
	}

	if err := Save(ctx, m2); err != nil {
		t.Error("Error saving to Datastore", err.Error())
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
		t.Error("Retrieved Ointment.Expiry is different from saved")
	}
	if o2.Name != m2.Name {
		t.Error("Retrieved Ointment.Name is different from saved")
	}
}

type Package struct {
	Weight float64
	Type   *Ointment
}

func TestEquality(t *testing.T) {
	t1, _ := time.Parse(time.RFC3339, "2007-06-05T16:03:02+08:00")
	t2, _ := time.Parse(time.RFC3339, "2008-06-05T16:03:02+08:00")
	t3, _ := time.Parse(time.RFC3339, "2009-06-05T16:03:02+08:00")
	o1 := Ointment{nil, 100, DateTime{t1}, "ml"}
	o2 := Ointment{nil, 200, DateTime{t2}, "cc"}
	o1a := Ointment{nil, 100, DateTime{t3}, "ml"}

	if o1 == o2 {
		fmt.Println("o1 is equal to o2")
	} else {
		fmt.Println("o1 is NOT equal to o2")
	}
	if o1 == o1a {
		fmt.Println("o1 is equal to o1a")
	} else {
		fmt.Println("o1 is NOT equal to o1a")
	}

	p1 := Package{12, &o1}
	p2 := Package{12, &o2}
	p1a := Package{12, &o1a}

	if p1 == p2 {
		fmt.Println("p1 is equal to p2")
	} else {
		fmt.Println("p1 is NOT equal to p2")
	}
	if p1 == p1a {
		fmt.Println("p1 is equal to p1a")
	} else {
		fmt.Println("p1 is NOT equal to p1a")
	}

}

func TestDateTime(t *testing.T) {
	t1 := DateTime{time.Time{}}
	j1, _ := t1.MarshalJSON()

	if string(j1) != `""` {
		t.Errorf("expected empty string for zeroed time; got %v", string(j1))
	}

	t1a := DateTime{time.Time{}}
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

	//test converting partly invalid JSON (
	t4, err := NewDateTime(`"2016-07-32T10:33:00+08:00"`)
	if err == nil {
		t.Errorf("expect unmarshalling to return error for invalid timestamp; converted to %v", t4)
	} else {
		fmt.Println("err", err)
	}
}
