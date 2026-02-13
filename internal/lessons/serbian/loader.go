package serbian

import (
	"embed"
	"encoding/json"
	"fmt"
	"sort"
)

//go:embed data/*.json
var lessonData embed.FS

type Lesson struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Order        int       `json:"order"`
	Prerequisite *string   `json:"prerequisite"`
	Illustration string    `json:"illustration"`
	Sections     []Section `json:"sections"`
	Quiz         Quiz      `json:"quiz"`
}

type Section struct {
	Type        string        `json:"type"`
	Title       string        `json:"title"`
	Items       []VocabItem   `json:"items,omitempty"`
	Explanation string        `json:"explanation,omitempty"`
	Examples    []Example     `json:"examples,omitempty"`
	Content     string        `json:"content,omitempty"`
}

type VocabItem struct {
	ID                string           `json:"id"`
	English           string           `json:"english"`
	SerbianCyrillic   string           `json:"serbian_cyrillic"`
	SerbianLatin      string           `json:"serbian_latin"`
	PronunciationHint string           `json:"pronunciation_hint"`
	AudioOverride     *string          `json:"audio_override"`
	ExampleSentence   *ExampleSentence `json:"example_sentence"`
}

type ExampleSentence struct {
	English         string `json:"english"`
	SerbianCyrillic string `json:"serbian_cyrillic"`
	SerbianLatin    string `json:"serbian_latin"`
}

type Example struct {
	English         string `json:"english"`
	SerbianCyrillic string `json:"serbian_cyrillic"`
	SerbianLatin    string `json:"serbian_latin"`
}

type Quiz struct {
	Questions []Question `json:"questions"`
}

type Question struct {
	Type           string   `json:"type"`
	Question       string   `json:"question,omitempty"`
	Options        []string `json:"options,omitempty"`
	Correct        int      `json:"correct,omitempty"`
	Prompt         string   `json:"prompt,omitempty"`
	CorrectAnswers []string `json:"correct_answers,omitempty"`
	Pairs          []Pair   `json:"pairs,omitempty"`
	WordID         string   `json:"word_id,omitempty"`

	// Computed fields for template rendering
	ShuffledSerbian []string `json:"-"`
	AudioText       string   `json:"-"`
}

type Pair struct {
	English string `json:"english"`
	Serbian string `json:"serbian"`
}

var allLessons []*Lesson
var lessonMap map[string]*Lesson

func init() {
	lessonMap = make(map[string]*Lesson)
	if err := loadLessons(); err != nil {
		panic(fmt.Sprintf("failed to load serbian lessons: %v", err))
	}
}

func loadLessons() error {
	entries, err := lessonData.ReadDir("data")
	if err != nil {
		return fmt.Errorf("read lesson dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := lessonData.ReadFile("data/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read %s: %w", entry.Name(), err)
		}

		var lesson Lesson
		if err := json.Unmarshal(data, &lesson); err != nil {
			return fmt.Errorf("parse %s: %w", entry.Name(), err)
		}

		// Compute audio text for listen_and_choose questions
		for i := range lesson.Quiz.Questions {
			q := &lesson.Quiz.Questions[i]
			if q.Type == "listen_and_choose" && q.WordID != "" {
				q.AudioText = findWordText(&lesson, q.WordID)
				if q.AudioText == "" && len(q.Options) > q.Correct {
					q.AudioText = q.Options[q.Correct]
				}
			}
			if q.Type == "match_pairs" {
				// Create shuffled serbian list (simple reverse for determinism)
				q.ShuffledSerbian = make([]string, len(q.Pairs))
				for j, p := range q.Pairs {
					q.ShuffledSerbian[len(q.Pairs)-1-j] = p.Serbian
				}
			}
		}

		allLessons = append(allLessons, &lesson)
		lessonMap[lesson.ID] = &lesson
	}

	sort.Slice(allLessons, func(i, j int) bool {
		return allLessons[i].Order < allLessons[j].Order
	})

	return nil
}

func findWordText(lesson *Lesson, wordID string) string {
	for _, section := range lesson.Sections {
		for _, item := range section.Items {
			if item.ID == wordID {
				return item.SerbianLatin
			}
		}
	}
	return ""
}

func GetAllLessons() []*Lesson {
	return allLessons
}

func GetLesson(id string) *Lesson {
	return lessonMap[id]
}

func GetNextLessonID(currentID string) string {
	current := lessonMap[currentID]
	if current == nil {
		return ""
	}
	for _, l := range allLessons {
		if l.Order == current.Order+1 {
			return l.ID
		}
	}
	return ""
}
