// 文件传输接收端实现
package transfer

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"hash"
	"net"
	"os"
	"path/filepath"
	"strings"

	"go-engine/pkg/protocol"
)

// Receiver 文件接收器
type Receiver struct {
	targetDir    string
	conn         net.Conn
	currentFile  *os.File
	currentHash  string
	currentPath  string
	currentSize  int64
	receivedSize int64
	hasher       hash.Hash
}

// NewReceiver 创建新的接收器
func NewReceiver(targetDir string) *Receiver {
	return &Receiver{targetDir: targetDir}
}

// Receive 接收文件传输
func (r *Receiver) Receive(conn net.Conn) error {
	r.conn = conn
	for {
		header, err := protocol.DecodeHeader(conn)
		if err != nil {
			if strings.Contains(err.Error(), "EOF") {
				return nil
			}
			return fmt.Errorf("decode header: %w", err)
		}

		switch header.Type {
		case protocol.TypeTaskInfo:
			if err := r.handleTaskInfo(header.Length); err != nil {
				return err
			}
		case protocol.TypeFileHeader:
			if err := r.handleFileHeader(header.Length); err != nil {
				return err
			}
		case protocol.TypeChunk:
			if err := r.handleChunk(header.Length); err != nil {
				return err
			}
		case protocol.TypeDone:
			return r.finalize()
		default:
			if header.Length > 0 {
				protocol.ReadPayload(conn, header.Length)
			}
		}
	}
}

// resetCurrentFile 重置当前文件状态
func (r *Receiver) resetCurrentFile() {
	if r.currentFile != nil {
		r.currentFile.Close()
		r.currentFile = nil
	}
	r.currentHash = ""
	r.currentPath = ""
	r.currentSize = 0
	r.receivedSize = 0
	r.hasher = nil
}

// handleTaskInfo 处理任务信息消息
func (r *Receiver) handleTaskInfo(length uint64) error {
	payload, err := protocol.ReadPayload(r.conn, length)
	if err != nil {
		return fmt.Errorf("read task info payload: %w", err)
	}
	var ti protocol.TaskInfo
	if err := json.Unmarshal(payload, &ti); err != nil {
		return fmt.Errorf("unmarshal task info: %w", err)
	}
	fmt.Printf(`{"type":"task","id":"%s","total_size":%d,"item_count":%d}`+"\n",
		ti.ID, ti.TotalSize, ti.ItemCount)
	return nil
}

// handleFileHeader 处理文件头消息
func (r *Receiver) handleFileHeader(length uint64) error {
	r.resetCurrentFile()

	payload, err := protocol.ReadPayload(r.conn, length)
	if err != nil {
		return fmt.Errorf("read file header payload: %w", err)
	}
	var fh protocol.FileHeader
	if err := json.Unmarshal(payload, &fh); err != nil {
		return fmt.Errorf("unmarshal file header: %w", err)
	}

	// 安全检查：防止路径遍历攻击
	if strings.Contains(fh.Path, "..") {
		return r.sendAck("error", 5, "path traversal detected")
	}

	fullPath := filepath.Join(r.targetDir, filepath.FromSlash(fh.Path))
	cleanTarget, _ := filepath.Abs(r.targetDir)
	cleanTarget = filepath.Clean(cleanTarget)
	cleanPath := filepath.Clean(fullPath)
	if !strings.HasPrefix(cleanPath, cleanTarget+string(filepath.Separator)) && cleanPath != cleanTarget {
		return r.sendAck("error", 5, "path traversal detected")
	}

	// 创建目录和文件
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return r.sendAck("error", 1, fmt.Sprintf("mkdir parent: %v", err))
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return r.sendAck("error", 2, fmt.Sprintf("create file: %v", err))
	}

	r.currentFile = f
	r.currentHash = fh.Hash
	r.currentPath = fh.Path
	r.currentSize = fh.Size
	r.receivedSize = 0
	r.hasher = sha256.New()

	fmt.Printf(`{"type":"progress","path":"%s","kind":"file","size":%d}`+"\n", fh.Path, fh.Size)

	return r.sendAck("ok", 0, "")
}

// handleChunk 处理文件数据块
func (r *Receiver) handleChunk(length uint64) error {
	payload, err := protocol.ReadPayload(r.conn, length)
	if err != nil {
		return fmt.Errorf("read chunk payload: %w", err)
	}

	if r.currentFile == nil {
		return fmt.Errorf("received chunk without active file header")
	}

	// 写入文件并更新校验和
	if _, err := r.currentFile.Write(payload); err != nil {
		return fmt.Errorf("write chunk: %w", err)
	}

	if r.hasher != nil {
		r.hasher.Write(payload)
	}

	r.receivedSize += int64(length)

	// 检查文件是否接收完成
	if r.receivedSize >= r.currentSize {
		return r.verifyAndSendFinalAck()
	}

	return nil
}

// finalize 完成传输，清理资源
func (r *Receiver) finalize() error {
	r.resetCurrentFile()
	fmt.Println(`{"type":"complete"}`)
	return nil
}

// verifyAndSendFinalAck 校验文件完整性并发送最终确认
func (r *Receiver) verifyAndSendFinalAck() error {
	computed := fmt.Sprintf("sha256:%x", r.hasher.Sum(nil))

	// 校验SHA256哈希值
	if r.currentHash != "" && computed != r.currentHash {
		r.currentFile.Close()
		os.Remove(r.currentFile.Name())
		r.resetCurrentFile()
		return r.sendAck("error", 4,
			fmt.Sprintf("hash mismatch: expected %s, got %s", r.currentHash, computed))
	}

	r.currentFile.Close()
	r.currentFile = nil
	r.hasher = nil

	return r.sendAck("ok", 0, "")
}

// ListenAndServe 启动TCP服务器监听传入连接
func (r *Receiver) ListenAndServe(addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "accept error: %v\n", err)
			continue
		}
		// 为每个连接启动独立的接收协程
		go func() {
			rc := NewReceiver(r.targetDir)
			if err := rc.Receive(conn); err != nil {
				fmt.Fprintf(os.Stderr, "receive error: %v\n", err)
			}
		}()
	}
}

// sendAck 发送确认消息给发送端
func (r *Receiver) sendAck(status string, code int, msg string) error {
	ack := protocol.Ack{Status: status, Code: code, Msg: msg}
	return protocol.SendJSON(r.conn, protocol.TypeAck, ack)
}
