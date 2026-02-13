-- name: CreateUser :one
INSERT INTO users (username, email, password_hash, display_name)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = ?;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;

-- name: GetLessonProgress :one
SELECT * FROM lesson_progress
WHERE user_id = ? AND language = ? AND lesson_id = ?;

-- name: ListLessonProgress :many
SELECT * FROM lesson_progress
WHERE user_id = ? AND language = ?
ORDER BY lesson_id;

-- name: UpsertLessonProgress :one
INSERT INTO lesson_progress (user_id, language, lesson_id, status, best_score, attempts, last_accessed, completed_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id, language, lesson_id)
DO UPDATE SET
    status = excluded.status,
    best_score = CASE WHEN excluded.best_score > lesson_progress.best_score THEN excluded.best_score ELSE lesson_progress.best_score END,
    attempts = excluded.attempts,
    last_accessed = excluded.last_accessed,
    completed_at = COALESCE(excluded.completed_at, lesson_progress.completed_at)
RETURNING *;

-- name: CreateQuizAttempt :one
INSERT INTO quiz_attempts (user_id, language, lesson_id, score, total_questions, correct_answers)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListQuizAttempts :many
SELECT * FROM quiz_attempts
WHERE user_id = ? AND language = ? AND lesson_id = ?
ORDER BY attempted_at DESC;

-- name: UpsertVocabProgress :one
INSERT INTO vocab_progress (user_id, language, word_id, times_correct, times_incorrect, mastery_level, last_reviewed)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id, language, word_id)
DO UPDATE SET
    times_correct = excluded.times_correct,
    times_incorrect = excluded.times_incorrect,
    mastery_level = excluded.mastery_level,
    last_reviewed = excluded.last_reviewed
RETURNING *;

-- name: GetVocabProgress :many
SELECT * FROM vocab_progress
WHERE user_id = ? AND language = ?;

-- name: GetVocabProgressByWord :one
SELECT * FROM vocab_progress
WHERE user_id = ? AND language = ? AND word_id = ?;

-- name: CountCompletedLessons :one
SELECT COUNT(*) FROM lesson_progress
WHERE user_id = ? AND language = ? AND status = 'completed';

-- name: GetTotalScore :one
SELECT COALESCE(SUM(best_score), 0) FROM lesson_progress
WHERE user_id = ? AND language = ?;
