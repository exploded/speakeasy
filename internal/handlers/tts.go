package handlers

import (
	"net/http"

	"speakeasy/internal/tts"
)

type TTSHandler struct {
	client *tts.Client
}

func NewTTSHandler(c *tts.Client) *TTSHandler {
	return &TTSHandler{client: c}
}

func (h *TTSHandler) ServeAudio(w http.ResponseWriter, r *http.Request) {
	text := r.URL.Query().Get("text")
	lang := r.URL.Query().Get("lang")
	gender := r.URL.Query().Get("gender")

	// Support POST for long texts
	if r.Method == http.MethodPost {
		r.ParseForm()
		if v := r.FormValue("text"); v != "" {
			text = v
		}
		if v := r.FormValue("lang"); v != "" {
			lang = v
		}
		if v := r.FormValue("gender"); v != "" {
			gender = v
		}
	}

	if text == "" {
		http.Error(w, "text parameter required", http.StatusBadRequest)
		return
	}
	if lang == "" {
		lang = "sr"
	}

	data, contentType, err := h.client.GetAudio(text, lang, gender)
	if err != nil {
		http.Error(w, "TTS error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(data)
}
