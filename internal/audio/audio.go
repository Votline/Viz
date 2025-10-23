package audio

import (
	"time"

	"go.uber.org/zap"
	"github.com/gordonklaus/portaudio"

	"Viz/internal/compressor"
)

type AudioStream struct {
	log *zap.Logger
	dur time.Duration
	bitrate int
	channels int
	bufferSize int
	sampleRate float64
	stopChan chan bool
	audioChan chan []byte
	playAb *audioBuffer
	recordAb *audioBuffer
	playCmpr *compressor.Compressor
	recordCmpr *compressor.Compressor
}

func NewAS(log *zap.Logger) *AudioStream {
	playAb := newAB(1.0, log)
	recordAb := newAB(1.0, log)
	
	playCmpr, err := compressor.NewCmpr(32000, 48000, 1, log)
	if err != nil {
		log.Error("Create compressor error: ", zap.Error(err))
		return nil
	}

	recordCmpr, err := compressor.NewCmpr(32000, 48000, 1, log)
	if err != nil {
		log.Error("Create compressor error: ", zap.Error(err))
		return nil
	}

	return &AudioStream{
		log: log,
		dur: 300,
		channels: 1,
		bitrate: 32000,
		bufferSize: 2048,
		sampleRate: 48000.0,
		stopChan: make(chan bool),
		audioChan: make(chan []byte, 10),
		playAb: playAb,
		recordAb: recordAb,
		playCmpr: playCmpr,
		recordCmpr: recordCmpr,
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

	samplesPerMs := int(as.sampleRate * float64(as.dur)/1000 * float64(as.channels))

	as.recordAb.resetPCM(samplesPerMs)
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
			as.recordAb.write(in)
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
			startTime := time.Now()
			for !as.recordAb.recorded() {
				if time.Since(startTime) > as.dur * time.Millisecond {
					as.log.Warn("Record timeout")
					break
				}
				as.log.Info("Waiting for buffer")
				time.Sleep(1 * time.Millisecond)
			}

			as.log.Info("Going to compress")

			zstdChunk, err := as.recordCmpr.CompressVoice(int(as.sampleRate), as.channels, as.recordAb.data())
			if err != nil {
				as.log.Error("Compress voice error: ", zap.Error(err))
				continue
			}

			as.log.Info("Before compress", zap.Int("chunk: ", len(zstdChunk)))

			select {
			case as.audioChan <- zstdChunk:
			case <- time.After(100 * time.Millisecond):
				as.log.Warn("Channel full, dropping packet")
			}

			as.recordAb.resetPCM(samplesPerMs)
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
			as.playAb.read(out)
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

			as.log.Info("Going to decompress")

			pcm, err := as.playCmpr.DecompressAudio(as.bufferSize, int(as.sampleRate), as.channels, zstdChunk)
			if err != nil {
				as.log.Error("Decompress audio error: ", zap.Error(err))
				continue
			}

			as.log.Info("Before decompress", zap.Int("pcm: ", len(pcm)))

			as.playAb.setPCM(pcm)
			as.playAb.resetPlay()

			startTime := time.Now()
			for !as.playAb.played() {
				if time.Since(startTime) > as.dur*time.Millisecond {
					as.log.Warn("Play timeout")
					break
				}
				as.log.Info("Playback")
				time.Sleep(1 * time.Millisecond)
			}
		}
	}
}
