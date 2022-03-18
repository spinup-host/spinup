package misc

import "testing"

type SliceContainsStringData struct {
	Slice    []string
	Contains string
	Expected bool
}

func TestSliceContainsString(t *testing.T) {
	/* 	actual := SliceContainsString([]string{"hello", "world"}, "world")
	   	expected := true
	   	if actual != expected {
	   		t.Errorf("actual %t, expected %t", actual, expected)
	   	} */
	SliceContainsStrings := []SliceContainsStringData{
		{
			Slice:    []string{"hello", "world"},
			Contains: "world",
			Expected: true,
		},
		{
			Slice:    []string{"hello", "world"},
			Contains: "word",
			Expected: false,
		},
		{
			Slice:    []string{"/hello", "/world"},
			Contains: "/world",
			Expected: true,
		},
	}
	for _, iter := range SliceContainsStrings {
		if actual := SliceContainsString(iter.Slice, iter.Contains); actual != iter.Expected {
			t.Errorf("actual %t, expected %t", actual, iter.Expected)
		}
	}
}
