# SpeakEasy

A web-based language tutor for adult English speakers. Learn vocabulary, grammar, and cultural context through progressive lessons with interactive quizzes and text-to-speech audio. Currently supports **Serbian**, **Croatian**, and **Indonesian**.

**Demo:** [speakeasy.mchugh.au](https://speakeasy.mchugh.au/)

## Features

- **3 languages** — Serbian (5 lessons), Croatian (5 lessons), Indonesian (6 lessons)
- **Dual-script support** — toggle between Cyrillic, Latin, or both for Serbian
- **4 quiz types** — multiple choice, type answer, match pairs, listen & choose
- **Text-to-speech** — native pronunciation via Google Cloud TTS with server-side caching
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
  lessons/                   Shared types, registry, and per-language loaders with embedded JSON
  tts/                       Google Cloud TTS client with file-based caching
web/
  templates/                 Go HTML templates (layout + 6 page templates)
  static/                    CSS, JS, and SVG assets
deploy/                      systemd service and nginx config for production
```

Lesson content is defined as JSON files (e.g. `internal/lessons/serbian/data/lesson01-05.json`) containing vocabulary items, pronunciation hints, example sentences, grammar notes, cultural context, and quiz questions. New languages self-register via Go's `init()` pattern — just add a package with a loader and JSON data, import it, and rebuild.

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

### Deploy to a Debian/Ubuntu server

Reference deployment files for systemd and nginx are in the `deploy/` directory.

#### 1. Cross-compile the binary

```bash
# From macOS or Linux
GOOS=linux GOARCH=amd64 go build -o speakeasy-linux ./cmd/server/

# From Windows (Command Prompt)
set GOOS=linux&& set GOARCH=amd64&& go build -o speakeasy-linux ./cmd/server/
```

#### 2. Upload files to the server

Upload `speakeasy-linux` and the `web/` directory to your home directory on the server (e.g. via `scp`, `rsync`, or SFTP):

```bash
scp speakeasy-linux user@yourserver:~/
scp -r web/ user@yourserver:~/web/
```

#### 3. First-time setup (only needed once)

SSH into the server and create the application directory:

```bash
# Create the app directory and set ownership
sudo mkdir -p /var/www/speakeasy
sudo chown www-data:www-data /var/www/speakeasy

# Copy files into place
sudo cp ~/speakeasy-linux /var/www/speakeasy/
sudo cp -r ~/web /var/www/speakeasy/
sudo chown -R www-data:www-data /var/www/speakeasy
sudo chmod 755 /var/www/speakeasy/speakeasy-linux

# Install the systemd service
sudo cp deploy/speakeasy.service /etc/systemd/system/
# Edit the service file to set your Google TTS API key:
sudo nano /etc/systemd/system/speakeasy.service
# Change: Environment=GOOGLE_TTS_API_KEY=your-api-key-here

# Enable and start the service
sudo systemctl daemon-reload
sudo systemctl enable speakeasy
sudo systemctl start speakeasy

# Install the nginx config (adjust server_name as needed)
sudo cp deploy/speakeasy.conf /etc/nginx/conf.d/
sudo nginx -t && sudo systemctl reload nginx
```

The app runs on port 8282 behind nginx, which proxies requests from port 80. The SQLite database (`speakeasy.db`) and TTS cache (`tts_cache/`) are created automatically in `/var/www/speakeasy/` on first run.

#### 4. Deploying updates

After building a new binary (and updating `web/` if templates or static files changed):

```bash
# Upload the new files
scp speakeasy-linux user@yourserver:~/
scp -r web/ user@yourserver:~/web/    # only if templates/static changed

# SSH in and deploy
ssh user@yourserver

# Stop the service, copy files, fix permissions, restart
sudo systemctl stop speakeasy
sudo cp ~/speakeasy-linux /var/www/speakeasy/
sudo cp -r ~/web /var/www/speakeasy/   # only if templates/static changed
sudo chown -R www-data:www-data /var/www/speakeasy
sudo chmod 755 /var/www/speakeasy/speakeasy-linux
sudo systemctl start speakeasy

# Verify it's running
sudo systemctl status speakeasy
```

#### 5. Useful commands

```bash
# Check service status
sudo systemctl status speakeasy

# View application logs
sudo journalctl -u speakeasy -f

# Restart after config changes
sudo systemctl daemon-reload
sudo systemctl restart speakeasy

# Check nginx config syntax
sudo nginx -t
```

#### 6. SSL with Let's Encrypt (optional)

```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d speakeasy.mchugh.au
```

Certbot will modify the nginx config to handle HTTPS and set up auto-renewal.

## License

[MIT](LICENSE)
