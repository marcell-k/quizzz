import { t } from "./strings";
import { connect } from "./ws";
import type { ServerMsg } from "./protocol";

const h1 = document.createElement("h1");
h1.textContent = t.appTitle + ": " + t.admin;
const hint = document.createElement("p");
hint.textContent = t.scanToJoin;
const img = document.createElement("img");
img.src = "/qr.png";
img.width = 256;
img.height = 256;
const url = document.createElement("p");
const info = document.createElement("p");
const cancel = document.createElement("button");
cancel.textContent = t.cancelBuzz;
const plTitle = document.createElement("h2");
plTitle.textContent = t.players;
const list = document.createElement("ul");
document.body.replaceChildren(h1, hint, img, url, info, cancel, plTitle, list);

fetch("/info")
  .then((r) => r.text())
  .then((u) => {
    url.textContent = u;
  })
  .catch(() => {
    url.textContent = "?";
  });

function onMsg(m: ServerMsg): void {
  if (m.type !== "state") return;
  const by = m.players.find((p) => p.id === m.buzzedBy);
  info.textContent = by
    ? `${t.buzzer}: ${m.buzzer} (${m.teams[by.team] ?? "?"}: ${by.name})`
    : `${t.buzzer}: ${m.buzzer}`;
  list.replaceChildren(
    ...m.players.map((p) => {
      const li = document.createElement("li");
      li.textContent = `${m.teams[p.team] ?? "?"}: ${p.name} (${p.online ? t.online : t.offline})`;
      return li;
    }),
  );
}

const conn = connect(
  "/ws/admin",
  onMsg,
  () => {},
  () => {},
);
cancel.onclick = () => {
  conn.send({ type: "cancel_buzz" });
};
