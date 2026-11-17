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

const headTitle = el("h1", "display-head-title", t.appTitle);
const head = el("header", "display-head");
head.append(logo("display-logo"), headTitle);

const main = el("main", "display-main");

const timerCard = el("div", "display-timer");
const timerName = el("span", "name", t.gameTime);
const timerClock = el("span", "clock", "--:--");
timerCard.append(timerName, timerClock);

const teams = el("div", "display-teams");
const qr = el("img", "display-qr");
qr.src = "/qr.png";
qr.alt = "";

const side = el("aside", "display-side");
side.append(timerCard, teams, qr);

const body = el("div", "display-body");
body.append(main, side);

const sponsors = el("footer", "display-sponsors hidden");
const connEl = el("p", "display-conn hidden", t.disconnected);

const page = el("div", "display");
page.append(head, body, sponsors, connEl);
document.body.classList.add("display-page");
document.body.replaceChildren(page);

// ---- sponsors ----

function loadSponsors(): void {
  fetch("/api/sponsors")
    .then((r) => {
      if (!r.ok) throw new Error(String(r.status));
      return r.json() as Promise<string[]>;
    })
    .then((names) => {
      sponsors.replaceChildren(
        ...names.map((name) => {
          const img = el("img", "display-sponsor");
          img.src = `/media/sponsors/${encodeURIComponent(name)}`;
          img.alt = "";
          img.onerror = () => img.remove();
          return img;
        }),
      );
      sponsors.classList.toggle("hidden", names.length === 0);
    })
    .catch(() => sponsors.classList.add("hidden"));
}

// ---- render ----

let timerState: TimerInfo | null = null;

function renderTimer(): void {
  timerClock.textContent = timerState ? formatClock(remainingSeconds(timerState)) : "--:--";
}

function renderTeams(m: StateMsg): void {
  const buzzTeam = m.players.find((p) => p.id === m.buzzedBy)?.team ?? -1;
  teams.replaceChildren(
    ...m.teams.flatMap((team, i) => {
      if (team.name === "") return [];
      const card = el("div", "display-team");
      const row = el("div", "row");
      const dot = el("span", "dot");
      dot.style.background = teamColor(i);
      row.append(dot, el("span", "name", team.name));
      card.append(row, el("div", "score", fmt(team.score)));
      if (m.buzzer === "buzzed" && i === buzzTeam) {
        card.classList.add("buzzing");
        card.style.borderColor = teamColor(i);
      }
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
  board.style.gridTemplateColumns = `repeat(${m.topics.length}, minmax(0, 1fr))`;
  board.style.gridTemplateRows = `auto repeat(${n}, minmax(0, 1fr))`;

  // row 1: topic names. Then one row per question level.
  const cells: HTMLElement[] = m.topics.map((topic) => el("div", "display-topic", topic));
  for (let q = 0; q < n; q++) {
    for (let ti = 0; ti < m.topics.length; ti++) {
      const used = m.used[ti]?.[q] ?? false;
      const cell = el("div", "display-cell", fmt(m.points[q] ?? 0));
      if (used) cell.classList.add("used");
      cells.push(cell);
    }
  }
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
    el("h1", "display-title", t.scanToJoin),
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
  headTitle.textContent = m.settings.title || t.appTitle;
  timerName.textContent = m.round === "lightning" ? t.lightningTime : t.gameTime;
  renderTimer();
  renderTeams(m);

  // setup: QR takes the sidebar slot. Other rounds: clock + teams.
  const setup = m.round === "setup";
  timerCard.classList.toggle("hidden", setup);
  teams.classList.toggle("hidden", setup);
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
    loadSponsors();
  },
  () => {
    connEl.classList.remove("hidden");
  },
);

startTick(renderTimer);
