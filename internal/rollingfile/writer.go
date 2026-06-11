package rollingfile

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	DefaultMaxSizeBytes = 100 * 1024 * 1024
	DefaultMaxBackups   = 5
)

type Writer struct {
	mu           sync.Mutex
	path         string
	maxSizeBytes int64
	maxBackups   int
	file         *os.File
	size         int64
}

func New(dir string, fileName string, maxSizeBytes int64, maxBackups int) (*Writer, error) {
	if dir == "" {
		dir = "logs"
	}
	if fileName == "" {
		fileName = "stellar.log"
	}
	if maxSizeBytes <= 0 {
		maxSizeBytes = DefaultMaxSizeBytes
	}
	if maxBackups <= 0 {
		maxBackups = DefaultMaxBackups
	}

	writer := &Writer{
		path:         filepath.Join(dir, fileName),
		maxSizeBytes: maxSizeBytes,
		maxBackups:   maxBackups,
	}
	if err := writer.open(); err != nil {
		return nil, err
	}
	return writer, nil
}

func (w *Writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		if err := w.open(); err != nil {
			return 0, err
		}
	}
	if w.size+int64(len(p)) > w.maxSizeBytes {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func (w *Writer) open() error {
	if err := os.MkdirAll(filepath.Dir(w.path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return err
	}
	w.file = file
	w.size = info.Size()
	return nil
}

func (w *Writer) rotate() error {
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return err
		}
		w.file = nil
	}

	oldest := backupPath(w.path, w.maxBackups)
	if err := os.Remove(oldest); err != nil && !os.IsNotExist(err) {
		return err
	}
	for i := w.maxBackups - 1; i >= 1; i-- {
		src := backupPath(w.path, i)
		dst := backupPath(w.path, i+1)
		if err := os.Rename(src, dst); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.Rename(w.path, backupPath(w.path, 1)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return w.open()
}

func backupPath(path string, index int) string {
	ext := filepath.Ext(path)
	base := path[:len(path)-len(ext)]
	return fmt.Sprintf("%s.%d%s", base, index, ext)
}
