package emacs_test

import (
	"os/exec"
	"testing"
)

// TestOrgBabel runs the ob-dfd ERT suite, which builds dfd and drives
// real org source blocks through it.
func TestOrgBabel(t *testing.T) {
	if _, err := exec.LookPath("emacs"); err != nil {
		t.Skip("emacs not installed; skipping the org-babel suite")
	}
	out, err := exec.Command("./run-tests.sh").CombinedOutput()
	if err != nil {
		t.Fatalf("ob-dfd tests failed: %v\n%s", err, out)
	}
	t.Logf("%s", out)
}
