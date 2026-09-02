package ym_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Sentinels belong in ymerrors so a caller matching with errors.Is needs one
// import, not one per service package. The rule is stated in AGENTS.md, and
// this PR broke it sixteen times while writing it down — a released sentinel
// cannot be moved without breaking imports or keeping an alias forever, so the
// placement is checked here rather than remembered.
func TestSentinelsAreDeclaredOnlyInYmerrors(t *testing.T) {
	var offenders []string
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if strings.HasPrefix(filepath.ToSlash(path), "ymerrors/") {
			return nil
		}

		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(src), "\n") {
			if strings.HasPrefix(line, "var Err") {
				offenders = append(offenders, filepath.ToSlash(path)+":"+itoa(i+1)+" "+strings.TrimSpace(line))
			}
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	if len(offenders) > 0 {
		t.Fatalf("sentinels must be declared in ymerrors, found %d elsewhere:\n%s",
			len(offenders), strings.Join(offenders, "\n"))
	}
}
