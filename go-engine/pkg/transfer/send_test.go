// transfer包的单元测试
// 测试文件发送和接收功能
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

// startReceiver 启动测试用的接收服务器
// 返回端口号、完成通道和错误通道
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

// fileHash 计算文件的SHA256哈希值
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

// TestSendFile 测试单文件发送功能
// 验证文件内容和哈希值在传输后保持一致
func TestSendFile(t *testing.T) {
	tmpDir := t.TempDir()
	rcvDir := filepath.Join(tmpDir, "rcv")
	if err := os.MkdirAll(rcvDir, 0755); err != nil {
		t.Fatalf("mkdir rcv: %v", err)
	}

	// 创建测试文件
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

	// 等待接收端完成，超时5秒
	select {
	case <-done:
	case e := <-errChan:
		t.Fatalf("receiver error: %v", e)
	case <-time.After(5 * time.Second):
		t.Fatal("receiver timed out")
	}

	received := filepath.Join(rcvDir, "hello.txt")
	// 验证点：接收端必须成功创建目标文件
	if _, err := os.Stat(received); os.IsNotExist(err) {
		t.Fatal("received file does not exist")
	}

	gotHash := fileHash(t, received)
	// 验证点：接收文件的SHA256哈希必须与源文件一致（确保数据完整性）
	if gotHash != expectedHash {
		t.Errorf("hash mismatch: got %s, want %s", gotHash, expectedHash)
	}

	gotContent, _ := os.ReadFile(received)
	// 验证点：接收文件的内容必须与源文件完全一致
	if string(gotContent) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", string(gotContent), string(content))
	}
}

// TestSendFileEmpty 测试空文件发送
// 验证零字节文件的正确传输
func TestSendFileEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	rcvDir := filepath.Join(tmpDir, "rcv")
	if err := os.MkdirAll(rcvDir, 0755); err != nil {
		t.Fatalf("mkdir rcv: %v", err)
	}

	// 创建空文件（0字节）
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
	// 验证点：接收的文件必须保持0字节，不能有额外数据
	if len(got) != 0 {
		t.Errorf("expected empty file, got %d bytes", len(got))
	}
}

// TestSendFileRejectsDirectory 测试SendFile拒绝目录输入
// 验证传入目录路径时返回错误
func TestSendFileRejectsDirectory(t *testing.T) {
	sender := &Sender{conn: nil} // 不需要实际连接
	err := sender.SendFile(t.TempDir())
	// 验证点：当传入目录路径时，SendFile必须返回错误（防止误将目录当文件发送）
	if err == nil {
		t.Fatal("expected error when sending directory as file, got nil")
	}
}

// TestSendDirectory 测试目录发送功能
// 验证目录结构、子目录和文件内容的完整传输
func TestSendDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	rcvDir := filepath.Join(tmpDir, "rcv")
	if err := os.MkdirAll(rcvDir, 0755); err != nil {
		t.Fatalf("mkdir rcv: %v", err)
	}

	// 创建测试目录结构（包含子目录）
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

	// 计算所有源文件的哈希值用于后续验证
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

	// 验证点：每个源文件必须在接收端存在且哈希值一致
	for rel, wantHash := range expected {
		receivedPath := filepath.Join(rcvDir, "src", rel)
		// 验证点：接收端必须创建对应的文件
		if _, err := os.Stat(receivedPath); os.IsNotExist(err) {
			t.Errorf("missing received file: %s", rel)
			continue
		}
		gotHash := fileHash(t, receivedPath)
		// 验证点：文件内容的SHA256哈希必须与源文件一致
		if gotHash != wantHash {
			t.Errorf("hash mismatch for %s: got %s, want %s", rel, gotHash, wantHash)
		}
	}

	// 验证点：子目录结构必须在接收端被正确创建
	subDirRcv := filepath.Join(rcvDir, "src", "sub")
	if info, err := os.Stat(subDirRcv); err != nil || !info.IsDir() {
		t.Error("subdirectory was not created on receiver")
	}
}
