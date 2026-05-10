package gateway

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestExtractZipRejectsSymlinkDestinationEscape(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink escape check uses Unix symlink semantics")
	}

	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("create outside dir: %v", err)
	}

	archivePath := filepath.Join(workspace, "payload.zip")
	createZipWithFile(t, archivePath, "payload.txt", "escaped")

	dst := filepath.Join(workspace, "scratch", "link-out")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("create scratch dir: %v", err)
	}
	if err := os.Symlink(outside, dst); err != nil {
		t.Fatalf("create destination symlink: %v", err)
	}

	if err := extractZip(archivePath, dst); err == nil {
		t.Fatalf("expected symlink destination escape to be rejected")
	}
	if _, err := os.Stat(filepath.Join(outside, "payload.txt")); !os.IsNotExist(err) {
		t.Fatalf("outside payload should not be written, stat err=%v", err)
	}
}

func TestExtractTarGZRejectsSymlinkDestinationEscape(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink escape check uses Unix symlink semantics")
	}

	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("create outside dir: %v", err)
	}

	archivePath := filepath.Join(workspace, "payload.tar.gz")
	createTarGZWithFile(t, archivePath, "payload.txt", "escaped")

	dst := filepath.Join(workspace, "scratch", "link-out")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("create scratch dir: %v", err)
	}
	if err := os.Symlink(outside, dst); err != nil {
		t.Fatalf("create destination symlink: %v", err)
	}

	if err := extractTarGZ(archivePath, dst); err == nil {
		t.Fatalf("expected symlink destination escape to be rejected")
	}
	if _, err := os.Stat(filepath.Join(outside, "payload.txt")); !os.IsNotExist(err) {
		t.Fatalf("outside payload should not be written, stat err=%v", err)
	}
}

func TestExtractArchivesRejectSymlinkEntries(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink entry check uses Unix symlink modes")
	}

	root := t.TempDir()
	zipPath := filepath.Join(root, "links.zip")
	createZipWithSymlink(t, zipPath, "link-out", "/tmp/outside")
	if err := extractZip(zipPath, filepath.Join(root, "zip-dst")); err == nil {
		t.Fatalf("expected zip symlink entry to be rejected")
	}

	tarPath := filepath.Join(root, "links.tar.gz")
	createTarGZWithSymlink(t, tarPath, "link-out", "/tmp/outside")
	if err := extractTarGZ(tarPath, filepath.Join(root, "tar-dst")); err == nil {
		t.Fatalf("expected tar symlink entry to be rejected")
	}
}

func TestExtractZipCreatesRegularDestination(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	archivePath := filepath.Join(root, "payload.zip")
	createZipWithFile(t, archivePath, "nested/payload.txt", "ok")

	dst := filepath.Join(root, "new-destination")
	if err := extractZip(archivePath, dst); err != nil {
		t.Fatalf("extract zip: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(dst, "nested", "payload.txt"))
	if err != nil {
		t.Fatalf("read extracted payload: %v", err)
	}
	if string(body) != "ok" {
		t.Fatalf("unexpected payload body %q", string(body))
	}
}

func createZipWithFile(t *testing.T, path, name, body string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create(name)
	if err != nil {
		_ = zw.Close()
		_ = f.Close()
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := w.Write([]byte(body)); err != nil {
		_ = zw.Close()
		_ = f.Close()
		t.Fatalf("write zip entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		_ = f.Close()
		t.Fatalf("close zip writer: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close zip file: %v", err)
	}
}

func createZipWithSymlink(t *testing.T, path, name, target string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	zw := zip.NewWriter(f)
	header := &zip.FileHeader{Name: name}
	header.SetMode(os.ModeSymlink | 0o777)
	w, err := zw.CreateHeader(header)
	if err != nil {
		_ = zw.Close()
		_ = f.Close()
		t.Fatalf("create zip symlink entry: %v", err)
	}
	if _, err := w.Write([]byte(target)); err != nil {
		_ = zw.Close()
		_ = f.Close()
		t.Fatalf("write zip symlink entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		_ = f.Close()
		t.Fatalf("close zip writer: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close zip file: %v", err)
	}
}

func createTarGZWithFile(t *testing.T, path, name, body string) {
	t.Helper()
	writeTarGZ(t, path, func(tw *tar.Writer) {
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(body)),
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatalf("write tar file header: %v", err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatalf("write tar file body: %v", err)
		}
	})
}

func createTarGZWithSymlink(t *testing.T, path, name, target string) {
	t.Helper()
	writeTarGZ(t, path, func(tw *tar.Writer) {
		header := &tar.Header{
			Name:     name,
			Mode:     0o777,
			Typeflag: tar.TypeSymlink,
			Linkname: target,
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatalf("write tar symlink header: %v", err)
		}
	})
}

func writeTarGZ(t *testing.T, path string, write func(*tar.Writer)) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create tar.gz: %v", err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	write(tw)
	if err := tw.Close(); err != nil {
		_ = gw.Close()
		_ = f.Close()
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gw.Close(); err != nil {
		_ = f.Close()
		t.Fatalf("close gzip writer: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close tar.gz file: %v", err)
	}
}
