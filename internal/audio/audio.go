package audio

import (
	"time"

	"go.uber.org/zap"
	"github.com/gordonklaus/portaudio"

	"Viz/internal/compressor"
)

type AudioStream struct {
	log *zap.Logger
	bitrate int
	channels int
	bufferSize int
	sampleRate float64
	stopChan chan bool
	audioChan chan []byte
	ab *audioBuffer
	cmpr *compressor.Compressor
}

func NewAS(log *zap.Logger) *AudioStream {
	ab := newAB(1.0, log)
	cmpr, err := compressor.NewCmpr(16000, 48000, 1, log)
	if err != nil {
		log.Error("Create compressor error: ", zap.Error(err))
		return nil
	}

	return &AudioStream{
		ab: ab,
		log: log,
		cmpr: cmpr,
		channels: 1,
		bitrate: 32000,
		bufferSize: 2048,
		sampleRate: 48000.0,
		stopChan: make(chan bool),
		audioChan: make(chan []byte, 10),
	}
}

func (as *AudioStream) Start() {
	go as.recordStream()
	go as.playStream()
}

func (as *AudioStream) recordStream() {
	dev, err := portaudio.DefaultInputDevice()
	if err != nil {
		as.log.Error("Couldn't get default output device")
		return
	}

	samplesPerMs := int(as.sampleRate * 0.3 * float64(as.channels))

	as.ab.resetPCM(samplesPerMs)
	stream, err := portaudio.OpenStream(
		portaudio.StreamParameters{
			Input: portaudio.StreamDeviceParameters{
				Device: dev,
				Channels: as.channels,
			},
			Output: portaudio.StreamDeviceParameters{
				Channels: 0,
			},
			SampleRate: as.sampleRate,
			FramesPerBuffer: as.bufferSize,
		},
		func (in []float32) {
			as.ab.write(in)
		},
	)
	if err != nil {
		return
	}
	defer stream.Close()

	if err := stream.Start(); err != nil {
		as.log.Error("Start recording stream error: ", zap.Error(err))
		return
	}

	for {
		select {
		case <- as.stopChan:
			as.log.Info("Stopping recording")
			return
		default:
			time.Sleep(450 * time.Millisecond)
			for !as.ab.recorded() {
				time.Sleep(10 * time.Millisecond)
			}

			zstdChunk, err := as.cmpr.CompressVoice(int(as.sampleRate), as.channels, as.ab.data())
			if err != nil {
				as.log.Error("Compress voice error: ", zap.Error(err))
				continue
			}

			select {
			case as.audioChan <- zstdChunk:
			case <- time.After(100 * time.Millisecond):
				as.log.Warn("Channel full, dropping packet")
			}

			as.ab.resetPCM(samplesPerMs)
		}
	}
}

func (as *AudioStream) playStream() {
	dev, err := portaudio.DefaultOutputDevice()
	if err != nil {
		as.log.Error("Couldn't get default output device")
		return
	}

	stream, err := portaudio.OpenStream(
		portaudio.StreamParameters{
			Input: portaudio.StreamDeviceParameters{
				Channels: 0,
			},
			Output: portaudio.StreamDeviceParameters{
				Device: dev,
				Channels: as.channels,
			},
			SampleRate: as.sampleRate,
			FramesPerBuffer: as.bufferSize,
		},
		func (in, out []float32) {
			as.ab.read(out)
		},
	)
	if err != nil {
		as.log.Error("Open output stream err: ", zap.Error(err))
		return
	}
	defer stream.Close()

	if err := stream.Start(); err != nil {
		as.log.Error("Open output stream error: ", zap.Error(err))
	}

	for {
		select {
		case <- as.stopChan:
			as.log.Info("Stopping playback stream")
			return
		case zstdChunk, ok := <- as.audioChan:
			if !ok {
				as.log.Info("Channel is closed")
				return
			}

			pcm, err := as.cmpr.DecompressAudio(as.bufferSize, int(as.sampleRate), as.channels, zstdChunk)
			if err != nil {
				as.log.Error("Decompress audio error: ", zap.Error(err))
				continue
			}

			as.ab.setPCM(pcm)
			as.ab.resetPlay()

			for !as.ab.played() {
				time.Sleep(10 * time.Millisecond)
			}

			time.Sleep(50 * time.Millisecond)
		}
	}
}
