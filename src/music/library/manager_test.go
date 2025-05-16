package library

import "testing"

func TestOps(t *testing.T) {
	mm := NewMusicManager()

	if mm == nil {
		t.Errorf("music manager should not be nil")
	}

	if mm.Len() != 0 {
		t.Errorf("music manager should be empty")
	}

	m1 := &MusicEntry{
		Id:     "1",
		Name:   "a",
		Artist: "Celine Dion",
		Source: "http://qbox.me/24501234",
		Type:   "Pop",
	}

	mm.Add(m1)
	if mm.Len() != 1 {
		t.Errorf("music manager should have 1 music")
	}

	m, err := mm.Find("a")
	if err != nil {
		t.Errorf("failed to find music: %v", err)
	}
	if m.Id != m1.Id || m.Artist != m1.Artist ||
		m.Name != m1.Name ||
		m.Source != m1.Source || m.Type != m1.Type {
		t.Error("MusicManager.Find() failed. Found item mismatch.")
	}

	m2, err := mm.Get(0)

	if m2 == nil {
		t.Error("MusicManager.Get() failed.", err)
	}

	m3 := mm.Remove(0)
	if mm.Len() != 0 || m3 == nil {
		t.Error("MusicManager.Remove() failed.", err)
	}
}
