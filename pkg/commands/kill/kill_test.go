package kill

import "testing"

func TestIsPID(t *testing.T) {
	valid := []string{"1", "42", "999999", "0"}
	for _, s := range valid {
		if !isPID(s) {
			t.Errorf("isPID(%q) = false, want true", s)
		}
	}

	invalid := []string{"", " ", "12a", "a12", "-1", "1.0", "1 2", "PID", "١٢٣"}
	for _, s := range invalid {
		if isPID(s) {
			t.Errorf("isPID(%q) = true, want false", s)
		}
	}
}

func TestShouldExclude(t *testing.T) {
	excluded := []string{"systemd", "init", "kthreadd"}

	for _, name := range excluded {
		if !shouldExclude(name, excluded) {
			t.Errorf("shouldExclude(%q) = false, want true", name)
		}
	}

	for _, name := range []string{"firefox", "go", ""} {
		if shouldExclude(name, excluded) {
			t.Errorf("shouldExclude(%q) = true, want false", name)
		}
	}

	if shouldExclude("firefox", nil) {
		t.Error("nothing is excluded when the list is empty")
	}
}
