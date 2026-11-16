package main

import (
	"fmt"
	"path/filepath"
	"testing"
)

func testGame() *Game {
	n := 0
	return NewGame([]string{"A", "B", "C", "", ""}, func() string {
		n++
		return fmt.Sprintf("tok%d", n)
	})
}

func testQuiz(t *testing.T) *Quiz {
	t.Helper()
	q, _, err := LoadQuiz(filepath.Join("testdata", "sample.txt"), 5, 6)
	if err != nil {
		t.Fatalf("load test quiz: %v", err)
	}
	return q
}

func boardGame(t *testing.T) *Game {
	t.Helper()
	g := testGame()
	g.SetQuiz(testQuiz(t))
	return g
}

func mustJoin(t *testing.T, g *Game, team int, name, token string) *Player {
	t.Helper()
	p, err := g.Join(team, name, token)
	if err != nil {
		t.Fatalf("join %q: %v", name, err)
	}
	return p
}

func TestJoinNew(t *testing.T) {
	g := testGame()
	a := mustJoin(t, g, 0, "  Anna ", "")
	if a.Name != "Anna" || a.Team != 0 || !a.Online || a.Token == "" || a.ID == 0 {
		t.Fatalf("bad player: %+v", a)
	}
}

func TestJoinBad(t *testing.T) {
	g := testGame()
	cases := []struct {
		team int
		name string
		want error
	}{
		{-1, "x", ErrBadTeam},
		{5, "x", ErrBadTeam},
		{3, "x", ErrBadTeam}, // unused slot
		{0, "", ErrBadName},
		{0, "   ", ErrBadName},
		{0, "áááááááááááááááááááááááááááááá" + "á", ErrBadName}, // 31 runes
	}
	for _, c := range cases {
		if _, err := g.Join(c.team, c.name, ""); err != c.want {
			t.Errorf("join(%d,%q) = %v, want %v", c.team, c.name, err, c.want)
		}
	}
}

func TestJoinNameInUse(t *testing.T) {
	g := testGame()
	mustJoin(t, g, 0, "Anna", "")
	if _, err := g.Join(0, "ANNA", ""); err != ErrNameInUse {
		t.Fatalf("got %v, want ErrNameInUse", err)
	}
}

func TestSameNameOtherTeam(t *testing.T) {
	g := testGame()
	a := mustJoin(t, g, 0, "Anna", "")
	b := mustJoin(t, g, 1, "Anna", "")
	if a.ID == b.ID {
		t.Fatal("same ID across teams")
	}
}

func TestJoinTakeoverRotatesToken(t *testing.T) {
	g := testGame()
	a := mustJoin(t, g, 0, "Éva", "")
	id, old := a.ID, a.Token
	g.Disconnect(id)
	b := mustJoin(t, g, 0, " éva ", "")
	if b.ID != id {
		t.Fatalf("got new ID %d, want %d", b.ID, id)
	}
	if b.Token == old {
		t.Fatal("token not rotated")
	}
	if len(g.Players) != 1 {
		t.Fatalf("players = %d, want 1", len(g.Players))
	}
}

func TestJoinTokenRestore(t *testing.T) {
	g := testGame()
	a := mustJoin(t, g, 0, "Anna", "")
	// token wins even while online, ignores team/name
	b := mustJoin(t, g, 2, "", a.Token)
	if b.ID != a.ID || b.Team != 0 {
		t.Fatalf("token did not restore identity: %+v", b)
	}
}

func TestNewGameLocked(t *testing.T) {
	g := testGame()
	if g.Buzzer != BuzzerLocked {
		t.Fatalf("buzzer = %v, want locked", g.Buzzer)
	}
	if g.Round != RoundBoard {
		t.Fatalf("round = %v, want board", g.Round)
	}
	if g.Quiz != nil {
		t.Fatal("quiz set before load")
	}
}

func TestSetQuiz(t *testing.T) {
	g := testGame()
	g.SetQuiz(testQuiz(t))
	if len(g.Board) != 5 {
		t.Fatalf("board topics = %d, want 5", len(g.Board))
	}
	for i, row := range g.Board {
		if len(row) != 6 {
			t.Fatalf("board row %d = %d, want 6", i, len(row))
		}
	}
	if len(g.Points) != 6 || g.Points[5] != 8000 {
		t.Fatalf("points = %v", g.Points)
	}
	if g.Current != nil || g.Buzzer != BuzzerLocked {
		t.Fatalf("current = %+v, buzzer = %v", g.Current, g.Buzzer)
	}
}

func TestSetQuizCustomSize(t *testing.T) {
	g := testGame()
	q := &Quiz{
		Topics:    []string{"T1", "T2"},
		Questions: [][]string{{"a", "b", "c"}, {"d", "e", "f"}},
		HasImage:  [][]bool{{false, false, false}, {false, false, false}},
	}
	g.SetQuiz(q)
	if len(g.Board) != 2 || len(g.Board[0]) != 3 {
		t.Fatalf("board = %dx%d, want 2x3", len(g.Board), len(g.Board[0]))
	}
	if len(g.Points) != 3 || g.Points[2] != 3000 {
		t.Fatalf("points = %v, want 3 with 3000 last", g.Points)
	}
	if !g.Select(1, 2) || g.Current.Points != 3000 {
		t.Fatalf("custom size select failed: %+v", g.Current)
	}
}

func TestSelect(t *testing.T) {
	g := boardGame(t)
	if !g.Select(1, 2) {
		t.Fatal("select returned false")
	}
	if !g.Board[1][2] {
		t.Fatal("cell not marked used")
	}
	if g.Current == nil || g.Current.Topic != 1 || g.Current.Q != 2 || g.Current.Points != 3000 {
		t.Fatalf("current = %+v", g.Current)
	}
	if g.Buzzer != BuzzerOpen || g.BuzzedBy != 0 {
		t.Fatalf("buzzer = %v by %d, want open", g.Buzzer, g.BuzzedBy)
	}
}

func TestSelectInvalid(t *testing.T) {
	g := boardGame(t)
	if g.Select(-1, 0) || g.Select(0, -1) || g.Select(5, 0) || g.Select(0, 6) {
		t.Fatal("out of range select accepted")
	}
	if !g.Select(0, 0) {
		t.Fatal("valid select rejected")
	}
	if g.Select(0, 0) {
		t.Fatal("used cell accepted")
	}
	if g.Select(1, 1) {
		t.Fatal("second question accepted while busy")
	}
	if !g.None() {
		t.Fatal("none failed")
	}
	if g.Select(0, 0) {
		t.Fatal("used cell accepted after clear")
	}
	if !g.Select(0, 1) {
		t.Fatal("unused cell rejected after clear")
	}

	g2 := boardGame(t)
	g2.Round = RoundLightning
	if g2.Select(0, 0) {
		t.Fatal("select accepted outside board round")
	}

	g3 := testGame()
	if g3.Select(0, 0) {
		t.Fatal("select accepted without quiz")
	}
}

func TestSelectPoints(t *testing.T) {
	g := boardGame(t)
	if !g.Select(4, 5) {
		t.Fatal("select failed")
	}
	if g.Current.Points != 8000 {
		t.Fatalf("points = %d, want 8000", g.Current.Points)
	}
}

func TestPlus(t *testing.T) {
	g := boardGame(t)
	g.Select(0, 0)
	if !g.Plus(0) {
		t.Fatal("plus failed")
	}
	if g.Teams[0].Score != 1000 {
		t.Fatalf("score = %d, want 1000", g.Teams[0].Score)
	}
	if g.Current != nil || g.Buzzer != BuzzerLocked || g.BuzzedBy != 0 {
		t.Fatalf("question not cleared: %+v %v %d", g.Current, g.Buzzer, g.BuzzedBy)
	}
	if !g.Board[0][0] {
		t.Fatal("cell un-used after plus")
	}
}

func TestPlusNoCurrent(t *testing.T) {
	g := boardGame(t)
	if g.Plus(0) {
		t.Fatal("plus accepted without current question")
	}
}

func TestPlusBadTeam(t *testing.T) {
	g := boardGame(t)
	g.Select(0, 0)
	for _, team := range []int{-1, 3, 4, 5} {
		if g.Plus(team) {
			t.Fatalf("plus accepted team %d", team)
		}
	}
	if g.Current == nil {
		t.Fatal("invalid plus cleared the question")
	}
	if g.Teams[0].Score != 0 {
		t.Fatalf("score changed: %d", g.Teams[0].Score)
	}
}

func TestMinusNegative(t *testing.T) {
	g := boardGame(t)
	g.Select(0, 0)
	if !g.Minus(0) {
		t.Fatal("minus failed")
	}
	if g.Teams[0].Score != -1000 {
		t.Fatalf("score = %d, want -1000", g.Teams[0].Score)
	}
	if g.Current != nil || g.Buzzer != BuzzerLocked {
		t.Fatal("question not cleared")
	}
}

func TestScoreTeamIsolation(t *testing.T) {
	g := boardGame(t)
	g.Select(0, 0)
	g.Plus(0)
	g.Select(1, 0)
	g.Plus(1)
	if g.Teams[0].Score != 1000 || g.Teams[1].Score != 1000 {
		t.Fatalf("scores = %d, %d", g.Teams[0].Score, g.Teams[1].Score)
	}
	if g.Teams[2].Score != 0 {
		t.Fatalf("unrelated team score = %d", g.Teams[2].Score)
	}
}

func TestNone(t *testing.T) {
	g := boardGame(t)
	if g.None() {
		t.Fatal("none accepted without current question")
	}
	g.Select(2, 3)
	if !g.None() {
		t.Fatal("none failed")
	}
	if g.Current != nil || g.Buzzer != BuzzerLocked || g.BuzzedBy != 0 {
		t.Fatalf("question not cleared: %+v %v %d", g.Current, g.Buzzer, g.BuzzedBy)
	}
	if !g.Board[2][3] {
		t.Fatal("cell un-used after none")
	}
	if g.Teams[0].Score != 0 {
		t.Fatalf("score changed: %d", g.Teams[0].Score)
	}
}

func TestCancelBuzzKeepsCurrent(t *testing.T) {
	g := boardGame(t)
	a := mustJoin(t, g, 0, "Anna", "")
	g.Select(1, 1)
	if r := g.Buzz(a.ID); r != BuzzWon {
		t.Fatalf("buzz = %v", r)
	}
	if !g.CancelBuzz() {
		t.Fatal("cancel failed")
	}
	if g.Current == nil || g.Current.Topic != 1 || g.Current.Q != 1 {
		t.Fatalf("current lost: %+v", g.Current)
	}
	if g.Buzzer != BuzzerOpen || g.BuzzedBy != 0 {
		t.Fatalf("buzzer = %v by %d", g.Buzzer, g.BuzzedBy)
	}
	if g.CancelBuzz() {
		t.Fatal("second cancel returned true")
	}
}

func TestSetQuizResetsBoardKeepsScores(t *testing.T) {
	g := boardGame(t)
	g.Select(0, 0)
	g.Plus(0)
	g.SetQuiz(testQuiz(t))
	if g.Teams[0].Score != 1000 {
		t.Fatalf("score reset: %d", g.Teams[0].Score)
	}
	if g.Current != nil || g.Buzzer != BuzzerLocked {
		t.Fatal("current/buzzer not reset")
	}
	for ti, row := range g.Board {
		for qi, used := range row {
			if used {
				t.Fatalf("cell %d,%d still used", ti, qi)
			}
		}
	}
}

func TestCurrentQuestion(t *testing.T) {
	g := testGame()
	q := testQuiz(t)
	q.HasImage[1][2] = true
	g.SetQuiz(q)
	g.Select(1, 2)
	text, has := g.CurrentQuestion()
	if text != "Kérdés23" || !has {
		t.Fatalf("text = %q, hasImage = %v", text, has)
	}
	g.None()
	text, has = g.CurrentQuestion()
	if text != "" || has {
		t.Fatal("cleared question still returns text")
	}
}

func TestBuzzFirstWins(t *testing.T) {
	g := testGame()
	g.Buzzer = BuzzerOpen
	a := mustJoin(t, g, 0, "Anna", "")
	b := mustJoin(t, g, 1, "Bela", "")
	if r := g.Buzz(a.ID); r != BuzzWon {
		t.Fatalf("a: %v, want won", r)
	}
	if r := g.Buzz(b.ID); r != BuzzLate {
		t.Fatalf("b: %v, want late", r)
	}
	if r := g.Buzz(a.ID); r != BuzzLate {
		t.Fatalf("a again: %v, want late", r)
	}
	if g.Buzzer != BuzzerBuzzed || g.BuzzedBy != a.ID {
		t.Fatalf("state: %v by %d", g.Buzzer, g.BuzzedBy)
	}
}

func TestBuzzLocked(t *testing.T) {
	g := testGame()
	a := mustJoin(t, g, 0, "Anna", "")
	if r := g.Buzz(a.ID); r != BuzzLate {
		t.Fatalf("got %v, want late", r)
	}
	if g.BuzzedBy != 0 {
		t.Fatal("BuzzedBy set while locked")
	}
}

func TestBuzzUnknownPlayer(t *testing.T) {
	g := testGame()
	before := g.Buzzer
	if r := g.Buzz(99); r != BuzzUnknown {
		t.Fatalf("got %v, want unknown", r)
	}
	if g.Buzzer != before {
		t.Fatal("unknown buzz changed state")
	}
}

func TestCancelBuzz(t *testing.T) {
	g := testGame()
	g.Buzzer = BuzzerOpen
	a := mustJoin(t, g, 0, "Anna", "")
	b := mustJoin(t, g, 1, "Bela", "")
	if g.CancelBuzz() {
		t.Fatal("cancel with no buzz returned true")
	}
	g.Buzz(a.ID)
	if !g.CancelBuzz() {
		t.Fatal("cancel returned false")
	}
	if g.Buzzer != BuzzerOpen || g.BuzzedBy != 0 {
		t.Fatalf("state: %v by %d", g.Buzzer, g.BuzzedBy)
	}
	if r := g.Buzz(b.ID); r != BuzzWon {
		t.Fatalf("b after cancel: %v, want won", r)
	}
}
