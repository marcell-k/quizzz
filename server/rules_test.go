package main

import (
	"testing"
	"time"
)

func rulesGame(t *testing.T) *Game {
	t.Helper()
	return boardGame(t)
}

func anyLockedTeam(g *Game) bool {
	for _, locked := range g.LockedTeams {
		if locked {
			return true
		}
	}
	return false
}

func TestMinusWithoutSubtractKeepsScore(t *testing.T) {
	g := rulesGame(t)
	g.Settings.Rules.SubtractPoints = false
	a := mustJoin(t, g, 0, "Anna", "")
	if !g.Select(0, 0) {
		t.Fatal("select failed")
	}
	g.Buzz(a.ID)
	if !g.Minus(0) {
		t.Fatal("minus failed")
	}
	if g.Teams[0].Score != 0 {
		t.Fatalf("score = %d, want 0", g.Teams[0].Score)
	}
	if g.Current != nil || g.Buzzer != BuzzerLocked {
		t.Fatalf("question not cleared: current=%v buzzer=%v", g.Current, g.Buzzer)
	}
}

func TestMinusReopenLocksOnlyAnswerer(t *testing.T) {
	g := rulesGame(t)
	g.Settings.Rules.ReopenOnWrong = true
	anna := mustJoin(t, g, 0, "Anna", "")
	bela := mustJoin(t, g, 0, "Béla", "")
	cili := mustJoin(t, g, 1, "Cili", "")

	if !g.Select(0, 0) {
		t.Fatal("select failed")
	}
	if g.Buzz(anna.ID) != BuzzWon {
		t.Fatal("first buzz failed")
	}
	if !g.Minus(0) {
		t.Fatal("minus failed")
	}
	if g.Teams[0].Score != -1000 {
		t.Fatalf("score = %d, want -1000", g.Teams[0].Score)
	}
	if g.Current == nil {
		t.Fatal("question cleared with reopening on")
	}
	if g.Buzzer != BuzzerOpen {
		t.Fatalf("buzzer = %v, want open", g.Buzzer)
	}
	if g.Buzz(anna.ID) != BuzzLate {
		t.Fatal("wrong answerer buzz accepted")
	}
	if g.Buzz(bela.ID) != BuzzWon {
		t.Fatal("same team mate blocked with player-only lockout")
	}
	g.CancelBuzz()
	if g.Buzz(cili.ID) != BuzzWon {
		t.Fatal("other team blocked")
	}
}

func TestMinusReopenLockoutTeam(t *testing.T) {
	g := rulesGame(t)
	g.Settings.Rules.ReopenOnWrong = true
	g.Settings.Rules.LockoutTeam = true
	anna := mustJoin(t, g, 0, "Anna", "")
	bela := mustJoin(t, g, 0, "Béla", "")
	cili := mustJoin(t, g, 1, "Cili", "")

	g.Select(0, 0)
	g.Buzz(anna.ID)
	g.Minus(0)
	if g.Buzz(bela.ID) != BuzzLate {
		t.Fatal("team mate not locked out")
	}
	if g.Buzz(cili.ID) != BuzzWon {
		t.Fatal("other team blocked")
	}
}

func TestPlusClearsLockouts(t *testing.T) {
	g := rulesGame(t)
	g.Settings.Rules.ReopenOnWrong = true
	anna := mustJoin(t, g, 0, "Anna", "")
	cili := mustJoin(t, g, 1, "Cili", "")

	g.Select(0, 0)
	g.Buzz(anna.ID)
	g.Minus(0)
	if len(g.LockedPlayers) == 0 {
		t.Fatal("no lockout after wrong answer")
	}
	g.Buzz(cili.ID)
	if !g.Plus(1) {
		t.Fatal("plus failed")
	}
	if g.Teams[1].Score != 1000 {
		t.Fatalf("score = %d", g.Teams[1].Score)
	}
	if len(g.LockedPlayers) != 0 || anyLockedTeam(g) {
		t.Fatal("lockouts kept after clearing the question")
	}
	if !g.Select(0, 1) || g.Buzz(anna.ID) != BuzzWon {
		t.Fatal("lockout survived into next question")
	}
}

func TestNoneClearsLockouts(t *testing.T) {
	g := rulesGame(t)
	g.Settings.Rules.ReopenOnWrong = true
	anna := mustJoin(t, g, 0, "Anna", "")
	g.Select(0, 0)
	g.Buzz(anna.ID)
	g.Minus(0)
	if !g.None() {
		t.Fatal("none failed")
	}
	if g.Current != nil || g.Buzzer != BuzzerLocked {
		t.Fatalf("buzzer = %v, current = %v", g.Buzzer, g.Current)
	}
	if len(g.LockedPlayers) != 0 {
		t.Fatal("lockouts kept after no-answer")
	}
}

func TestAutoOpenOffNeedsManualOpen(t *testing.T) {
	g := rulesGame(t)
	g.Settings.Rules.AutoOpen = false
	anna := mustJoin(t, g, 0, "Anna", "")
	if !g.Select(0, 0) {
		t.Fatal("select failed")
	}
	if g.Buzzer != BuzzerLocked {
		t.Fatalf("buzzer = %v, want locked", g.Buzzer)
	}
	if g.Buzz(anna.ID) != BuzzLate {
		t.Fatal("buzz accepted while locked")
	}
	if !g.OpenBuzz() {
		t.Fatal("manual open failed")
	}
	if g.Buzz(anna.ID) != BuzzWon {
		t.Fatal("buzz after manual open failed")
	}
	if g.OpenBuzz() {
		t.Fatal("open accepted while buzzed")
	}
	g.None()
	if g.OpenBuzz() {
		t.Fatal("open accepted without a question")
	}
}

func TestLightningHasNoLockouts(t *testing.T) {
	g := rulesGame(t)
	g.Settings.Rules.ReopenOnWrong = true
	g.Settings.Rules.LockoutTeam = true
	anna := mustJoin(t, g, 0, "Anna", "")
	if !g.StartLightning(time.Now()) {
		t.Fatal("lightning failed")
	}
	g.Buzz(anna.ID)
	if !g.Minus(0) {
		t.Fatal("minus failed")
	}
	if g.Teams[0].Score != -3000 {
		t.Fatalf("score = %d, want -3000", g.Teams[0].Score)
	}
	if g.Buzzer != BuzzerOpen || g.BuzzedBy != 0 {
		t.Fatalf("buzzer not reopened: %v by %d", g.Buzzer, g.BuzzedBy)
	}
	if len(g.LockedPlayers) != 0 || anyLockedTeam(g) {
		t.Fatal("lockout in lightning")
	}
}

func TestCancelBuzzKeepsLockout(t *testing.T) {
	g := rulesGame(t)
	g.Settings.Rules.ReopenOnWrong = true
	anna := mustJoin(t, g, 0, "Anna", "")
	bela := mustJoin(t, g, 0, "Béla", "")
	g.Select(0, 0)
	g.Buzz(anna.ID)
	g.Minus(0)
	g.Buzz(bela.ID)
	if !g.CancelBuzz() {
		t.Fatal("cancel failed")
	}
	if g.Buzz(anna.ID) != BuzzLate {
		t.Fatal("lockout lost after cancel")
	}
}
