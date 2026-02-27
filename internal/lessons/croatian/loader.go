package croatian

import (
	"embed"

	"speakeasy/internal/lessons"
)

//go:embed data/*.json
var lessonData embed.FS

func init() {
	lessons.RegisterFromFS(lessonData, lessons.Language{
		Slug:          "croatian",
		DisplayName:   "Croatian",
		TTSCode:       "hr",
		HasDualScript: false,
		ScriptLabel:   "Latin",
	})
}
