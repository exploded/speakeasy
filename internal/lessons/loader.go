package lessons

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
)

// RegisterFromFS loads all lesson JSON files from the "data/" subdirectory of
// the given filesystem and registers them under the given language config.
//
// This is the standard entry point for language packages. Each language package
// should embed its data directory and call this from init():
//
//	//go:embed data/*.json
//	var lessonData embed.FS
//
//	func init() {
//	    lessons.RegisterFromFS(lessonData, lessons.Language{...})
//	}
func RegisterFromFS(lessonFS fs.FS, lang Language) {
	entries, err := fs.ReadDir(lessonFS, "data")
	if err != nil {
		panic(fmt.Sprintf("speakeasy: load language %q: %v", lang.Slug, err))
	}

	var result []*Lesson
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := fs.ReadFile(lessonFS, "data/"+entry.Name())
		if err != nil {
			panic(fmt.Sprintf("speakeasy: load %s/%s: %v", lang.Slug, entry.Name(), err))
		}
		var lesson Lesson
		if err := json.Unmarshal(data, &lesson); err != nil {
			panic(fmt.Sprintf("speakeasy: parse %s/%s: %v", lang.Slug, entry.Name(), err))
		}

		// Compute derived fields for quiz questions
		for i := range lesson.Quiz.Questions {
			q := &lesson.Quiz.Questions[i]
			switch q.Type {
			case "listen_and_choose":
				if q.WordID != "" {
					q.AudioText = findWordInLesson(&lesson, q.WordID)
				}
				// Fallback: use the correct option text if word not found
				if q.AudioText == "" && len(q.Options) > q.Correct {
					q.AudioText = q.Options[q.Correct]
				}
			case "match_pairs":
				// Shuffle = deterministic reversal so GET and POST are consistent
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

	Register(lang, result)
}

func findWordInLesson(lesson *Lesson, wordID string) string {
	for _, section := range lesson.Sections {
		for _, item := range section.Items {
			if item.ID == wordID {
				return item.TargetPrimary
			}
		}
	}
	return ""
}
