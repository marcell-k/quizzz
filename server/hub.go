package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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
	Type  string `json:"type"`
	Team  int    `json:"team"`
	Name  string `json:"name"`
	Token string `json:"token"`
}

type statePlayer struct {
	ID     int    `json:"id"`
	Team   int    `json:"team"`
	Name   string `json:"name"`
	Online bool   `json:"online"`
}

type stateMsg struct {
	Type     string        `json:"type"`
	Teams    []string      `json:"teams"`
	Players  []statePlayer `json:"players"`
	Buzzer   string        `json:"buzzer"`
	BuzzedBy int           `json:"buzzedBy"`
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
}

func NewHub(g *Game) *Hub {
	return &Hub{
		events:  make(chan event, 64),
		game:    g,
		clients: make(map[*client]struct{}),
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

func (h *Hub) stateJSON() []byte {
	g := h.game
	ps := make([]statePlayer, 0, len(g.Players))
	for _, p := range g.Players {
		ps = append(ps, statePlayer{ID: p.ID, Team: p.Team, Name: p.Name, Online: p.Online})
	}
	return mustJSON(stateMsg{
		Type:     "state",
		Teams:    g.Teams,
		Players:  ps,
		Buzzer:   g.Buzzer.String(),
		BuzzedBy: g.BuzzedBy,
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
		return h.game.Teams[i]
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
		}
	default:
		h.sendError(c, "bad_message")
	}
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
