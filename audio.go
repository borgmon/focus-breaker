package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
)

// Global audio context singleton
var (
	globalAudioCtx     *oto.Context
	globalAudioCtxOnce sync.Once
	audioCtxReady      bool
)

// AudioPlayer manages alarm sound playback with cancellation support
type AudioPlayer struct {
	stopChan chan struct{}
	player   *oto.Player
}

// initAudioContext initializes the global audio context once
func initAudioContext(format *wavFormat) {
	globalAudioCtxOnce.Do(func() {
		op := &oto.NewContextOptions{
			SampleRate:   format.SampleRate,
			ChannelCount: format.Channels,
			Format:       oto.FormatSignedInt16LE,
		}

		ctx, readyChan, err := oto.NewContext(op)
		if err != nil {
			log.Printf("Failed to initialize audio context: %v", err)
			return
		}

		// Wait for the hardware audio devices to be ready
		<-readyChan

		globalAudioCtx = ctx
		audioCtxReady = true
		log.Println("Audio context initialized successfully")
	})
}

// playAlarmSound plays the alarm.wav file from bundled resources and returns an AudioPlayer
func playAlarmSound() *AudioPlayer {
	// Use the bundled resource instead of reading from filesystem
	wavData := resourceAlarmWav.Content()

	// Parse WAV header to get audio format
	format, audioData, err := parseWAV(wavData)
	if err != nil {
		log.Printf("Failed to parse WAV file: %v", err)
		return nil
	}

	// Initialize global audio context if not already done
	initAudioContext(format)

	if !audioCtxReady || globalAudioCtx == nil {
		log.Printf("Audio context not ready")
		return nil
	}

	ap := &AudioPlayer{
		stopChan: make(chan struct{}),
	}

	// Play the sound in a goroutine so it doesn't block
	go func() {
		// Create a new player that will handle our sound
		ap.player = globalAudioCtx.NewPlayer(bytes.NewReader(audioData))

		// Play starts playing the sound and returns without waiting
		ap.player.Play()

		// Wait for the sound to finish playing or stop signal
		for ap.player.IsPlaying() {
			select {
			case <-ap.stopChan:
				// Stop requested, cleanup and exit
				ap.player.Close()
				return
			case <-time.After(time.Millisecond):
				// Continue checking
			}
		}

		// Cleanup
		err = ap.player.Close()
		if err != nil {
			log.Printf("Failed to close audio player: %v", err)
		}
	}()

	return ap
}

// Stop stops the audio playback
func (ap *AudioPlayer) Stop() {
	if ap != nil {
		close(ap.stopChan)
	}
}

// wavFormat holds WAV file format information
type wavFormat struct {
	SampleRate int
	Channels   int
	BitDepth   int
}

// parseWAV parses a WAV file and returns the format and audio data
func parseWAV(data []byte) (*wavFormat, []byte, error) {
	reader := bytes.NewReader(data)

	// Read RIFF header
	riff := make([]byte, 4)
	if _, err := reader.Read(riff); err != nil {
		return nil, nil, err
	}

	// Skip file size
	reader.Seek(4, io.SeekCurrent)

	// Read WAVE header
	wave := make([]byte, 4)
	if _, err := reader.Read(wave); err != nil {
		return nil, nil, err
	}

	format := &wavFormat{}
	var dataStart int64
	var dataSize uint32

	// Read chunks
	for {
		chunkID := make([]byte, 4)
		if _, err := reader.Read(chunkID); err != nil {
			if err == io.EOF {
				break
			}
			return nil, nil, err
		}

		var chunkSize uint32
		if err := binary.Read(reader, binary.LittleEndian, &chunkSize); err != nil {
			return nil, nil, err
		}

		chunkIDStr := string(chunkID)

		if chunkIDStr == "fmt " {
			// Read format chunk
			var audioFormat uint16
			binary.Read(reader, binary.LittleEndian, &audioFormat)

			var numChannels uint16
			binary.Read(reader, binary.LittleEndian, &numChannels)
			format.Channels = int(numChannels)

			var sampleRate uint32
			binary.Read(reader, binary.LittleEndian, &sampleRate)
			format.SampleRate = int(sampleRate)

			// Skip byte rate and block align
			reader.Seek(6, io.SeekCurrent)

			var bitsPerSample uint16
			binary.Read(reader, binary.LittleEndian, &bitsPerSample)
			format.BitDepth = int(bitsPerSample)

			// Skip any extra format bytes
			remaining := chunkSize - 16
			if remaining > 0 {
				reader.Seek(int64(remaining), io.SeekCurrent)
			}
		} else if chunkIDStr == "data" {
			// Found data chunk
			dataStart, _ = reader.Seek(0, io.SeekCurrent)
			dataSize = chunkSize
			break
		} else {
			// Skip unknown chunk
			reader.Seek(int64(chunkSize), io.SeekCurrent)
		}
	}

	// Read audio data
	audioData := make([]byte, dataSize)
	reader.Seek(dataStart, io.SeekStart)
	reader.Read(audioData)

	return format, audioData, nil
}
