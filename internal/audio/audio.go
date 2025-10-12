package audio

import (
	"time"
	
	"go.uber.org/zap"
	"github.com/gordonklaus/portaudio"
)

func Start(log *zap.Logger) {
	pcm, err := record(log)
	if err != nil {
		log.Fatal("\nRecord audio err: ", zap.Error(err))
	}
	if err := play(pcm, log); err != nil {
		log.Fatal("\nPlay audio err: ", zap.Error(err))
	}
}

func record(log *zap.Logger) ([]int16, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, err
	}
	defer portaudio.Terminate()

	dev, err := portaudio.DefaultInputDevice()
	if err != nil {
		return nil, err
	}

	var dur time.Duration
	dur = 5*time.Second

	channels := 1
	bufferSize := 8192
	sampleRate := 44100.0
	totalSamples := int(dur.Seconds() * sampleRate * float64(channels))
	pcm := make([]int16, totalSamples)

	var recIdx int
	stream, err := portaudio.OpenStream(
		portaudio.StreamParameters{
			Input: portaudio.StreamDeviceParameters{
				Device: dev,
				Channels: channels,
			},
			Output: portaudio.StreamDeviceParameters{
				Channels: 0,
			},
			SampleRate: sampleRate,
			FramesPerBuffer: bufferSize,
		},
		func (in []float32) {
			for _, sample := range in {
				if recIdx < len(pcm) {
					pcm[recIdx] = int16(sample*32767)
					recIdx++
				}
			}
		},
	)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	log.Info("Recording")
	if err := stream.Start(); err != nil {
		return nil, err
	}
	for recIdx < len(pcm) {
		time.Sleep(10*time.Millisecond)
	}
	stream.Stop()

	return pcm, nil
}

func play(pcm []int16, log *zap.Logger) error {
	if err := portaudio.Initialize(); err != nil {
		return err
	}

	dev, err := portaudio.DefaultOutputDevice()
	if err != nil {
		return err
	}

	channels := 1
	bufferSize := 8192
	sampleRate := 44100.0

	var playIdx int
	stream, err := portaudio.OpenStream(
		portaudio.StreamParameters{
			Input: portaudio.StreamDeviceParameters{
				Channels: 0,
			},
			Output: portaudio.StreamDeviceParameters{
				Device: dev,
				Channels: channels,
			},
			SampleRate: sampleRate,
			FramesPerBuffer: bufferSize,
		},
		func (in, out []float32) {
			for i := range out {
				if playIdx < len(pcm) {
					out[i] = float32(pcm[playIdx])/32767.0 * 2.0
					playIdx++
				}
			}
		},
	)
	defer stream.Close()

	log.Info("Playing")
	stream.Start()
	for playIdx < len(pcm) {
		time.Sleep(10*time.Millisecond)
	}
	stream.Stop()

	return nil
}
