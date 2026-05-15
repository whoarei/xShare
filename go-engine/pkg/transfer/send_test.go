package transfer

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func startReceiver(t *testing.T, targetDir string) (port int, done chan struct{}, errChan chan error) {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port = l.Addr().(*net.TCPAddr).Port

	done = make(chan struct{})
	errChan = make(chan error, 1)

	go func() {
		conn, err := l.Accept()
		if err != nil {
			errChan <- fmt.Errorf("accept: %w", err)
			return
		}
		l.Close()

		rc := NewReceiver(targetDir)
		err = rc.Receive(conn)
		conn.Close()
		if err != nil {
			errChan <- err
		} else {
			close(done)
		}
	}()

	// Give the listener time to start
	time.Sleep(50 * time.Millisecond)
	return
}

func fileHash(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		t.Fatalf("hash %s: %v", path, err)
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil))
}

func TestSendFile(t *testing.T) {
	tmpDir := t.TempDir()
	rcvDir := filepath.Join(tmpDir, "rcv")
	if err := os.MkdirAll(rcvDir, 0755); err != nil {
		t.Fatalf("mkdir rcv: %v", err)
	}

	// Create a test file
	srcFile := filepath.Join(tmpDir, "hello.txt")
	content := []byte("Hello xShare single file test!")
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	expectedHash := fileHash(t, srcFile)

	port, done, errChan := startReceiver(t, rcvDir)

	sender, err := NewSender(fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("NewSender: %v", err)
	}

	if err := sender.SendFile(srcFile); err != nil {
		sender.Close()
		t.Fatalf("SendFile: %v", err)
	}
	sender.Close()

	// Wait for receiver to finish
	select {
	case <-done:
	case e := <-errChan:
		t.Fatalf("receiver error: %v", e)
	case <-time.After(5 * time.Second):
		t.Fatal("receiver timed out")
	}

	received := filepath.Join(rcvDir, "hello.txt")
	if _, err := os.Stat(received); os.IsNotExist(err) {
		t.Fatal("received file does not exist")
	}

	gotHash := fileHash(t, received)
	if gotHash != expectedHash {
		t.Errorf("hash mismatch: got %s, want %s", gotHash, expectedHash)
	}

	gotContent, _ := os.ReadFile(received)
	if string(gotContent) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", string(gotContent), string(content))
	}
}

func TestSendFileEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	rcvDir := filepath.Join(tmpDir, "rcv")
	if err := os.MkdirAll(rcvDir, 0755); err != nil {
		t.Fatalf("mkdir rcv: %v", err)
	}

	srcFile := filepath.Join(tmpDir, "empty.txt")
	if err := os.WriteFile(srcFile, []byte{}, 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	port, done, errChan := startReceiver(t, rcvDir)

	sender, err := NewSender(fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("NewSender: %v", err)
	}

	if err := sender.SendFile(srcFile); err != nil {
		sender.Close()
		t.Fatalf("SendFile: %v", err)
	}
	sender.Close()

	select {
	case <-done:
	case e := <-errChan:
		t.Fatalf("receiver error: %v", e)
	case <-time.After(5 * time.Second):
		t.Fatal("receiver timed out")
	}

	received := filepath.Join(rcvDir, "empty.txt")
	got, _ := os.ReadFile(received)
	if len(got) != 0 {
		t.Errorf("expected empty file, got %d bytes", len(got))
	}
}

func TestSendFileRejectsDirectory(t *testing.T) {
	sender := &Sender{conn: nil} // won't be used
	err := sender.SendFile(t.TempDir())
	if err == nil {
		t.Fatal("expected error when sending directory as file, got nil")
	}
}

func TestSendDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	rcvDir := filepath.Join(tmpDir, "rcv")
	if err := os.MkdirAll(rcvDir, 0755); err != nil {
		t.Fatalf("mkdir rcv: %v", err)
	}

	// Create test directory structure
	subDir := filepath.Join(srcDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}
	files := map[string][]byte{
		filepath.Join(srcDir, "a.txt"):           []byte("file a"),
		filepath.Join(srcDir, "sub", "b.txt"):    []byte("file b"),
		filepath.Join(srcDir, "sub", "c.dat"):    []byte("file c content"),
	}
	for p, data := range files {
		if err := os.WriteFile(p, data, 0644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}

	// Compute expected hashes
	expected := make(map[string]string)
	for p := range files {
		rel, _ := filepath.Rel(srcDir, p)
		expected[rel] = fileHash(t, p)
	}

	port, done, errChan := startReceiver(t, rcvDir)

	sender, err := NewSender(fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("NewSender: %v", err)
	}

	if err := sender.SendDirectory(srcDir); err != nil {
		sender.Close()
		t.Fatalf("SendDirectory: %v", err)
	}
	sender.Close()

	select {
	case <-done:
	case e := <-errChan:
		t.Fatalf("receiver error: %v", e)
	case <-time.After(5 * time.Second):
		t.Fatal("receiver timed out")
	}

	for rel, wantHash := range expected {
		receivedPath := filepath.Join(rcvDir, "src", rel)
		if _, err := os.Stat(receivedPath); os.IsNotExist(err) {
			t.Errorf("missing received file: %s", rel)
			continue
		}
		gotHash := fileHash(t, receivedPath)
		if gotHash != wantHash {
			t.Errorf("hash mismatch for %s: got %s, want %s", rel, gotHash, wantHash)
		}
	}

	subDirRcv := filepath.Join(rcvDir, "src", "sub")
	if info, err := os.Stat(subDirRcv); err != nil || !info.IsDir() {
		t.Error("subdirectory was not created on receiver")
	}
}
