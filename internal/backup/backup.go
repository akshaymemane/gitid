package backup

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gitid/gitid/internal/config"
)

func Create(path, label string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	expanded, err := config.ExpandPath(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(expanded)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("cannot back up directory %s", expanded)
	}
	backupDir, err := config.BackupDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s-%s", sanitize(label), time.Now().Format("20060102-150405.000000000"))
	dest := filepath.Join(backupDir, name)
	if err := copyFile(expanded, dest, info.Mode().Perm()); err != nil {
		return "", err
	}
	return dest, nil
}

func RestoreLatest(label, dest string) (string, error) {
	expandedDest, err := config.ExpandPath(dest)
	if err != nil {
		return "", err
	}
	backupDir, err := config.BackupDir()
	if err != nil {
		return "", err
	}
	matches, err := filepath.Glob(filepath.Join(backupDir, sanitize(label)+"-*"))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no backup found for %s", label)
	}
	sort.Strings(matches)
	src := matches[len(matches)-1]
	info, err := os.Stat(src)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(expandedDest), 0o700); err != nil {
		return "", err
	}
	if err := copyFile(src, expandedDest, info.Mode().Perm()); err != nil {
		return "", err
	}
	return src, nil
}

func copyFile(src, dest string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func sanitize(label string) string {
	label = strings.ToLower(strings.TrimSpace(label))
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", ".", "_")
	label = replacer.Replace(label)
	if label == "" {
		return "file"
	}
	return label
}
