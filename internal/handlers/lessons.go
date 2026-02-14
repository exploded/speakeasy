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
	"speakeasy/internal/lessons"
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

type LanguageSummary struct {
	Slug        string
	DisplayName string
	Completed   int64
	Total       int
	Percent     int
}

func (h *LessonHandler) Home(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	if userID == 0 {
		var summaries []LanguageSummary
		for _, lang := range lessons.GetLanguages() {
			summaries = append(summaries, LanguageSummary{
				Slug:        lang.Slug,
				DisplayName: lang.DisplayName,
				Total:       len(lessons.GetAllLessons(lang.Slug)),
			})
		}
		h.tmpl.Render(w, "home.html", map[string]interface{}{
			"Title":         "Home",
			"LangSummaries": summaries,
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

	allLanguages := lessons.GetLanguages()
	var langSummaries []LanguageSummary

	totalCompleted := int64(0)
	totalLessons := 0
	totalWordsLearned := 0
	totalScoreSum := int64(0)
	totalCompletedForAvg := int64(0)

	for _, lang := range allLanguages {
		langLessons := lessons.GetAllLessons(lang.Slug)
		completed, _ := h.queries.CountCompletedLessons(r.Context(), db.CountCompletedLessonsParams{
			UserID:   userID,
			Language: lang.Slug,
		})

		total := len(langLessons)
		pct := 0
		if total > 0 {
			pct = int(completed) * 100 / total
		}

		langSummaries = append(langSummaries, LanguageSummary{
			Slug:        lang.Slug,
			DisplayName: lang.DisplayName,
			Completed:   completed,
			Total:       total,
			Percent:     pct,
		})

		totalCompleted += completed
		totalLessons += total

		if completed > 0 {
			score, _ := h.queries.GetTotalScore(r.Context(), db.GetTotalScoreParams{
				UserID:   userID,
				Language: lang.Slug,
			})
			if ts, ok := score.(int64); ok {
				totalScoreSum += ts
				totalCompletedForAvg += completed
			}
		}

		vocabProgress, _ := h.queries.GetVocabProgress(r.Context(), db.GetVocabProgressParams{
			UserID:   userID,
			Language: lang.Slug,
		})
		for _, vp := range vocabProgress {
			if vp.MasteryLevel.Valid && vp.MasteryLevel.Int64 >= 1 {
				totalWordsLearned++
			}
		}
	}

	progressPercent := 0
	if totalLessons > 0 {
		progressPercent = int(totalCompleted) * 100 / totalLessons
	}

	avgScore := 0
	if totalCompletedForAvg > 0 {
		avgScore = int(totalScoreSum) / int(totalCompletedForAvg)
	}

	h.tmpl.Render(w, "home.html", map[string]interface{}{
		"Title":            "Dashboard",
		"User":             user,
		"Languages":        allLanguages,
		"LangSummaries":    langSummaries,
		"CompletedLessons": totalCompleted,
		"TotalLessons":     totalLessons,
		"ProgressPercent":  progressPercent,
		"AvgScore":         avgScore,
		"WordsLearned":     totalWordsLearned,
	})
}

func (h *LessonHandler) LessonList(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	langSlug := extractLanguage(r.URL.Path)

	langConfig := lessons.GetLanguage(langSlug)
	if langConfig == nil {
		http.NotFound(w, r)
		return
	}

	allLessons := lessons.GetAllLessons(langSlug)

	progressList, _ := h.queries.ListLessonProgress(r.Context(), db.ListLessonProgressParams{
		UserID:   userID,
		Language: langSlug,
	})

	progressMap := make(map[string]db.LessonProgress)
	for _, p := range progressList {
		progressMap[p.LessonID] = p
	}

	var lessonItems []LessonListItem
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

		lessonItems = append(lessonItems, LessonListItem{
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
		Language: langSlug,
	})
	totalLessons := len(allLessons)
	progressPercent := 0
	if totalLessons > 0 {
		progressPercent = int(completed) * 100 / totalLessons
	}

	h.tmpl.Render(w, "lesson_list.html", map[string]interface{}{
		"Title":           langConfig.DisplayName + " Lessons",
		"Lessons":         lessonItems,
		"ProgressPercent": progressPercent,
		"User":            getUser(r.Context(), h.queries, userID),
		"LanguageSlug":    langSlug,
		"LanguageName":    langConfig.DisplayName,
		"LanguageConfig":  langConfig,
	})
}

func (h *LessonHandler) LessonView(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	langSlug := extractLanguage(r.URL.Path)
	lessonID := extractLessonID(r.URL.Path)

	langConfig := lessons.GetLanguage(langSlug)
	if langConfig == nil {
		http.NotFound(w, r)
		return
	}

	lesson := lessons.GetLesson(langSlug, lessonID)
	if lesson == nil {
		http.NotFound(w, r)
		return
	}

	// Mark as in_progress
	now := sql.NullTime{Time: time.Now(), Valid: true}
	h.queries.UpsertLessonProgress(r.Context(), db.UpsertLessonProgressParams{
		UserID:       userID,
		Language:     langSlug,
		LessonID:     lessonID,
		Status:       "in_progress",
		LastAccessed: now,
	})

	h.tmpl.Render(w, "lesson.html", map[string]interface{}{
		"Title":          lesson.Title,
		"Lesson":         lesson,
		"User":           getUser(r.Context(), h.queries, userID),
		"LanguageSlug":   langSlug,
		"LanguageName":   langConfig.DisplayName,
		"LanguageConfig": langConfig,
	})
}

// extractLanguage extracts the language slug from a URL path like /lessons/serbian/...
func extractLanguage(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

func extractLessonID(path string) string {
	// /lessons/{language}/lesson01 or /lessons/{language}/lesson01/quiz
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

func initLessonProgress(ctx context.Context, q *db.Queries, userID int64, langSlug string) {
	allLessons := lessons.GetAllLessons(langSlug)
	for _, l := range allLessons {
		status := "locked"
		if l.Order == 1 {
			status = "available"
		}
		q.UpsertLessonProgress(ctx, db.UpsertLessonProgressParams{
			UserID:   userID,
			Language: langSlug,
			LessonID: l.ID,
			Status:   status,
		})
	}
}
