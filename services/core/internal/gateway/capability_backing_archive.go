package gateway

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (t *capabilityBackingTool) createTarGZ(req Request) (string, error) {
	out := inputString(req.Input, "output")
	outputProvided := out != ""
	if out == "" {
		out = filepath.Join(nonEmpty(t.dataDir, os.TempDir()), "snapshots", fmt.Sprintf("%s-%d.tar.gz", strings.ReplaceAll(t.capability.ID, ".", "-"), time.Now().UnixMilli()))
	}
	if !filepath.IsAbs(out) {
		out = filepath.Join(t.workspace, out)
	}
	if outputProvided {
		if err := validateWorkspacePath(t.workspace, out); err != nil {
			return "", err
		}
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return "", err
	}
	if outputProvided {
		if err := validateWorkspacePath(t.workspace, out); err != nil {
			return "", err
		}
	}
	f, err := os.Create(out)
	if err != nil {
		return "", err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	paths := req.Paths
	if len(paths) == 0 {
		paths = []string{"."}
	}
	for _, raw := range paths {
		target, err := firstWorkspacePath([]string{raw}, t.workspace)
		if err != nil {
			return "", err
		}
		if err := addPathToTar(tw, t.workspace, target); err != nil {
			return "", err
		}
	}
	return out, nil
}

func addPathToTar(tw *tar.Writer, root, target string) error {
	return filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("archive target %q includes symlink path", path)
		}
		if err := validateWorkspacePath(root, path); err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		header.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
}

func (t *capabilityBackingTool) extractArchive(req Request) (string, error) {
	src := inputString(req.Input, "archive")
	if src == "" && len(req.Paths) > 0 {
		src = req.Paths[0]
	}
	if src == "" {
		return "", errors.New("extract requires input.archive or a path")
	}
	if !filepath.IsAbs(src) {
		src = filepath.Join(t.workspace, src)
	}
	dst := inputString(req.Input, "destination")
	if dst == "" {
		dst = "scratch/extracted"
	}
	if !filepath.IsAbs(dst) {
		dst = filepath.Join(t.workspace, dst)
	}
	if !pathContains(t.workspace, dst) {
		return "", fmt.Errorf("extract destination %q outside workspace", dst)
	}
	if strings.HasSuffix(src, ".zip") {
		return dst, extractZip(src, dst)
	}
	return dst, extractTarGZ(src, dst)
}

func extractTarGZ(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	if err := prepareArchiveDestination(dst); err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dst, filepath.Clean(header.Name))
		if err := validateArchiveEntryTarget(dst, target, header.Name); err != nil {
			return err
		}
		if header.FileInfo().Mode()&os.ModeType != 0 && !header.FileInfo().IsDir() {
			return fmt.Errorf("archive entry %q has unsupported file type", header.Name)
		}
		if header.FileInfo().IsDir() {
			if err := safeMkdirAllForArchive(dst, target); err != nil {
				return err
			}
			continue
		}
		if err := safeMkdirAllForArchive(dst, filepath.Dir(target)); err != nil {
			return err
		}
		if err := validateArchiveEntryTarget(dst, target, header.Name); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, header.FileInfo().Mode())
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, tr)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
}

func extractZip(src, dst string) error {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer zr.Close()
	if err := prepareArchiveDestination(dst); err != nil {
		return err
	}
	for _, file := range zr.File {
		target := filepath.Join(dst, filepath.Clean(file.Name))
		if err := validateArchiveEntryTarget(dst, target, file.Name); err != nil {
			return err
		}
		if file.FileInfo().Mode()&os.ModeType != 0 && !file.FileInfo().IsDir() {
			return fmt.Errorf("archive entry %q has unsupported file type", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := safeMkdirAllForArchive(dst, target); err != nil {
				return err
			}
			continue
		}
		if err := safeMkdirAllForArchive(dst, filepath.Dir(target)); err != nil {
			return err
		}
		if err := validateArchiveEntryTarget(dst, target, file.Name); err != nil {
			return err
		}
		in, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, file.FileInfo().Mode())
		if err != nil {
			_ = in.Close()
			return err
		}
		_, copyErr := io.Copy(out, in)
		_ = in.Close()
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func prepareArchiveDestination(root string) error {
	if info, err := os.Lstat(root); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("archive destination %q is a symlink", root)
		}
		if !info.IsDir() {
			return fmt.Errorf("archive destination %q is not a directory", root)
		}
		return validateArchiveEntryTarget(root, root, root)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	parent := filepath.Dir(filepath.Clean(root))
	resolvedParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return err
	}
	absParent, err := filepath.Abs(parent)
	if err != nil {
		absParent = parent
	}
	if filepath.Clean(resolvedParent) != filepath.Clean(absParent) {
		return fmt.Errorf("archive destination parent %q resolves through a symlink", parent)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	return validateArchiveEntryTarget(root, root, root)
}

func safeMkdirAllForArchive(root, target string) error {
	if err := validateArchiveEntryTarget(root, target, target); err != nil {
		return err
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	return validateArchiveEntryTarget(root, target, target)
}

func validateArchiveEntryTarget(root, target, entryName string) error {
	if !pathContains(root, target) {
		return fmt.Errorf("archive entry %q escapes destination", entryName)
	}
	if info, err := os.Lstat(target); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("archive entry %q targets a symlink path", entryName)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	existing, err := nearestExistingPath(target)
	if err != nil {
		return err
	}
	resolved, err := filepath.EvalSymlinks(existing)
	if err != nil {
		return err
	}
	if !pathContains(root, resolved) {
		return fmt.Errorf("archive entry %q escapes destination through symlink path", entryName)
	}
	return nil
}

func nearestExistingPath(path string) (string, error) {
	current := filepath.Clean(path)
	for {
		if _, err := os.Lstat(current); err == nil {
			return current, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		next := filepath.Dir(current)
		if next == current {
			return "", fmt.Errorf("no existing parent for archive target %q", path)
		}
		current = next
	}
}
