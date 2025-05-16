package library

import (
	"errors"
	"fmt"
	"slices"
)

type MusicEntry struct {
	Id     string
	Name   string
	Artist string
	Source string
	Type   string
}

type MusicManager struct {
	musics []MusicEntry
}

func NewMusicManager() *MusicManager {
	return &MusicManager{make([]MusicEntry, 0)}
}

func (m *MusicManager) Len() int {
	return len(m.musics)
}

func (m *MusicManager) Get(index int) (music *MusicEntry, err error) {
	if index < 0 || index >= len(m.musics) {
		return nil, errors.New("index out of range")
	}
	return &m.musics[index], nil
}

func (m *MusicManager) Find(name string) (music *MusicEntry, err error) {
	if m.Len() == 0 {
		return nil, errors.New("no music entries available")
	}
	for _, music := range m.musics {
		fmt.Println(music.Name, name, music.Name == name)
		if music.Name == name {
			fmt.Println("Found music: ", music.Name)
			return &music, nil
		}
	}
	return nil, errors.New("music not found")
}

func (m *MusicManager) Add(music *MusicEntry) {
	m.musics = append(m.musics, *music)
}

func (m *MusicManager) Remove(index int) *MusicEntry {
	if index < 0 || index >= len(m.musics) {
		return nil
	}
	music := m.musics[index]
	m.musics = slices.Delete(m.musics, index, index+1)
	return &music
}
