package launch

import "testing"

func TestQuitNotRequestedByDefault(t *testing.T) {
	if QuitRequested() {
		t.Fatal("expected QuitRequested to be false by default")
	}
	if UserHidLauncher() {
		t.Fatal("expected UserHidLauncher to be false by default")
	}
}
