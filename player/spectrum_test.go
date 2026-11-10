package player

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"testing"

	"github.com/ericonyourphone/sidea/spectrum"
)

type fakePCMPlayer struct {
	reader     io.Reader
	audio      bytes.Buffer
	done       chan struct{}
	bufferSize int
}

func (p *fakePCMPlayer) Play()               { go func() { _, _ = io.Copy(&p.audio, p.reader); close(p.done) }() }
func (p *fakePCMPlayer) Close() error        { <-p.done; return nil }
func (p *fakePCMPlayer) SetBufferSize(n int) { p.bufferSize = n }

func TestSpectrumForwardsPCMAndCleansUp(t *testing.T) {
	s := &Sidecar{cmd: exec.Command("unused")}
	var sink *fakePCMPlayer
	err := s.startSpectrumWith(io.Discard, func(r io.Reader) (pcmPlayer, error) {
		sink = &fakePCMPlayer{reader: r, done: make(chan struct{})}
		return sink, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	data := make([]byte, 4096*4)
	for i := range data {
		data[i] = byte(i)
	}
	if n, err := s.cmd.Stdout.Write(data); err != nil || n != len(data) {
		t.Fatal(n, err)
	}
	// Closing both pipe ends releases any blocked read before closing the player.
	s.stopSpectrum()
	if !bytes.Equal(data, sink.audio.Bytes()) {
		t.Fatal("audio bytes changed")
	}
	if sink.bufferSize != spectrum.SampleRate*4/20 {
		t.Fatal("unexpected buffer size")
	}
	if !spectrum.Default.Snapshot().Updated.IsZero() {
		t.Fatal("audio state survived shutdown")
	}
}

func TestSpectrumAudioInitializationError(t *testing.T) {
	s := &Sidecar{cmd: exec.Command("unused")}
	want := errors.New("no audio device")
	err := s.startSpectrumWith(io.Discard, func(io.Reader) (pcmPlayer, error) { return nil, want })
	if !errors.Is(err, want) || s.output != nil {
		t.Fatal("audio initialization error was not returned cleanly", err)
	}
}
