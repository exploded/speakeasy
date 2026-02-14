package lessons

import (
	"sort"
	"sync"
)

var (
	mu        sync.RWMutex
	languages = make(map[string]*registeredLanguage)
)

type registeredLanguage struct {
	config  Language
	lessons []*Lesson
	byID    map[string]*Lesson
}

// Register adds a language and its lessons to the global registry.
// Typically called from a language package's init() function.
func Register(lang Language, lessons []*Lesson) {
	mu.Lock()
	defer mu.Unlock()

	sorted := make([]*Lesson, len(lessons))
	copy(sorted, lessons)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Order < sorted[j].Order
	})

	byID := make(map[string]*Lesson, len(sorted))
	for _, l := range sorted {
		byID[l.ID] = l
	}

	languages[lang.Slug] = &registeredLanguage{
		config:  lang,
		lessons: sorted,
		byID:    byID,
	}
}

// GetLanguages returns all registered language configs, sorted by display name.
func GetLanguages() []Language {
	mu.RLock()
	defer mu.RUnlock()

	var result []Language
	for _, rl := range languages {
		result = append(result, rl.config)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].DisplayName < result[j].DisplayName
	})
	return result
}

// GetLanguage returns the config for a given language slug, or nil if not found.
func GetLanguage(slug string) *Language {
	mu.RLock()
	defer mu.RUnlock()

	rl, ok := languages[slug]
	if !ok {
		return nil
	}
	lang := rl.config
	return &lang
}

// GetAllLessons returns all lessons for a language, sorted by order.
func GetAllLessons(slug string) []*Lesson {
	mu.RLock()
	defer mu.RUnlock()

	rl, ok := languages[slug]
	if !ok {
		return nil
	}
	return rl.lessons
}

// GetLesson returns a specific lesson by language slug and lesson ID.
func GetLesson(slug, id string) *Lesson {
	mu.RLock()
	defer mu.RUnlock()

	rl, ok := languages[slug]
	if !ok {
		return nil
	}
	return rl.byID[id]
}

// GetNextLessonID returns the ID of the next lesson after currentID, or "" if none.
func GetNextLessonID(slug, currentID string) string {
	mu.RLock()
	defer mu.RUnlock()

	rl, ok := languages[slug]
	if !ok {
		return ""
	}
	current, ok := rl.byID[currentID]
	if !ok {
		return ""
	}
	for _, l := range rl.lessons {
		if l.Order == current.Order+1 {
			return l.ID
		}
	}
	return ""
}
