# Quiz Rewrite: Delphi → Browser + Web Buzzer (final plan)

---

## 1. Goal

Replace 20+ year old Delphi quiz app with browser-based system.

1. Teacher runs **one `.exe`**. No install, no runtime.
2. Players use **own phones** as buzzers. Scan QR → pick team → enter name → red button. Replaces physical buzzer hardware.
3. Quiz screen (board, question, timer, results) works in browser, shown on projector.
4. All on same local network. No internet needed.
5. Old question files keep working (after one-time conversion to UTF-8).

Non-goals (v1): accounts, cloud, internet play, question editor UI, persistent history, sounds, hardware buzzer (design allows later), score display on phones.

---

## 2. What old Delphi app does (from source)

Source read: `Project1.dpr`, `MvM.pas` (main), `vegeredmeny.pas` (results), `about.pas`, `UEredmenyhirdetes_IvN.pas`, `UVillamkerdesek_IvN.pas`. **No `.dfm` seen**, layout and button wiring unknown.

App title in source: _Kárpát-medencei földrajz és földtudományi verseny 2022._

- **Buzzer:** none in code. External hardware, teacher scores by hand.
- **Setup:** 5 player names, 5 topic names, 5×6 question texts, game length (min), lightning length (min), load file, new game, about.
- **Round 1 (board, Jeopardy style):** 5 topics × 6 questions. Points 1000, 2000, 3000, 4000, 5000, **8000**. Pick cell → question shown, cell used, timer starts. Optional image per question. `+`/`−` per player adds or subtracts question points, clears question. "Nincs válasz" = nobody knew. All 30 used → prompt for lightning.
- **Lightning:** board hidden, static image, no question data (teacher reads from paper), fixed 3000, `+`/`−` repeatable, own countdown, manual "game end".
- **Timer:** display only, mm:ss, click = pause/resume, never enforces anything.
- **Results:** sorted by score, modal. Number format `12.000`.
- **Question file** (`.txt`): 5 blocks × 8 lines: topic, q1..q6, blank (old fixed size; new loader reads size from settings). Questions only, no answers. Images `<file>_<topic><q>.jpg`.

---

## 3. Big plan

One Go server holds all state. Browsers are thin views.

```
Teacher laptop: quiz.exe
┌─────────────────────────────────┐
│ HTTP :8080                      │        Phones (players)
│  static files (embedded)        │◄─WiFi─►  /        team + name, red button
│  WebSocket /ws                  │
│  game state (memory)            │        Projector (mirrored from laptop)
│  quizzes + images (disk)        │        browser on /display
│  QR generator                   │
└─────────────────────────────────┘        Teacher browser (same laptop)
                                   ◄──────►  /admin  localhost only
```

Teacher flow:

1. Double click `quiz.exe` (allow firewall once). Browser opens `/admin` automatically. If `autosave.json` exists, prompt "Continue previous game?" (continue / new game).
2. Admin shows QR. Set title, team names, lengths. Load quiz file.
3. Players scan QR, pick team, type name.
4. Teacher switches to `/display` tab for audience. Projector freeze button (hardware) hides admin moments.
5. Run game. Press "Start lightning" when ready. End game → results.

---

## 4. Architecture decisions

| #   | Decision                                                                                         | Why                                                                                         |
| --- | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------- |
| D1  | **Go backend**, whole thing                                                                      | Single static `.exe`, no runtime, easy WebSocket, `go:embed`                                |
| D2  | **TypeScript frontend**, vanilla                                                                 | Typed protocol. Three small pages need no framework                                         |
| D3  | **esbuild** + `tsc --noEmit`                                                                     | Fast, no config. Build only on dev machine                                                  |
| D4  | Frontend **embedded in exe**                                                                     | One file to ship                                                                            |
| D5  | Quizzes, images, UI assets, settings **on disk next to exe**                                     | Swap without rebuild                                                                        |
| D6  | **Server authoritative.** Phones send only `buzz`                                                | Client clocks untrusted                                                                     |
| D7  | **Single goroutine owns state**, input via channel                                               | No locks, buzz order deterministic                                                          |
| D8  | Plain **HTTP**                                                                                   | Nothing needs secure context                                                                |
| D9  | **Admin = loopback only**                                                                        | Phones cannot open `/admin` or send admin messages                                          |
| D10 | Reconnect via **token in `localStorage`**, fallback same team + same name                        | Refresh or WiFi drop keeps identity                                                         |
| D11 | Buzz on **`pointerdown`**                                                                        | No 300 ms delay                                                                             |
| D12 | Keep **legacy `.txt` format**, **strict UTF-8**                                                  | Old files converted once by user. Loader errors on non-UTF-8, no silent garbling            |
| D13 | Scoring **teacher-driven** (`+`/`−`). Buzzer only reveals who was first                          | Matches old flow                                                                            |
| D14 | **Timer on server**, clients render `endsAt`                                                     | No drift                                                                                    |
| D15 | State **in memory, autosaved to `autosave.json`** on every change (write temp file, then rename) | Crash or accidental close loses nothing. Players, tokens, scores, board, timer all restored |
| D16 | **All buzz sources call one function** `Buzz(playerId)`                                          | Hardware buzzer can be added later without redesign                                         |
| D17 | **Score belongs to team**, buzz belongs to **player**                                            | See section 6                                                                               |
| D18 | **Rules configurable** in `/admin`, saved to `settings.json`                                     | Different quizzes, different rules                                                          |
| D19 | **Board size configurable** (topics × questions per topic), default 5 × 6, max 9 × 9             | Other quizzes may differ. Max 9 keeps image names `<topic><q>` unambiguous                  |
| D20 | **Lock file** `quiz.lock`. Second `quiz.exe` refuses to start with message                       | Two servers would corrupt `autosave.json`                                                   |
| D21 | **Team colors hardcoded** (5-color palette)                                                      | Simple. Used on display and on phone buzzer strip                                           |

### Board size and file format

- Settings: `topics` (default 5), `questionsPerTopic` (default 6), both 1–9.
- Block = 1 topic line + N question lines + 1 blank line. Default 5 × (1 + 6 + 1) = 40 lines, same as old file.
- Points list length must equal `questionsPerTopic`. Admin validates. Default 1000, 2000, 3000, 4000, 5000, 8000. Changing N resets list to 1000 × q, last one editable.
- Loader: not valid UTF-8 → error "File not UTF-8. Convert first.". Fewer lines than needed → error with line number. Extra lines ignored with warning. Missing trailing blank on last block tolerated.
- Display board and admin board layout adapt to grid size.

Rejected: Node + `pkg` (bigger exe), Delphi + Indy (maintenance), JSON-only questions (breaks old files), latency compensation (not needed).

---

## 5. Folders and files (next to exe)

```
quiz.exe
settings.json          title, team names, lengths, rule toggles, points, last quiz
quiz.lock              exists while running, prevents second instance
autosave.json          game state: players + tokens, scores, board, current question, round, timer, undo log
ui/                    look and feel, user editable
  theme.css            colors, fonts
  logo.png
  background.jpg
quizzes/               per quiz content
  quiz2022.txt
  quiz2022_11.jpg      image for topic 1 question 1
  quiz2022_23.jpg      image for topic 2 question 3
  quiz2022_lightning.jpg   optional lightning image (default: text screen "Villámkérdések")
```

- Missing file → embedded default.
- Image extensions accepted: `.jpg`, `.jpeg`, `.png`.
- Image name: `<txt name without extension>_<topic><q>.<ext>`.
- Resize images to ≤ 1280 px before use.
- Settings saved on every change in `/admin`. Loaded at start.
- Autosave written after every state change. Atomic (temp file + rename), so crash mid-write cannot corrupt it. Unreadable file → start fresh, show warning.
- Quiz file named in autosave missing at start → start fresh, show warning.

---

## 6. Pages

| Route      | Who                | Content                                                                                                                                                                                                                                                                       |
| ---------- | ------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `/`        | Players            | Step 1: pick team (from teacher's list). Step 2: type name. Red button. States only: locked (grey), open (red), you buzzed, someone else buzzed, too late. No scores on phones. Team color strip. Wake lock. Vibrate later                                                    |
| `/admin`   | Teacher, localhost | QR. Setup (title, teams, lengths, rules, quiz file). Board. Current question, `+`/`−` per team, "Nincs válasz", "Cancel buzz", timer, "Start lightning", "End game", "Undo", manual score edit, player list (kick). Unrevealed question text hidden by default, "peek" button |
| `/display` | Audience           | Board (used cells greyed). Question + points. Image. Who buzzed: "Team A: Anna". Timer. Team scores. Final results. **QR code in corner before game starts, hidden once game starts**                                                                                         |
| `/help`    | Teacher            | Question file format, image naming, assets, settings. Replaces old "About"                                                                                                                                                                                                    |
| `/qr.png`  | Admin              | QR of `http://<LAN-IP>:8080/`                                                                                                                                                                                                                                                 |
| `/media/*` | All                | Images from `quizzes/` and `ui/`                                                                                                                                                                                                                                              |

UI language: Hungarian. All labels in one strings file.

---

## 7. Teams and players

- Teacher defines up to **5 teams** (names) in `/admin`. Empty names hidden.
- Phone joins: pick team, type own name.
- **Many players per team**, no cap. Joining anytime.
- **Buzz belongs to player.** Display shows "Team A: Anna". Only Anna answers.
- **Score belongs to team.** Results show teams.
- **Identity:** same team + same name = same player. Token in `localStorage` restores identity. Lost phone / cleared browser: pick same team + same name again.
- **Same name, same team, currently connected:** rejected ("name in use"). If old connection is gone, new device takes over. Teacher can kick a player in `/admin`.
- Names compared case-insensitive, trimmed.
- **Players are saved** (autosave). After server restart or laptop reboot, phones reconnect with token and keep identity. Teams, players and scores survive.
- Admin has "remove player" and "clear all players".

---

## 8. Game state (server)

```
settings   title, teams[5], gameMinutes, lightningMinutes, rules, points[6], lightningPoints
teams[]    name, score, color (hardcoded palette)
players[]  id, team, name, online, token
quiz       topics[5], questions[5][6], image flags, lightning image flag
board      used[5][6]
current    {topic, q, points, hasImage, imageVisible} | none
round      SETUP | BOARD | LIGHTNING | ENDED
buzzer     LOCKED | OPEN | BUZZED(playerId)
lockedOut  set of teams for current question (only if rule enabled)
timer      endsAt, paused, remaining
log        score changes (for undo)
```

**New game:** zero scores, clear board, current question, undo log, lockouts. **Keeps** team names, players, settings, loaded quiz. Rounds → BOARD.

**Resume after restart:** round, board, current question, scores restored. Buzzer starts LOCKED if it was BUZZED (open stays open). Timer restored **paused** with remaining time. Teacher resumes timer.

### Round 1 flow

```
pick cell ─► show question/image ─► buzzer OPEN (auto, setting)
                                       │
                          player buzzes ─► BUZZED(p)
                                       │
          ┌────────────────────────────┼─────────────────────────┐
        "+" team                    "−" team                  "Cancel buzz"
     add points                  subtract points              back to OPEN
     clear question              (setting) then:              no score change
     buzzer LOCKED                 reopen off → clear, LOCKED
     back to board                 reopen on  → OPEN, team (or
                                              player's team) locked out
   "Nincs válasz" (any time): clear question, buzzer LOCKED, cell stays used
   all 30 used ─► confirm prompt "Start lightning?"
```

Timer starts on first cell pick.

### Lightning flow

```
"Start lightning" ─► board hidden, lightning image/title, timer starts, buzzer OPEN
                          │
              player buzzes ─► BUZZED(p)
                          │
        teacher: "+" or "−" (3000, configurable) on the team
                          │
                 buzzer OPEN again immediately
                 (teacher reads next question from notes)
   "Cancel buzz": back to OPEN, no score change
   "End game" ─► confirm ─► results
```

No lockouts in lightning. No cell picking. Timer 00:00 does not stop anything (display only).

### Rules (in `/admin`, saved to `settings.json`)

| Setting                                                    | Default                                 |
| ---------------------------------------------------------- | --------------------------------------- |
| Round 1: wrong answer reopens buzzer for other teams       | **off**                                 |
| Lock out whole team after wrong answer (only if reopen on) | **off** (only the player is locked out) |
| Wrong answer subtracts points                              | **on**                                  |
| Auto-open buzzer on cell pick                              | **on**                                  |
| Points per question 1–6                                    | 1000, 2000, 3000, 4000, 5000, 8000      |
| Lightning points                                           | 3000                                    |
| Board size (topics × questions)                            | 5 × 6                                   |

Results: teams sorted by score descending. **Ties: shared rank.** Number format `1.000`.

---

## 9. Protocol (WebSocket, JSON)

Player → server: `join {team, name, token?}`, `buzz`
Admin → server: `setup`, `load_quiz`, `select {topic,q}`, `plus {team}`, `minus {team}`, `none`, `cancel_buzz`, `toggle_image`, `timer {pause|resume}`, `start_lightning`, `end_game`, `new_game`, `resume`, `remove_player {player}`, `clear_players`, `set_score {team,value}`, `undo`, `kick {player}`
Server → all: `state` (full snapshot), `joined {id, token}`, `too_late`, `error {msg}`

Server sends full snapshot on every change. Small (≤ 30 cells, ≤ 5 teams, ≤ ~30 players).
Visibility: `/admin` gets all question text. `/display` and phones get current question and buzzer state only.
Types hand-written in `protocol.ts`, discriminated unions.

---

## 10. Repo layout

```
quiz/
  server/
    main.go        routes, embed
    hub.go         websocket hub, state goroutine
    game.go        pure logic (testable)
    loader.go      legacy .txt loader, strict UTF-8
    settings.go    settings.json
    qr.go          LAN IP + QR
    web/dist/      generated, embedded
  web/
    src/ player.ts admin.ts display.ts help.ts protocol.ts ws.ts strings.ts
    static/ *.html style.css
    package.json tsconfig.json
  Makefile
```

Build: `cd web && npm run build`, then `GOOS=windows GOARCH=amd64 go build -ldflags "-s -w"`.
Target: Windows 10+.

---

## 11. Network, deployment, build order

**Biggest risk: client isolation** on school/guest WiFi. Plan: laptop and phones on the **same venue WiFi**. Test it first with a phone before any further coding.
Fallbacks (documented in `/help`): phone hotspot, travel router, Windows mobile hotspot.
Also: firewall prompt (document with screenshot), multiple NICs (pick private IP, admin can override), port busy → next port.
Scale: ≤ 30 phones, typically ~10.

Build order:

1. Go server + WebSocket echo. **Test phone on real venue network.**
2. Join (team + name), buzz, first-wins. Log winner.
3. Player page.
4. Loader (strict UTF-8, tests with sample file).
5. Board state, select, `+`/`−`/none/cancel. Admin page.
6. Display page.
7. Server timer.
8. Lightning.
9. Results, undo, manual score edit.
10. Settings file, rules toggles, assets folders, QR.
11. `/help`, polish, reconnect, wake lock.
12. Release build. Test on clean Windows machine.

Go tests for `game.go`: buzz order, lockout rules, scoring per settings, undo, board completion, lightning loop.

---

## 12. Decisions summary and assumptions to verify

### Decided by you

- Autosave yes. Resume prompt on start (ask). Players saved. New game keeps teams and players.
- Same-name rejoin: reject while old connection alive. Teacher can kick.
- Lightning can start any time. Buzzer reopens right after each `+`/`−`.
- Players cleared manually only.
- Board size configurable in settings, default 5 × 6.
- Admin peek button kept. QR on display before game start. Team colors hardcoded. Phones show buzzer state only. Browser auto-opens `/admin`.
- Lightning screen default text. Old look: neutral theme, you edit `ui/theme.css` later. Hardware buzzer skipped.
- Network: same WiFi.

### Defaults I took (you did not answer, verify)

| Item                       | Default taken                                         |
| -------------------------- | ----------------------------------------------------- |
| Timers at 00:00            | Display only, teacher ends game manually (as old app) |
| Negative scores            | Allowed                                               |
| After game end             | Results stay on display until new game                |
| Second instance            | Blocked by `quiz.lock`                                |
| Grid size limit            | Max 9 × 9 (image name `<topic><q>` stays unambiguous) |
| Points list on grid change | Reset to 1000 × q, editable                           |
| Teams                      | Max 5                                                 |

### Still needed from you

- **Network test** at real venue: phone + laptop same WiFi, can phone open `http://<laptop-ip>:8080`? Do this when step 1 skeleton is ready.
- One real question file converted to UTF-8, for loader test (can be anonymised, accents needed).
- Later, optional: `.dfm` files or screenshots for old look, old lightning picture.

---

## 13. Next step

Build step 1: Go server with WebSocket echo, LAN IP detection, QR page. Run on laptop, scan with phone on venue WiFi, confirm connection.
