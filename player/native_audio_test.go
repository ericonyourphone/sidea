package player

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"testing"
	"time"

	"github.com/ericonyourphone/sidea/spectrum"
	"github.com/ebitengine/oto/v3"
)

// An explicit opt-in listening check; ordinary tests never play audio.
func TestNativeAudioDevice(t *testing.T) {
	if os.Getenv("SIDEA_TEST_NATIVE_AUDIO") != "1" {
		t.Skip("requires an explicit listening check")
	}
	data := make([]byte, spectrum.SampleRate*4)
	for i := range spectrum.SampleRate {
		v := int16(1600 * math.Sin(float64(i)*2*math.Pi*440/spectrum.SampleRate))
		binary.LittleEndian.PutUint16(data[i*4:], uint16(v))
		binary.LittleEndian.PutUint16(data[i*4+2:], uint16(v))
	}
	player, err := newNativePlayer(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer player.Close()
	player.Play()
	p := player.(*oto.Player)
	deadline := time.Now().Add(5 * time.Second)
	for p.IsPlaying() && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if err = p.Err(); err != nil {
		t.Fatal(err)
	}
	if p.IsPlaying() {
		t.Fatal("audio did not drain")
	}
	if err = nativeAudio.ctx.Err(); err != nil {
		t.Fatal(err)
	}
	t.Log("Native audio accepted and played one second of PCM without device errors")
}
