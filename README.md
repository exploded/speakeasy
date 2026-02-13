# SpeakEasy

A web-based Serbian language tutor for adult English speakers. Learn vocabulary, grammar, and cultural context through 5 progressive lessons with interactive quizzes and text-to-speech audio.

**Demo:** [speakeasy.mchugh.au](https://speakeasy.mchugh.au/)

## Features

- **5 progressive lessons** covering greetings, numbers, common phrases, family, and food
- **Dual-script support** — toggle between Cyrillic, Latin, or both side-by-side
- **4 quiz types** — multiple choice, type answer, match pairs, listen & choose
- **Text-to-speech** — native Serbian pronunciation via Google Cloud TTS with server-side caching
- **Progress tracking** — score 70% or higher to unlock the next lesson, with per-word mastery tracking
- **User accounts** — registration, login, and per-user progress with bcrypt password hashing and session cookies

## Tech Stack

SpeakEasy is a server-rendered Go application with zero JavaScript framework dependencies. The entire frontend is vanilla HTML, CSS, and JS, enhanced with htmx for interactivity.

| Layer | Technology | Notes |
|-------|-----------|-------|
| **Server** | Go `net/http` | Standard library HTTP server, no web framework |
| **Templates** | Go `html/template` | Server-rendered pages with a shared layout and per-page content blocks |
| **Database** | SQLite | Via [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) — a pure Go driver, no CGO required |
| **SQL** | [sqlc](https://sqlc.dev/) | Type-safe Go code generated from raw SQL queries |
| **Frontend** | htmx + vanilla JS | Minimal client-side code; htmx handles dynamic interactions |
| **Styling** | Vanilla CSS | Custom responsive design with no CSS framework |
| **Illustrations** | Inline SVG | Hand-crafted owl mascot ("Mila") and per-lesson artwork |
| **TTS** | Google Cloud Text-to-Speech | Serbian audio with server-side MP3 caching to minimize API calls |
| **Auth** | bcrypt + session cookies | Secure password hashing with in-memory session store |

### Architecture

The project follows a clean Go project layout:

```
cmd/server/main.go          Entry point, routing, schema init
internal/
  handlers/                  HTTP handlers (auth, lessons, quiz, progress, TTS)
  middleware/                 Session store and auth middleware
  db/                        sqlc-generated database layer (schema.sql, queries.sql)
  lessons/serbian/           Lesson data loader with embedded JSON
  tts/                       Google Cloud TTS client with file-based caching
web/
  templates/                 Go HTML templates (layout + 6 page templates)
  static/                    CSS, JS, and SVG assets
deploy/                      systemd service and nginx config for production
```

Lesson content is defined as JSON files (`internal/lessons/serbian/data/lesson01-05.json`) containing vocabulary items with Cyrillic and Latin script, pronunciation hints, example sentences, grammar notes, cultural context, and quiz questions. This makes it straightforward to add new lessons or languages without touching Go code.

The TTS system uses a layered lookup — pre-recorded audio overrides, then cached API responses, then live Google Cloud TTS calls. Failed API calls are never cached, so transient errors don't permanently break audio for a word.

## How It Was Built

SpeakEasy was built entirely through pair programming with [Claude Code](https://claude.ai/code) (Anthropic's AI coding assistant). The entire application — backend, frontend, lesson content, SVG artwork, and deployment configuration — was developed conversationally in a series of sessions.

## Getting Started

### Prerequisites

- **Go 1.24+** — [install instructions](https://go.dev/doc/install). The project uses the pure Go SQLite driver, so no C compiler or CGO is needed.
- **Google Cloud TTS API key** — get one from the [Google Cloud Console](https://console.cloud.google.com/apis/library/texttospeech.googleapis.com). Enable the "Cloud Text-to-Speech API" and create an API key. The app works without it, but pronunciation audio won't be available.
- **sqlc** (optional) — only needed if you modify the SQL queries in `internal/db/queries.sql`. Install from [sqlc.dev](https://sqlc.dev/) and run `sqlc generate` to regenerate the Go code. The generated code is already checked in.

No other dependencies are required. There is no Node.js, no npm, no build step for frontend assets. Just `go build` and run.

### Run locally

```bash
git clone https://github.com/exploded/speakeasy.git
cd speakeasy

export GOOGLE_TTS_API_KEY=your-api-key-here
go run ./cmd/server/
```

The app will be available at http://localhost:8080. A SQLite database file (`speakeasy.db`) is created automatically on first run.

On Windows, edit `start.bat` with your API key and run it instead.

### Deploy to a Linux server

Deployment files for systemd and nginx are in the `deploy/` directory.

```bash
# Cross-compile from any OS
GOOS=linux GOARCH=amd64 go build -o speakeasy-linux ./cmd/server/
```

Copy the binary and the `web/` directory to your server. Set `GOOGLE_TTS_API_KEY` in the systemd service file. See `deploy/speakeasy.service` and `deploy/speakeasy.conf` for reference configs.

## License

[MIT](LICENSE)
