package main

import (
	"errors"
	"strings"
)

type BuzzerState int

const (
	BuzzerLocked BuzzerState = iota
	BuzzerOpen
	BuzzerBuzzed
)

func (b BuzzerState) String() string {
	switch b {
	case BuzzerOpen:
		return "open"
	case BuzzerBuzzed:
		return "buzzed"
	default:
		return "locked"
	}
}

type BuzzResult int

const (
	BuzzWon BuzzResult = iota
	BuzzLate
	BuzzUnknown
)

var (
	ErrBadTeam   = errors.New("bad team")
	ErrBadName   = errors.New("bad name")
	ErrNameInUse = errors.New("name in use")
)

const maxNameLen = 30

type Player struct {
	ID     int
	Team   int
	Name   string
	Token  string
	Online bool
}

type Game struct {
	Teams    []string // empty name = unused slot
	Players  []*Player
	Buzzer   BuzzerState
	BuzzedBy int // player ID, 0 = none
	nextID   int
	newToken func() string
}

// NewGame starts with buzzer OPEN. Temporary: step 5 adds board flow that opens/locks it.
func NewGame(teams []string, newToken func() string) *Game {
	return &Game{
		Teams:    teams,
		Buzzer:   BuzzerOpen,
		nextID:   1,
		newToken: newToken,
	}
}

func (g *Game) Player(id int) *Player {
	for _, p := range g.Players {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// Join order: token match, then same team+name, then new player.
func (g *Game) Join(team int, name, token string) (*Player, error) {
	if token != "" {
		for _, p := range g.Players {
			if p.Token == token {
				p.Online = true
				return p, nil
			}
		}
	}
	name = strings.TrimSpace(name)
	if team < 0 || team >= len(g.Teams) || g.Teams[team] == "" {
		return nil, ErrBadTeam
	}
	if n := len([]rune(name)); n == 0 || n > maxNameLen {
		return nil, ErrBadName
	}
	for _, p := range g.Players {
		if p.Team == team && strings.EqualFold(p.Name, name) {
			if p.Online {
				return nil, ErrNameInUse
			}
			// takeover from new device: rotate token, old device loses identity
			p.Token = g.newToken()
			p.Online = true
			return p, nil
		}
	}
	p := &Player{ID: g.nextID, Team: team, Name: name, Token: g.newToken(), Online: true}
	g.nextID++
	g.Players = append(g.Players, p)
	return p, nil
}

func (g *Game) Disconnect(id int) {
	if p := g.Player(id); p != nil {
		p.Online = false
	}
}

// Buzz is the single entry for all buzz sources. First call while OPEN wins.
func (g *Game) Buzz(playerID int) BuzzResult {
	if g.Player(playerID) == nil {
		return BuzzUnknown
	}
	if g.Buzzer != BuzzerOpen {
		return BuzzLate
	}
	g.Buzzer = BuzzerBuzzed
	g.BuzzedBy = playerID
	return BuzzWon
}

// CancelBuzz: BUZZED -> OPEN, no score change. Returns false if nobody buzzed.
func (g *Game) CancelBuzz() bool {
	if g.Buzzer != BuzzerBuzzed {
		return false
	}
	g.Buzzer = BuzzerOpen
	g.BuzzedBy = 0
	return true
}
