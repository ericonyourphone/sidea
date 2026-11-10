// Package spectrum analyzes the same raw PCM sent to the audio output.
package spectrum

import (
	"encoding/binary"
	"math"
	"math/cmplx"
	"sync"
	"time"
)

const (
	Bands      = 32
	SampleRate = 44100
	windowSize = 4096
)

// Default is shared by the local player and terminal renderer. No audio is saved.
var Default Analyzer

type Frame struct {
	Levels  [Bands]float64
	Updated time.Time
}

type Analyzer struct {
	mu      sync.Mutex
	frame   Frame
	pending []byte
}

// Write accepts arbitrarily split, interleaved stereo S16LE PCM.
func (a *Analyzer) Write(p []byte) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	n := len(p)
	for len(p) > 0 {
		take := min(len(p), windowSize*4-len(a.pending))
		a.pending = append(a.pending, p[:take]...)
		p = p[take:]
		if len(a.pending) == windowSize*4 {
			a.frame = analyze(a.pending)
			a.pending = a.pending[:0]
		}
	}
	return n, nil
}

func (a *Analyzer) Snapshot() Frame {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.frame
}

func (a *Analyzer) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.frame = Frame{}
	a.pending = nil
}

func analyze(p []byte) Frame {
	var channels [2][windowSize]complex128
	for i := range windowSize {
		hann := 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(windowSize-1))
		for c := range 2 {
			sample := int16(binary.LittleEndian.Uint16(p[i*4+c*2:]))
			channels[c][i] = complex(float64(sample)/32768*hann, 0)
		}
	}
	for c := range 2 {
		fft(channels[c][:])
	}
	var out Frame
	out.Updated = time.Now()
	for b := range Bands {
		lo := 40 * math.Pow(16000.0/40, float64(b)/Bands)
		hi := 40 * math.Pow(16000.0/40, float64(b+1)/Bands)
		first := max(1, int(math.Round(lo*windowSize/SampleRate)))
		last := min(windowSize/2, max(first+1, int(math.Round(hi*windowSize/SampleRate))))
		power := 0.0
		for k := first; k < last; k++ {
			// Combine channel power, preserving out-of-phase stereo content.
			l, r := cmplx.Abs(channels[0][k]), cmplx.Abs(channels[1][k])
			power = max(power, (l*l+r*r)/2)
		}
		amplitude := math.Sqrt(power) * 4 / windowSize
		db := 20 * math.Log10(max(amplitude, 1e-10))
		out.Levels[b] = max(0, min(1, (db+65)/65))
	}
	return out
}

func fft(x []complex128) {
	n := len(x)
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for j&bit != 0 {
			j ^= bit
			bit >>= 1
		}
		j ^= bit
		if i < j {
			x[i], x[j] = x[j], x[i]
		}
	}
	for size := 2; size <= n; size <<= 1 {
		step := cmplx.Exp(complex(0, -2*math.Pi/float64(size)))
		for start := 0; start < n; start += size {
			w := complex(1, 0)
			for j := 0; j < size/2; j++ {
				u, v := x[start+j], w*x[start+j+size/2]
				x[start+j], x[start+j+size/2] = u+v, u-v
				w *= step
			}
		}
	}
}
