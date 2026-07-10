package provider

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// RepoRoot returns the git top-level dir (cwd is expected to be the repo).
func RepoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --show-toplevel: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func ArtifactsDir(root string) string { return filepath.Join(root, "artifacts") }
func IndexPath(root string) string    { return filepath.Join(ArtifactsDir(root), ".index.jsonl") }
func PluginDir(root string) string    { return filepath.Join(root, ".tools") }
func ManifestPath(root string) string { return filepath.Join(PluginDir(root), "manifest.json") }

// SHA256File returns the lowercase hex sha256 of a file.
func SHA256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// AppendIndex appends one staging cell to artifacts/.index.jsonl.
func AppendIndex(root string, c StageCell) error {
	if err := os.MkdirAll(ArtifactsDir(root), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(IndexPath(root), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

// ReadIndex reads all staging cells. Missing index is an empty slice, not an error.
func ReadIndex(root string) ([]StageCell, error) {
	b, err := os.ReadFile(IndexPath(root))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cells []StageCell
	for _, line := range strings.Split(strings.TrimRight(string(b), "\n"), "\n") {
		if line == "" {
			continue
		}
		var c StageCell
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			return nil, fmt.Errorf("bad index line %q: %w", line, err)
		}
		cells = append(cells, c)
	}
	return cells, nil
}

// ResetIndex truncates the staging index (the `clean` task wipes artifacts/
// wholesale; this is for safety on a fresh mirror).
func ResetIndex(root string) error {
	if err := os.MkdirAll(ArtifactsDir(root), 0o755); err != nil {
		return err
	}
	return os.WriteFile(IndexPath(root), nil, 0o644)
}

// ReleaseTag is the publish tag scheme: sandbox-<YYYYMMDD>-<short-sha>.
func ReleaseTag(root string) (string, error) {
	date := time.Now().UTC().Format("20060102")
	sha, err := exec.Command("git", "-C", root, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --short HEAD: %w", err)
	}
	return "sandbox-" + date + "-" + strings.TrimSpace(string(sha)), nil
}

// Run executes a command, wiring stdio, and returns combined output on error.
func Run(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// RunCapture runs a command and returns trimmed stdout.
func RunCapture(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// splitLines splits trimmed output into non-empty lines.
func splitLines(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
