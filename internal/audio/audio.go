package audio

import (
	"time"
	"sync"
	"fmt"
	
	"go.uber.org/zap"
	"github.com/gordonklaus/portaudio"
)

type audioBuffer struct {
	vol float32
	pcm []int16
	wPos int
	rPos int
	mu sync.Mutex
}
func (b *audioBuffer) write(samples []float32) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, sample := range samples {
		if b.wPos < len(b.pcm) {
			b.pcm[b.wPos] = int16(sample*32767)
			b.wPos++
		}
	}
}
func (b *audioBuffer) data() []int16 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.pcm[:b.wPos]
}
func (b *audioBuffer) read(out []float32) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i := range out {
		if b.rPos < len(b.pcm) {
			out[i] = float32(b.pcm[b.rPos])/32767 * b.vol
			b.rPos++
		} else {
			out[i] = 0
		}
	}
}
func (b *audioBuffer) recorded() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.wPos < len(b.pcm)
}
func (b *audioBuffer) played() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.rPos >= len(b.pcm)
}

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
	bufferSize := 2048
	sampleRate := 44100.0
	totalSamples := int(dur.Seconds() * sampleRate * float64(channels))

	buffer := &audioBuffer{pcm: make([]int16, totalSamples)}
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
			buffer.write(in)
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
	for buffer.recorded() {
		time.Sleep(10*time.Millisecond)
	}
	stream.Stop()

	return buffer.data(), nil
}

func play(pcm []int16, log *zap.Logger) error {
	fmt.Scanln()
	if err := portaudio.Initialize(); err != nil {
		return err
	}
	defer portaudio.Terminate()

	dev, err := portaudio.DefaultOutputDevice()
	if err != nil {
		return err
	}

	channels := 1
	bufferSize := 2048
	sampleRate := 44100.0

	buffer := &audioBuffer{pcm: pcm, vol: 1.0}
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
			buffer.read(out)
		},
	)
	defer stream.Close()

	log.Info("Playing")
	stream.Start()
	for !buffer.played() {
		time.Sleep(10*time.Millisecond)
	}
	time.Sleep(100*time.Millisecond)
	stream.Stop()

	return nil
}
