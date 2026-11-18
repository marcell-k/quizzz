package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// autosaveFile is written next to the exe after every state change (plan D15).
const autosaveFile = "autosave.json"

const autosaveVersion = 1

type savedTeam struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

type savedPlayer struct {
	ID    int    `json:"id"`
	Team  int    `json:"team"`
	Name  string `json:"name"`
	Token string `json:"token"`
}

type savedCurrent struct {
	Topic  int `json:"topic"`
	Q      int `json:"q"`
	Points int `json:"points"`
}

// savedTimer keeps whole seconds and no wall clock: after a restart the timer
// comes back paused with the remaining time (plan section 8).
type savedTimer struct {
	Started   bool `json:"started"`
	Remaining int  `json:"remaining"`
}

type savedState struct {
	Version       int           `json:"version"`
	SavedAt       time.Time     `json:"savedAt"`
	QuizFile      string        `json:"quizFile"`
	Round         string        `json:"round"`
	Buzzer        string        `json:"buzzer"`
	Current       *savedCurrent `json:"current"`
	Teams         []savedTeam   `json:"teams"`
	Players       []savedPlayer `json:"players"`
	Used          [][]bool      `json:"used"`
	Timer         savedTimer    `json:"timer"`
	ScoreLog      []ScoreChange `json:"scoreLog"`
	LockedPlayers []int         `json:"lockedPlayers"`
	LockedTeams   []bool        `json:"lockedTeams"`
}

// HasProgress reports whether there is anything worth resuming: a loaded quiz,
// joined players, scores, or used cells. An untouched game is not saved.
func (g *Game) HasProgress() bool {
	if g.Quiz != nil || g.QuizFile != "" || len(g.Players) > 0 {
		return true
	}
	for _, t := range g.Teams {
		if t.Score != 0 {
			return true
		}
	}
	for _, row := range g.Board {
		for _, used := range row {
			if used {
				return true
			}
		}
	}
	return false
}

// snapshot saves everything plan section 8 lists as resumable.
func snapshotGame(g *Game, now time.Time) savedState {
	sv := savedState{
		Version:     autosaveVersion,
		SavedAt:     now.UTC(),
		QuizFile:    g.QuizFile,
		Round:       g.Round.String(),
		Buzzer:      g.Buzzer.String(),
		Timer:       savedTimer{Started: g.Timer.Started, Remaining: g.Timer.RemainingSecondsAt(now)},
		ScoreLog:    append([]ScoreChange(nil), g.ScoreLog...),
		LockedTeams: append([]bool(nil), g.LockedTeams...),
	}
	for _, t := range g.Teams {
		sv.Teams = append(sv.Teams, savedTeam{Name: t.Name, Score: t.Score})
	}
	for _, p := range g.Players {
		sv.Players = append(sv.Players, savedPlayer{ID: p.ID, Team: p.Team, Name: p.Name, Token: p.Token})
	}
	if g.Current != nil {
		sv.Current = &savedCurrent{Topic: g.Current.Topic, Q: g.Current.Q, Points: g.Current.Points}
	}
	for _, row := range g.Board {
		sv.Used = append(sv.Used, append([]bool(nil), row...))
	}
	for id := range g.LockedPlayers {
		sv.LockedPlayers = append(sv.LockedPlayers, id)
	}
	sort.Ints(sv.LockedPlayers)
	return sv
}

// SaveAutosave writes autosave.json atomically (temp file, then rename).
func SaveAutosave(path string, g *Game, now time.Time) error {
	if !g.HasProgress() {
		return nil
	}
	b, err := json.MarshalIndent(snapshotGame(g, now), "", "  ")
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

// LoadAutosave reads autosave.json. A missing file returns (nil, nil).
func LoadAutosave(path string) (*savedState, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var sv savedState
	if err := json.Unmarshal(b, &sv); err != nil {
		return nil, fmt.Errorf("autosave %s: %w", path, err)
	}
	if sv.Version != autosaveVersion {
		return nil, fmt.Errorf("autosave %s: unsupported version %d", path, sv.Version)
	}
	return &sv, nil
}

// RemoveAutosave deletes autosave.json, ignoring a missing file.
func RemoveAutosave(path string) error {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func parseRound(s string) (Round, bool) {
	switch s {
	case "setup":
		return RoundSetup, true
	case "board":
		return RoundBoard, true
	case "lightning":
		return RoundLightning, true
	case "ended":
		return RoundEnded, true
	default:
		return RoundSetup, false
	}
}

// RestoreState puts a saved game back on top of the current settings. The
// buzzer comes back locked when it was buzzed; the timer comes back paused.
func (g *Game) RestoreState(sv savedState, quiz *Quiz) error {
	if quiz == nil {
		return errors.New("resume: no quiz")
	}
	questions := 0
	if len(quiz.Questions) > 0 {
		questions = len(quiz.Questions[0])
	}
	if len(sv.Used) != len(quiz.Topics) {
		return fmt.Errorf("resume: board is %dxN, quiz has %d topics", len(sv.Used), len(quiz.Topics))
	}
	for _, row := range sv.Used {
		if len(row) != questions {
			return fmt.Errorf("resume: board row has %d cells, quiz has %d questions", len(row), questions)
		}
	}
	round, ok := parseRound(sv.Round)
	if !ok {
		return fmt.Errorf("resume: bad round %q", sv.Round)
	}
	buzzer := BuzzerLocked
	if sv.Buzzer == "open" {
		buzzer = BuzzerOpen
	}

	g.Quiz = quiz
	g.QuizFile = sv.QuizFile
	g.Board = make([][]bool, len(sv.Used))
	for t := range sv.Used {
		g.Board[t] = append([]bool(nil), sv.Used[t]...)
	}
	g.Points = append([]int(nil), g.Settings.Points...)
	if len(g.Points) != questions {
		g.Points = pointsForQuestions(questions)
	}

	for i := range g.Teams {
		g.Teams[i].Score = 0
		if i < len(sv.Teams) {
			g.Teams[i].Score = sv.Teams[i].Score
		}
	}

	g.Players = nil
	g.nextID = 1
	for _, sp := range sv.Players {
		if sp.ID <= 0 || sp.Team < 0 || sp.Team >= len(sv.Teams) || sp.Team >= len(g.Teams) || g.Teams[sp.Team].Name == "" {
			continue
		}
		name := strings.TrimSpace(sp.Name)
		if name == "" || len([]rune(name)) > maxNameLen {
			continue
		}
		g.Players = append(g.Players, &Player{
			ID:    sp.ID,
			Team:  sp.Team,
			Name:  name,
			Token: sp.Token,
		})
		if sp.ID >= g.nextID {
			g.nextID = sp.ID + 1
		}
	}

	g.Current = nil
	if sv.Current != nil && round == RoundBoard {
		t, q := sv.Current.Topic, sv.Current.Q
		if t >= 0 && t < len(g.Board) && q >= 0 && q < len(g.Board[t]) && g.Board[t][q] {
			g.Current = &Current{Topic: t, Q: q, Points: sv.Current.Points}
		}
	}

	g.Round = round
	g.Buzzer = buzzer
	g.BuzzedBy = 0

	remaining := time.Duration(sv.Timer.Remaining) * time.Second
	if remaining < 0 {
		remaining = 0
	}
	g.Timer = Timer{Started: sv.Timer.Started, Remaining: remaining}

	g.ScoreLog = append([]ScoreChange(nil), sv.ScoreLog...)

	g.LockedPlayers = make(map[int]bool, len(sv.LockedPlayers))
	for _, id := range sv.LockedPlayers {
		if g.Player(id) != nil {
			g.LockedPlayers[id] = true
		}
	}
	g.LockedTeams = make([]bool, len(g.Teams))
	for i, locked := range sv.LockedTeams {
		if i < len(g.LockedTeams) {
			g.LockedTeams[i] = locked
		}
	}
	return nil
}
