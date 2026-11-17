import { t } from "./strings";
import { connect } from "./ws";
import { teamColor } from "./colors";
import { logo } from "./logo";
import { formatClock, remainingSeconds, startTick } from "./timer";
import type { ServerMsg, StateMsg, TimerInfo } from "./protocol";

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

function fmt(n: number): string {
  return String(n).replace(/\B(?=(\d{3})+(?!\d))/g, ".");
}

// ---- DOM ----

const main = el("main", "display-main");
const scores = el("footer", "display-scores");
const timerCard = el("div", "display-timer");
const timerName = el("span", "name", t.gameTime);
const timerClock = el("span", "clock", "--:--");
timerCard.append(timerName, timerClock);
const qr = el("img", "display-qr");
qr.src = "/qr.png";
qr.alt = "";
const connEl = el("p", "display-conn hidden", t.disconnected);

const page = el("div", "display");
page.append(main, scores, logo("display-logo"), qr, connEl);
document.body.classList.add("display-page");
document.body.replaceChildren(page);

// ---- render ----

let timerState: TimerInfo | null = null;

function renderTimer(): void {
  timerClock.textContent = timerState ? formatClock(remainingSeconds(timerState)) : "--:--";
}



function renderScores(m: StateMsg): void {
  scores.replaceChildren(
    timerCard,
    ...m.teams.flatMap((team, i) => {
      if (team.name === "") return [];
      const card = el("div", "display-team");
      const dot = el("span", "dot");
      dot.style.background = teamColor(i);
      card.append(dot, el("span", "name", team.name), el("span", "score", fmt(team.score)));
      return [card];
    }),
  );
}

function renderBoard(m: StateMsg): void {
  const n = m.points.length;
  if (m.topics.length === 0 || n === 0) {
    main.replaceChildren(el("p", "display-empty", t.noQuiz));
    return;
  }
  const board = el("div", "display-board");
  board.style.gridTemplateColumns = `minmax(9rem, 1.3fr) repeat(${n}, minmax(0, 1fr))`;
  board.style.gridTemplateRows = `auto repeat(${m.topics.length}, minmax(0, 1fr))`;

  const cells: HTMLElement[] = [el("div", "display-col", "")];
  for (let q = 0; q < n; q++) cells.push(el("div", "display-col", `#${q + 1}`));

  m.topics.forEach((topic, ti) => {
    cells.push(el("div", "display-topic", topic));
    for (let q = 0; q < n; q++) {
      const used = m.used[ti]?.[q] ?? false;
      const cell = el("div", "display-cell", fmt(m.points[q] ?? 0));
      if (used) cell.classList.add("used");
      cells.push(cell);
    }
  });
  board.replaceChildren(...cells);
  main.replaceChildren(board);
}

function whoLine(m: StateMsg): HTMLElement {
  const line = el("p", "display-who");
  const by = m.players.find((p) => p.id === m.buzzedBy);
  if (by) {
    const dot = el("span", "dot");
    dot.style.background = teamColor(by.team);
    const team = m.teams[by.team]?.name ?? "?";
    line.append(dot, el("span", "", `${team}: ${by.name}`));
  } else if (m.buzzer === "open") {
    line.classList.add("open");
    line.textContent = t.buzzOpen;
  }
  return line;
}

function renderQuestion(m: StateMsg): void {
  const cur = m.current;
  if (!cur) {
    renderBoard(m);
    return;
  }
  const box = el("div", "display-question");
  const topic = m.topics[cur.topic] ?? "?";
  box.append(el("p", "display-qmeta", `${topic} · ${fmt(cur.points)} ${t.points}`));
  box.append(el("p", "display-qtext", cur.text));
  if (cur.image !== "") {
    const img = el("img", "display-qimg");
    img.src = cur.image;
    img.alt = "";
    img.onerror = () => img.remove();
    box.append(img);
  }
  box.append(whoLine(m));
  main.replaceChildren(box);
}

function renderLightning(m: StateMsg): void {
  const box = el("div", "display-question");
  if (m.lightningImage !== "") {
    const img = el("img", "display-qimg display-lightning-img");
    img.src = m.lightningImage;
    img.alt = "";
    img.onerror = () => {
      img.remove();
      box.prepend(el("h1", "display-title", t.lightningTitle));
    };
    box.append(img);
  } else {
    box.append(el("h1", "display-title", t.lightningTitle));
  }
  box.append(whoLine(m));
  main.replaceChildren(box);
}

function renderSetup(m: StateMsg): void {
  const box = el("div", "display-question");
  const online = m.players.filter((p) => p.online).length;
  box.append(
    el("h1", "display-title", m.settings.title),
    el("p", "display-qmeta", t.scanToJoin),
    el("p", "display-qmeta", `${t.players}: ${online}`),
  );
  main.replaceChildren(box);
}

function renderEnded(m: StateMsg): void {
  const box = el("div", "display-results");
  box.append(el("h1", "display-title", t.results));
  const list = el("div", "display-result-list");
  for (const r of m.results ?? []) {
    const row = el("div", "display-result");
    const dot = el("span", "dot");
    dot.style.background = teamColor(r.team);
    row.append(
      el("span", "rank", String(r.rank)),
      dot,
      el("span", "name", m.teams[r.team]?.name ?? "?"),
      el("span", "score", fmt(r.score)),
    );
    list.append(row);
  }
  box.append(list);
  main.replaceChildren(box);
}

function render(m: StateMsg): void {
  timerState = m.timer;
  timerName.textContent = m.round === "lightning" ? t.lightningTime : t.gameTime;
  renderTimer();
  renderScores(m);
  const setup = m.round === "setup";
  scores.classList.toggle("hidden", setup);
  qr.classList.toggle("hidden", !setup);

  switch (m.round) {
	case "setup":
      renderSetup(m);
      break;
    case "ended":
      renderEnded(m);
      break;
    case "lightning":
      renderLightning(m);
      break;
    default:
      renderQuestion(m);
  }
}

function onMsg(m: ServerMsg): void {
  if (m.type === "state") render(m);
}

connect(
  "/ws",
  onMsg,
  () => {
    connEl.classList.add("hidden");
  },
  () => {
    connEl.classList.remove("hidden");
  },
);

startTick(renderTimer);
