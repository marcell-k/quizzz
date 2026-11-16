#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")"
(cd web && npm install && npm run build)
(cd server && GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o ../quiz.exe .)
echo "built quiz.exe"
