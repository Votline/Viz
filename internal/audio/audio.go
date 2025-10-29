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
	
	VoiceChan chan []byte
	audioChan chan []byte

	AudioQueue *audioQueue
	playAb *audioBuffer
	recordAb *audioBuffer
	playCmpr *compressor.Compressor
	recordCmpr *compressor.Compressor
}

func NewAS(log *zap.Logger) (*AudioStream, error) {
	playAb := newAB(1.0, log)
	recordAb := newAB(1.0, log)
	
	playCmpr, err := compressor.NewCmpr(32000, 48000, 1, log)
	if err != nil {
		log.Error("Create compressor error: ", zap.Error(err))
		return nil, err
	}

	recordCmpr, err := compressor.NewCmpr(32000, 48000, 1, log)
	if err != nil {
		log.Error("Create compressor error: ", zap.Error(err))
		return nil, err
	}
	queue := newAQ()

	return &AudioStream{
		log: log,
		dur: 300,
		
		channels: 1,
		bitrate: 32000,
		bufferSize: 2048,
		sampleRate: 48000.0,
		
		VoiceChan: make(chan []byte, 10),
		audioChan: make(chan []byte, 10),
		
		AudioQueue: queue,
		playAb: playAb,
		recordAb: recordAb,
		playCmpr: playCmpr,
		recordCmpr: recordCmpr,
	}, err
}

func (as *AudioStream) RecordStream() {
	defer func(){
		if r := recover(); r != nil {
			as.log.Error("PANIC in record audio", zap.Any("recover: ", r))
		}
	}()

	dev, err := portaudio.DefaultInputDevice()
	if err != nil {
		as.log.Error("Couldn't get default output device")
		return
	}

	samplesPerMs := int(as.sampleRate * float64(as.dur)/1000 * float64(as.channels))

	as.log.Debug("Starting recording",
		zap.Int("durationMs: ", int(as.dur)),
		zap.Int("samples: ", samplesPerMs),
		zap.Float64("sampleRate: ", as.sampleRate))

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
		startTime := time.Now()
		for !as.recordAb.recorded() {
			if time.Since(startTime) > as.dur * time.Millisecond {
				as.log.Warn("Record timeout - resetting buffer")
				as.recordAb.resetPCM(samplesPerMs)
				startTime = time.Now()
				continue
			}
			as.log.Info("Waiting for buffer")
			time.Sleep(5 * time.Millisecond)
		}

		as.log.Debug("Buffer filled, compressing...",
			zap.Int("samples: ", len(as.recordAb.data())))

		pcmData := as.recordAb.data()
		if len(pcmData) == 0 {
			as.log.Warn("Empty PCM data")
			as.recordAb.resetPCM(samplesPerMs)
			continue
		}

		zstdChunk, err := as.recordCmpr.CompressVoice(int(as.sampleRate), as.channels, pcmData)
		if err != nil {
			as.log.Error("Compress voice error: ", zap.Error(err))
			continue
		}

		as.log.Debug("Compressing completed",
			zap.Int("inputSamples: ", len(pcmData)),
			zap.Int("outputBytes: ", len(zstdChunk)))

		select {
		case as.VoiceChan <- zstdChunk:
		case <- time.After(100 * time.Millisecond):
			as.log.Warn("Channel full, dropping packet")
		}

		as.recordAb.resetPCM(samplesPerMs)
	}
}

func (as *AudioStream) PlayStream() {
	defer func(){
		if r := recover(); r != nil {
			as.log.Error("PANIC in audio playback", zap.Any("recvoer: ", r))
		}
	}()

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
		zstdChunk := as.AudioQueue.pop()
		if zstdChunk == nil {
			time.Sleep(1*time.Millisecond)
			continue
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
