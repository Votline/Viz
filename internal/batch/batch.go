package batch

import (
	"errors"
)

func PackBatch(batch [][]byte) []byte {
	result := []byte{byte(len(batch))}

	for _, frame := range batch {
		result = append(result, byte(len(frame)>>8), byte(len(frame)))
		result = append(result, frame...)
	}
	
	return result
}

func UnpackBatch(data []byte) ([][]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("Empty data")
	}

	frameCount := int(data[0])
	frames := make([][]byte, 0, frameCount)

	pos := 1
	for i := 0; i < frameCount; i++ {
		if pos+2 >= len(data) {
			return nil, errors.New("Invalid packet")
		}

		frameSize := int(data[pos])<<8 | int(data[pos+1])
		pos += 2

		if pos+frameSize > len(data) {
			return nil, errors.New("Frame bigger than packet")
		}

		frame := make([]byte, frameSize)
		copy(frame, data[pos:pos+frameSize])
		frames = append(frames, frame)

		pos += frameSize
	}

	return frames, nil
}
