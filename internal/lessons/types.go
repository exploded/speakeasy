package lessons

// Language describes a language available in SpeakEasy.
type Language struct {
	Slug        string // URL-safe identifier, e.g. "serbian"
	DisplayName string // Human-readable name, e.g. "Serbian"
	TTSCode     string // BCP-47 language code for TTS, e.g. "sr"
	HasDualScript bool   // true if the language has an alternate script (e.g. Cyrillic)
	ScriptLabel   string // label for the primary script, e.g. "Latin"
	AltScriptLabel string // label for the alternate script, e.g. "Cyrillic"
}

// Lesson represents a single lesson in any language.
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
	Type        string      `json:"type"`
	Title       string      `json:"title"`
	Items       []VocabItem `json:"items,omitempty"`
	Explanation string      `json:"explanation,omitempty"`
	Examples    []Example   `json:"examples,omitempty"`
	Content     string      `json:"content,omitempty"`
}

type VocabItem struct {
	ID                string           `json:"id"`
	English           string           `json:"english"`
	TargetPrimary     string           `json:"target_primary"`
	TargetAlt         string           `json:"target_alt,omitempty"`
	PronunciationHint string           `json:"pronunciation_hint"`
	AudioOverride     *string          `json:"audio_override"`
	ExampleSentence   *ExampleSentence `json:"example_sentence"`
}

type ExampleSentence struct {
	English       string `json:"english"`
	TargetPrimary string `json:"target_primary"`
	TargetAlt     string `json:"target_alt,omitempty"`
}

type Example struct {
	English       string `json:"english"`
	TargetPrimary string `json:"target_primary"`
	TargetAlt     string `json:"target_alt,omitempty"`
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
	ShuffledTarget []string `json:"-"`
	AudioText      string   `json:"-"`
}

type Pair struct {
	English string `json:"english"`
	Target  string `json:"target"`
}
