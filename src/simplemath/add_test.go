package simplemath

import "testing"

func TestAddInt(t *testing.T) {
	result := Add(3, 5)
	if result != 8 {
		t.Errorf("Add(3, 5) failed. Expected 8, got %v", result)
	}
}

func TestAddFloat(t *testing.T) {
	result := Add(3.2, 5.8)
	if result != 9.0 {
		t.Errorf("Add(3.2, 5.8) failed. Expected 9.0, got %v", result)
	}
}
