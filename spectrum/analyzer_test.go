package spectrum

import (
	"encoding/binary"
	"math"
	"testing"
)

func tone(freq float64, invert bool) []byte {
	p := make([]byte, windowSize*4)
	for i := range windowSize {
		v := int16(24000 * math.Sin(2*math.Pi*freq*float64(i)/SampleRate))
		r := v
		if invert {
			r = -v
		}
		binary.LittleEndian.PutUint16(p[i*4:], uint16(v))
		binary.LittleEndian.PutUint16(p[i*4+2:], uint16(r))
	}
	return p
}

func TestFrequencyAndStereo(t *testing.T) {
	for _, freq := range []float64{100, 1000, 8000} {
		for _, inverted := range []bool{false, true} {
			var a Analyzer
			p := tone(freq, inverted)
			// Include byte splits inside samples and stereo frames.
			for len(p) > 0 {
				n := min(137, len(p))
				_, _ = a.Write(p[:n])
				p = p[n:]
			}
			f := a.Snapshot()
			best := 0
			for b, v := range f.Levels {
				if math.IsNaN(v) || v < 0 || v > 1 {
					t.Fatal("invalid level", v)
				}
				if v > f.Levels[best] {
					best = b
				}
			}
			want := int(math.Log(freq/40) / math.Log(16000.0/40) * Bands)
			if abs(best-want) > 1 || f.Levels[best] < 0.8 {
				t.Errorf("%g Hz, inverse=%v: strongest band %d, want %d", freq, inverted, best, want)
			}
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func TestSilenceResetAndPartialFrame(t *testing.T) {
	var a Analyzer
	_, _ = a.Write(make([]byte, windowSize*4-1))
	if !a.Snapshot().Updated.IsZero() {
		t.Fatal("published partial frame")
	}
	_, _ = a.Write([]byte{0})
	f := a.Snapshot()
	if f.Updated.IsZero() {
		t.Fatal("missing silence frame")
	}
	for _, v := range f.Levels {
		if v != 0 {
			t.Fatal("silence must be zero", v)
		}
	}
	a.Reset()
	if !a.Snapshot().Updated.IsZero() || len(a.pending) != 0 {
		t.Fatal("reset retained data")
	}
}
