import { t, errorText } from "./strings";
import { connect } from "./ws";
import { teamColor } from "./colors";
import { logo } from "./logo";
import { formatClock, remainingSeconds, startTick } from "./timer";
import type {
  BuzzerState,
  Round,
  ServerMsg,
  Settings,
  StateMsg,
  TimerInfo,
} from "./protocol";

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

function numInput(min: number, max: number): HTMLInputElement {
  const i = el("input");
  i.type = "number";
  i.min = String(min);
  i.max = String(max);
  i.inputMode = "numeric";
  return i;
}

function field(label: string, control: HTMLElement): HTMLLabelElement {
  const l = el("label", "field");
  l.append(el("span", "", label), control);
  return l;
}

function checkRow(label: string): { input: HTMLInputElement; row: HTMLLabelElement } {
  const input = el("input");
  input.type = "checkbox";
  const row = el("label", "check");
  row.append(input, el("span", "", label));
  return { input, row };
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
const startGameBtn = el("button", "btn", t.startGame);
startGameBtn.disabled = true;

const noQuiz = el("p", "hint", t.noQuiz);
noQuiz.classList.add("hidden");
const board = el("div", "board");

const qBox = el("div", "qbox");
const noneBtn = el("button", "btn", t.noAnswer);
const cancelBtn = el("button", "btn", t.cancelBuzz);
const openBtn = el("button", "btn", t.openBuzz);
const opMsg = el("p", "err");
noneBtn.disabled = true;
cancelBtn.disabled = true;
openBtn.disabled = true;
const actions = el("div", "actions");
actions.append(noneBtn, cancelBtn, openBtn);

const teamList = el("div", "team-list-admin");
const undoBtn = el("button", "btn undo-btn", t.undo);
undoBtn.disabled = true;
const playerList = el("ul", "player-list");

const resultsList = el("div", "results-admin");
const newGameBtn = el("button", "btn", t.newGame);
const resultsPanel = panel(t.results, resultsList, newGameBtn);
resultsPanel.classList.add("wide", "hidden");

const timerClock = el("p", "timer-clock", "00:00");
const timerBtn = el("button", "btn timer-btn", t.pauseTimer);
timerBtn.disabled = true;

// ---- settings panel ----

const setTitle = el("input");
setTitle.type = "text";
setTitle.maxLength = 80;
setTitle.autocomplete = "off";

const teamInputs = Array.from({ length: 5 }, () => {
  const i = el("input");
  i.type = "text";
  i.maxLength = 30;
  i.autocomplete = "off";
  return i;
});

const gameMin = numInput(1, 600);
const lightMin = numInput(0, 600);
const lightningPts = numInput(1, 10000000);

function sizeSelect(): HTMLSelectElement {
  const s = el("select");
  for (let n = 1; n <= 9; n++) {
    const o = el("option", "", String(n));
    o.value = String(n);
    s.appendChild(o);
  }
  return s;
}

const topicsSel = sizeSelect();
const questionsSel = sizeSelect();

const pointsRow = el("div", "points-row");
const pointInputs: HTMLInputElement[] = [];

const ruleReopen = checkRow(t.ruleReopenOnWrong);
const ruleLockout = checkRow(t.ruleLockoutTeam);
const ruleSubtract = checkRow(t.ruleSubtractPoints);
const ruleAuto = checkRow(t.ruleAutoOpen);

const namesRow = el("div", "team-names");
namesRow.append(...teamInputs.map((input, i) => field(`${t.team} ${i + 1}`, input)));

const checks = el("div", "checks");
checks.append(ruleReopen.row, ruleLockout.row, ruleSubtract.row, ruleAuto.row);

const settingsGrid = el("div", "settings-grid");
settingsGrid.append(
  field(t.title, setTitle),
  field(t.gameMinutes, gameMin),
  field(t.lightningMinutes, lightMin),
  field(t.topics, topicsSel),
  field(t.questionsPerTopic, questionsSel),
  field(t.lightningPoints, lightningPts),
);

const settingsPanel = panel(t.settings);
settingsPanel.classList.add("wide");
settingsPanel.append(
  settingsGrid,
  el("h3", "", t.teamNames),
  namesRow,
  el("h3", "", t.questionPoints),
  pointsRow,
  el("h3", "", t.rules),
  checks,
  el("p", "hint", t.boardSizeHint),
);

const startLightningBtn = el("button", "btn", t.startLightning);
const endGameBtn = el("button", "btn", t.endGame);
startLightningBtn.disabled = true;
endGameBtn.disabled = true;

const boardPanel = panel(t.board, noQuiz, board);
boardPanel.classList.add("wide");
const questionPanel = panel(t.currentQuestion, status, qBox, actions, opMsg);
questionPanel.classList.add("wide");
const teamsPanel = panel(t.teams, teamList, undoBtn);
teamsPanel.classList.add("wide");

const resumeText = el("p", "status");
const resumeContinueBtn = el("button", "btn", t.resumeContinue);
const resumeNewBtn = el("button", "btn", t.resumeNew);
const resumeActions = el("div", "actions");
resumeActions.append(resumeContinueBtn, resumeNewBtn);
const resumePanel = el("section", "panel resume-panel hidden");
resumePanel.append(resumeText, resumeActions);

const qrPanel = panel(t.scanToJoin, qr, url);
const quizPanel = panel(t.quizFile, quizSel, loadBtn, startGameBtn, quizMsg);
const timerPanel = panel(t.gameTime, timerClock, timerBtn);
const flowPanel = panel(t.gameFlow, startLightningBtn, endGameBtn);

// setup round: QR, quiz, settings. Other rounds: game panels.
const setupPanels: HTMLElement[] = [qrPanel, quizPanel, settingsPanel];
const gamePanels: HTMLElement[] = [timerPanel, flowPanel, boardPanel, questionPanel, teamsPanel];

const grid = el("div", "admin");
grid.append(
  qrPanel,
  quizPanel,
  settingsPanel,
  timerPanel,
  flowPanel,
  boardPanel,
  questionPanel,
  resultsPanel,
  teamsPanel,
  panel(t.players, playerList),
);
const head = el("div", "admin-head");
head.append(logo(), h1, connEl);
document.body.replaceChildren(head, resumePanel, grid);

// ---- state ----

let peekFor = "";
let loadPending = false;
let timerState: TimerInfo | null = null;
let currentRound: Round = "board";
let lastState: StateMsg | null = null;
let lightningPrompted = false;
let quizPicked = false;

function validInt(v: number, min: number, max: number): boolean {
  return Number.isInteger(v) && v >= min && v <= max;
}

function renderPoints(points: number[]): void {
  const ae = document.activeElement;
  if (ae instanceof HTMLInputElement && ae.type === "number" && pointsRow.contains(ae)) return;
  if (pointInputs.length === points.length) {
    points.forEach((p, i) => {
      const input = pointInputs[i];
      if (input) input.value = String(p);
    });
    return;
  }
  pointInputs.length = 0;
  pointsRow.replaceChildren(
    ...points.map((p, i) => {
      const input = numInput(1, 10000000);
      input.value = String(p);
      pointInputs.push(input);
      return field(`#${i + 1}`, input);
    }),
  );
}

function renderSettings(s: Settings): void {
  const ae = document.activeElement;
  if (
    ae instanceof HTMLInputElement &&
    (ae.type === "text" || ae.type === "number") &&
    settingsPanel.contains(ae)
  ) {
    return;
  }
  setTitle.value = s.title;
  s.teams.forEach((name, i) => {
    const input = teamInputs[i];
    if (input) input.value = name;
  });
  gameMin.value = String(s.gameMinutes);
  lightMin.value = String(s.lightningMinutes);
  lightningPts.value = String(s.lightningPoints);
  topicsSel.value = String(s.topics);
  questionsSel.value = String(s.questions);
  ruleReopen.input.checked = s.rules.reopenOnWrong;
  ruleLockout.input.checked = s.rules.lockoutTeam;
  ruleLockout.input.disabled = !s.rules.reopenOnWrong;
  ruleSubtract.input.checked = s.rules.subtractPoints;
  ruleAuto.input.checked = s.rules.autoOpen;
  renderPoints(s.points);
  h1.textContent = `${s.title}: ${t.admin}`;
}

function readSettings(): Settings | null {
  if (!lastState) return null;
  const topics = Number(topicsSel.value);
  const questions = Number(questionsSel.value);
  const gameMinutes = Number(gameMin.value);
  const lightningMinutes = Number(lightMin.value);
  const lightningPoints = Number(lightningPts.value);
  const points = pointInputs.map((input) => Number(input.value));
  if (
    !validInt(topics, 1, 9) ||
    !validInt(questions, 1, 9) ||
    !validInt(gameMinutes, 1, 600) ||
    !validInt(lightningMinutes, 0, 600) ||
    !validInt(lightningPoints, 1, 10000000) ||
    points.length !== questions ||
    points.some((p) => !validInt(p, 1, 10000000))
  ) {
    return null;
  }
  return {
    title: setTitle.value.trim(),
    teams: teamInputs.map((input) => input.value.trim()),
    topics,
    questions,
    points,
    lightningPoints,
    gameMinutes,
    lightningMinutes,
    rules: {
      reopenOnWrong: ruleReopen.input.checked,
      lockoutTeam: ruleLockout.input.checked,
      subtractPoints: ruleSubtract.input.checked,
      autoOpen: ruleAuto.input.checked,
    },
    lastQuiz: lastState.settings.lastQuiz,
  };
}

function applyLastQuiz(file: string): void {
  if (quizPicked || file === "" || quizSel.disabled) return;
  const opt = Array.from(quizSel.options).find((o) => o.value === file);
  if (opt) {
    quizSel.value = file;
    quizPicked = true;
  }
}

function boardFull(m: StateMsg): boolean {
  return m.used.length > 0 && m.used.every((row) => row.length > 0 && row.every(Boolean));
}

function maybePromptLightning(m: StateMsg): void {
  if (!boardFull(m)) {
    lightningPrompted = false;
    return;
  }
  if (m.round !== "board" || lightningPrompted) return;
  lightningPrompted = true;
  if (confirm(t.confirmLightningFull)) conn.send({ type: "start_lightning" });
}

function renderTimer(): void {
  if (!timerState) {
    timerClock.textContent = "--:--";
    timerBtn.disabled = true;
    return;
  }
  timerClock.textContent = formatClock(remainingSeconds(timerState));
  timerBtn.disabled = !timerState.started || currentRound === "ended";
  timerBtn.textContent = timerState.paused ? t.resumeTimer : t.pauseTimer;
}

function renderQuestion(m: StateMsg): void {
  if (m.round === "lightning") {
    qBox.replaceChildren(
      el("p", "qmeta", `${t.lightningTitle} · ${fmt(m.lightningPoints)}`),
      el("p", "hint", t.lightningHint),
    );
    return;
  }
  if (m.round === "ended") {
    qBox.replaceChildren(el("p", "hint", t.gameEnded));
    return;
  }
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
      b.disabled = used || m.current !== null || m.round !== "board";
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

function editScore(team: number, name: string, current: number): void {
  const raw = prompt(`${t.editScore}: ${name}`, String(current));
  if (raw === null) return;
  const text = raw.trim();
  if (!/^-?\d+$/.test(text) || !Number.isSafeInteger(Number(text))) {
    opMsg.textContent = errorText("bad_score");
    return;
  }
  const value = Number(text);
  opMsg.textContent = "";
  if (value !== current) conn.send({ type: "set_score", team, value });
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
    plus.disabled = m.current === null && m.round !== "lightning";
    plus.onclick = () => {
      opMsg.textContent = "";
      conn.send({ type: "plus", team: i });
    };
    const minus = el("button", "score-btn", "−");
    minus.type = "button";
    minus.disabled = m.current === null && m.round !== "lightning";
    minus.onclick = () => {
      opMsg.textContent = "";
      conn.send({ type: "minus", team: i });
    };
    const score = el("button", "score score-edit", fmt(team.score));
    score.type = "button";
    score.title = t.editScore;
    score.onclick = () => {
      opMsg.textContent = "";
      editScore(i, team.name, team.score);
    };
    row.append(dot, el("span", "tname", team.name), score, plus, minus);
    rows.push(row);
  });
  teamList.replaceChildren(...rows);
}

function renderResults(m: StateMsg): void {
  const results = m.results ?? [];
  resultsList.replaceChildren(
    ...results.map((r) => {
      const row = el("div", "result-row");
      const dot = el("span", "dot");
      dot.style.background = teamColor(r.team);
      row.append(
        el("span", "rank", String(r.rank)),
        dot,
        el("span", "tname", m.teams[r.team]?.name ?? "?"),
        el("span", "score", fmt(r.score)),
      );
      return row;
    }),
  );
}

function renderPlayers(m: StateMsg): void {
  playerList.replaceChildren(
    ...m.players.map((p) => {
      const li = el("li");
      const out = m.lockedPlayers.includes(p.id) || m.lockedTeams[p.team] === true;
      const status = p.online ? t.online : t.offline;
      li.textContent = `${m.teams[p.team]?.name ?? "?"}: ${p.name} (${status}${out ? `, ${t.out}` : ""})`;
      return li;
    }),
  );
}

function resumePrompt(warning: string): string {
  switch (warning) {
    case "resume_quiz_missing":
      return t.resumeQuizMissing;
    case "resume_broken":
      return t.resumeBroken;
    default:
      return warning;
  }
}

function renderResume(m: StateMsg): void {
  const warning = m.resumeWarning;
  resumePanel.classList.toggle("hidden", !m.resumeAvailable && warning === "");
  resumeText.textContent = warning !== "" ? resumePrompt(warning) : t.resumeTitle;
  resumeActions.classList.toggle("hidden", !m.resumeAvailable);
  resumeContinueBtn.disabled = false;
  resumeNewBtn.disabled = false;
}

function renderVisibility(m: StateMsg): void {
  const setup = m.round === "setup";
  for (const p of setupPanels) p.classList.toggle("hidden", !setup);
  for (const p of gamePanels) p.classList.toggle("hidden", setup);
  startGameBtn.disabled = !setup || m.topics.length === 0;
}

function render(m: StateMsg): void {
  currentRound = m.round;
  timerState = m.timer;
  renderResume(m);
  renderTimer();
  renderVisibility(m);
  const label = buzzerLabels[m.buzzer];
  const by = m.players.find((p) => p.id === m.buzzedBy);
  status.textContent = by
    ? `${t.buzzer}: ${label} — ${m.teams[by.team]?.name ?? "?"}: ${by.name}`
    : `${t.buzzer}: ${label}`;
  noneBtn.disabled = m.current === null;
  cancelBtn.disabled = m.buzzer !== "buzzed";
  openBtn.disabled = !(m.round === "board" && m.current !== null && m.buzzer === "locked");
  startLightningBtn.disabled = m.round !== "board";
  endGameBtn.disabled = m.round === "ended" || m.round === "setup";
  undoBtn.disabled = m.undoCount === 0;
  undoBtn.textContent = t.undo;
  resultsPanel.classList.toggle("hidden", m.round !== "ended");
  newGameBtn.disabled = m.round !== "ended";
  renderSettings(m.settings);
  applyLastQuiz(m.settings.lastQuiz);
  renderBoard(m);
  renderQuestion(m);
  renderResults(m);
  renderTeams(m);
  renderPlayers(m);
}

function onMsg(m: ServerMsg): void {
  switch (m.type) {
    case "state":
      loadPending = false;
      quizMsg.textContent = "";
      lastState = m;
      render(m);
      maybePromptLightning(m);
      break;
    case "error": {
      const text = errorText(m.msg);
	  if (loadPending || currentRound === "setup") {
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

settingsPanel.addEventListener("change", () => {
  ruleLockout.input.disabled = !ruleReopen.input.checked;
  const settings = readSettings();
  if (!settings) {
    if (lastState) renderSettings(lastState.settings);
	  quizMsg.textContent = errorText("bad_settings");
    return;
  }
  quizMsg.textContent = "";
  conn.send({ type: "setup", settings });
});

startGameBtn.onclick = () => {
  quizMsg.textContent = "";
  conn.send({ type: "start_game" });
};

resumeContinueBtn.onclick = () => {
  resumeContinueBtn.disabled = true;
  resumeNewBtn.disabled = true;
  conn.send({ type: "resume" });
};

resumeNewBtn.onclick = () => {
  resumeContinueBtn.disabled = true;
  resumeNewBtn.disabled = true;
  conn.send({ type: "new_game" });
};

quizSel.addEventListener("change", () => {
  quizPicked = true;
});

loadBtn.onclick = () => {
  const file = quizSel.value;
  if (!file) return;
  quizPicked = true;
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

openBtn.onclick = () => {
  opMsg.textContent = "";
  conn.send({ type: "open_buzz" });
};

timerBtn.onclick = () => {
  if (!timerState?.started) return;
  conn.send({ type: "timer", action: timerState.paused ? "resume" : "pause" });
};

startLightningBtn.onclick = () => {
  if (!lastState) return;
  opMsg.textContent = "";
  if (!confirm(boardFull(lastState) ? t.confirmLightningFull : t.confirmLightning)) return;
  conn.send({ type: "start_lightning" });
};

endGameBtn.onclick = () => {
  opMsg.textContent = "";
  if (!confirm(t.confirmEndGame)) return;
  conn.send({ type: "end_game" });
};

undoBtn.onclick = () => {
  opMsg.textContent = "";
  conn.send({ type: "undo" });
};

newGameBtn.onclick = () => {
  opMsg.textContent = "";
  if (!confirm(t.confirmNewGame)) return;
  conn.send({ type: "new_game" });
};

startTick(renderTimer);
