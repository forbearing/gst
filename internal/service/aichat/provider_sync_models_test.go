package serviceaichat

import (
	"testing"

	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	"github.com/stretchr/testify/assert"
)

func TestDetermineOpenAIModelType(t *testing.T) {
	tests := []struct {
		modelID  string
		expected modelaichat.ModelType
	}{
		{"gpt-4", modelaichat.ModelTypeChat},
		{"gpt-3.5-turbo", modelaichat.ModelTypeChat},
		{"dall-e-3", modelaichat.ModelTypeImage},
		{"dall-e-2", modelaichat.ModelTypeImage},
		{"tts-1", modelaichat.ModelTypeAudio},
		{"tts-1-hd", modelaichat.ModelTypeAudio},
		{"whisper-1", modelaichat.ModelTypeAudio},
		{"text-embedding-3-small", modelaichat.ModelTypeEmbedding},
		{"text-embedding-ada-002", modelaichat.ModelTypeEmbedding},
		{"gpt-4-vision-preview", modelaichat.ModelTypeChat},
		{"davinci-002", modelaichat.ModelTypeChat},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			got := determineOpenAIModelType(tt.modelID)
			assert.Equal(t, tt.expected, got)
		})
	}
}
