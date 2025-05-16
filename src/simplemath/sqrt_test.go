package simplemath

import "testing"

func TestSqrtInt(t *testing.T) {
	result := Sqrt(9)
	if result != 3 {
		t.Errorf("Sqrt(9) failed. Expected 3, got %v", result)
	}
}

func TestSqrtFloat64(t *testing.T) {
	result := Sqrt(9.0)
	if result != 3 {
		t.Errorf("Sqrt(9.0) failed. Expected 3, got %v", result)
	}
}
