# Project: Quiz Rewrite (Delphi → Go + TypeScript web buzzer)

## What this is

Rewrite of 20+ year old Delphi quiz app. Teacher runs one quiz.exe on Windows 10+. Players use phones as buzzers (scan QR, pick team, type name, red button). Quiz board shown in browser on projector. All on same local WiFi, no internet needed.

## Sources

- quiz-plan-final.md: source of truth for decisions, rules, protocol, folders, build order.
- GitHub repo: current code.
- If plan and code disagree, say so. Ask which is right. Never silently pick.
- If a new decision changes the plan, say so and give exact plan text to update.

## Stack (fixed, do not suggest alternatives)

- Backend: Go, whole thing. github.com/coder/websocket, github.com/skip2/go-qrcode. Single static exe, frontend embedded with go:embed.
- Frontend: TypeScript, vanilla, no framework. esbuild bundles, tsc --noEmit checks. Three pages: / (player), /admin (teacher, loopback only), /display (projector). Plus /help.
- Hungarian UI. All labels in strings.ts.

## Core rules (details in plan)

- Server authoritative. Phones send only buzz. First message processed wins.
- Single goroutine owns game state. Input via channel. No mutex.
- All buzz sources call one function Buzz(playerId). Hardware buzzer may come later.
- Score belongs to team. Buzz belongs to player. Display shows "Team A: Anna".
- Board default 5 topics x 6 questions, configurable (max 9x9). Points 1000-5000, 8000. Lightning 3000.
- Round 1: pick cell, question shows, buzzer opens at once. Teacher presses + or -. Rules toggles in settings.json.
- Lightning: buzzer opens at start, reopens right after each + or -. No lockouts, no question data.
- Question file: legacy .txt, strict UTF-8, topic line + N questions + blank line per block. Images <quiz>_<topic><q>.jpg.
- Files next to exe: settings.json, autosave.json (atomic write), quiz.lock, history.csv, ui/, quizzes/.
- History: on End game, append one row per team to history.csv. Semicolon separator, UTF-8 with BOM, header on creation. Columns: date_time; title; quiz_file; rank; team; score; players. Ties share rank. Locked file (Excel open) = show retry message, result stays pending.
- Admin = loopback only. Players never get admin rights.

## How to work

- Follow plan build order. One step at a time. Do not jump ahead.
- Give complete files or exact diffs. Code must compile. No placeholders like "..." in code.
- Go: pure logic in game.go, testable. Write tests for rules (buzz order, lockout, scoring, undo, lightning loop, history rows).
- TS: strict mode. Protocol types in protocol.ts as discriminated unions. Keep in sync with Go structs, flag any change on both sides.
- Windows 10+ target. Check paths, file locking, line endings (CRLF in old files).
- Old Delphi source (MvM.pas etc.) is reference for rules. Never invent rules. If unclear, ask.
- Missing info: ask max 1-3 questions with defaults. Do not block on small things.
- Do not add features outside plan. Suggest them separately, short.
- Biggest risk is network (client isolation). Remind when relevant.

## Reply style

Drop articles, filler, pleasantries, hedging. Fragments OK. Short words. Technical terms exact. Code blocks unchanged. Errors quoted exact. Pattern: [thing] [action] [reason]. [next step].
