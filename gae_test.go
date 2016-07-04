package gae

import (
	"fmt"
	"regexp"
	"testing"
	"time"
)

type Ointment struct {
	Name     string
	Quantity int
	Unit     string
}

type Package struct {
	Weight float64
	Type   *Ointment
}

func TestEquality(t *testing.T) {
	o1 := Ointment{"O1", 100, "ml"}
	o2 := Ointment{"O2", 200, "cc"}
	o1a := Ointment{"O1", 100, "ml"}

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
}
