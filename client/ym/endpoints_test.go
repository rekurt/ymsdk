package ym_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Endpoint paths belong in endpoints.go so each has one definition. A literal
// elsewhere is invisible to behavioural tests — url.Parse strips the query, so
// "/bot/v1/messages/getFile/?file_id=" yields the same Path as the constant —
// and can drift from it silently. Literals slipped past three separate manual
// sweeps, so the rule is checked here instead.
func TestNoEndpointPathLiteralsOutsideEndpointsGo(t *testing.T) {
	root := "."

	var offenders []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		base := filepath.Base(path)
		if base == "endpoints.go" || strings.HasSuffix(base, "_test.go") {
			return nil
		}

		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(src), "\n") {
			if strings.Contains(line, `"/bot/v1/`) {
				offenders = append(offenders, filepath.ToSlash(path)+":"+itoa(i+1)+" "+strings.TrimSpace(line))
			}
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	if len(offenders) > 0 {
		t.Fatalf("endpoint paths must come from endpoints.go, found %d literal(s):\n%s",
			len(offenders), strings.Join(offenders, "\n"))
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}

	return string(digits)
}
