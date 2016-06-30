package gae

import (
	"fmt"
	"testing"
	"time"
)

func TestDateTime(t *testing.T) {
	t1 := DateTime{time.Time{}}
	j1, _ := t1.MarshalJSON()

	fmt.Printf("j1: %v\n", string(j1))
}
