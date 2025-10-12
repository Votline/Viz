package audio

import (
	"log"
	"time"
	
	"github.com/gordonklaus/portaudio"
)

func Start() {
	_, err := record()
	if err != nil {
		log.Fatalf("\nRecord audio err: %s\n", err.Error())
	}
}

func record() ([]int16, error) {
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

	log.Printf("Recording")
	if err := stream.Start(); err != nil {
		return nil, err
	}
	for recIdx < len(pcm) {
		time.Sleep(10*time.Millisecond)
	}
	stream.Stop()

	return pcm, nil
}
