package main

import (
	"reflect"
	"testing"
)

func TestGetBitfield(t *testing.T) {
	var b byte = 0x0
	if present, err := BitField.Has(BitField{b}, 0); err != nil || present {
		t.Fatalf("Bitfield 0 is set: %v, %v.", present, err)
	} else {
		t.Log("present = ", present, ", err = ", err)
	}
}

func TestGetBitfield2(t *testing.T) {
	var b byte = 0x0
	if _, err := BitField.Has(BitField{b}, 8); err == nil {
		t.Fatalf("Bitfield 0 is set, err = %v.", err)
	} else {
		t.Log("err = ", err)
	}
}

func TestGetBitfield3(t *testing.T) {
	var b byte = 0b10000000
	for i := 0; i < 8; i++ {
		if pres, err := BitField.Has(BitField{b}, i); err != nil {
			t.Fatalf("Bitfield 0 is set, but we have an error, err = %v.", err)
		} else {
			if i == 0 && !pres {
				t.Fatalf("First piece index is declared not present, while it should be. pres = %v.", pres)
			} else if i != 0 && pres {
				t.Fatalf("Piece index different that the first one is declared as present, while it should not be. i=%v, pres = %v.", i, pres)
			}
		}
	}
}

func TestGetBitfield4(t *testing.T) {
	var b byte = 0b00000001
	for i := 0; i < 8; i++ {
		if pres, err := BitField.Has(BitField{b}, i); err != nil {
			t.Fatalf("Bitfield 0 is set but we have an error, err = %v.", err)
		} else {
			if i == 7 && !pres {
				t.Fatalf("Last piece index is declared not present, while it should be. pres = %v.", pres)
			} else if i != 7 && pres {
				t.Fatalf("Piece index different that the first one is declared as present, while it should not be. i=%v, pres = %v.", i, pres)
			}
		}
	}
}

func TestGetBitfield5(t *testing.T) {
	b := BitField{0b10000000, 0b00000000}
	if present, err := b.Has(0); err != nil || !present {
		t.Fatal("Expecting to have the first bit field set")
	}
}

func TestSetBitfield1(t *testing.T) {
	b := BitField{0b00000000}
	BitField.Set(b, 0)
	want := BitField{0b10000000}
	if !reflect.DeepEqual(b, want) {
		t.Fatalf("%b is the computed value, while expecting %v", b, want)
	}
}

func TestSetBitfield2(t *testing.T) {
	b := BitField{0b00000000}
	BitField.Set(b, 7)
	want := BitField{0b00000001}
	if !reflect.DeepEqual(b, want) {
		t.Fatalf("%b is the computed value, while expecting %v", b, want)
	}
}
