package multiplexer

import (
	"testing"
)

func TestFrame(t *testing.T) {
	f := Frame{
		ID:     56789,
		Status: 54321,
		Data:   []byte("hello, world"),
	}
	newF := NewFrame(f.bytes())
	if f.ID != newF.ID {
		t.Fatalf("id not equal")
	}

	if f.Status != newF.Status {
		t.Fatalf("status not equal")
	}
	if string(f.Data) != string(newF.Data) {
		t.Fatalf("data not equal")
	}
}
