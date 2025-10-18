package audio

import (
	"fmt"
	"time"
	
	"go.uber.org/zap"
	"github.com/gordonklaus/portaudio"
)

func Start(log *zap.Logger) {
	opusBuffer, err := record(log)
	if err != nil {
		log.Fatal("\nRecord audio err: ", zap.Error(err))
	}
	if err := play(opusBuffer, log); err != nil {
		log.Fatal("\nPlay audio err: ", zap.Error(err))
	}
}

func record(log *zap.Logger) ([]byte, error) {
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
	bitrate := 32000
	bufferSize := 2048
	sampleRate := 48000.0
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

	opusBuffer, err := buffer.encodeOPUS(int(sampleRate), channels, bitrate, buffer.data(), log)
	if err != nil {
		log.Error("Encode PCM to OPUS failed: ", zap.Error(err))
		return nil, err
	}
	return opusBuffer, nil
}

func play(opusBuffer []byte, log *zap.Logger) error {
	if len(opusBuffer) == 0 {
		log.Error("Empty OPUS buffer")
		return nil
	}
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
	sampleRate := 48000.0

	buffer := &audioBuffer{vol: 1.0}
	pcm, err := buffer.decodeOPUS(bufferSize, int(sampleRate), channels, opusBuffer, log)
	if err != nil {
		log.Error("Decode OPUS to PCM failed: ", zap.Error(err))
		return err
	}
	buffer.setPCM(pcm)

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
