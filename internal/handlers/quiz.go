package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"speakeasy/internal/db"
	"speakeasy/internal/lessons/serbian"
	"speakeasy/internal/middleware"
)

type QuizHandler struct {
	queries *db.Queries
	tmpl    *TemplateRenderer
}

func NewQuizHandler(q *db.Queries, t *TemplateRenderer) *QuizHandler {
	return &QuizHandler{queries: q, tmpl: t}
}

func (h *QuizHandler) QuizPage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	lessonID := extractLessonID(r.URL.Path)

	lesson := serbian.GetLesson(lessonID)
	if lesson == nil {
		http.NotFound(w, r)
		return
	}

	h.tmpl.Render(w, "quiz.html", map[string]interface{}{
		"Title":  "Quiz: " + lesson.Title,
		"Lesson": lesson,
		"User":   getUser(r.Context(), h.queries, userID),
	})
}

func (h *QuizHandler) SubmitQuiz(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	lessonID := extractLessonID(r.URL.Path)

	lesson := serbian.GetLesson(lessonID)
	if lesson == nil {
		http.NotFound(w, r)
		return
	}

	r.ParseForm()

	totalStr := r.FormValue("total")
	total, _ := strconv.Atoi(totalStr)
	if total == 0 {
		total = len(lesson.Quiz.Questions)
	}

	correct := 0
	for i := 0; i < total; i++ {
		if i >= len(lesson.Quiz.Questions) {
			break
		}
		q := lesson.Quiz.Questions[i]
		answer := r.FormValue("answer-" + strconv.Itoa(i))

		switch q.Type {
		case "multiple_choice", "listen_and_choose":
			idx, err := strconv.Atoi(answer)
			if err == nil && idx == q.Correct {
				correct++
				updateVocabCorrect(r, h.queries, userID, q, true)
			} else {
				updateVocabCorrect(r, h.queries, userID, q, false)
			}

		case "type_answer":
			answer = strings.TrimSpace(answer)
			for _, ca := range q.CorrectAnswers {
				if strings.EqualFold(answer, ca) {
					correct++
					break
				}
			}

		case "match_pairs":
			if isMatchCorrect(answer, q.Pairs, q.ShuffledSerbian) {
				correct++
			}
		}
	}

	score := 0
	if total > 0 {
		score = correct * 100 / total
	}

	// Save quiz attempt
	now := sql.NullTime{Time: time.Now(), Valid: true}
	h.queries.CreateQuizAttempt(r.Context(), db.CreateQuizAttemptParams{
		UserID:         userID,
		Language:       "serbian",
		LessonID:       lessonID,
		Score:          int64(score),
		TotalQuestions: int64(total),
		CorrectAnswers: int64(correct),
	})

	// Update lesson progress
	status := "in_progress"
	var completedAt sql.NullTime
	if score >= 70 {
		status = "completed"
		completedAt = now

		// Unlock next lesson
		nextID := serbian.GetNextLessonID(lessonID)
		if nextID != "" {
			// Check if next lesson is locked
			progress, err := h.queries.GetLessonProgress(r.Context(), db.GetLessonProgressParams{
				UserID:   userID,
				Language: "serbian",
				LessonID: nextID,
			})
			if err != nil || progress.Status == "locked" {
				h.queries.UpsertLessonProgress(r.Context(), db.UpsertLessonProgressParams{
					UserID:   userID,
					Language: "serbian",
					LessonID: nextID,
					Status:   "available",
				})
			}
		}
	}

	// Get current attempts count
	existing, _ := h.queries.GetLessonProgress(r.Context(), db.GetLessonProgressParams{
		UserID:   userID,
		Language: "serbian",
		LessonID: lessonID,
	})
	attempts := int64(1)
	if existing.Attempts.Valid {
		attempts = existing.Attempts.Int64 + 1
	}

	h.queries.UpsertLessonProgress(r.Context(), db.UpsertLessonProgressParams{
		UserID:       userID,
		Language:     "serbian",
		LessonID:     lessonID,
		Status:       status,
		BestScore:    sql.NullInt64{Int64: int64(score), Valid: true},
		Attempts:     sql.NullInt64{Int64: attempts, Valid: true},
		LastAccessed: now,
		CompletedAt:  completedAt,
	})

	nextLessonID := ""
	if score >= 70 {
		nextLessonID = serbian.GetNextLessonID(lessonID)
	}

	h.tmpl.Render(w, "results.html", map[string]interface{}{
		"Title":        "Quiz Results",
		"Lesson":       lesson,
		"Score":        score,
		"Correct":      correct,
		"Total":        total,
		"Passed":       score >= 70,
		"Perfect":      score >= 100,
		"Excellent":    score >= 90,
		"HalfWay":      score >= 50,
		"NextLessonID": nextLessonID,
		"User":         getUser(r.Context(), h.queries, userID),
	})
}

func isMatchCorrect(answer string, pairs []serbian.Pair, shuffled []string) bool {
	if answer == "" {
		return false
	}

	var matched []struct {
		English int `json:"english"`
		Serbian int `json:"serbian"`
	}
	if err := json.Unmarshal([]byte(answer), &matched); err != nil {
		return false
	}

	if len(matched) != len(pairs) {
		return false
	}

	for _, m := range matched {
		if m.English < 0 || m.English >= len(pairs) || m.Serbian < 0 || m.Serbian >= len(shuffled) {
			return false
		}
		expected := pairs[m.English].Serbian
		actual := shuffled[m.Serbian]
		if expected != actual {
			return false
		}
	}

	return true
}

func updateVocabCorrect(r *http.Request, q *db.Queries, userID int64, question serbian.Question, isCorrect bool) {
	wordID := question.WordID
	if wordID == "" {
		return
	}

	existing, err := q.GetVocabProgressByWord(r.Context(), db.GetVocabProgressByWordParams{
		UserID:   userID,
		Language: "serbian",
		WordID:   wordID,
	})

	timesCorrect := int64(0)
	timesIncorrect := int64(0)
	if err == nil {
		if existing.TimesCorrect.Valid {
			timesCorrect = existing.TimesCorrect.Int64
		}
		if existing.TimesIncorrect.Valid {
			timesIncorrect = existing.TimesIncorrect.Int64
		}
	}

	if isCorrect {
		timesCorrect++
	} else {
		timesIncorrect++
	}

	mastery := int64(0)
	total := timesCorrect + timesIncorrect
	if total > 0 {
		ratio := float64(timesCorrect) / float64(total)
		if ratio >= 0.9 && timesCorrect >= 5 {
			mastery = 5
		} else if ratio >= 0.8 && timesCorrect >= 3 {
			mastery = 4
		} else if ratio >= 0.7 && timesCorrect >= 2 {
			mastery = 3
		} else if ratio >= 0.5 {
			mastery = 2
		} else if timesCorrect >= 1 {
			mastery = 1
		}
	}

	q.UpsertVocabProgress(r.Context(), db.UpsertVocabProgressParams{
		UserID:         userID,
		Language:       "serbian",
		WordID:         wordID,
		TimesCorrect:   sql.NullInt64{Int64: timesCorrect, Valid: true},
		TimesIncorrect: sql.NullInt64{Int64: timesIncorrect, Valid: true},
		MasteryLevel:   sql.NullInt64{Int64: mastery, Valid: true},
		LastReviewed:   sql.NullTime{Time: time.Now(), Valid: true},
	})
}
