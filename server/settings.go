package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// settingsFile lives next to the exe (plan section 5).
const settingsFile = "settings.json"

const (
	defaultTitle            = "Kvíz"
	defaultTopics           = 5
	defaultQuestions        = 6
	defaultGameMinutes      = 30
	defaultGameDuration     = defaultGameMinutes * time.Minute
	defaultLightningMinutes = 3
	defaultLightningPoints  = 3000
	maxTeams                = 5
	maxGameMinutes          = 600
	maxLightningMinutes     = 600
)

var defaultPoints = []int{1000, 2000, 3000, 4000, 5000, 8000}

// Rules are the toggles from plan section 8.
type Rules struct {
	ReopenOnWrong  bool `json:"reopenOnWrong"`  // wrong answer reopens buzzer (default off)
	LockoutTeam    bool `json:"lockoutTeam"`    // lock whole team, not only player (default off)
	SubtractPoints bool `json:"subtractPoints"` // wrong answer subtracts points (default on)
	AutoOpen       bool `json:"autoOpen"`       // opening a cell opens the buzzer (default on)
}

// Settings is settings.json: everything the teacher can change about the game
// shape. Loaded at start, saved after every change.
type Settings struct {
	Title            string   `json:"title"`
	Teams            []string `json:"teams"`
	Topics           int      `json:"topics"`
	Questions        int      `json:"questions"`
	Points           []int    `json:"points"`
	LightningPoints  int      `json:"lightningPoints"`
	GameMinutes      int      `json:"gameMinutes"`
	LightningMinutes int      `json:"lightningMinutes"`
	Rules            Rules    `json:"rules"`
	LastQuiz         string   `json:"lastQuiz"`
}

// GameDuration is the round-1 countdown length.
func (s Settings) GameDuration() time.Duration {
	return time.Duration(s.GameMinutes) * time.Minute
}

// LightningDuration is the lightning countdown length.
func (s Settings) LightningDuration() time.Duration {
	return time.Duration(s.LightningMinutes) * time.Minute
}

// DefaultSettings returns the plan defaults: 5 teams, 5x6 board, 30 + 3 min,
// points 1000-8000, subtract on, auto-open on, both wrong-answer rules off.
func DefaultSettings() Settings {
	return Settings{
		Title:            defaultTitle,
		Teams:            []string{"A csapat", "B csapat", "C csapat", "D csapat", "E csapat"},
		Topics:           defaultTopics,
		Questions:        defaultQuestions,
		Points:           append([]int(nil), defaultPoints...),
		LightningPoints:  defaultLightningPoints,
		GameMinutes:      defaultGameMinutes,
		LightningMinutes: defaultLightningMinutes,
		Rules: Rules{
			SubtractPoints: true,
			AutoOpen:       true,
		},
	}
}

// normalize repairs a loaded or incoming settings value. Out-of-range values
// fall back to defaults instead of failing the start.
func (s *Settings) normalize() {
	s.Title = strings.TrimSpace(s.Title)
	if s.Title == "" {
		s.Title = defaultTitle
	}

	teams := make([]string, maxTeams)
	for i := range teams {
		if i < len(s.Teams) {
			teams[i] = strings.TrimSpace(s.Teams[i])
		}
	}
	s.Teams = teams

	if s.Topics < minBoardSize || s.Topics > maxBoardSize {
		s.Topics = defaultTopics
	}
	if s.Questions < minBoardSize || s.Questions > maxBoardSize {
		s.Questions = defaultQuestions
	}

	pointsOK := len(s.Points) == s.Questions
	if pointsOK {
		for _, p := range s.Points {
			if p <= 0 {
				pointsOK = false
				break
			}
		}
	}
	if !pointsOK {
		s.Points = pointsForQuestions(s.Questions)
	}

	if s.LightningPoints <= 0 {
		s.LightningPoints = defaultLightningPoints
	}
	if s.GameMinutes < 1 || s.GameMinutes > maxGameMinutes {
		s.GameMinutes = defaultGameMinutes
	}
	if s.LightningMinutes < 0 || s.LightningMinutes > maxLightningMinutes {
		s.LightningMinutes = defaultLightningMinutes
	}

	s.LastQuiz = strings.TrimSpace(s.LastQuiz)
	if s.LastQuiz != filepath.Base(s.LastQuiz) || strings.ContainsAny(s.LastQuiz, `/\`) {
		s.LastQuiz = ""
	}
}

// LoadSettings reads settings.json. A missing file is not an error: defaults
// are used and saved on the first change.
func LoadSettings(path string) (Settings, error) {
	s := DefaultSettings()
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return DefaultSettings(), fmt.Errorf("settings %s: %w", path, err)
	}
	s.normalize()
	return s, nil
}

// SaveSettings writes settings.json atomically (temp file, then rename), so a
// crash mid-write cannot leave a half-written file.
func SaveSettings(path string, s Settings) error {
	s.normalize()
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
