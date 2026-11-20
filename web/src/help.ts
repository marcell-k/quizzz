import { t } from "./strings";
import { logo } from "./logo";

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

function p(...children: (Node | string)[]): HTMLParagraphElement {
  const e = el("p");
  e.append(...children);
  return e;
}

function code(text: string): HTMLElement {
  return el("code", "", text);
}

function item(...children: (Node | string)[]): HTMLLIElement {
  const li = el("li");
  li.append(...children);
  return li;
}

function list(ordered: boolean, ...items: HTMLLIElement[]): HTMLElement {
  const e = el(ordered ? "ol" : "ul");
  e.append(...items);
  return e;
}

function pre(text: string): HTMLPreElement {
  const e = el("pre");
  e.append(code(text));
  return e;
}

function table(headers: string[], rows: (Node | string)[][]): HTMLTableElement {
  const e = el("table");
  const thead = el("thead");
  const headRow = el("tr");
  for (const h of headers) headRow.append(el("th", "", h));
  thead.append(headRow);

  const tbody = el("tbody");
  for (const row of rows) {
    const tr = el("tr");
    for (const cell of row) {
      const td = el("td");
      td.append(cell);
      tr.append(td);
    }
    tbody.append(tr);
  }
  e.append(thead, tbody);
  return e;
}

function section(title: string, ...children: (Node | string)[]): HTMLElement {
  const s = el("section", "panel help-section");
  s.append(el("h2", "", title), ...children);
  return s;
}

function link(href: string, text: string): HTMLAnchorElement {
  const a = el("a", "link", text);
  a.href = href;
  return a;
}

document.title = `${t.help} — ${t.appTitle}`;

const h1 = el("h1", "", t.help);
const head = el("header", "help-head");
head.append(logo(), h1);

const nav = el("nav", "help-nav");
nav.append(link("/", t.player), link("/admin", t.admin), link("/display", t.display));

const quickStart = section(
  t.helpQuickStart,
  list(
    true,
    item("Indítsd el a ", code("quiz.exe"), " programot. A böngésző az ", code("/admin"), " oldalon nyílik."),
    item(
      "Ha van korábbi mentés, válaszd a ",
      el("strong", "", "Folytatás"),
      " vagy az ",
      el("strong", "", "Új játék"),
      " lehetőséget.",
    ),
    item(
      "A ",
      el("strong", "", t.settings),
      " panelen add meg a címet, a csapatneveket, az időket és a tábla méretét.",
    ),
    item("Tölts be egy feladatsort a ", code("quizzes/"), " mappából."),
    item("Nyisd meg a ", code("/display"), " oldalt a kivetítőn. A QR-kód a sarokban jelenik meg."),
    item("A játékosok beolvassák a QR-kódot, kiválasztják a csapatot, és beírják a nevüket."),
    item(
      "Válassz kérdést a táblán, pontozz a ",
      code("+"),
      " és ",
      code("−"),
      " gombokkal, majd a ",
      el("strong", "", t.endGame),
      " gombbal zárd le a játékot.",
    ),
  ),
);

const quizFormat = section(
  t.helpQuizFormat,
  p(
    "A feladatsor egyszerű szövegfájl (",
    code(".txt"),
    "), a ",
    code("quizzes/"),
    " mappában. Minden témablokk egy témasorból, a beállított számú kérdésből és egy üres sorból áll.",
  ),
  pre(`Első téma
Első témához tartozó 1. kérdés
Első témához tartozó 2. kérdés

Második téma
Második témához tartozó 1. kérdés
Második témához tartozó 2. kérdés`),
  p(
    "A példa 2 témás és 2 kérdéses. Az alapértelmezett tábla 5 × 6, ezért 5 × (1 + 6 + 1) = 40 soros fájlt vár.",
  ),
  list(
    false,
    item("A fájl legyen UTF-8. A régi kódolású fájlt előbb konvertálni kell (pl. Jegyzettömb: Mentés másként → UTF-8)."),
    item("A CRLF és a CR soremelés, valamint az elején lévő UTF-8 BOM elfogadott."),
    item("A sorok végéről a szóközök és tabulátorok levágódnak."),
    item("Üres kérdéssor nem lehet. Kevesebb sor vagy üres kérdés hibát ad a pontos sorszámmal."),
    item("A szükségesnél több kérdéssor figyelmeztetéssel kimarad."),
    item("A témák száma 1 és 7, a kérdéseké 1 és 6 között állítható be."),
  ),
);

const images = section(
  t.helpImages,
  p("A képek a feladatsor mellett, ugyanabban a ", code("quizzes/"), " mappában legyenek."),
  pre(`<feladatsor>_<témaszám><kérdésszám>.jpg

példa: feladatsor_23.jpg   → 2. téma, 3. kérdés
        feladatsor_11.png   → 1. téma, 1. kérdés`),
  list(
    false,
    item("Elfogadott kiterjesztés: ", code(".jpg"), ", ", code(".jpeg"), ", ", code(".png"), "."),
    item("A témaszám 1-től 7-ig, a kérdésszám 1-től 6-ig terjed, egyjegyű, vezető nulla nélkül."),
    item("A kép csak akkor jelenik meg, ha a fájlnév pontosan egyezik a feladatsor nevével. A kis- és nagybetű, valamint az ékezet is számít."),
    item("Kész minta: ", code("sample.txt"), ", ", code("sample_11.png"), ", ", code("sample_23.png"), ", ", code("sample_lightning.png"), "."),
    item("Ajánlott a képet legfeljebb 1280 px szélesre átméretezni."),
    item(
      "Villámkörhöz nem kötelező kép: ",
      code("<feladatsor>_lightning.jpg"),
      ". Ha nincs, a ",
      el("strong", "", t.lightningTitle),
      " szöveg jelenik meg.",
    ),
  ),
);

const look = section(
  t.helpLook,
  p("A ", code("ui/"), " mappa nem kötelező. Hiányzó fájl esetén az alapértelmezett kinézet marad."),
  pre(`ui/
  logo.png      logó a tanári fejlécen, a kivetítőn és a telefonon
  background.jpg  háttérkép a kivetítőn (nem kötelező)
  theme.css     színek és betűtípus felülírása
  sponsors/     szponzorképek a kivetítő alján (nem kötelező)`),
  p("Példa a ", code("ui/theme.css"), " fájlra:"),
  pre(`:root {
  --accent: #0a6;
  --font: Georgia, serif;
  --display-bg: #f3f1ec;
  --display-text: #161616;
  --bg-image: url(/media/ui/background.jpg);
  --bg-blur: 6px;
  --bg-overlay: rgb(243 241 236 / 0.6);
  --team-1: #d33;
}`),
  list(
    false,
    item(code("--accent"), ": a jelentkező gomb és a kiemelések színe."),
    item(code("--font"), ": a teljes felület betűtípusa."),
    item(code("--display-bg"), " és ", code("--display-text"), ": a kivetítő háttere és szövegszíne."),
    item(code("--bg-image"), ": háttérkép, alapból ", code("url(/media/ui/background.jpg)"), ". Másik fájlhoz abszolút útvonal kell, pl. ", code("url(/media/ui/hatter.png)"), ". Háttér nélkül: ", code("none"), "."),
    item(code("--bg-blur"), ": a háttér elmosása (pl. ", code("8px"), ", kikapcsolva ", code("0px"), ")."),
    item(code("--bg-overlay"), ": a háttérre kerülő színes réteg, a szöveg olvashatósága miatt."),
    item(code("--panel-glass"), ": a kérdés mögötti áttetsző kártya színe."),
    item(code("--panel-bg"), ", ", code("--panel-text"), ", ", code("--panel-border"), ": csapatkártyák és eredménysorok."),
    item(code("--topic-bg"), ", ", code("--topic-text"), ", ", code("--cell-bg"), ", ", code("--cell-text"), ", ", code("--cell-used-bg"), ", ", code("--cell-used-text"), ": a tábla színei."),
    item(code("--timer-bg"), ", ", code("--timer-text"), ", ", code("--muted"), ": időmérő és halvány szöveg."),
    item(code("--team-1"), " … ", code("--team-10"), ": csapatszínek. Alapértelmezett a beépített 10 színű paletta."),
    item("Ajánlott háttérkép: legfeljebb 1920 px széles."),
    item("A ", code("ui/sponsors/"), " képeit a kivetítő alján jeleníti meg, fájlnév szerint rendezve."),
  ),
);

const settings = section(
  t.helpSettings,
  p(
    "A beállításokat a ",
    code("/admin"),
    " oldalon módosíthatod. Minden változás azonnal a ",
    code("settings.json"),
    " fájlba mentődik.",
  ),
  table(
    [t.helpSetting, t.helpMeaning],
    [
      [t.title, "A kivetítő fejlécében és a böngésző címében látható név."],
      [t.teamNames, "2 és 10 közötti csapat. Az üresen hagyott csapat nem jelenik meg."],
      [
        t.boardSize,
        `Témák × kérdések: témák 1–7, kérdések 1–6. A méret módosítása kiüríti a betöltött feladatsort.`,
      ],
      [
        t.questionPoints,
        "Pontok kérdésenként. A lista hossza mindig megegyezik a kérdések számával.",
      ],
      [t.lightningPoints, "A villámkörben adható vagy levonható pont."],
      [t.gameTime, "A tábla körének visszaszámlálása. Csak kijelzés, 00:00 után sem áll le a játék."],
      [t.lightningTime, "A villámkör visszaszámlálása. Szintén csak kijelzés."],
    ],
  ),
  el("h3", "", t.rules),
  table(
    [t.helpSetting, t.helpDefault],
    [
      [t.ruleReopenOnWrong, "kikapcsolva"],
      [t.ruleLockoutTeam, "kikapcsolva"],
      [t.ruleSubtractPoints, "bekapcsolva"],
      [t.ruleAutoOpen, "bekapcsolva"],
    ],
  ),
  p(
    "A táblaméret módosításakor a pontlista az alapértelmezett értékekre áll vissza, de kézzel szerkeszthető marad.",
  ),
);

const gameFlow = section(
  t.helpGameFlow,
  list(
    true,
    item(
      "Válassz egy mezőt a táblán. A mező felhasználtnak jelölődik, a kérdés megjelenik a kivetítőn, és az első kiválasztáskor indul a játékidő.",
    ),
    item(
      "Ha a ",
      el("strong", "", t.ruleAutoOpen),
      " be van kapcsolva, a jelentkezés azonnal nyílik. A játékosok a telefonjukon nyomják meg a piros gombot.",
    ),
    item(
      "A kivetítő kiírja, ki jelentkezett elsőként, például: ",
      el("strong", "", "A csapat: Anna"),
      ". A pont a csapaté, a jelentkezés a játékosé.",
    ),
    item(
      "Pontozás: ",
      code("+"),
      " hozzáad, ",
      code("−"),
      " levon. A ",
      el("strong", "", t.cancelBuzz),
      " visszaadja a jelentkezést pontváltozás nélkül.",
    ),
    item(
      "A ",
      el("strong", "", t.noAnswer),
      " lezárja a kérdést: nincs pontváltozás, a mező felhasznált marad.",
    ),
    item(
      "A ",
      el("strong", "", t.undo),
      " az utolsó pontváltozást vonja vissza. Kézi pontjavítás is lehetséges a csapat sorában.",
    ),
    item(
      "Ha minden mező felhasználódott, a program felajánlja a villámkört. A ",
      el("strong", "", t.startLightning),
      " gombbal bármikor indítható.",
    ),
    item(
      "Villámkörben nincs tábla és nincs kimaradás. A tanár felolvassa a kérdést, a jelentkezés nyílik, pontozás után azonnal újra nyílik.",
    ),
    item(
      "A ",
      el("strong", "", t.endGame),
      " gomb lezárja a játékot és megjeleníti az eredményt. Azonos pontszámnál a helyezés megosztott.",
    ),
    item(
      "Az ",
      el("strong", "", t.newGame),
      " nullázza a pontokat és a táblát, de megtartja a csapatokat, a játékosokat, a beállításokat és a betöltött feladatsort.",
    ),
  ),
);

const network = section(
  t.helpNetwork,
  p(
    "A tanári gép és a telefonok ugyanazon a helyi hálózaton legyenek. Internet nem kell. A QR-kód a gép helyi IP-címére mutat, például: ",
    code("http://192.168.1.20:8080/"),
  ),
  list(
    false,
    item(
      "Az első indításkor a Windows tűzfal engedélyt kér. Engedélyezd a ",
      el("strong", "", "magánhálózati"),
      " hozzáférést.",
    ),
    item(
      "A ",
      code("/admin"),
      " oldal csak a tanári gépről érhető el. A telefonok a gyökér (",
      code("/"),
      ") oldalt használják.",
    ),
    item(
      "A ",
      code("8080"),
      " foglalt port esetén a program a következő szabad portot választja (maximum ",
      code("8099"),
      "). A QR-kód mindig az aktuális címet tartalmazza.",
    ),
    item(
      "Vendég Wi-Fi gyakran tiltja az eszközök közötti kapcsolatot (client isolation). Ezt a helyszínen egy telefonnal előre teszteld.",
    ),
    item(
      "Ha a telefon nem éri el a szervert, próbáld ezeket: telefon hotspot, saját utazó router, vagy a Windows mobil hotspot funkciója.",
    ),
    item(
      "Több hálózati kártya esetén a program az első privát IPv4-címet választja. Ha ez rossz, kapcsold ki a felesleges adaptert, vagy használj külön hotspotot.",
    ),
  ),
);

const files = section(
  t.helpFiles,
  pre(`quiz.exe        a program
settings.json   beállítások, automatikusan mentve
autosave.json   folyamatban lévő játék mentése
quiz.lock       futás közben létezik
ui/             logó, téma, szponzorképek
quizzes/        feladatsorok és képek`),
  list(
    false,
    item(
      "A program minden állapotváltozás után menti az ",
      code("autosave.json"),
      " fájlt. A mentés először ideiglenes fájlba ír, majd átnevezi, így hirtelen leállás sem rontja el.",
    ),
    item(
      "Indításkor, ha van olvasható mentés, a program felajánlja a folytatást. A játékosok, a tokenek, a pontok, a tábla és a visszavonási lista is visszaáll.",
    ),
    item("Olvashatatlan mentés vagy hiányzó feladatsor esetén figyelmeztetés mellett új játék indul."),
    item(
      "A ",
      code("quiz.lock"),
      " megakadályozza, hogy második példány induljon, mert az felülírhatná a mentést.",
    ),
    item("A ", code("settings.json"), " kézi szerkesztése futás közben nem ajánlott, mert a program felülírja."),
  ),
);

const page = el("div", "help-page");
page.append(head, nav, quickStart, quizFormat, images, look, settings, gameFlow, network, files);

document.body.replaceChildren(page);
