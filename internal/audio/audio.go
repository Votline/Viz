package audio

import (
	"sync"
	"time"
	"errors"

	"go.uber.org/zap"
	"github.com/gordonklaus/portaudio"

	"Viz/internal/compressor"
)

type AudioStream struct {
	mu sync.Mutex

	log        *zap.Logger
	dur        time.Duration
	waitTime   time.Duration

	bitrate    int
	channels   int
	bufferSize int
	sampleRate float64
	
	VoiceChan  chan []byte
	audioChan  chan []byte
	
	Queues     *allQueue
	
	playAb     *audioBuffer
	recordAb   *audioBuffer
	
	playCmpr   *compressor.Compressor
	recordCmpr *compressor.Compressor

	inputDevice *portaudio.DeviceInfo
	outputDevice *portaudio.DeviceInfo
}

func NewAS(log *zap.Logger) (*AudioStream, error) {
	playAb := newAB(1.0, log)
	recordAb := newAB(1.0, log)
	chs := 1
	dur := 40
	btr := 64000
	smpR := 48000.0
	bufS := int(smpR * float64(dur) / 1000) * chs
	playCmpr, err := compressor.NewCmpr(btr, int(smpR), chs, dur, log)
	if err != nil {
		log.Error("Create compressor error: ", zap.Error(err))
		return nil, err
	}
	recordCmpr, err := compressor.NewCmpr(btr, int(smpR), chs, dur, log)
	if err != nil {
		log.Error("Create compressor error: ", zap.Error(err))
		return nil, err
	}
	queue := newQueue()
	as := &AudioStream{
		log:        log,
		dur:        time.Duration(dur),
		waitTime:   5,
		
		channels:   chs,
		bitrate:    btr,
		bufferSize: bufS,
		sampleRate: smpR,
		
		VoiceChan:  make(chan []byte, 100),
		audioChan:  make(chan []byte, 100),
		
		Queues:     queue,
		
		playAb:     playAb,
		recordAb:   recordAb,
		
		playCmpr:   playCmpr,
		recordCmpr: recordCmpr,
	}

	maxAttempts := 0
	for (as.inputDevice == nil || as.outputDevice == nil) && maxAttempts <= 10 {
		input, err := as.initInputDevice()
		if err == nil {
			as.inputDevice = input
		}
		output, err := as.initOutputDevice()
		if err != nil {
			as.outputDevice = output
		}
		maxAttempts++
		time.Sleep(100 * time.Millisecond)
	}

	return as, nil
}

func (as *AudioStream) initInputDevice() (dev *portaudio.DeviceInfo, err error) {
	defer func(){
		if r := recover(); r != nil {
			as.log.Error("Panic in init input device", zap.Any("recover",r))
			err = errors.New("panic in get input device")
		}
	}()
	dev, err = portaudio.DefaultInputDevice()
	return
}

func (as *AudioStream) initOutputDevice() (*portaudio.DeviceInfo, error) {
	defer func(){
		if r := recover(); r != nil {
			as.log.Error("Panic in init input device", zap.Any("recover",r))
		}
	}()
	return portaudio.DefaultOutputDevice()
}

func (as *AudioStream) RecordStream() {
	dev, err := portaudio.DefaultInputDevice()
	if err != nil {
		as.log.Error("Couldn't get default output device")
		return
	}

	ringBufSize := int(as.sampleRate * 1.0)
	as.recordAb.resetPCM(ringBufSize)
	samplesPerMs := int(as.sampleRate * float64(as.dur) / 1000) * as.channels

	stream, err := portaudio.OpenStream(
		portaudio.StreamParameters{
			Input: portaudio.StreamDeviceParameters{
				Device:   dev,
				Channels: as.channels,
			},
			Output: portaudio.StreamDeviceParameters{
				Channels: 0,
			},
			SampleRate:      as.sampleRate,
			FramesPerBuffer: samplesPerMs,
		},
		func(in []float32) {
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

	ticker := time.NewTicker(as.dur * time.Millisecond)
	for range ticker.C {
		available := as.recordAb.available()
		if available < samplesPerMs {
			as.log.Warn("Not enough data in ring buffer",
				zap.Int("available", available),
				zap.Int("required", samplesPerMs))
			continue
		}

		chunk := make([]int16, samplesPerMs)
		copied := as.recordAb.copyChunk(chunk)

		if copied == samplesPerMs {
			go func(pcmData []int16){
				if len(pcmData) == 0 {
					as.log.Warn("Empty PCM data")
					return
				}

				zstdChunk, err := as.recordCmpr.CompressVoice(pcmData)
				if err != nil {
					as.log.Error("Compress voice error: ", zap.Error(err))
					return
				}

				as.log.Debug("Compressing completed",
					zap.Int("inputSamples", len(pcmData)),
					zap.Int("outputBytes", len(zstdChunk)))
				as.VoiceChan <-zstdChunk
			}(chunk)
		} else {
			as.log.Warn("Failed to copy full chunk",
				zap.Int("copied", copied),
				zap.Int("required", samplesPerMs))
			time.Sleep(as.waitTime * time.Millisecond)
		}
	}
}

func (as *AudioStream) PlayStream() {
	go func() {
		for {
			zstdChunk, ok := as.Queues.pop(as.Queues.AQ).([]byte)
			if zstdChunk != nil && ok {

				as.log.Debug("Prebuffer filled, decompressing...",
					zap.Int("samples", len(as.recordAb.data())))

				pcm, err := as.playCmpr.DecompressAudio(as.bufferSize, zstdChunk)
				if err != nil {
					as.log.Error("Decompress audio error: ", zap.Error(err))
					continue
				}

				as.log.Debug("Decompress successfully",
					zap.Int("pcm", len(pcm)))

				as.Queues.Push(pcm, as.Queues.pQ)
			} else {
				time.Sleep(as.waitTime * time.Millisecond)
			}
		}
	}()

	dev, err := portaudio.DefaultOutputDevice()
	if err != nil {
		as.log.Error("Couldn't get default output device")
		return
	}
	
	for as.Queues.length(as.Queues.pQ) < 3 {
		time.Sleep(10 * time.Millisecond)
	}

	ringBufSize := int(as.sampleRate * 1.0)
	as.playAb.resetPCM(ringBufSize)
	go func(){
		for {
			if as.Queues.length(as.Queues.pQ) > 0 {
				pcm, ok := as.Queues.pop(as.Queues.pQ).([]int16)
				if pcm != nil && ok{
					as.playAb.appendPCM(pcm)

					as.log.Debug("Chunk playback completed", 
						zap.Int("samplesPlayed", as.playAb.getReadPos()),
						zap.Int("totalSamples", len(pcm)))
				}
			} else {
				time.Sleep(as.waitTime * time.Millisecond)
			}
		}
	}()

	stream, err := portaudio.OpenStream(
		portaudio.StreamParameters{
			Input: portaudio.StreamDeviceParameters{
				Channels: 0,
			},
			Output: portaudio.StreamDeviceParameters{
				Device:   dev,
				Channels: as.channels,
			},
			SampleRate:      as.sampleRate,
			FramesPerBuffer: as.bufferSize,
		},
		func(in, out []float32) {
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

	as.log.Debug("Prebuffer filled, starting playback")

	select{}
}

