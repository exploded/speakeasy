package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"speakeasy/internal/db"
	"speakeasy/internal/handlers"
	"speakeasy/internal/middleware"
	"speakeasy/internal/tts"

	_ "modernc.org/sqlite"
)

func main() {
	// Determine data directory
	dataDir := os.Getenv("SPEAKEASY_DATA_DIR")
	if dataDir == "" {
		dataDir = "."
	}

	// Open SQLite database (pure Go driver)
	dbPath := filepath.Join(dataDir, "speakeasy.db")
	database, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Enable WAL mode for better concurrency
	database.Exec("PRAGMA journal_mode=WAL")
	database.Exec("PRAGMA foreign_keys=ON")

	// Initialize schema
	if err := initSchema(database); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	queries := db.New(database)
	sessions := middleware.NewSessionStore()

	// Determine template and static directories
	webDir := "web"
	if _, err := os.Stat(webDir); os.IsNotExist(err) {
		// Try relative to executable
		exe, _ := os.Executable()
		webDir = filepath.Join(filepath.Dir(exe), "web")
	}

	templatesDir := filepath.Join(webDir, "templates")
	staticDir := filepath.Join(webDir, "static")

	// TTS client
	cacheDir := filepath.Join(dataDir, "tts_cache")
	audioDir := filepath.Join(staticDir, "audio")
	ttsClient := tts.NewClient(cacheDir, audioDir)

	// Template renderer
	tmpl := handlers.NewTemplateRenderer(templatesDir)

	// Handlers
	authHandler := handlers.NewAuthHandler(queries, sessions, tmpl)
	lessonHandler := handlers.NewLessonHandler(queries, tmpl)
	quizHandler := handlers.NewQuizHandler(queries, tmpl)
	progressHandler := handlers.NewProgressHandler(sessions)
	ttsHandler := handlers.NewTTSHandler(ttsClient)

	// Mux
	mux := http.NewServeMux()

	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// Public routes
	mux.HandleFunc("/", lessonHandler.Home)
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			authHandler.Login(w, r)
		} else {
			authHandler.LoginPage(w, r)
		}
	})
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			authHandler.Register(w, r)
		} else {
			authHandler.RegisterPage(w, r)
		}
	})
	mux.HandleFunc("/logout", authHandler.Logout)

	// Protected routes
	mux.HandleFunc("/lessons/serbian", middleware.RequireAuth(lessonHandler.LessonList))
	mux.HandleFunc("/lessons/serbian/", middleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		// Route to quiz or lesson view
		if pathEndsWithQuiz(r.URL.Path) {
			if r.Method == http.MethodPost {
				quizHandler.SubmitQuiz(w, r)
			} else {
				quizHandler.QuizPage(w, r)
			}
		} else {
			lessonHandler.LessonView(w, r)
		}
	}))

	// API routes
	mux.HandleFunc("/api/tts", ttsHandler.ServeAudio)
	mux.HandleFunc("/api/preference/script", middleware.RequireAuth(progressHandler.SetScriptPreference))

	// Wrap with auth middleware
	handler := sessions.AuthMiddleware(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("SpeakEasy server starting on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func pathEndsWithQuiz(path string) bool {
	// Check if path ends with /quiz
	clean := filepath.ToSlash(path)
	if len(clean) > 0 && clean[len(clean)-1] == '/' {
		clean = clean[:len(clean)-1]
	}
	return len(clean) > 5 && clean[len(clean)-5:] == "/quiz"
}

func initSchema(database *sql.DB) error {
	_, err := database.Exec(db.SchemaSQL)
	return err
}
