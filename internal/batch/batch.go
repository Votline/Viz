package batch

import (
	"errors"
	"encoding/binary"
)

func PackBatch(batch [][]byte) []byte {
	totalSize := 4
	for _, chunk := range batch {
		totalSize += 4 + len(chunk)
	}

	buffer := make([]byte, totalSize)
	binary.BigEndian.PutUint32(buffer[0:4], uint32(len(batch)))

	offset := 4
	for _, chunk := range batch {
		binary.BigEndian.PutUint32(buffer[offset:offset+4], uint32(len(chunk)))
		offset += 4

		copy(buffer[offset:offset+len(chunk)], chunk)
		offset += len(chunk)

	}

	return buffer
}

func UnpackBatch(data []byte) ([][]byte, error) {
	if len(data) < 4 {
		return nil, errors.New("Invalid batch data")
	}
	
	frameCount := int(binary.BigEndian.Uint32(data[0:4]))
	frames := make([][]byte, 0, frameCount)

	offset := 4
	for i := 0; i < frameCount; i++ {
		if offset+4 > len(data) {
			return nil, errors.New("Invalid batch format")
		}

		frameSize := int(binary.BigEndian.Uint32(data[offset:offset+4]))
		offset += 4

		if offset+frameSize > len(data) {
			return nil, errors.New("Frame size exceeds data")
		}

		frame := make([]byte, frameSize)
		copy(frame, data[offset:offset+frameSize])
		frames = append(frames, frame)

		offset += frameSize
	}

	return frames, nil
}
