package indonesian

import (
	"embed"

	"speakeasy/internal/lessons"
)

//go:embed data/*.json
var lessonData embed.FS

func init() {
	lessons.RegisterFromFS(lessonData, lessons.Language{
		Slug:          "indonesian",
		DisplayName:   "Indonesian",
		TTSCode:       "id",
		HasDualScript: false,
		ScriptLabel:   "Latin",
	})
}
