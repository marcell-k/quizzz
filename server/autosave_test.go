package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAutosaveRoundTrip(t *testing.T) {
	g := boardGame(t)
	g.Settings.Rules.ReopenOnWrong = true
	anna := mustJoin(t, g, 0, "Anna", "")
	bela := mustJoin(t, g, 1, "Béla", "")

	t0 := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	if !g.SelectAt(0, 0, t0) {
		t.Fatal("select failed")
	}
	if g.Buzz(anna.ID) != BuzzWon {
		t.Fatal("buzz failed")
	}
	if !g.Minus(0) {
		t.Fatal("minus failed")
	}

	path := filepath.Join(t.TempDir(), "autosave.json")
	if err := SaveAutosave(path, g, t0.Add(5*time.Minute)); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temp file left: %v", err)
	}
	sv, err := LoadAutosave(path)
	if err != nil || sv == nil {
		t.Fatalf("load: %v, %v", sv, err)
	}

	g2 := NewGame(g.Settings, func() string { return "new" })
	if err := g2.RestoreState(*sv, testQuiz(t)); err != nil {
		t.Fatalf("restore: %v", err)
	}

	if len(g2.Players) != 2 {
		t.Fatalf("players = %d", len(g2.Players))
	}
	restored := g2.Player(anna.ID)
	if restored == nil || restored.Name != "Anna" || restored.Token != anna.Token {
		t.Fatalf("anna not restored: %+v", restored)
	}
	if restored.Online {
		t.Fatal("restored player is online")
	}
	if g2.Teams[0].Score != -1000 || g2.Teams[1].Score != 0 {
		t.Fatalf("scores = %d / %d", g2.Teams[0].Score, g2.Teams[1].Score)
	}
	if !g2.Board[0][0] {
		t.Fatal("used cell lost")
	}
	if g2.Current == nil || g2.Current.Topic != 0 || g2.Current.Q != 0 {
		t.Fatalf("current = %+v", g2.Current)
	}
	if g2.Round != RoundBoard {
		t.Fatalf("round = %v", g2.Round)
	}
	if g2.Buzzer != BuzzerOpen || g2.BuzzedBy != 0 {
		t.Fatalf("buzzer = %v by %d", g2.Buzzer, g2.BuzzedBy)
	}
	if !g2.Timer.Started || g2.Timer.Running() {
		t.Fatalf("timer not restored paused: %+v", g2.Timer)
	}
	if got := g2.Timer.Remaining; got != 25*time.Minute {
		t.Fatalf("remaining = %v, want 25m", got)
	}
	if len(g2.LockedPlayers) != 1 || !g2.LockedPlayers[anna.ID] {
		t.Fatalf("lockout lost: %+v", g2.LockedPlayers)
	}
	if len(g2.ScoreLog) != 1 {
		t.Fatalf("undo log = %+v", g2.ScoreLog)
	}
	if _, ok := g2.Undo(); !ok || g2.Teams[0].Score != 0 {
		t.Fatalf("undo after resume failed: %d", g2.Teams[0].Score)
	}
	if bela.Team != 1 {
		t.Fatalf("bela team = %d", bela.Team)
	}
}

func TestAutosaveBuzzedComesBackLocked(t *testing.T) {
	g := boardGame(t)
	anna := mustJoin(t, g, 0, "Anna", "")
	g.Select(0, 0)
	g.Buzz(anna.ID)

	path := filepath.Join(t.TempDir(), "autosave.json")
	if err := SaveAutosave(path, g, time.Now()); err != nil {
		t.Fatal(err)
	}
	sv, err := LoadAutosave(path)
	if err != nil || sv == nil {
		t.Fatalf("load: %v", err)
	}
	g2 := NewGame(g.Settings, func() string { return "new" })
	if err := g2.RestoreState(*sv, testQuiz(t)); err != nil {
		t.Fatal(err)
	}
	if g2.Buzzer != BuzzerLocked || g2.BuzzedBy != 0 {
		t.Fatalf("buzzer = %v by %d, want locked", g2.Buzzer, g2.BuzzedBy)
	}
	if g2.Current == nil {
		t.Fatal("current question lost")
	}
}

func TestAutosaveSkipsEmptyGame(t *testing.T) {
	g := NewGame(DefaultSettings(), func() string { return "tok" })
	path := filepath.Join(t.TempDir(), "autosave.json")
	if err := SaveAutosave(path, g, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("empty game saved: %v", err)
	}
}

func TestAutosaveLoadErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "autosave.json")
	if err := os.WriteFile(path, []byte("{nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadAutosave(path); err == nil {
		t.Fatal("broken json accepted")
	}
	if err := os.WriteFile(path, []byte(`{"version":99}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadAutosave(path); err == nil {
		t.Fatal("unknown version accepted")
	}
	sv, err := LoadAutosave(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil || sv != nil {
		t.Fatalf("missing file: %v, %v", sv, err)
	}
}

func TestRestoreRejectsMismatchedBoard(t *testing.T) {
	g := boardGame(t)
	anna := mustJoin(t, g, 0, "Anna", "")
	g.Select(0, 0)
	g.Buzz(anna.ID)
	path := filepath.Join(t.TempDir(), "autosave.json")
	if err := SaveAutosave(path, g, time.Now()); err != nil {
		t.Fatal(err)
	}
	sv, err := LoadAutosave(path)
	if err != nil || sv == nil {
		t.Fatalf("load: %v", err)
	}
	sv.Used = sv.Used[:2]

	g2 := NewGame(g.Settings, func() string { return "new" })
	if err := g2.RestoreState(*sv, testQuiz(t)); err == nil {
		t.Fatal("board size mismatch accepted")
	}
}

func TestRestoreRejectsBadRound(t *testing.T) {
	g := boardGame(t)
	g.Select(0, 0)
	path := filepath.Join(t.TempDir(), "autosave.json")
	if err := SaveAutosave(path, g, time.Now()); err != nil {
		t.Fatal(err)
	}
	sv, err := LoadAutosave(path)
	if err != nil || sv == nil {
		t.Fatalf("load: %v", err)
	}
	sv.Round = "bogus"
	g2 := NewGame(g.Settings, func() string { return "new" })
	if err := g2.RestoreState(*sv, testQuiz(t)); err == nil {
		t.Fatal("setup round accepted")
	}
}

func TestRemoveAutosave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "autosave.json")
	if err := RemoveAutosave(path); err != nil {
		t.Fatalf("remove missing: %v", err)
	}
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveAutosave(path); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("file not removed")
	}
}

func TestRestoreSkipsPlayersOfRemovedTeam(t *testing.T) {
	g := boardGame(t)
	anna := mustJoin(t, g, 0, "Anna", "")
	bela := mustJoin(t, g, 1, "Béla", "")

	path := filepath.Join(t.TempDir(), "autosave.json")
	if err := SaveAutosave(path, g, time.Now()); err != nil {
		t.Fatal(err)
	}
	sv, err := LoadAutosave(path)
	if err != nil || sv == nil {
		t.Fatalf("load: %v, %v", sv, err)
	}

	st := g.Settings
	st.Teams = copyTeams(st.Teams)
	st.Teams[0] = ""
	g2 := NewGame(st, func() string { return "new" })
	if err := g2.RestoreState(*sv, testQuiz(t)); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if g2.Player(anna.ID) != nil {
		t.Fatal("player of removed team restored")
	}
	if g2.Player(bela.ID) == nil {
		t.Fatal("other team's player lost")
	}
}
