package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"speakeasy/internal/db"
	"speakeasy/internal/lessons/serbian"
	"speakeasy/internal/middleware"
)

type TemplateRenderer struct {
	templates map[string]*template.Template
}

func NewTemplateRenderer(templatesDir string) *TemplateRenderer {
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"optionLetter": func(i int) string {
			return string(rune('A' + i))
		},
	}

	layoutFile := filepath.Join(templatesDir, "layout.html")
	pages := []string{
		"home.html",
		"login.html",
		"lesson.html",
		"lesson_list.html",
		"quiz.html",
		"results.html",
	}

	templates := make(map[string]*template.Template)
	for _, page := range pages {
		tmpl := template.Must(
			template.New("").Funcs(funcMap).ParseFiles(layoutFile, filepath.Join(templatesDir, page)),
		)
		templates[page] = tmpl
	}

	return &TemplateRenderer{templates: templates}
}

func (t *TemplateRenderer) Render(w http.ResponseWriter, name string, data interface{}) {
	tmpl, ok := t.templates[name]
	if !ok {
		http.Error(w, "template not found: "+name, http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}

type LessonHandler struct {
	queries *db.Queries
	tmpl    *TemplateRenderer
}

func NewLessonHandler(q *db.Queries, t *TemplateRenderer) *LessonHandler {
	return &LessonHandler{queries: q, tmpl: t}
}

type LessonListItem struct {
	ID           string
	Title        string
	Description  string
	Order        int
	Illustration string
	Status       string
	BestScore    int64
}

func (h *LessonHandler) Home(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	if userID == 0 {
		h.tmpl.Render(w, "home.html", map[string]interface{}{
			"Title": "Home",
		})
		return
	}

	user, err := h.queries.GetUserByID(r.Context(), userID)
	if err != nil {
		h.tmpl.Render(w, "home.html", map[string]interface{}{
			"Title": "Home",
		})
		return
	}

	allLessons := serbian.GetAllLessons()
	progressList, _ := h.queries.ListLessonProgress(r.Context(), db.ListLessonProgressParams{
		UserID:   userID,
		Language: "serbian",
	})

	progressMap := make(map[string]db.LessonProgress)
	for _, p := range progressList {
		progressMap[p.LessonID] = p
	}

	completed, _ := h.queries.CountCompletedLessons(r.Context(), db.CountCompletedLessonsParams{
		UserID:   userID,
		Language: "serbian",
	})

	totalLessons := len(allLessons)
	progressPercent := 0
	if totalLessons > 0 {
		progressPercent = int(completed) * 100 / totalLessons
	}

	avgScore := 0
	if completed > 0 {
		totalScore, _ := h.queries.GetTotalScore(r.Context(), db.GetTotalScoreParams{
			UserID:   userID,
			Language: "serbian",
		})
		if ts, ok := totalScore.(int64); ok && completed > 0 {
			avgScore = int(ts) / int(completed)
		}
	}

	vocabProgress, _ := h.queries.GetVocabProgress(r.Context(), db.GetVocabProgressParams{
		UserID:   userID,
		Language: "serbian",
	})
	wordsLearned := 0
	for _, vp := range vocabProgress {
		if vp.MasteryLevel.Valid && vp.MasteryLevel.Int64 >= 1 {
			wordsLearned++
		}
	}

	// Find next available lesson
	var nextLesson *serbian.Lesson
	for _, l := range allLessons {
		p, ok := progressMap[l.ID]
		if !ok || p.Status == "available" || p.Status == "in_progress" {
			nextLesson = l
			break
		}
	}

	h.tmpl.Render(w, "home.html", map[string]interface{}{
		"Title":            "Dashboard",
		"User":             user,
		"CompletedLessons": completed,
		"TotalLessons":     totalLessons,
		"ProgressPercent":  progressPercent,
		"AvgScore":         avgScore,
		"WordsLearned":     wordsLearned,
		"NextLesson":       nextLesson,
	})
}

func (h *LessonHandler) LessonList(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	allLessons := serbian.GetAllLessons()

	progressList, _ := h.queries.ListLessonProgress(r.Context(), db.ListLessonProgressParams{
		UserID:   userID,
		Language: "serbian",
	})

	progressMap := make(map[string]db.LessonProgress)
	for _, p := range progressList {
		progressMap[p.LessonID] = p
	}

	var lessons []LessonListItem
	for _, l := range allLessons {
		status := "locked"
		var bestScore int64
		if p, ok := progressMap[l.ID]; ok {
			status = p.Status
			if p.BestScore.Valid {
				bestScore = p.BestScore.Int64
			}
		} else if l.Order == 1 {
			status = "available"
		}

		lessons = append(lessons, LessonListItem{
			ID:           l.ID,
			Title:        l.Title,
			Description:  l.Description,
			Order:        l.Order,
			Illustration: l.Illustration,
			Status:       status,
			BestScore:    bestScore,
		})
	}

	completed, _ := h.queries.CountCompletedLessons(r.Context(), db.CountCompletedLessonsParams{
		UserID:   userID,
		Language: "serbian",
	})
	totalLessons := len(allLessons)
	progressPercent := 0
	if totalLessons > 0 {
		progressPercent = int(completed) * 100 / totalLessons
	}

	h.tmpl.Render(w, "lesson_list.html", map[string]interface{}{
		"Title":           "Serbian Lessons",
		"Lessons":         lessons,
		"ProgressPercent": progressPercent,
		"User":            getUser(r.Context(), h.queries, userID),
	})
}

func (h *LessonHandler) LessonView(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	lessonID := extractLessonID(r.URL.Path)

	lesson := serbian.GetLesson(lessonID)
	if lesson == nil {
		http.NotFound(w, r)
		return
	}

	// Mark as in_progress
	now := sql.NullTime{Time: time.Now(), Valid: true}
	h.queries.UpsertLessonProgress(r.Context(), db.UpsertLessonProgressParams{
		UserID:       userID,
		Language:     "serbian",
		LessonID:     lessonID,
		Status:       "in_progress",
		LastAccessed: now,
	})

	h.tmpl.Render(w, "lesson.html", map[string]interface{}{
		"Title":  lesson.Title,
		"Lesson": lesson,
		"User":   getUser(r.Context(), h.queries, userID),
	})
}

func extractLessonID(path string) string {
	// /lessons/serbian/lesson01 or /lessons/serbian/lesson01/quiz
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

func getUser(ctx context.Context, q *db.Queries, userID int64) *db.User {
	if userID == 0 {
		return nil
	}
	user, err := q.GetUserByID(ctx, userID)
	if err != nil {
		return nil
	}
	return &user
}

func initLessonProgress(ctx context.Context, q *db.Queries, userID int64) {
	allLessons := serbian.GetAllLessons()
	for _, l := range allLessons {
		status := "locked"
		if l.Order == 1 {
			status = "available"
		}
		q.UpsertLessonProgress(ctx, db.UpsertLessonProgressParams{
			UserID:   userID,
			Language: "serbian",
			LessonID: l.ID,
			Status:   status,
		})
	}
}
