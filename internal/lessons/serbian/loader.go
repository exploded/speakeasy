package serbian

import (
	"embed"

	"speakeasy/internal/lessons"
)

//go:embed data/*.json
var lessonData embed.FS

func init() {
	lessons.RegisterFromFS(lessonData, lessons.Language{
		Slug:           "serbian",
		DisplayName:    "Serbian",
		TTSCode:        "sr",
		HasDualScript:  true,
		ScriptLabel:    "Latin",
		AltScriptLabel: "Cyrillic",
	})
}
