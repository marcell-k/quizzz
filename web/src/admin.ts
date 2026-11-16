import { t, errorText } from "./strings";
import { connect } from "./ws";
import { teamColor } from "./colors";
import { formatClock, remainingSeconds, startTick } from "./timer";
import type { BuzzerState, ServerMsg, StateMsg, TimerInfo } from "./protocol";

function el<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  cls?: string,
  text?: string,
): HTMLElementTagNameMap[K] {
  const e = document.createElement(tag);
  if (cls) e.className = cls;
  if (text !== undefined) e.textContent = text;
  return e;
}

function panel(title: string, ...children: (Node | string)[]): HTMLElement {
  const s = el("section", "panel");
  s.append(el("h2", "", title), ...children);
  return s;
}

function fmt(n: number): string {
  return String(n).replace(/\B(?=(\d{3})+(?!\d))/g, ".");
}

const buzzerLabels: Record<BuzzerState, string> = {
  locked: t.buzzerLocked,
  open: t.buzzerOpen,
  buzzed: t.buzzerBuzzed,
};

// ---- DOM ----

const h1 = el("h1", "", `${t.appTitle}: ${t.admin}`);
const status = el("p", "status");
const connEl = el("p", "conn");
const url = el("p", "hint");
const qr = el("img");
qr.src = "/qr.png";
qr.width = 220;
qr.height = 220;

const quizSel = el("select");
const loadBtn = el("button", "btn", t.loadQuiz);
const quizMsg = el("p", "err");
quizSel.disabled = true;
loadBtn.disabled = true;

const noQuiz = el("p", "hint", t.noQuiz);
noQuiz.classList.add("hidden");
const board = el("div", "board");

const qBox = el("div", "qbox");
const noneBtn = el("button", "btn", t.noAnswer);
const cancelBtn = el("button", "btn", t.cancelBuzz);
const opMsg = el("p", "err");
noneBtn.disabled = true;
cancelBtn.disabled = true;
const actions = el("div", "actions");
actions.append(noneBtn, cancelBtn);

const teamList = el("div", "team-list-admin");
const playerList = el("ul", "player-list");

const timerClock = el("p", "timer-clock", "00:00");
const timerBtn = el("button", "btn timer-btn", t.pauseTimer);
timerBtn.disabled = true;

const boardPanel = panel(t.board, noQuiz, board);
boardPanel.classList.add("wide");
const questionPanel = panel(t.currentQuestion, qBox, actions, opMsg);
questionPanel.classList.add("wide");
const teamsPanel = panel(t.teams, teamList);
teamsPanel.classList.add("wide");

const grid = el("div", "admin");
grid.append(
  panel(t.scanToJoin, qr, url, status, connEl),
  panel(t.quizFile, quizSel, loadBtn, quizMsg),
  panel(t.gameTime, timerClock, timerBtn),
  boardPanel,
  questionPanel,
  teamsPanel,
  panel(t.players, playerList),
);
document.body.replaceChildren(h1, grid);

// ---- state ----

let peekFor = "";
let loadPending = false;
let timerState: TimerInfo | null = null;

function renderTimer(): void {
  if (!timerState) {
    timerClock.textContent = "--:--";
    timerBtn.disabled = true;
    return;
  }
  timerClock.textContent = formatClock(remainingSeconds(timerState));
  timerBtn.disabled = !timerState.started;
  timerBtn.textContent = timerState.paused ? t.resumeTimer : t.pauseTimer;
}

function renderQuestion(m: StateMsg): void {
  const cur = m.current;
  if (!cur) {
    qBox.replaceChildren(el("p", "hint", t.noCurrent));
    return;
  }
  const key = `${cur.topic}:${cur.q}`;
  const parts: HTMLElement[] = [
    el("p", "qmeta", `${m.topics[cur.topic] ?? "?"} · ${fmt(cur.points)}`),
  ];
  if (peekFor === key) {
    parts.push(el("p", "qtext", cur.text));
    const hide = el("button", "link", t.hide);
    hide.onclick = () => {
      peekFor = "";
      renderQuestion(m);
    };
    parts.push(hide);
  } else {
    const peek = el("button", "btn peek", t.peek);
    peek.onclick = () => {
      peekFor = key;
      renderQuestion(m);
    };
    parts.push(peek);
  }
  if (cur.hasImage) parts.push(el("p", "hint", t.image));
  qBox.replaceChildren(...parts);
}

function renderBoard(m: StateMsg): void {
  const n = m.points.length;
  if (m.topics.length === 0 || n === 0) {
    board.replaceChildren();
    board.style.gridTemplateColumns = "";
    noQuiz.classList.remove("hidden");
    return;
  }
  noQuiz.classList.add("hidden");
  board.style.gridTemplateColumns = `minmax(7rem, 1.2fr) repeat(${n}, minmax(3.2rem, 1fr))`;
  const cells: HTMLElement[] = [el("div", "col", "")];
  for (let q = 0; q < n; q++) cells.push(el("div", "col", `#${q + 1}`));
  m.topics.forEach((topic, ti) => {
    cells.push(el("div", "topic", topic));
    for (let q = 0; q < n; q++) {
      const used = m.used[ti]?.[q] ?? false;
      const b = el("button", "cell", fmt(m.points[q] ?? 0));
      b.type = "button";
      b.disabled = used || m.current !== null;
      if (used) b.classList.add("used");
      b.onclick = () => {
        opMsg.textContent = "";
        conn.send({ type: "select", topic: ti, q });
      };
      cells.push(b);
    }
  });
  board.replaceChildren(...cells);
}

function renderTeams(m: StateMsg): void {
  const rows: HTMLElement[] = [];
  m.teams.forEach((team, i) => {
    if (team.name === "") return;
    const row = el("div", "team-row");
    const dot = el("span", "dot");
    dot.style.background = teamColor(i);
    const plus = el("button", "score-btn", "+");
    plus.type = "button";
    plus.disabled = m.current === null;
    plus.onclick = () => {
      opMsg.textContent = "";
      conn.send({ type: "plus", team: i });
    };
    const minus = el("button", "score-btn", "−");
    minus.type = "button";
    minus.disabled = m.current === null;
    minus.onclick = () => {
      opMsg.textContent = "";
      conn.send({ type: "minus", team: i });
    };
    row.append(dot, el("span", "tname", team.name), el("span", "score", fmt(team.score)), plus, minus);
    rows.push(row);
  });
  teamList.replaceChildren(...rows);
}

function renderPlayers(m: StateMsg): void {
  playerList.replaceChildren(
    ...m.players.map((p) => {
      const li = el("li");
      li.textContent = `${m.teams[p.team]?.name ?? "?"}: ${p.name} (${p.online ? t.online : t.offline})`;
      return li;
    }),
  );
}

function render(m: StateMsg): void {
  timerState = m.timer;
  renderTimer();
  const label = buzzerLabels[m.buzzer];
  const by = m.players.find((p) => p.id === m.buzzedBy);
  status.textContent = by
    ? `${t.buzzer}: ${label} — ${m.teams[by.team]?.name ?? "?"}: ${by.name}`
    : `${t.buzzer}: ${label}`;
  noneBtn.disabled = m.current === null;
  cancelBtn.disabled = m.buzzer !== "buzzed";
  renderBoard(m);
  renderQuestion(m);
  renderTeams(m);
  renderPlayers(m);
}

function onMsg(m: ServerMsg): void {
  switch (m.type) {
    case "state":
      loadPending = false;
      quizMsg.textContent = "";
      render(m);
      break;
    case "error": {
      const text = errorText(m.msg);
      if (loadPending) {
        loadPending = false;
        quizMsg.textContent = text;
      } else {
        opMsg.textContent = text;
      }
      break;
    }
  }
}

// ---- quiz picker ----

function fillQuizzes(names: string[]): void {
  quizSel.replaceChildren();
  if (names.length === 0) {
    quizSel.appendChild(el("option", "", t.noQuizFiles));
    quizSel.disabled = true;
    loadBtn.disabled = true;
    return;
  }
  for (const name of names) {
    const o = el("option", "", name);
    o.value = name;
    quizSel.appendChild(o);
  }
  quizSel.disabled = false;
  loadBtn.disabled = false;
}

fetch("/api/quizzes")
  .then((r) => {
    if (!r.ok) throw new Error(String(r.status));
    return r.json() as Promise<string[]>;
  })
  .then(fillQuizzes)
  .catch(() => fillQuizzes([]));

fetch("/info")
  .then((r) => r.text())
  .then((u) => {
    url.textContent = u;
  })
  .catch(() => {
    url.textContent = "?";
  });

// ---- actions ----

const conn = connect(
  "/ws/admin",
  onMsg,
  () => {
    connEl.textContent = t.connected;
    connEl.classList.remove("offline");
  },
  () => {
    connEl.textContent = t.disconnected;
    connEl.classList.add("offline");
  },
);

loadBtn.onclick = () => {
  const file = quizSel.value;
  if (!file) return;
  quizMsg.textContent = "";
  opMsg.textContent = "";
  loadPending = true;
  conn.send({ type: "load_quiz", file });
};

noneBtn.onclick = () => {
  opMsg.textContent = "";
  conn.send({ type: "none" });
};

cancelBtn.onclick = () => {
  opMsg.textContent = "";
  conn.send({ type: "cancel_buzz" });
};

timerBtn.onclick = () => {
  if (!timerState?.started) return;
  conn.send({ type: "timer", action: timerState.paused ? "resume" : "pause" });
};

startTick(renderTimer);
