package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/charmbracelet/log"

	"github.com/hayasedb/hayase/internal/paths"
)

type Entry struct {
	AnimeID     string    `json:"anime_id"`
	AnimeName   string    `json:"anime_name"`
	LastSeason  int       `json:"last_season"`
	LastEpisode int       `json:"last_episode"`
	ProgressPct int       `json:"progress_pct"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Manager struct {
	entries map[string]*Entry
	path    string
}

func New() *Manager {
	path, err := paths.DataFile("history.json")
	if err != nil {
		log.Warn("history path error", "error", err)
		path = ""
	}

	m := &Manager{
		entries: make(map[string]*Entry),
		path:    path,
	}

	if err := m.Load(); err != nil {
		log.Debug("history load error", "error", err)
	}

	return m
}

func (m *Manager) Load() error {
	if m.path == "" {
		return nil
	}

	data, err := os.ReadFile(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var entries []*Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}

	m.entries = make(map[string]*Entry)
	for _, e := range entries {
		m.entries[e.AnimeID] = e
	}

	log.Debug("history loaded", "count", len(m.entries))
	return nil
}

func (m *Manager) Save() error {
	if m.path == "" {
		return nil
	}

	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	entries := m.List()
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := m.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, m.path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	log.Debug("history saved", "count", len(entries))
	return nil
}

func (m *Manager) Get(animeID string) *Entry {
	return m.entries[animeID]
}

func (m *Manager) Update(animeID, animeName string, season, episode, progressPct int) {
	entry := m.entries[animeID]
	if entry == nil {
		entry = &Entry{AnimeID: animeID}
		m.entries[animeID] = entry
	}

	entry.AnimeName = animeName
	entry.LastSeason = season
	entry.LastEpisode = episode
	entry.ProgressPct = progressPct
	entry.UpdatedAt = time.Now()

	if err := m.Save(); err != nil {
		log.Warn("history save error", "error", err)
	}
}

func (m *Manager) List() []*Entry {
	entries := make([]*Entry, 0, len(m.entries))
	for _, e := range m.entries {
		entries = append(entries, e)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].UpdatedAt.After(entries[j].UpdatedAt)
	})

	return entries
}

func (m *Manager) Remove(animeID string) {
	delete(m.entries, animeID)
	if err := m.Save(); err != nil {
		log.Warn("history save error", "error", err)
	}
}
