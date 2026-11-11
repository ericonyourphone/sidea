package spotify

import "testing"

func TestApiTrackToSourceUnavailable(t *testing.T) {
	got := listItemToSource(apiTrack{})
	if !got.IsSeparator || got.URI != "" || got.Name == "" {
		t.Fatalf("empty track should be a labelled non-selectable row, got %+v", got)
	}
}
