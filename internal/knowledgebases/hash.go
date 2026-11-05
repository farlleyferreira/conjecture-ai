package knowledgebases

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// ComputeContentHash computes a knowledge base's required content hash
// (see [Base.ContentHash]) from the file or directory at path: the
// SHA-256 of a single file's own bytes, or, for a directory - the
// specification's own example registers one ("./normas/") - of every
// regular file within it, each one's path (relative to path itself,
// so the hash does not depend on where the tree happens to sit) and
// content in [filepath.WalkDir]'s own documented lexical order, with a
// NUL separator after each of a file's path and its content so two
// different trees can never hash the same by accident.
func ComputeContentHash(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("knowledgebases: %w", err)
	}

	h := sha256.New()
	if !info.IsDir() {
		if err := hashFile(h, path); err != nil {
			return nil, fmt.Errorf("knowledgebases: %w", err)
		}
		return h.Sum(nil), nil
	}

	err = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(path, p)
		if err != nil {
			return err
		}
		h.Write([]byte(filepath.ToSlash(rel)))
		h.Write([]byte{0})
		if err := hashFile(h, p); err != nil {
			return err
		}
		h.Write([]byte{0})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("knowledgebases: %w", err)
	}
	return h.Sum(nil), nil
}

func hashFile(h io.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = io.Copy(h, f)
	return err
}
