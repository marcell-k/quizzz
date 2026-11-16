package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

func newToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ---- wire types (keep in sync with web/src/protocol.ts) ----

type inMsg struct {
	Type   string `json:"type"`
	Team   int    `json:"team"`
	Name   string `json:"name"`
	Token  string `json:"token"`
	Topic  int    `json:"topic"`
	Q      int    `json:"q"`
	File   string `json:"file"`
	Action string `json:"action"`
}

type statePlayer struct {
	ID     int    `json:"id"`
	Team   int    `json:"team"`
	Name   string `json:"name"`
	Online bool   `json:"online"`
}

type stateTeam struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

type stateTimer struct {
	Started   bool  `json:"started"`
	Paused    bool  `json:"paused"`
	Remaining int   `json:"remaining"`
	EndsAt    int64 `json:"endsAt"`
}

type stateCurrent struct {
	Topic    int    `json:"topic"`
	Q        int    `json:"q"`
	Points   int    `json:"points"`
	Text     string `json:"text"`
	HasImage bool   `json:"hasImage"`
	Image    string `json:"image"`
}

type stateMsg struct {
	Type     string        `json:"type"`
	Teams    []stateTeam   `json:"teams"`
	Players  []statePlayer `json:"players"`
	Buzzer   string        `json:"buzzer"`
	BuzzedBy int           `json:"buzzedBy"`
	Round    string        `json:"round"`
	Topics   []string      `json:"topics"`
	Points   []int         `json:"points"`
	Used     [][]bool      `json:"used"`
	Current  *stateCurrent `json:"current"`
	Timer    stateTimer    `json:"timer"`
}

type joinedMsg struct {
	Type  string `json:"type"`
	ID    int    `json:"id"`
	Token string `json:"token"`
	Team  int    `json:"team"`
	Name  string `json:"name"`
}

type errorMsg struct {
	Type string `json:"type"`
	Msg  string `json:"msg"` // error code, UI maps it to text
}

type plainMsg struct {
	Type string `json:"type"`
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// ---- hub: one goroutine owns game and clients ----

type client struct {
	send     chan []byte
	admin    bool
	playerID int // 0 = not joined. Touched by hub goroutine only.
}

type eventKind int

const (
	evConnect eventKind = iota
	evDisconnect
	evMessage
)

type event struct {
	kind eventKind
	c    *client
	data []byte
}

type Hub struct {
	events  chan event
	game    *Game
	clients map[*client]struct{}
	quizDir string
}

func NewHub(g *Game, quizDir string) *Hub {
	return &Hub{
		events:  make(chan event, 64),
		game:    g,
		clients: make(map[*client]struct{}),
		quizDir: quizDir,
	}
}

func (h *Hub) Run() {
	for ev := range h.events {
		switch ev.kind {
		case evConnect:
			h.clients[ev.c] = struct{}{}
			h.sendTo(ev.c, h.stateJSON())
		case evDisconnect:
			if _, ok := h.clients[ev.c]; ok {
				h.drop(ev.c)
				h.broadcastState()
			}
		case evMessage:
			h.handle(ev.c, ev.data)
		}
	}
}

// drop removes client, marks its player offline. Hub goroutine only.
func (h *Hub) drop(c *client) {
	if _, ok := h.clients[c]; !ok {
		return
	}
	delete(h.clients, c)
	close(c.send)
	if c.playerID != 0 {
		h.game.Disconnect(c.playerID)
		log.Printf("player left: id=%d", c.playerID)
		c.playerID = 0
	}
}

func (h *Hub) sendTo(c *client, b []byte) {
	select {
	case c.send <- b:
	default:
		log.Printf("client too slow, dropped")
		h.drop(c)
	}
}

func (h *Hub) broadcastState() {
	b := h.stateJSON()
	for c := range h.clients {
		h.sendTo(c, b)
	}
}

func timerState(t Timer, now time.Time) stateTimer {
	st := stateTimer{
		Started:   t.Started,
		Paused:    t.Paused(),
		Remaining: t.RemainingSecondsAt(now),
	}
	if t.Running() {
		st.EndsAt = t.EndsAt.UnixMilli()
	}
	return st
}

func (h *Hub) stateJSON() []byte {
	g := h.game
	now := time.Now()
	ps := make([]statePlayer, 0, len(g.Players))
	for _, p := range g.Players {
		ps = append(ps, statePlayer{ID: p.ID, Team: p.Team, Name: p.Name, Online: p.Online})
	}
	ts := make([]stateTeam, len(g.Teams))
	for i, t := range g.Teams {
		ts[i] = stateTeam{Name: t.Name, Score: t.Score}
	}

	topics := []string{}
	used := [][]bool{}
	points := g.Points
	if points == nil {
		points = []int{}
	}
	if g.Quiz != nil {
		topics = g.Quiz.Topics
	}
	if g.Board != nil {
		used = g.Board
	}

	var current *stateCurrent
	if g.Current != nil {
		text, hasImage := g.CurrentQuestion()
		image := ""
		if name := g.CurrentImage(); name != "" {
			image = "/media/quizzes/" + name
		}
		current = &stateCurrent{
			Topic:    g.Current.Topic,
			Q:        g.Current.Q,
			Points:   g.Current.Points,
			Text:     text,
			HasImage: hasImage,
			Image:    image,
		}
	}

	return mustJSON(stateMsg{
		Type:     "state",
		Teams:    ts,
		Players:  ps,
		Buzzer:   g.Buzzer.String(),
		BuzzedBy: g.BuzzedBy,
		Round:    g.Round.String(),
		Topics:   topics,
		Points:   points,
		Used:     used,
		Current:  current,
		Timer:    timerState(g.Timer, now),
	})
}

func (h *Hub) sendError(c *client, code string) {
	h.sendTo(c, mustJSON(errorMsg{Type: "error", Msg: code}))
}

func joinErrCode(err error) string {
	switch err {
	case ErrBadTeam:
		return "bad_team"
	case ErrBadName:
		return "bad_name"
	case ErrNameInUse:
		return "name_in_use"
	default:
		return "bad_message"
	}
}

func (h *Hub) teamName(i int) string {
	if i >= 0 && i < len(h.game.Teams) {
		return h.game.Teams[i].Name
	}
	return "?"
}

func (h *Hub) handle(c *client, data []byte) {
	var m inMsg
	if err := json.Unmarshal(data, &m); err != nil {
		h.sendError(c, "bad_message")
		return
	}
	switch m.Type {
	case "join":
		if c.admin {
			h.sendError(c, "forbidden")
			return
		}
		h.onJoin(c, m)
	case "buzz":
		if c.admin {
			h.sendError(c, "forbidden")
			return
		}
		h.onBuzz(c)
	case "cancel_buzz":
		if !c.admin {
			h.sendError(c, "forbidden")
			return
		}
		if h.game.CancelBuzz() {
			log.Printf("buzz cancelled")
			h.broadcastState()
		} else {
			h.sendError(c, "no_buzz")
		}
	case "timer":
		if !c.admin {
			h.sendError(c, "forbidden")
			return
		}
		switch m.Action {
		case "pause":
			h.game.Timer.Pause(time.Now())
		case "resume":
			h.game.Timer.Resume(time.Now())
		default:
			h.sendError(c, "bad_message")
			return
		}
		log.Printf("timer %s", m.Action)
		h.broadcastState()
	case "select":
		if !c.admin {
			h.sendError(c, "forbidden")
			return
		}
		h.onSelect(c, m)
	case "plus":
		if !c.admin {
			h.sendError(c, "forbidden")
			return
		}
		h.onScore(c, m.Team, 1)
	case "minus":
		if !c.admin {
			h.sendError(c, "forbidden")
			return
		}
		h.onScore(c, m.Team, -1)
	case "none":
		if !c.admin {
			h.sendError(c, "forbidden")
			return
		}
		if h.game.None() {
			log.Printf("no answer")
			h.broadcastState()
		} else {
			h.sendError(c, "no_question")
		}
	case "load_quiz":
		if !c.admin {
			h.sendError(c, "forbidden")
			return
		}
		h.onLoadQuiz(c, m.File)
	default:
		h.sendError(c, "bad_message")
	}
}

func (h *Hub) onSelect(c *client, m inMsg) {
	if h.game.Quiz == nil {
		h.sendError(c, "no_quiz")
		return
	}
	if h.game.Current != nil {
		h.sendError(c, "busy")
		return
	}
	if !h.game.SelectAt(m.Topic, m.Q, time.Now()) {
		h.sendError(c, "bad_cell")
		return
	}
	log.Printf("cell selected: topic=%d q=%d points=%d", m.Topic+1, m.Q+1, h.game.Current.Points)
	h.broadcastState()
}

func (h *Hub) onScore(c *client, team, sign int) {
	if h.game.Current == nil {
		h.sendError(c, "no_question")
		return
	}
	points := h.game.Current.Points
	var ok bool
	if sign > 0 {
		ok = h.game.Plus(team)
	} else {
		ok = h.game.Minus(team)
	}
	if !ok {
		h.sendError(c, "bad_team")
		return
	}
	log.Printf("score %+d: %s -> %d", sign*points, h.teamName(team), h.game.Teams[team].Score)
	h.broadcastState()
}

func (h *Hub) onLoadQuiz(c *client, file string) {
	path, err := quizPath(h.quizDir, file)
	if err != nil {
		h.sendError(c, "bad_file")
		return
	}
	q, warnings, err := LoadQuiz(path, defaultTopics, defaultQuestions)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotUTF8):
			h.sendError(c, "not_utf8")
		default:
			var fe *FormatError
			if errors.As(err, &fe) {
				log.Printf("quiz %s: %v", file, err)
				h.sendError(c, "quiz_format")
			} else {
				log.Printf("quiz %s: %v", file, err)
				h.sendError(c, "load_failed")
			}
		}
		return
	}
	for _, w := range warnings {
		log.Printf("quiz %s: %s", file, w)
	}
	h.game.SetQuiz(q)
	log.Printf("quiz loaded: %s (%d topics)", file, len(q.Topics))
	h.broadcastState()
}

func (h *Hub) onJoin(c *client, m inMsg) {
	if c.playerID != 0 {
		h.sendError(c, "already_joined")
		return
	}
	p, err := h.game.Join(m.Team, m.Name, m.Token)
	if err != nil {
		h.sendError(c, joinErrCode(err))
		return
	}
	// same player on another live connection: new one replaces it
	for o := range h.clients {
		if o != c && o.playerID == p.ID {
			o.playerID = 0
			h.drop(o)
		}
	}
	c.playerID = p.ID
	log.Printf("player joined: id=%d %s: %s", p.ID, h.teamName(p.Team), p.Name)
	h.sendTo(c, mustJSON(joinedMsg{Type: "joined", ID: p.ID, Token: p.Token, Team: p.Team, Name: p.Name}))
	h.broadcastState()
}

func (h *Hub) onBuzz(c *client) {
	p := h.game.Player(c.playerID)
	if c.playerID == 0 || p == nil {
		h.sendError(c, "not_joined")
		return
	}
	switch h.game.Buzz(p.ID) {
	case BuzzWon:
		log.Printf("BUZZ WIN: %s: %s", h.teamName(p.Team), p.Name)
		h.broadcastState()
	case BuzzLate:
		log.Printf("buzz late: %s: %s", h.teamName(p.Team), p.Name)
		h.sendTo(c, mustJSON(plainMsg{Type: "too_late"}))
	default:
		h.sendError(c, "not_joined")
	}
}

// ---- websocket plumbing ----

func (h *Hub) playerHandler(w http.ResponseWriter, r *http.Request) { h.serve(w, r, false) }
func (h *Hub) adminHandler(w http.ResponseWriter, r *http.Request)  { h.serve(w, r, true) }

func (h *Hub) serve(w http.ResponseWriter, r *http.Request, admin bool) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Printf("ws accept: %v", err)
		return
	}
	defer conn.CloseNow()
	conn.SetReadLimit(4096)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	c := &client{send: make(chan []byte, 16), admin: admin}
	h.events <- event{kind: evConnect, c: c}

	go func() {
		defer cancel()
		for msg := range c.send {
			wctx, wcancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Write(wctx, websocket.MessageText, msg)
			wcancel()
			if err != nil {
				return
			}
		}
		// channel closed by hub: end connection
		conn.Close(websocket.StatusPolicyViolation, "replaced")
	}()

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			break
		}
		h.events <- event{kind: evMessage, c: c, data: data}
	}
	h.events <- event{kind: evDisconnect, c: c}
}
