package audio

import (
	"sync"
	"time"

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
	
	VoiceChan  chan []int16
	audioChan  chan []int16
	
	Queues     *allQueue
	
	playAb     *audioBuffer
	recordAb   *audioBuffer
	
	playCmpr   *compressor.Compressor
	recordCmpr *compressor.Compressor
}

func NewAS(log *zap.Logger) (*AudioStream, error) {
	playAb := newAB(1.0, log)
	recordAb := newAB(1.0, log)
	chs := 1
	bufS := 1024
	btr := 32000
	smpR := 48000.0
	playCmpr, err := compressor.NewCmpr(btr, int(smpR), chs, log)
	if err != nil {
		log.Error("Create compressor error: ", zap.Error(err))
		return nil, err
	}
	recordCmpr, err := compressor.NewCmpr(btr, int(smpR), chs, log)
	if err != nil {
		log.Error("Create compressor error: ", zap.Error(err))
		return nil, err
	}
	queue := newQueue()
	return &AudioStream{
		log:        log,
		dur:        300,
		waitTime:   1,
		
		channels:   chs,
		bitrate:    btr,
		bufferSize: bufS,
		sampleRate: smpR,
		
		VoiceChan:  make(chan []int16, 100),
		audioChan:  make(chan []int16, 100),
		
		Queues:     queue,
		
		playAb:     playAb,
		recordAb:   recordAb,
		
		playCmpr:   playCmpr,
		recordCmpr: recordCmpr,
	}, err
}

func (as *AudioStream) RecordStream() {
	dev, err := portaudio.DefaultInputDevice()
	if err != nil {
		as.log.Error("Couldn't get default output device")
		return
	}

	samplesPerMs := int(as.sampleRate * float64(as.dur) / 1000 * float64(as.channels))

	as.log.Debug("Starting recording",
		zap.Int("durationMs", int(as.dur)),
		zap.Int("samples", samplesPerMs),
		zap.Float64("sampleRate", as.sampleRate))

	as.recordAb.resetPCM(samplesPerMs)

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
			FramesPerBuffer: as.bufferSize,
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

	cleanupTicker := time.NewTicker(30 * time.Second)
	for {
		as.recordAb.resetPCM(samplesPerMs)

		startTime := time.Now()
		for !as.recordAb.recorded() {
			if time.Since(startTime) > as.dur*time.Millisecond*2 {
				as.log.Warn("Record timeout")
				as.recordAb.stopRecording()
				break
			}
			as.log.Debug("Waiting for buffer")
			time.Sleep(as.waitTime*time.Millisecond)
		}
		as.recordAb.stopRecording()

		as.log.Debug("Buffer filled, compressing...",
			zap.Int("samples", len(as.playAb.data())))

		pcmData := as.recordAb.data()
		if len(pcmData) == 0 {
			as.log.Warn("Empty PCM data")
			continue
		}

		zstdChunk, err := as.recordCmpr.CompressVoice(pcmData)
		if err != nil {
			as.log.Error("Compress voice error: ", zap.Error(err))
			continue
		}

		as.log.Debug("Compressing completed",
			zap.Int("inputSamples", len(pcmData)),
			zap.Int("outputBytes", len(zstdChunk)))

		select{
		case as.VoiceChan <-zstdChunk:
		case <-time.After(50 * time.Millisecond):
			as.log.Warn("Channel full, dropping packet")
		}
		select{
		case <-cleanupTicker.C:
			as.recordAb.cleanup()
		default:
		}
	}
}

func (as *AudioStream) PlayStream() {
	go func() {
		for {
			zstdChunk, ok := as.Queues.pop(as.Queues.AQ).([]int16)
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

				as.log.Debug("Decoded and queued PCM",
					zap.Int("pcmSamples", len(pcm)),
					zap.Int("queueSize", as.Queues.length(as.Queues.pQ)))
			}
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

	for {
		if as.Queues.length(as.Queues.pQ) > 2 {
			pcm, ok := as.Queues.pop(as.Queues.pQ).([]int16)
			if pcm != nil && ok{
				as.playAb.setPCM(pcm)
				as.playAb.resetPlay()

				startTime := time.Now()
				samplesToPlay := len(pcm)

				for as.playAb.getReadPos() < samplesToPlay-(samplesToPlay/10) {
					if time.Since(startTime) > as.dur*time.Millisecond*2 {
						as.log.Warn("Play timeout")
						break
					}
					time.Sleep(as.waitTime * time.Millisecond)

					if as.playAb.getReadPos() >= len(pcm) {
						break
					}
				}
				as.log.Debug("Chunk playback completed", 
					zap.Int("samplesPlayed", as.playAb.getReadPos()),
					zap.Int("totalSamples", samplesToPlay))
			}
		} else {
			as.log.Debug("Waiting for pcm queue...")
			time.Sleep(as.waitTime * time.Millisecond)
		}
	}
}
