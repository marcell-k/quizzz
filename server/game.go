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

type Round int

const (
	RoundSetup Round = iota
	RoundBoard
	RoundLightning
	RoundEnded
)

func (r Round) String() string {
	switch r {
	case RoundBoard:
		return "board"
	case RoundLightning:
		return "lightning"
	case RoundEnded:
		return "ended"
	default:
		return "setup"
	}
}

var (
	ErrBadTeam   = errors.New("bad team")
	ErrBadName   = errors.New("bad name")
	ErrNameInUse = errors.New("name in use")
)

const (
	maxNameLen = 30

	// Temporary settings defaults until step 10 adds settings.json.
	defaultTopics    = 5
	defaultQuestions = 6
)

var defaultPoints = []int{1000, 2000, 3000, 4000, 5000, 8000}

type Player struct {
	ID     int
	Team   int
	Name   string
	Token  string
	Online bool
}

// Team owns the score. Buzz belongs to Player (plan D17).
type Team struct {
	Name  string
	Score int
}

// Current is the open question. Board cell is already marked used.
type Current struct {
	Topic  int
	Q      int
	Points int
}

type Game struct {
	Teams    []Team // empty name = unused slot
	Players  []*Player
	Quiz     *Quiz
	Board    [][]bool // used[topic][q]
	Current  *Current
	Round    Round
	Points   []int // points per question index, temporary until settings
	Buzzer   BuzzerState
	BuzzedBy int // player ID, 0 = none
	nextID   int
	newToken func() string
}

// NewGame starts on the board with a locked buzzer. A quiz is loaded later.
func NewGame(teams []string, newToken func() string) *Game {
	ts := make([]Team, len(teams))
	for i, name := range teams {
		ts[i] = Team{Name: name}
	}
	return &Game{
		Teams:    ts,
		Points:   pointsForQuestions(defaultQuestions),
		Round:    RoundBoard,
		Buzzer:   BuzzerLocked,
		nextID:   1,
		newToken: newToken,
	}
}

func pointsForQuestions(n int) []int {
	ps := make([]int, n)
	for i := range ps {
		if i < len(defaultPoints) {
			ps[i] = defaultPoints[i]
		} else {
			ps[i] = 1000 * (i + 1)
		}
	}
	return ps
}

func (g *Game) Player(id int) *Player {
	for _, p := range g.Players {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// SetQuiz replaces the loaded quiz: clear board and current question.
// Scores and players stay (plan: new game clears scores, loading does not).
func (g *Game) SetQuiz(q *Quiz) {
	questions := 0
	if len(q.Questions) > 0 {
		questions = len(q.Questions[0])
	}
	g.Quiz = q
	g.Points = pointsForQuestions(questions)
	g.Board = make([][]bool, len(q.Topics))
	for t := range g.Board {
		g.Board[t] = make([]bool, questions)
	}
	g.Current = nil
	g.Buzzer = BuzzerLocked
	g.BuzzedBy = 0
	g.Round = RoundBoard
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
	if team < 0 || team >= len(g.Teams) || g.Teams[team].Name == "" {
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

// Select opens a board cell. Valid only on the board, with no current
// question, and only for an unused cell. Buzzer opens immediately.
func (g *Game) Select(topic, q int) bool {
	if g.Quiz == nil || g.Round != RoundBoard || g.Current != nil {
		return false
	}
	if topic < 0 || topic >= len(g.Board) || q < 0 || q >= len(g.Board[topic]) {
		return false
	}
	if g.Board[topic][q] || q >= len(g.Points) {
		return false
	}
	g.Board[topic][q] = true
	g.Current = &Current{Topic: topic, Q: q, Points: g.Points[q]}
	g.Buzzer = BuzzerOpen
	g.BuzzedBy = 0
	return true
}

// Plus awards the current question points to a team, then clears the question.
func (g *Game) Plus(team int) bool {
	if g.Current == nil || !g.validTeam(team) {
		return false
	}
	g.Teams[team].Score += g.Current.Points
	g.clearQuestion()
	return true
}

// Minus subtracts the current question points (rule default: on), then clears.
func (g *Game) Minus(team int) bool {
	if g.Current == nil || !g.validTeam(team) {
		return false
	}
	g.Teams[team].Score -= g.Current.Points
	g.clearQuestion()
	return true
}

// None is "Nincs válasz": clear the question, lock the buzzer. Cell stays used.
func (g *Game) None() bool {
	if g.Current == nil {
		return false
	}
	g.clearQuestion()
	return true
}

func (g *Game) clearQuestion() {
	g.Current = nil
	g.Buzzer = BuzzerLocked
	g.BuzzedBy = 0
}

func (g *Game) validTeam(team int) bool {
	return team >= 0 && team < len(g.Teams) && g.Teams[team].Name != ""
}

// CurrentQuestion returns the text and image flag of the open question.
func (g *Game) CurrentQuestion() (text string, hasImage bool) {
	if g.Current == nil || g.Quiz == nil {
		return "", false
	}
	t, q := g.Current.Topic, g.Current.Q
	if t < len(g.Quiz.Questions) && q < len(g.Quiz.Questions[t]) {
		text = g.Quiz.Questions[t][q]
	}
	if t < len(g.Quiz.HasImage) && q < len(g.Quiz.HasImage[t]) {
		hasImage = g.Quiz.HasImage[t][q]
	}
	return text, hasImage
}
