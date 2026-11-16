package main

import (
	"fmt"
	"testing"
)

func testGame() *Game {
	n := 0
	return NewGame([]string{"A", "B", "C", "", ""}, func() string {
		n++
		return fmt.Sprintf("tok%d", n)
	})
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

func TestBuzzFirstWins(t *testing.T) {
	g := testGame()
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
	g.Buzzer = BuzzerLocked
	if r := g.Buzz(a.ID); r != BuzzLate {
		t.Fatalf("got %v, want late", r)
	}
	if g.BuzzedBy != 0 {
		t.Fatal("BuzzedBy set while locked")
	}
}

func TestBuzzUnknownPlayer(t *testing.T) {
	g := testGame()
	if r := g.Buzz(99); r != BuzzUnknown {
		t.Fatalf("got %v, want unknown", r)
	}
	if g.Buzzer != BuzzerOpen {
		t.Fatal("unknown buzz changed state")
	}
}

func TestCancelBuzz(t *testing.T) {
	g := testGame()
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
