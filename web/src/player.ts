import { t, errorText } from "./strings";
import { connect } from "./ws";
import type { ServerMsg } from "./protocol";

const TOKEN_KEY = "quiz.token";

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

const h1 = document.createElement("h1");
h1.textContent = t.appTitle + ": " + t.player;
const status = document.createElement("p");
status.textContent = t.disconnected;
const teamSel = document.createElement("select");
const nameIn = document.createElement("input");
nameIn.placeholder = t.name;
nameIn.maxLength = 30;
const joinBtn = document.createElement("button");
joinBtn.textContent = t.join;
const who = document.createElement("p");
const info = document.createElement("p");
const buzzBtn = document.createElement("button");
buzzBtn.textContent = t.buzz;
buzzBtn.disabled = true;
buzzBtn.style.cssText =
  "width:100%;height:40vh;font-size:2.5rem;background:#c00;color:#fff;border:0;border-radius:1rem;opacity:0.4";
const log = document.createElement("pre");
document.body.replaceChildren(h1, status, teamSel, nameIn, joinBtn, who, info, buzzBtn, log);

function append(line: string): void {
  log.textContent = line + "\n" + (log.textContent ?? "");
}

let teams: string[] = [];
let lastTeams = "";
let rejoining = false;

function onMsg(m: ServerMsg): void {
  switch (m.type) {
    case "state": {
      teams = m.teams;
      const key = m.teams.join("|");
      if (key !== lastTeams) {
        lastTeams = key;
        const prev = teamSel.value;
        teamSel.replaceChildren();
        m.teams.forEach((name, i) => {
          if (name === "") return;
          const o = document.createElement("option");
          o.value = String(i);
          o.textContent = name;
          teamSel.appendChild(o);
        });
        if (prev !== "") teamSel.value = prev;
      }
      const by = m.players.find((p) => p.id === m.buzzedBy);
      info.textContent = by
        ? `${t.buzzer}: ${m.buzzer} (${m.teams[by.team] ?? "?"}: ${by.name})`
        : `${t.buzzer}: ${m.buzzer}`;
      buzzBtn.style.opacity = m.buzzer === "open" ? "1" : "0.4";
      break;
    }
    case "joined":
      rejoining = false;
      saveToken(m.token);
      teamSel.disabled = true;
      nameIn.disabled = true;
      joinBtn.disabled = true;
      buzzBtn.disabled = false;
      who.textContent = `${teams[m.team] ?? "?"}: ${m.name}`;
      break;
    case "too_late":
      append(t.tooLate);
      break;
    case "error":
      if (rejoining) {
        // saved token unknown to server: show the form
        rejoining = false;
        saveToken(null);
      } else {
        append(errorText(m.msg));
      }
      break;
  }
}

const conn = connect(
  "/ws",
  onMsg,
  () => {
    status.textContent = t.connected;
    const token = loadToken();
    if (token) {
      rejoining = true;
      conn.send({ type: "join", team: 0, name: "", token });
    }
  },
  () => {
    status.textContent = t.disconnected;
  },
);

joinBtn.onclick = () => {
  conn.send({ type: "join", team: Number(teamSel.value), name: nameIn.value });
};
buzzBtn.onpointerdown = () => {
  conn.send({ type: "buzz" });
};
