package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()
	if s.Title != defaultTitle {
		t.Fatalf("title = %q", s.Title)
	}
	if len(s.Teams) != maxTeams {
		t.Fatalf("teams = %d, want %d", len(s.Teams), maxTeams)
	}
	if s.Topics != 5 || s.Questions != 6 {
		t.Fatalf("board = %dx%d", s.Topics, s.Questions)
	}
	if len(s.Points) != 6 || s.Points[5] != 8000 {
		t.Fatalf("points = %v", s.Points)
	}
	if s.GameDuration() != 30*time.Minute || s.LightningDuration() != 3*time.Minute {
		t.Fatalf("durations = %v / %v", s.GameDuration(), s.LightningDuration())
	}
	want := Rules{SubtractPoints: true, AutoOpen: true}
	if s.Rules != want {
		t.Fatalf("rules = %+v, want %+v", s.Rules, want)
	}
}

func TestLoadSettingsMissingFile(t *testing.T) {
	s, err := LoadSettings(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if s.Title != defaultTitle || s.Topics != defaultTopics {
		t.Fatalf("not defaults: %+v", s)
	}
}

func TestLoadSettingsKeepsRuleDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"rules":{"reopenOnWrong":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := LoadSettings(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !s.Rules.ReopenOnWrong {
		t.Fatal("reopenOnWrong not set")
	}
	if !s.Rules.SubtractPoints || !s.Rules.AutoOpen {
		t.Fatalf("defaults lost: %+v", s.Rules)
	}
}

func TestLoadSettingsBadJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{nope`), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := LoadSettings(path)
	if err == nil {
		t.Fatal("expected error")
	}
	if s.Topics != defaultTopics || len(s.Teams) != maxTeams {
		t.Fatalf("not defaults after error: %+v", s)
	}
}

func TestSettingsNormalize(t *testing.T) {
	s := Settings{
		Title:            "  Verseny  ",
		Teams:            []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"},
		Topics:           0,
		Questions:        12,
		Points:           []int{1000, -5},
		LightningPoints:  0,
		GameMinutes:      -1,
		LightningMinutes: 9000,
		LastQuiz:         "../secret.txt",
	}
	s.normalize()
	if s.Title != "Verseny" {
		t.Fatalf("title = %q", s.Title)
	}
	if len(s.Teams) != maxTeams || s.Teams[9] != "J" {
		t.Fatalf("teams = %v", s.Teams)
	}
	if s.Topics != defaultTopics || s.Questions != defaultQuestions {
		t.Fatalf("board = %dx%d", s.Topics, s.Questions)
	}
	if len(s.Points) != defaultQuestions || s.Points[0] != 1000 {
		t.Fatalf("points = %v", s.Points)
	}
	if s.LightningPoints != defaultLightningPoints {
		t.Fatalf("lightning points = %d", s.LightningPoints)
	}
	if s.GameMinutes != defaultGameMinutes || s.LightningMinutes != defaultLightningMinutes {
		t.Fatalf("minutes = %d / %d", s.GameMinutes, s.LightningMinutes)
	}
	if s.LastQuiz != "" {
		t.Fatalf("lastQuiz = %q", s.LastQuiz)
	}
}

func TestSettingsNormalizeKeepsCustomPoints(t *testing.T) {
	s := DefaultSettings()
	s.Questions = 3
	s.Points = []int{500, 1500, 9000}
	s.normalize()
	if len(s.Points) != 3 || s.Points[2] != 9000 {
		t.Fatalf("points = %v", s.Points)
	}
}

func TestSaveSettingsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	want := DefaultSettings()
	want.Title = "Helyi verseny"
	want.Topics, want.Questions = 4, 3
	want.Points = []int{1000, 2000, 5000}
	want.GameMinutes = 45
	want.LightningMinutes = 2
	want.LightningPoints = 2500
	want.Rules = Rules{ReopenOnWrong: true, AutoOpen: true}
	want.LastQuiz = "quiz2022.txt"

	if err := SaveSettings(path, want); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temp file left behind: %v", err)
	}

	got, err := LoadSettings(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Title != want.Title || got.Topics != want.Topics || got.Questions != want.Questions {
		t.Fatalf("roundtrip = %+v", got)
	}
	if got.GameMinutes != 45 || got.LightningMinutes != 2 || got.LightningPoints != 2500 {
		t.Fatalf("lengths = %+v", got)
	}
	if got.Rules != want.Rules {
		t.Fatalf("rules = %+v, want %+v", got.Rules, want.Rules)
	}
	if got.LastQuiz != "quiz2022.txt" {
		t.Fatalf("lastQuiz = %q", got.LastQuiz)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var disk map[string]json.RawMessage
	if err := json.Unmarshal(raw, &disk); err != nil {
		t.Fatalf("bad json on disk: %v", err)
	}
	if !strings.HasSuffix(string(raw), "\n") {
		t.Fatal("file does not end with newline")
	}
}

func TestNewGameUsesSettings(t *testing.T) {
	st := DefaultSettings()
	st.Teams = []string{"Piros", "Kék", "", "", ""}
	st.Questions = 3
	st.Points = []int{100, 200, 300}
	st.GameMinutes = 12
	st.LightningMinutes = 1
	st.LightningPoints = 700
	st.Rules = Rules{ReopenOnWrong: true, LockoutTeam: true, AutoOpen: false}

	g := NewGame(st, func() string { return "tok" })
	if len(g.Teams) != maxTeams || g.Teams[1].Name != "Kék" || g.Teams[2].Name != "" {
		t.Fatalf("teams = %+v", g.Teams)
	}
	if len(g.Points) != 3 || g.Points[2] != 300 {
		t.Fatalf("points = %v", g.Points)
	}
	if g.Timer.Remaining != 12*time.Minute {
		t.Fatalf("timer = %v, want 12m", g.Timer.Remaining)
	}
	if g.Settings.LightningPoints != 700 || g.Settings.LightningDuration() != time.Minute {
		t.Fatalf("lightning = %+v", g.Settings)
	}
	if g.Settings.Rules.AutoOpen {
		t.Fatal("autoOpen lost")
	}
}

func TestApplySettings(t *testing.T) {
	g := boardGame(t)
	p := mustJoin(t, g, 0, "Anna", "")
	g.Teams[0].Score = 500

	st := g.Settings
	st.Teams = []string{"Piros", "", "", "", ""}
	st.GameMinutes = 45
	st.Points = []int{10, 20, 30, 40, 50, 60}
	st.Rules.AutoOpen = false
	if unloaded, _ := g.ApplySettings(st); unloaded {
		t.Fatal("quiz unloaded without a size change")
	}
	if g.Teams[0].Name != "Piros" || g.Teams[0].Score != 500 {
		t.Fatalf("team = %+v", g.Teams[0])
	}
	if g.Player(p.ID) != p || len(g.Players) != 1 {
		t.Fatal("players lost")
	}
	if g.Timer.Remaining != 45*time.Minute {
		t.Fatalf("timer = %v", g.Timer.Remaining)
	}
	if len(g.Points) != 6 || g.Points[5] != 60 {
		t.Fatalf("points = %v", g.Points)
	}
	if g.Settings.Rules.AutoOpen {
		t.Fatal("rule change lost")
	}

	st.Topics = 4
	if unloaded, _ := g.ApplySettings(st); !unloaded {
		t.Fatal("quiz not unloaded on size change")
	}
	if g.Quiz != nil || g.Board != nil || g.Current != nil {
		t.Fatal("quiz state kept after size change")
	}
	if g.Teams[0].Score != 500 {
		t.Fatal("scores reset on settings change")
	}
}

func TestApplySettingsKeepsStartedTimer(t *testing.T) {
	g := boardGame(t)
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	g.SelectAt(0, 0, t0)
	st := g.Settings
	st.GameMinutes = 99
	g.ApplySettings(st)
	if !g.Timer.Started || !g.Timer.EndsAt.Equal(t0.Add(30*time.Minute)) {
		t.Fatalf("running timer touched: %+v", g.Timer)
	}
}

func TestSetQuizKeepsMatchingPoints(t *testing.T) {
	st := DefaultSettings()
	st.Points = []int{111, 222, 333, 444, 555, 666}
	g := NewGame(st, func() string { return "tok" })
	q := &Quiz{
		Topics:    []string{"T1", "T2"},
		Questions: [][]string{{"a", "b", "c", "d", "e", "f"}, {"g", "h", "i", "j", "k", "l"}},
		HasImage:  [][]bool{{}, {}},
		Images:    [][]string{{}, {}},
	}
	g.SetQuiz(q)
	if len(g.Points) != 6 || g.Points[0] != 111 {
		t.Fatalf("points reset on matching size: %v", g.Points)
	}
}

func TestSetQuizResetsPointsOnSizeChange(t *testing.T) {
	g := NewGame(DefaultSettings(), func() string { return "tok" })
	q := &Quiz{
		Topics:    []string{"T1"},
		Questions: [][]string{{"a", "b", "c"}},
		HasImage:  [][]bool{{}},
		Images:    [][]string{{}},
	}
	g.SetQuiz(q)
	if len(g.Points) != 3 || g.Settings.Questions != 3 {
		t.Fatalf("points = %v, settings questions = %d", g.Points, g.Settings.Questions)
	}
	if g.Timer.Remaining != g.Settings.GameDuration() {
		t.Fatalf("timer = %v", g.Timer.Remaining)
	}
}

func copyTeams(names []string) []string {
	return append([]string(nil), names...)
}

func TestApplySettingsRenameKeepsTeam(t *testing.T) {
	g := boardGame(t)
	anna := mustJoin(t, g, 0, "Anna", "")
	g.Teams[0].Score = 500
	g.ScoreLog = append(g.ScoreLog, ScoreChange{Team: 0, Delta: 500})

	st := g.Settings
	st.Teams = copyTeams(st.Teams)
	st.Teams[0] = "Orange"
	_, removed := g.ApplySettings(st)

	if len(removed) != 0 {
		t.Fatalf("removed players = %v", removed)
	}
	if g.Teams[0].Name != "Orange" || g.Teams[0].Score != 500 {
		t.Fatalf("team = %+v", g.Teams[0])
	}
	if g.Player(anna.ID) == nil || g.Player(anna.ID).Team != 0 {
		t.Fatal("player lost on rename")
	}
	if len(g.ScoreLog) != 1 {
		t.Fatalf("undo log = %+v", g.ScoreLog)
	}
}

func TestApplySettingsRemovesTeamMembers(t *testing.T) {
	g := boardGame(t)
	anna := mustJoin(t, g, 0, "Anna", "")
	bela := mustJoin(t, g, 1, "Béla", "")
	g.Teams[0].Score = 500
	g.ScoreLog = append(g.ScoreLog,
		ScoreChange{Team: 0, Delta: 500},
		ScoreChange{Team: 1, Delta: 100},
	)
	g.LockedPlayers = map[int]bool{anna.ID: true}
	g.BuzzedBy = anna.ID
	g.Buzzer = BuzzerBuzzed

	st := g.Settings
	st.Teams = copyTeams(st.Teams)
	st.Teams[0] = ""
	_, removed := g.ApplySettings(st)

	if len(removed) != 1 || removed[0] != anna.ID {
		t.Fatalf("removed = %v, want [%d]", removed, anna.ID)
	}
	if g.Player(anna.ID) != nil {
		t.Fatal("removed player kept")
	}
	if g.Player(bela.ID) == nil {
		t.Fatal("other team's player lost")
	}
	if g.Teams[0].Score != 0 {
		t.Fatalf("removed team score = %d", g.Teams[0].Score)
	}
	if len(g.ScoreLog) != 1 || g.ScoreLog[0].Team != 1 {
		t.Fatalf("undo log = %+v", g.ScoreLog)
	}
	if g.LockedPlayers[anna.ID] {
		t.Fatal("lockout kept for removed player")
	}
	if g.BuzzedBy != 0 || g.Buzzer != BuzzerLocked {
		t.Fatalf("buzz kept: buzzer = %v, by %d", g.Buzzer, g.BuzzedBy)
	}
}

func TestJoinStaleTokenAfterTeamRemoved(t *testing.T) {
	g := boardGame(t)
	anna := mustJoin(t, g, 0, "Anna", "")

	st := g.Settings
	st.Teams = copyTeams(st.Teams)
	st.Teams[0] = ""
	g.ApplySettings(st)

	if _, err := g.Join(0, "", anna.Token); err != ErrBadTeam {
		t.Fatalf("stale token join error = %v, want ErrBadTeam", err)
	}
	if len(g.Players) != 0 {
		t.Fatalf("stale player kept: %+v", g.Players)
	}
}
