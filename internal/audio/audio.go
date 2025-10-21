package audio

import (
	"time"
	
	"go.uber.org/zap"
	"github.com/gordonklaus/portaudio"
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
}

func NewAS(log *zap.Logger) *AudioStream {
	ab, err := newAB(1.0, 32000, 48000, 1, log)
	if err != nil {
		log.Error("Create audio buffer error: ", zap.Error(err))
		return nil
	}

	return &AudioStream{
		ab: ab,
		log: log,
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
		return
	}

	samplesPer500ms := int(as.sampleRate * 0.5 * float64(as.channels))

	as.ab.resetPCM(samplesPer500ms)
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
				as.log.Info("DATA: \n", zap.Int("pcm", len(as.ab.pcm)), zap.Int("wPos", as.ab.wPos))
			}

			as.log.Info("I can record")

			opusChunk, err := as.ab.encodeOPUS(int(as.sampleRate), as.channels, as.ab.data())
			if err != nil {
				as.log.Error("Encode PCM to OPUS failedL ", zap.Error(err))
				continue
			}
			
			as.log.Info("I going to full channel ", zap.Int("opusChunk: ", len(opusChunk)))

			select {
			case as.audioChan <- opusChunk:
			case <- time.After(100 * time.Millisecond):
				as.log.Warn("Channel full, dropping packet")
			}

			as.ab.resetPCM(samplesPer500ms)
		}
	}
}

func (as *AudioStream) playStream() {
	dev, err := portaudio.DefaultOutputDevice()
	if err != nil {
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
		case opusChunk, ok := <- as.audioChan:
			if !ok {
				return
			}

			as.log.Info("Decoding new chunk: ", zap.Int("size", len(opusChunk)))

			pcm, err := as.ab.decodeOPUS(as.bufferSize, int(as.sampleRate), as.channels, opusChunk)
			if err != nil {
				as.log.Error("Decode OPUS chunk failed: ", zap.Error(err))
				continue
			}
			as.ab.setPCM(pcm)
			as.ab.resetPlay()

			as.log.Info("I going playback with ", zap.Int("pcm len:", len(as.ab.pcm)))
			
			for !as.ab.played() {
				time.Sleep(10 * time.Millisecond)
			}

			time.Sleep(50 * time.Millisecond)
		}
	}
}
