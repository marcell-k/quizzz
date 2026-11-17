package main

import (
	"errors"
	"sort"
	"strings"
	"time"
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

const maxNameLen = 30

// Timer is a display-only countdown. Clients render EndsAt; the server owns
// pause and resume. Zero does not stop the game.
type Timer struct {
	Started   bool
	Remaining time.Duration
	EndsAt    time.Time
}

func NewTimer(d time.Duration) Timer {
	if d < 0 {
		d = 0
	}
	return Timer{Remaining: d}
}

func (t *Timer) Reset(d time.Duration) {
	*t = NewTimer(d)
}

// Running reports whether EndsAt is set.
func (t *Timer) Running() bool {
	return !t.EndsAt.IsZero()
}

// Start begins the countdown once. Later calls do not reset it.
func (t *Timer) Start(now time.Time) {
	if t.Started {
		return
	}
	t.Started = true
	t.EndsAt = now.Add(t.Remaining)
}

// Pause stores the exact remaining time and clears EndsAt.
func (t *Timer) Pause(now time.Time) bool {
	if !t.Running() {
		return false
	}
	t.Remaining = t.RemainingAt(now)
	t.EndsAt = time.Time{}
	return true
}

// Resume restarts a paused timer. A zero or unstarted timer stays stopped.
func (t *Timer) Resume(now time.Time) bool {
	if !t.Started || t.Running() || t.Remaining <= 0 {
		return false
	}
	t.EndsAt = now.Add(t.Remaining)
	return true
}

func (t Timer) Paused() bool {
	return t.Started && t.EndsAt.IsZero()
}

func (t Timer) RemainingAt(now time.Time) time.Duration {
	if t.Running() {
		d := t.EndsAt.Sub(now)
		if d > 0 {
			return d
		}
		return 0
	}
	if t.Remaining > 0 {
		return t.Remaining
	}
	return 0
}

// RemainingSecondsAt rounds up, so 0.1 s left still displays as one second.
func (t Timer) RemainingSecondsAt(now time.Time) int {
	d := t.RemainingAt(now)
	if d <= 0 {
		return 0
	}
	s := int(d / time.Second)
	if d%time.Second != 0 {
		s++
	}
	return s
}

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

// ScoreChange is one entry of the undo log. Delta is what was added to the
// team score, so undo subtracts it again.
type ScoreChange struct {
	Team  int `json:"team"`
	Delta int `json:"delta"`
}

// Standing is one row of the final results. Ties share a rank.
type Standing struct {
	Team  int
	Rank  int
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
	QuizFile string   // loaded file name with .txt, for autosave/reload
	Board    [][]bool // used[topic][q]
	Current  *Current
	Round    Round
	Points   []int // points per question index, mirrors Settings.Points
	Buzzer   BuzzerState
	BuzzedBy int // player ID, 0 = none
	Timer    Timer
	ScoreLog []ScoreChange

	// Lockouts for the current question (plan section 8). Only filled when the
	// wrong-answer reopening rule is on. Reset with every new/cleared question.
	LockedTeams   []bool
	LockedPlayers map[int]bool

	// Settings is settings.json: title, teams, board size, points, lengths,
	// rule toggles, last quiz.
	Settings Settings

	nextID   int
	newToken func() string
}

// NewGame starts on the board with a locked buzzer. A quiz is loaded later.
func NewGame(st Settings, newToken func() string) *Game {
	st.normalize()
	ts := make([]Team, len(st.Teams))
	for i, name := range st.Teams {
		ts[i] = Team{Name: name}
	}
	return &Game{
		Settings:      st,
		Teams:         ts,
		Points:        append([]int(nil), st.Points...),
		Round:         RoundBoard,
		Buzzer:        BuzzerLocked,
		Timer:         NewTimer(st.GameDuration()),
		LockedTeams:   make([]bool, len(ts)),
		LockedPlayers: make(map[int]bool),
		nextID:        1,
		newToken:      newToken,
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
	g.QuizFile = ""
	if len(g.Points) != questions {
		g.Points = pointsForQuestions(questions)
		g.Settings.Questions = questions
		g.Settings.Points = append([]int(nil), g.Points...)
	}
	g.Board = make([][]bool, len(q.Topics))
	for t := range g.Board {
		g.Board[t] = make([]bool, questions)
	}
	g.Current = nil
	g.Buzzer = BuzzerLocked
	g.BuzzedBy = 0
	g.resetLockouts()
	g.Round = RoundBoard
	g.Timer.Reset(g.Settings.GameDuration())
}

// ApplySettings replaces the settings. Team names update in place, so scores
// and players stay attached. A board size that no longer matches the loaded
// quiz unloads the quiz: the teacher must load a matching file again.
func (g *Game) ApplySettings(st Settings) (quizUnloaded bool) {
	st.normalize()
	for i := range g.Teams {
		if i < len(st.Teams) {
			g.Teams[i].Name = st.Teams[i]
		}
	}
	g.Settings = st
	g.Points = append([]int(nil), st.Points...)
	if g.Round == RoundBoard && !g.Timer.Started {
		g.Timer.Reset(st.GameDuration())
	}
	if g.Quiz == nil {
		return false
	}
	topics := len(g.Quiz.Topics)
	questions := 0
	if len(g.Quiz.Questions) > 0 {
		questions = len(g.Quiz.Questions[0])
	}
	if topics == st.Topics && questions == st.Questions {
		return false
	}
	g.Quiz = nil
	g.QuizFile = ""
	g.Board = nil
	g.Current = nil
	g.Buzzer = BuzzerLocked
	g.BuzzedBy = 0
	g.resetLockouts()
	return true
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
	p := g.Player(playerID)
	if p == nil {
		return BuzzUnknown
	}
	if g.Buzzer != BuzzerOpen || g.isLockedOut(p) {
		return BuzzLate
	}
	g.Buzzer = BuzzerBuzzed
	g.BuzzedBy = playerID
	return BuzzWon
}

// isLockedOut reports whether the player sat out the current question (wrong
// answer with reopening on, whole-team lockout optional).
func (g *Game) isLockedOut(p *Player) bool {
	if g.LockedPlayers[p.ID] {
		return true
	}
	return p.Team >= 0 && p.Team < len(g.LockedTeams) && g.LockedTeams[p.Team]
}

// resetLockouts clears per-question lockouts. Safe on a zero value.
func (g *Game) resetLockouts() {
	g.LockedPlayers = make(map[int]bool)
	g.LockedTeams = make([]bool, len(g.Teams))
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
	return g.SelectAt(topic, q, time.Now())
}

// SelectAt is Select with an explicit clock, used by tests and the hub.
// The first successful pick starts the round-1 timer.
func (g *Game) SelectAt(topic, q int, now time.Time) bool {
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
	g.resetLockouts()
	g.BuzzedBy = 0
	if g.Settings.Rules.AutoOpen {
		g.Buzzer = BuzzerOpen
	} else {
		g.Buzzer = BuzzerLocked
	}
	g.Timer.Start(now)
	return true
}

// OpenBuzz opens the buzzer manually when auto-open is off. Returns false when
// there is nothing to open.
func (g *Game) OpenBuzz() bool {
	if g.Round != RoundBoard || g.Current == nil || g.Buzzer != BuzzerLocked {
		return false
	}
	g.Buzzer = BuzzerOpen
	return true
}

// StartLightning switches from the board to lightning questions. Allowed any
// time on the board, even with an open question: the cell stays used and the
// current question is discarded. The buzzer opens and the lightning clock
// resets and starts.
func (g *Game) StartLightning(now time.Time) bool {
	if g.Round != RoundBoard {
		return false
	}
	g.Round = RoundLightning
	g.Current = nil
	g.BuzzedBy = 0
	g.Buzzer = BuzzerOpen
	g.resetLockouts()
	g.Timer.Reset(g.Settings.LightningDuration())
	g.Timer.Start(now)
	return true
}

// EndGame stops the game: buzzer locked, current question cleared, timer
// paused. Results come from Standings; history is added in a later step.
func (g *Game) EndGame(now time.Time) bool {
	if g.Round == RoundEnded {
		return false
	}
	g.Round = RoundEnded
	g.Current = nil
	g.BuzzedBy = 0
	g.Buzzer = BuzzerLocked
	g.resetLockouts()
	g.Timer.Pause(now)
	return true
}

// StartNewGame resets scores, board, current question, undo log and timer.
// Teams, players, the loaded quiz and settings stay. Allowed only after the
// game has ended, so a running game cannot be wiped by accident.
func (g *Game) StartNewGame() bool {
	if g.Round != RoundEnded {
		return false
	}
	for i := range g.Teams {
		g.Teams[i].Score = 0
	}
	for t := range g.Board {
		for q := range g.Board[t] {
			g.Board[t][q] = false
		}
	}
	g.Current = nil
	g.Round = RoundBoard
	g.Buzzer = BuzzerLocked
	g.BuzzedBy = 0
	g.ScoreLog = nil
	g.resetLockouts()
	g.Timer.Reset(g.Settings.GameDuration())
	return true
}

// Plus awards the current question points to a team, then clears the question.
// In lightning it awards LightningPoints and reopens the buzzer instead.
func (g *Game) Plus(team int) bool { return g.score(team, 1) }

// Minus marks a wrong answer. Points are subtracted only when the setting is
// on. With reopening off the question clears; with reopening on both the
// question and the buzzer stay, and the answerer sits out (whole team
// optionally). In lightning it only subtracts and reopens.
func (g *Game) Minus(team int) bool { return g.score(team, -1) }

// score handles both rounds. On the board it uses Current question points. In
// lightning it uses LightningPoints and reopens the buzzer right after every
// press; no lockouts there.
func (g *Game) score(team, sign int) bool {
	if !g.validTeam(team) {
		return false
	}
	if g.Round == RoundLightning {
		g.addScore(team, sign*g.Settings.LightningPoints)
		g.Buzzer = BuzzerOpen
		g.BuzzedBy = 0
		return true
	}
	if g.Current == nil {
		return false
	}
	if sign > 0 || g.Settings.Rules.SubtractPoints {
		g.addScore(team, sign*g.Current.Points)
	}
	if sign < 0 && g.Settings.Rules.ReopenOnWrong {
		g.reopenAfterWrong(team)
		return true
	}
	g.clearQuestion()
	return true
}

// reopenAfterWrong keeps the question open and benches the answering player,
// or the whole team when that rule is on. Falls back to the pressed team when
// nobody buzzed.
func (g *Game) reopenAfterWrong(team int) {
	by := g.BuzzedBy
	g.Buzzer = BuzzerOpen
	g.BuzzedBy = 0
	p := g.Player(by)
	if g.Settings.Rules.LockoutTeam {
		locked := team
		if p != nil {
			locked = p.Team
		}
		if locked >= 0 && locked < len(g.LockedTeams) {
			g.LockedTeams[locked] = true
		}
		return
	}
	if p != nil {
		g.LockedPlayers[p.ID] = true
	}
}

// addScore applies delta and records it for undo. A zero delta is not logged.
func (g *Game) addScore(team, delta int) {
	g.Teams[team].Score += delta
	if delta != 0 {
		g.ScoreLog = append(g.ScoreLog, ScoreChange{Team: team, Delta: delta})
	}
}

// SetScore sets a team score directly (manual edit). The difference is logged
// so Undo can restore the previous value.
func (g *Game) SetScore(team, value int) bool {
	if !g.validTeam(team) {
		return false
	}
	g.addScore(team, value-g.Teams[team].Score)
	return true
}

// Undo reverts the last score change. It does not reopen the question, un-use
// the cell, or change the buzzer. Returns false when the log is empty.
func (g *Game) Undo() (ScoreChange, bool) {
	n := len(g.ScoreLog)
	if n == 0 {
		return ScoreChange{}, false
	}
	c := g.ScoreLog[n-1]
	g.ScoreLog = g.ScoreLog[:n-1]
	if g.validTeam(c.Team) {
		g.Teams[c.Team].Score -= c.Delta
	}
	return c, true
}

// Standings returns active teams by score, highest first. Ties share a rank:
// 100, 100, 50 -> 1, 1, 3. Unused team slots are skipped.
func (g *Game) Standings() []Standing {
	out := make([]Standing, 0, len(g.Teams))
	for i, t := range g.Teams {
		if t.Name == "" {
			continue
		}
		out = append(out, Standing{Team: i, Score: t.Score})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Score > out[j].Score
	})
	for i := range out {
		if i > 0 && out[i].Score == out[i-1].Score {
			out[i].Rank = out[i-1].Rank
		} else {
			out[i].Rank = i + 1
		}
	}
	return out
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
	g.resetLockouts()
}

func (g *Game) validTeam(team int) bool {
	return team >= 0 && team < len(g.Teams) && g.Teams[team].Name != ""
}

// CurrentImage returns the image file name of the open question, or "".
func (g *Game) CurrentImage() string {
	if g.Current == nil || g.Quiz == nil {
		return ""
	}
	t, q := g.Current.Topic, g.Current.Q
	if t < len(g.Quiz.Images) && q < len(g.Quiz.Images[t]) {
		return g.Quiz.Images[t][q]
	}
	return ""
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
