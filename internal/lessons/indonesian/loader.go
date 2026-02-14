package indonesian

import (
	"embed"
	"encoding/json"
	"fmt"
	"sort"

	"speakeasy/internal/lessons"
)

//go:embed data/*.json
var lessonData embed.FS

var allLessons []*lessons.Lesson

func init() {
	loaded, err := loadLessons()
	if err != nil {
		panic(fmt.Sprintf("failed to load indonesian lessons: %v", err))
	}
	allLessons = loaded

	lessons.Register(lessons.Language{
		Slug:          "indonesian",
		DisplayName:   "Indonesian",
		TTSCode:       "id",
		HasDualScript: false,
		ScriptLabel:   "Latin",
	}, allLessons)
}

func loadLessons() ([]*lessons.Lesson, error) {
	entries, err := lessonData.ReadDir("data")
	if err != nil {
		return nil, fmt.Errorf("read lesson dir: %w", err)
	}

	var result []*lessons.Lesson
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := lessonData.ReadFile("data/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}

		var lesson lessons.Lesson
		if err := json.Unmarshal(data, &lesson); err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}

		for i := range lesson.Quiz.Questions {
			q := &lesson.Quiz.Questions[i]
			if q.Type == "listen_and_choose" && q.WordID != "" {
				q.AudioText = findWordText(&lesson, q.WordID)
				if q.AudioText == "" && len(q.Options) > q.Correct {
					q.AudioText = q.Options[q.Correct]
				}
			}
			if q.Type == "match_pairs" {
				q.ShuffledTarget = make([]string, len(q.Pairs))
				for j, p := range q.Pairs {
					q.ShuffledTarget[len(q.Pairs)-1-j] = p.Target
				}
			}
		}

		result = append(result, &lesson)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Order < result[j].Order
	})

	return result, nil
}

func findWordText(lesson *lessons.Lesson, wordID string) string {
	for _, section := range lesson.Sections {
		for _, item := range section.Items {
			if item.ID == wordID {
				return item.TargetPrimary
			}
		}
	}
	return ""
}
