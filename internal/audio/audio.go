// Package audio uses the Go-audio library to record and play audio
package audio

import (
	"fmt"

	gAu "github.com/Votline/Go-audio"
	gAcl "github.com/Votline/Go-audio/pkg/audio"
	"go.uber.org/zap"
)

type AudioStream struct {
	*gAcl.AudioClient
}

func NewAudioStream(log *zap.Logger) (*AudioStream, error) {
	const op = "audio.NewAudioStream"

	acl, err := gAu.InitAudioClient(0, 0, 0, 0, 1, 64000, 48000, 40, true, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: Failed to init audio client: %w", op, err)
	}

	return &AudioStream{acl}, nil
}
