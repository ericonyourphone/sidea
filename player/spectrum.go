package player

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/ericonyourphone/sidea/spectrum"
	"github.com/ebitengine/oto/v3"
)

type pcmPlayer interface {
	Play()
	Close() error
	SetBufferSize(int)
}

type spectrumOutput struct {
	reader *io.PipeReader
	writer *io.PipeWriter
	player pcmPlayer
}

var nativeAudio struct {
	once sync.Once
	ctx  *oto.Context
	err  error
}

// Oto supports only one context per process; reuse it if the local player
// restarts after a connection error or the user changes devices.
func newNativePlayer(reader io.Reader) (pcmPlayer, error) {
	nativeAudio.once.Do(func() {
		var ready chan struct{}
		nativeAudio.ctx, ready, nativeAudio.err = oto.NewContext(&oto.NewContextOptions{
			SampleRate: spectrum.SampleRate, ChannelCount: 2, Format: oto.FormatSignedInt16LE,
			BufferSize: 50 * time.Millisecond,
		})
		if nativeAudio.err != nil {
			return
		}
		select {
		case <-ready:
			nativeAudio.err = nativeAudio.ctx.Err()
		case <-time.After(5 * time.Second):
			nativeAudio.err = fmt.Errorf("audio device initialization timed out")
		}
	})
	if nativeAudio.err != nil {
		return nil, nativeAudio.err
	}
	if err := nativeAudio.ctx.Err(); err != nil {
		return nil, err
	}
	return nativeAudio.ctx.NewPlayer(reader), nil
}

// Start a direct native audio stream. The FFT reads exactly the PCM that
// the output player consumes, without another playback process or recording.
func (s *Sidecar) startSpectrum(logw io.Writer) error {
	return s.startSpectrumWith(logw, newNativePlayer)
}

func (s *Sidecar) startSpectrumWith(logw io.Writer, open func(io.Reader) (pcmPlayer, error)) error {
	reader, writer := io.Pipe()
	p, err := open(io.TeeReader(reader, &spectrum.Default))
	if err != nil {
		_ = reader.Close()
		_ = writer.Close()
		return fmt.Errorf("open native audio output: %w", err)
	}
	p.SetBufferSize(spectrum.SampleRate * 4 / 20) // 50ms of stereo S16LE
	s.output = &spectrumOutput{reader: reader, writer: writer, player: p}
	spectrum.Default.Reset()
	s.cmd.Stdout = writer
	s.cmd.WaitDelay = 2 * time.Second
	p.Play()
	_, _ = fmt.Fprintln(logw, "Visualizer: native audio output ready (44100 Hz, stereo S16LE)")
	return nil
}

func (s *Sidecar) stopSpectrum() {
	if s.output == nil {
		return
	}
	_ = s.output.writer.Close()
	_ = s.output.reader.Close()
	_ = s.output.player.Close()
	spectrum.Default.Reset()
}
