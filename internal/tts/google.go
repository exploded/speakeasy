package tts

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Client struct {
	cacheDir   string
	audioDir   string
	mu         sync.Mutex
	httpClient *http.Client
}

func NewClient(cacheDir, audioDir string) *Client {
	os.MkdirAll(cacheDir, 0o755)
	return &Client{
		cacheDir: cacheDir,
		audioDir: audioDir,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) cacheKey(text, lang string) string {
	h := sha256.Sum256([]byte(lang + ":" + text))
	return hex.EncodeToString(h[:16])
}

// GetAudio returns audio data for the given text. It checks:
// 1. Pre-recorded audio overrides in audioDir
// 2. Cached TTS results
// 3. Google Cloud TTS API (if key set)
// 4. Returns error if nothing works
func (c *Client) GetAudio(text, lang string) ([]byte, string, error) {
	// Check for pre-recorded override
	key := c.cacheKey(text, lang)
	overridePath := filepath.Join(c.audioDir, key+".mp3")
	if data, err := os.ReadFile(overridePath); err == nil {
		return data, "audio/mpeg", nil
	}

	// Check cache
	cachePath := filepath.Join(c.cacheDir, key+".mp3")
	if data, err := os.ReadFile(cachePath); err == nil {
		return data, "audio/mpeg", nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check cache after acquiring lock
	if data, err := os.ReadFile(cachePath); err == nil {
		return data, "audio/mpeg", nil
	}

	// Try Google Cloud TTS if API key is available
	apiKey := os.Getenv("GOOGLE_TTS_API_KEY")
	if apiKey != "" {
		data, err := c.callGoogleTTS(text, lang, apiKey)
		if err != nil {
			log.Printf("TTS API error for %q: %v", text, err)
		} else {
			os.WriteFile(cachePath, data, 0o644)
			return data, "audio/mpeg", nil
		}
	}

	// No API key or API failed — don't cache failures, just return error
	return nil, "", fmt.Errorf("TTS unavailable for %q", text)
}

func (c *Client) callGoogleTTS(text, lang, apiKey string) ([]byte, error) {
	url := "https://texttospeech.googleapis.com/v1/text:synthesize?key=" + apiKey

	langCode := "sr-RS"
	if lang != "" && lang != "sr" {
		langCode = lang
	}

	reqBody := map[string]interface{}{
		"input": map[string]string{
			"text": text,
		},
		"voice": map[string]interface{}{
			"languageCode": langCode,
			"ssmlGender":   "FEMALE",
		},
		"audioConfig": map[string]string{
			"audioEncoding": "MP3",
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("TTS request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TTS API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		AudioContent string `json:"audioContent"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	audio, err := base64.StdEncoding.DecodeString(result.AudioContent)
	if err != nil {
		return nil, fmt.Errorf("decode audio: %w", err)
	}

	return audio, nil
}
