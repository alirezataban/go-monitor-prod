package collector

import "testing"

func TestTrackerSnapshotsDoNotExposeMutableState(t *testing.T) {
	tracker := NewTracker("https://example.com")
	tracker.resources = []*ResourceEntry{{URL: "https://example.com/app.js"}}
	tracker.downloads = []map[string]interface{}{{"filename": "app.exe"}}

	resources := tracker.Resources()
	resources[0].URL = "changed"
	downloads := tracker.Downloads()
	downloads[0]["filename"] = "changed"

	if got := tracker.Resources()[0].URL; got != "https://example.com/app.js" {
		t.Errorf("resource snapshot changed tracker state: %q", got)
	}
	if got := tracker.Downloads()[0]["filename"]; got != "app.exe" {
		t.Errorf("download snapshot changed tracker state: %v", got)
	}
}
