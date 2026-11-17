import { t } from "./strings";
import { logo } from "./logo";

const text = [
  `${t.appTitle} — ${t.help}`,
  "",
  "Saját kinézet (nem kötelező), a program melletti ui mappában:",
  "  ui/logo.png    logó: tanári fejléc, kivetítő, telefon",
  "  ui/theme.css   színek és betűtípus felülírása",
  "",
  "theme.css példa:",
  "  :root { --accent: #0a6; --font: Georgia, serif; }",
  "",
  "Kérdésképek a quizzes mappában:",
  "  <feladatsor>_<témaszám><kérdésszám>.jpg   példa: feladatsor_23.jpg",
  "  <feladatsor>_lightning.jpg                villámkör képe (nem kötelező)",
  "",
  "Elfogadott kép: .jpg, .jpeg, .png. Méret: legfeljebb 1280 px széles.",
].join("\n");

const pre = document.createElement("pre");
pre.className = "help";
pre.textContent = text;

const head = document.createElement("div");
head.className = "player-head";
head.append(logo());

document.body.replaceChildren(head, pre);
