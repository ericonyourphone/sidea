package app

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/ericonyourphone/sidea/source"
	"github.com/ericonyourphone/sidea/spectrum"
)

func TestSpectrumLayout(t *testing.T) {
	var v spectrumView
	v.status = "40 Hz                       16 kHz"
	for b := range spectrum.Bands {
		v.levels[b] = float64(b+1) / spectrum.Bands
		v.peaks[b] = v.levels[b]
	}
	for _, size := range [][2]int{{70, 35}, {36, 10}, {8, 3}, {2, 2}} {
		out := renderSpectrum(v, size[0], size[1])
		rows := strings.Split(out, "\n")
		if len(rows) != size[1] {
			t.Fatalf("size %v: %d rows", size, len(rows))
		}
		for _, line := range rows {
			if lipgloss.Width(line) > size[0] {
				t.Fatal("row overflow")
			}
		}
		if size[1] > 3 && !strings.Contains(ansi.Strip(out), "▪") {
			t.Fatal("missing dots")
		}
	}
}

func TestSpectrumToggleAndTick(t *testing.T) {
	m := newTestModel(&StubSource{})
	m.mode = ModeNowPlaying
	m.vinylMode = true
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = result.(Model)
	if !m.spectrumMode || m.vinylMode || m.lyricsView || !m.spectrumTicking || cmd == nil {
		t.Fatal("spectrum did not replace vinyl and start tick")
	}
	m.blurred = true
	result, _ = m.Update(spectrumTickMsg(time.Now()))
	m = result.(Model)
	if !m.spectrumTicking {
		t.Fatal("spectrum stopped when focus moved away")
	}
	m.mode = ModeNormal
	result, _ = m.Update(spectrumTickMsg(time.Now()))
	m = result.(Model)
	if m.spectrumTicking {
		t.Fatal("hidden visualizer kept running")
	}
	m.blurred = false
	m.mode = ModeNowPlaying
	result, _ = m.Update(noopMsg{})
	m = result.(Model)
	if !m.spectrumTicking {
		t.Fatal("focused spectrum did not restart")
	}
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = result.(Model)
	if m.spectrumMode || !m.vinylMode {
		t.Fatal("vinyl did not replace spectrum")
	}
}

func TestSpectrumSilencePauseAndRemote(t *testing.T) {
	spectrum.Default.Reset()
	m := newTestModel(&StubSource{})
	m.opts.PlayerName = "local"
	m.track = &source.Track{Playing: true, DeviceName: "phone"}
	for b := range spectrum.Bands {
		m.spectrum.levels[b] = 1
		m.spectrum.peaks[b] = 1
	}
	now := time.Now()
	for i := range 60 {
		m.advanceSpectrum(now.Add(time.Duration(i) * 40 * time.Millisecond))
	}
	for _, v := range m.spectrum.levels {
		if v != 0 {
			t.Fatal("remote device retained bars")
		}
	}
	if !strings.Contains(m.spectrum.status, "Press M") {
		t.Fatal("missing remote guidance")
	}
	m.track.Playing = false
	m.advanceSpectrum(now.Add(3 * time.Second))
	if m.spectrum.status != "Paused" {
		t.Fatal("missing paused state")
	}
}
