import { t, errorText } from "./strings";
import { connect } from "./ws";
import { teamColor } from "./colors";
import { logo } from "./logo";
import { formatClock, remainingSeconds, startTick } from "./timer";
import type {
  BuzzerState,
  PlayerInfo,
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

interface PlayersWidget {
  root: HTMLDetailsElement;
  summary: HTMLElement;
  list: HTMLUListElement;
}

function playersWidget(): PlayersWidget {
  const root = el("details", "players-details");
  const summary = el("summary", "", t.players);
  const list = el("ul", "player-list");
  root.append(summary, list);
  return { root, summary, list };
}

// ---- header ----

const h1 = el("h1", "", t.appTitle);
const helpLink = el("a", "link", t.help);
helpLink.href = "/help";
const connEl = el("p", "conn");
const urlText = el("span", "join-url");
const copyBtn = el("button", "link", t.copy);
copyBtn.type = "button";
const urlRow = el("div", "url-row hidden");
urlRow.append(urlText, copyBtn);
const connBox = el("div", "conn-box");
connBox.append(connEl, urlRow);
const head = el("div", "admin-head");
head.append(logo(), h1, helpLink, connBox);

const resumeText = el("p", "status");
const resumeContinueBtn = el("button", "btn", t.resumeContinue);
const resumeNewBtn = el("button", "btn", t.resumeNew);
const resumeActions = el("div", "actions");
resumeActions.append(resumeContinueBtn, resumeNewBtn);
const resumePanel = el("section", "panel resume-panel hidden");
resumePanel.append(resumeText, resumeActions);

// ---- setup view: 1 join, 2 quiz, 3 settings, sticky start bar ----

const joinCount = el("p", "status");
const setupPlayers = playersWidget();
const maxTeams = 10;
const minTeams = 2;
let teamCount = minTeams;

const teamInputs = Array.from({ length: maxTeams }, () => {
  const i = el("input");
  i.type = "text";
  i.maxLength = 30;
  i.autocomplete = "off";
  return i;
});
const teamRows = teamInputs.map((input, i) => {
  const row = el("div", "join-row");
  const dot = el("span", "dot");
  dot.style.background = teamColor(i);
  const count = el("span", "join-count", "0/0");
  row.append(dot, input, count);
  return { row, count };
});
const joinTeams = el("div", "join-teams");
joinTeams.append(...teamRows.map((r) => r.row));

const teamMinus = el("button", "step-btn", "−");
teamMinus.type = "button";
const teamPlus = el("button", "step-btn", "+");
teamPlus.type = "button";
const teamNum = el("span", "step-num", String(teamCount));
const teamHead = el("div", "team-head");
teamHead.append(el("h3", "", t.teamNames), teamMinus, teamNum, teamPlus);

function showTeamRows(count: number): void {
  teamCount = count;
  teamRows.forEach((r, i) => r.row.classList.toggle("hidden", i >= count));
  teamNum.textContent = String(count);
  teamMinus.disabled = count <= minTeams;
  teamPlus.disabled = count >= maxTeams;
}
showTeamRows(teamCount);
const joinPanel = panel(`1 · ${t.joinStep}`, joinCount, teamHead, joinTeams, setupPlayers.root);

const quizSel = el("select");
const loadBtn = el("button", "btn", t.loadQuiz);
const quizStatus = el("p", "hint", t.noQuiz);
const quizMsg = el("p", "err");
const topicList = el("ol", "topic-list");
quizSel.disabled = true;
loadBtn.disabled = true;
const quizRow = el("div", "quiz-row");
quizRow.append(quizSel, loadBtn);
const quizPanel = panel(`2 · ${t.quizFile}`, quizRow, quizStatus, quizMsg, topicList);

const setTitle = el("input");
setTitle.type = "text";
setTitle.maxLength = 80;
setTitle.autocomplete = "off";

const gameMin = numInput(1, 600);
const lightMin = numInput(0, 600);
const lightningPts = numInput(1, 10000000);

function sizeSelect(max: number): HTMLSelectElement {
  const s = el("select");
  for (let n = 1; n <= max; n++) {
    const o = el("option", "", String(n));
    o.value = String(n);
    s.appendChild(o);
  }
  return s;
}

const topicsSel = sizeSelect(7);
const questionsSel = sizeSelect(6);

const pointsRow = el("div", "points-row");
const pointInputs: HTMLInputElement[] = [];

const ruleReopen = checkRow(t.ruleReopenOnWrong);
const ruleLockout = checkRow(t.ruleLockoutTeam);
const ruleSubtract = checkRow(t.ruleSubtractPoints);
const ruleAuto = checkRow(t.ruleAutoOpen);

const checks = el("div", "checks");
checks.append(ruleReopen.row, ruleLockout.row, ruleSubtract.row, ruleAuto.row);

const settingsGrid = el("div", "settings-grid");
settingsGrid.append(
  field(t.title, setTitle),
  field(t.gameMinutes, gameMin),
  field(t.lightningMinutes, lightMin),
  field(t.lightningPoints, lightningPts),
);

const sizeRow = el("div", "size-row");
sizeRow.append(
  field(t.topics, topicsSel),
  field(t.questionsPerTopic, questionsSel),
  el("p", "hint", t.boardSizeHint),
);

const settingsPanel = panel(
  `3 · ${t.settings}`,
  settingsGrid,
  el("h3", "", t.boardSize),
  sizeRow,
  el("h3", "", t.questionPoints),
  pointsRow,
  el("h3", "", t.rules),
  checks,
);

const checkQuiz = el("span", "check-item");
const checkPlayers = el("span", "check-item");
const barInfo = el("div", "bar-info");
barInfo.append(checkQuiz, checkPlayers);
const startGameBtn = el("button", "btn btn-primary", `${t.startGame} ▶`);
startGameBtn.disabled = true;
const setupBar = el("div", "bar bar-bottom");
setupBar.append(barInfo, startGameBtn);

const setupCols = el("div", "setup-cols");
setupCols.append(joinPanel, quizPanel, settingsPanel);
const setupView = el("div", "admin-view setup-view");
setupView.append(setupCols, setupBar);

// ---- game view: top bar, board + teams, sticky question bar ----

const timerClock = el("span", "timer-clock", "00:00");
const timerBtn = el("button", "btn", t.pauseTimer);
timerBtn.disabled = true;
const startLightningBtn = el("button", "btn", t.startLightning);
const endGameBtn = el("button", "btn", t.endGame);
const undoBtn = el("button", "btn", `↶ ${t.undo}`);
startLightningBtn.disabled = true;
endGameBtn.disabled = true;
undoBtn.disabled = true;
const topbar = el("div", "bar");
topbar.append(
  timerClock,
  timerBtn,
  el("span", "spacer"),
  startLightningBtn,
  endGameBtn,
  undoBtn,
);

const noQuiz = el("p", "hint", t.noQuiz);
noQuiz.classList.add("hidden");
const board = el("div", "board");
const boardPanel = panel(t.board, noQuiz, board);

const resultsList = el("div", "results-admin");
const newGameBtn = el("button", "btn", t.newGame);
const resultsPanel = panel(t.results, resultsList, newGameBtn);
resultsPanel.classList.add("hidden");

const gameLeft = el("div", "game-left");
gameLeft.append(boardPanel, resultsPanel);

const teamList = el("div", "team-list-admin");
const gamePlayers = playersWidget();
const teamsPanel = panel(t.teams, teamList, gamePlayers.root);

const gameMain = el("div", "game-main");
gameMain.append(gameLeft, teamsPanel);

const buzzDot = el("span", "buzz-dot");
const buzzName = el("span", "buzz-name");
const buzzStatus = el("p", "buzz-status locked");
buzzStatus.append(buzzDot, buzzName);
const qBox = el("div", "qbox");
const opMsg = el("p", "err");
const noneBtn = el("button", "btn", t.noAnswer);
const cancelBtn = el("button", "btn", t.cancelBuzz);
const openBtn = el("button", "btn", t.openBuzz);
noneBtn.disabled = true;
cancelBtn.disabled = true;
openBtn.disabled = true;
const actions = el("div", "actions");
actions.append(noneBtn, cancelBtn, openBtn);
const qinfo = el("div", "qinfo");
qinfo.append(qBox, opMsg);
const qbar = el("div", "bar bar-bottom qbar");
qbar.append(qinfo, actions);

const gameView = el("div", "admin-view hidden");
gameView.append(topbar, gameMain, qbar);

document.body.classList.add("admin-page");
document.body.replaceChildren(head, resumePanel, setupView, gameView);

// ---- state ----

let loadPending = false;
let timerState: TimerInfo | null = null;
let currentRound: Round = "setup";
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
    (settingsPanel.contains(ae) || joinTeams.contains(ae))
  ) {
    return;
  }
  setTitle.value = s.title;
  s.teams.forEach((name, i) => {
    const input = teamInputs[i];
    if (input) input.value = name;
  });
  let count = minTeams;
  s.teams.forEach((name, i) => {
    if (name !== "") count = Math.max(count, i + 1);
  });
  showTeamRows(count);
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
  h1.textContent = s.title;
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
    !validInt(topics, 1, 7) ||
    !validInt(questions, 1, 6) ||
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
    teams: teamInputs.map((input, i) => (i < teamCount ? input.value.trim() : "")),
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

function questionHead(meta: string): HTMLElement {
  const row = el("div", "qhead");
  row.append(buzzStatus, el("span", "qmeta", meta));
  return row;
}

function renderQuestion(m: StateMsg): void {
  if (m.round === "lightning") {
    qBox.replaceChildren(
      questionHead(`${t.lightningTitle} · ${fmt(m.lightningPoints)}`),
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
    qBox.replaceChildren(questionHead(t.noCurrent));
    return;
  }
  const parts: HTMLElement[] = [
    questionHead(`${m.topics[cur.topic] ?? "?"} · ${fmt(cur.points)}`),
    el("p", "qtext", cur.text),
  ];
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
  board.style.gridTemplateColumns = `repeat(${m.topics.length}, minmax(5rem, 1fr))`;
  const cells: HTMLElement[] = m.topics.map((topic) => el("div", "topic", topic));
  for (let q = 0; q < n; q++) {
    for (let ti = 0; ti < m.topics.length; ti++) {
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
  }
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
  const buzzTeam = m.players.find((p) => p.id === m.buzzedBy)?.team ?? -1;
  const rows: HTMLElement[] = [];
  m.teams.forEach((team, i) => {
    if (team.name === "") return;
    const row = el("div", "team-row");
    if (m.buzzer === "buzzed" && i === buzzTeam) {
      row.classList.add("buzzing");
      row.style.borderColor = teamColor(i);
    }
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

function playerItem(m: StateMsg, p: PlayerInfo): HTMLLIElement {
  const li = el("li");
  const out = m.lockedPlayers.includes(p.id) || m.lockedTeams[p.team] === true;
  const status = p.online ? t.online : t.offline;
  li.textContent = `${m.teams[p.team]?.name ?? "?"}: ${p.name} (${status}${out ? `, ${t.out}` : ""})`;
  return li;
}

function renderPlayers(m: StateMsg): void {
  for (const w of [setupPlayers, gamePlayers]) {
    w.summary.textContent = `${t.players} (${m.players.length})`;
    w.list.replaceChildren(...m.players.map((p) => playerItem(m, p)));
  }

  const online = m.players.filter((p) => p.online).length;
  joinCount.textContent = `${t.players}: ${online}`;
  teamRows.forEach((r, i) => {
    const mine = m.players.filter((p) => p.team === i);
    const on = mine.filter((p) => p.online).length;
    r.count.textContent = `${on}/${mine.length}`;
  });
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

function renderSetupBar(m: StateMsg): void {
  const hasQuiz = m.topics.length > 0;
  const online = m.players.filter((p) => p.online).length;
  checkQuiz.textContent = `${hasQuiz ? "✓" : "✗"} ${t.quizReady}`;
  checkQuiz.classList.toggle("ok", hasQuiz);
  checkPlayers.textContent = `${online > 0 ? "✓" : "–"} ${online} ${t.players.toLowerCase()}`;
  checkPlayers.classList.toggle("ok", online > 0);
  startGameBtn.disabled = m.round !== "setup" || !hasQuiz;
  startGameBtn.title = hasQuiz ? "" : t.startNeedsQuiz;

  topicList.replaceChildren(...m.topics.map((name) => el("li", "", name)));

  if (hasQuiz) {
    const file = m.settings.lastQuiz;
    quizStatus.textContent = `✓ ${t.loaded}${file ? `: ${file}` : ""} — ${m.topics.length} ${t.topicsUnit} × ${m.points.length} ${t.questionsUnit}`;
    quizStatus.className = "hint quiz-ok";
  } else {
    quizStatus.textContent = t.noQuiz;
    quizStatus.className = "hint";
  }
}

function renderVisibility(m: StateMsg): void {
  const setup = m.round === "setup";
  const ended = m.round === "ended";
  setupView.classList.toggle("hidden", !setup);
  gameView.classList.toggle("hidden", setup);
  boardPanel.classList.toggle("hidden", ended);
  resultsPanel.classList.toggle("hidden", !ended);
  qbar.classList.toggle("hidden", ended);
}

function render(m: StateMsg): void {
  currentRound = m.round;
  timerState = m.timer;
  renderResume(m);
  renderTimer();
  renderVisibility(m);
  renderSetupBar(m);

  const by = m.players.find((p) => p.id === m.buzzedBy);
  const buzzText = by
    ? `${t.buzzer}: ${buzzerLabels[m.buzzer]} — ${m.teams[by.team]?.name ?? "?"}: ${by.name}`
    : `${t.buzzer}: ${buzzerLabels[m.buzzer]}`;
  buzzStatus.className = `buzz-status ${m.buzzer}`;
  buzzStatus.title = buzzText;
  buzzStatus.setAttribute("aria-label", buzzText);
  buzzName.textContent = by
    ? `${m.teams[by.team]?.name ?? "?"}: ${by.name}`
    : buzzerLabels[m.buzzer];
  buzzDot.style.background = by ? teamColor(by.team) : "";

  noneBtn.disabled = m.current === null;
  cancelBtn.disabled = m.buzzer !== "buzzed";
  openBtn.disabled = !(m.round === "board" && m.current !== null && m.buzzer === "locked");
  startLightningBtn.disabled = m.round !== "board";
  endGameBtn.disabled = m.round === "ended" || m.round === "setup";
  undoBtn.disabled = m.undoCount === 0;
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
  if (lastState) applyLastQuiz(lastState.settings.lastQuiz);
}

fetch("/api/quizzes")
  .then((r) => {
    if (!r.ok) throw new Error(String(r.status));
    return r.json() as Promise<string[]>;
  })
  .then(fillQuizzes)
  .catch(() => fillQuizzes([]));

fetch("/info")
  .then((r) => {
    if (!r.ok) throw new Error(String(r.status));
    return r.text();
  })
  .then((u) => {
    urlText.textContent = u;
    urlRow.classList.remove("hidden");
  })
  .catch(() => urlRow.classList.add("hidden"));

copyBtn.onclick = () => {
  navigator.clipboard
    .writeText(urlText.textContent ?? "")
    .then(() => {
      copyBtn.textContent = t.copied;
      setTimeout(() => {
        copyBtn.textContent = t.copy;
      }, 1200);
    })
    .catch(() => {
      /* clipboard blocked */
    });
};
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

function commitSettings(): void {
  ruleLockout.input.disabled = !ruleReopen.input.checked;
  const settings = readSettings();
  if (!settings) {
    if (lastState) renderSettings(lastState.settings);
    quizMsg.textContent = errorText("bad_settings");
    return;
  }
  quizMsg.textContent = "";
  conn.send({ type: "setup", settings });
}
settingsPanel.addEventListener("change", commitSettings);
joinTeams.addEventListener("change", (e) => {
  const input = e.target;
  if (input instanceof HTMLInputElement && input.value.trim() === "") {
    const i = teamInputs.indexOf(input);
    const hadName = (lastState?.teams[i]?.name ?? "") !== "";
    const hasPlayers = lastState?.players.some((p) => p.team === i) ?? false;
    if (hadName && hasPlayers && !confirm(t.confirmRemoveTeam)) {
      if (lastState) input.value = lastState.teams[i]?.name ?? "";
      return;
    }
  }
  commitSettings();
});

teamPlus.onclick = () => {
  if (teamCount >= maxTeams) return;
  const input = teamInputs[teamCount];
  if (input && input.value.trim() === "") {
    input.value = `${String.fromCharCode(65 + teamCount)} csapat`;
  }
  showTeamRows(teamCount + 1);
  commitSettings();
};

teamMinus.onclick = () => {
  if (teamCount <= minTeams) return;
  const last = teamCount - 1;
  if (lastState?.players.some((p) => p.team === last) && !confirm(t.confirmRemoveTeam)) return;
  showTeamRows(teamCount - 1);
  commitSettings();
};

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
