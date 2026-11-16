import { t, errorText } from "./strings";
import { connect } from "./ws";
import { teamColor } from "./colors";
import type { BuzzerState, ServerMsg } from "./protocol";

const TOKEN_KEY = "quiz.token";
const LAST_KEY = "quiz.last";

function loadToken(): string | null {
  try {
    return localStorage.getItem(TOKEN_KEY);
  } catch {
    return null;
  }
}

function saveToken(v: string | null): void {
  try {
    if (v === null) localStorage.removeItem(TOKEN_KEY);
    else localStorage.setItem(TOKEN_KEY, v);
  } catch {
    /* storage blocked: no reconnect identity */
  }
}

type Last = { team: number; name: string };

function loadLast(): Last | null {
  try {
    const raw = localStorage.getItem(LAST_KEY);
    if (!raw) return null;
    const v = JSON.parse(raw) as Partial<Last>;
    if (typeof v.team !== "number" || typeof v.name !== "string") return null;
    return { team: v.team, name: v.name };
  } catch {
    return null;
  }
}

function saveLast(v: Last): void {
  try {
    localStorage.setItem(LAST_KEY, JSON.stringify(v));
  } catch {
    /* storage blocked */
  }
}

// ---- DOM ----

const strip = document.createElement("div");
strip.className = "strip";

const title = document.createElement("h1");
title.textContent = t.appTitle;

const viewTeam = document.createElement("section");
viewTeam.className = "view";
const pickLabel = document.createElement("p");
pickLabel.className = "label";
pickLabel.textContent = t.pickTeam;
const teamList = document.createElement("div");
teamList.className = "team-list";
const noTeams = document.createElement("p");
noTeams.className = "hint hidden";
noTeams.textContent = t.noTeams;
viewTeam.append(pickLabel, teamList, noTeams);

const viewName = document.createElement("section");
viewName.className = "view hidden";
const chosen = document.createElement("p");
chosen.className = "chosen";
const nameIn = document.createElement("input");
nameIn.placeholder = t.name;
nameIn.maxLength = 30;
nameIn.autocomplete = "off";
nameIn.autocapitalize = "words";
const joinErr = document.createElement("p");
joinErr.className = "err";
const joinBtn = document.createElement("button");
joinBtn.className = "btn";
joinBtn.textContent = t.join;
const backBtn = document.createElement("button");
backBtn.className = "link";
backBtn.textContent = t.back;
viewName.append(chosen, nameIn, joinErr, joinBtn, backBtn);

const viewBuzz = document.createElement("section");
viewBuzz.className = "view hidden";
const who = document.createElement("p");
who.className = "who";
const buzzBtn = document.createElement("button");
buzzBtn.className = "buzz";
buzzBtn.textContent = t.buzz;
const buzzHint = document.createElement("p");
buzzHint.className = "hint";
viewBuzz.append(who, buzzBtn, buzzHint);

const connEl = document.createElement("p");
connEl.className = "conn hidden";
connEl.textContent = t.disconnected;

const page = document.createElement("div");
page.className = "player";
page.append(strip, title, viewTeam, viewName, viewBuzz, connEl);
document.body.replaceChildren(page);

// ---- local state ----

type View = "team" | "name" | "buzz";

let teams: string[] = [];
let lastTeamsKey = "";
let selectedTeam = -1;
let joined = false;
let online = false;
let rejoining = false;
let myID = 0;
let myTeam = -1;
let myName = "";
let lastBuzzer: BuzzerState = "locked";
let buzzedBy = 0;
let late = false;

function show(v: View): void {
  viewTeam.classList.toggle("hidden", v !== "team");
  viewName.classList.toggle("hidden", v !== "name");
  viewBuzz.classList.toggle("hidden", v !== "buzz");
  if (v === "team") {
    strip.style.background = "transparent";
  } else if (v === "name") {
    chosen.textContent = `${t.yourTeam}: ${teams[selectedTeam] ?? "?"}`;
    strip.style.background = selectedTeam >= 0 ? teamColor(selectedTeam) : "transparent";
  } else {
    renderWho();
    strip.style.background = myTeam >= 0 ? teamColor(myTeam) : "transparent";
  }
}

function renderTeams(): void {
  const items = teams
    .map((name, i) => ({ name, i }))
    .filter((x) => x.name !== "");
  noTeams.classList.toggle("hidden", items.length > 0);
  teamList.replaceChildren(
    ...items.map(({ name, i }) => {
      const b = document.createElement("button");
      b.className = "team-btn";
      b.type = "button";
      const dot = document.createElement("span");
      dot.className = "dot";
      dot.style.background = teamColor(i);
      const label = document.createElement("span");
      label.textContent = name;
      b.append(dot, label);
      b.onclick = () => {
        selectedTeam = i;
        joinErr.textContent = "";
        show("name");
        nameIn.focus();
      };
      return b;
    }),
  );
}

function renderWho(): void {
  who.textContent =
    joined && myTeam >= 0 ? `${teams[myTeam] ?? "?"}: ${myName}` : "";
}

function renderBuzz(): void {
  buzzBtn.className = "buzz";
  let label: string = t.buzz;
  let hint = "";
  let enabled = false;
  if (!online || !joined) {
    buzzBtn.classList.add("off");
  } else if (late) {
    buzzBtn.classList.add("late");
    hint = t.tooLate;
  } else if (lastBuzzer === "open") {
    enabled = true;
  } else if (lastBuzzer === "buzzed") {
    if (buzzedBy === myID) {
      buzzBtn.classList.add("mine");
      label = t.youBuzzed;
    } else {
      buzzBtn.classList.add("other");
    }
  } else {
    buzzBtn.classList.add("locked");
    hint = t.waiting;
  }
  buzzBtn.disabled = !enabled;
  buzzBtn.textContent = label;
  buzzHint.textContent = hint;
}

function setOnline(v: boolean): void {
  online = v;
  connEl.classList.toggle("hidden", v);
  renderBuzz();
}

// ---- messages ----

function onMsg(m: ServerMsg): void {
  switch (m.type) {
    case "state": {
      teams = m.teams;
      const key = teams.join("|");
      if (key !== lastTeamsKey) {
        lastTeamsKey = key;
        renderTeams();
        if (selectedTeam >= 0 && (teams[selectedTeam] ?? "") === "") {
          selectedTeam = -1;
          joinErr.textContent = "";
          if (!joined) show("team");
        } else if (!joined && !viewName.classList.contains("hidden")) {
          show("name");
        }
      }
      lastBuzzer = m.buzzer;
      buzzedBy = m.buzzedBy;
      late = false;
      if (joined) {
        renderWho();
        strip.style.background = myTeam >= 0 ? teamColor(myTeam) : "transparent";
      }
      renderBuzz();
      break;
    }
    case "joined":
      rejoining = false;
      joined = true;
      myID = m.id;
      myTeam = m.team;
      myName = m.name;
      saveToken(m.token);
      saveLast({ team: m.team, name: m.name });
      joinErr.textContent = "";
      joinBtn.disabled = false;
      late = false;
      show("buzz");
      renderBuzz();
      break;
    case "too_late":
      late = true;
      renderBuzz();
      break;
    case "error": {
      const text = errorText(m.msg);
      if (rejoining) {
        rejoining = false;
        saveToken(null);
        joined = false;
        const last = loadLast();
        if (last) {
          nameIn.value = last.name;
          if (last.team >= 0 && last.team < teams.length && (teams[last.team] ?? "") !== "") {
            selectedTeam = last.team;
          }
        }
        joinErr.textContent = "";
        show("team");
        renderBuzz();
      } else if (!viewName.classList.contains("hidden")) {
        joinErr.textContent = text;
        joinBtn.disabled = false;
      } else {
        buzzHint.textContent = text;
      }
      break;
    }
  }
}

const conn = connect(
  "/ws",
  onMsg,
  () => {
    setOnline(true);
    const token = loadToken();
    if (token) {
      rejoining = true;
      conn.send({ type: "join", team: 0, name: "", token });
    } else {
      joined = false;
      show("team");
      renderBuzz();
    }
  },
  () => {
    setOnline(false);
    joined = false;
    renderBuzz();
  },
);

// ---- actions ----

joinBtn.onclick = () => {
  const name = nameIn.value.trim();
  if (name === "") {
    joinErr.textContent = errorText("bad_name");
    return;
  }
  if (selectedTeam < 0) {
    show("team");
    return;
  }
  joinErr.textContent = "";
  joinBtn.disabled = true;
  conn.send({ type: "join", team: selectedTeam, name });
};

nameIn.onkeydown = (e: KeyboardEvent) => {
  if (e.key === "Enter") {
    e.preventDefault();
    joinBtn.click();
  }
};

backBtn.onclick = () => {
  joinErr.textContent = "";
  joinBtn.disabled = false;
  show("team");
};

buzzBtn.onpointerdown = (e: PointerEvent) => {
  e.preventDefault();
  if (buzzBtn.disabled || !joined || lastBuzzer !== "open" || late) return;
  lastBuzzer = "locked"; // local guard until server state arrives
  renderBuzz();
  conn.send({ type: "buzz" });
};

// ---- start ----

show("team");
renderBuzz();
