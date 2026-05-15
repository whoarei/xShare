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

func NewReceiver(targetDir string) *Receiver {
	return &Receiver{targetDir: targetDir}
}

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

func (r *Receiver) handleChunk(length uint64) error {
	payload, err := protocol.ReadPayload(r.conn, length)
	if err != nil {
		return fmt.Errorf("read chunk payload: %w", err)
	}

	if r.currentFile == nil {
		return fmt.Errorf("received chunk without active file header")
	}

	if _, err := r.currentFile.Write(payload); err != nil {
		return fmt.Errorf("write chunk: %w", err)
	}

	if r.hasher != nil {
		r.hasher.Write(payload)
	}

	r.receivedSize += int64(length)

	if r.receivedSize >= r.currentSize {
		return r.verifyAndSendFinalAck()
	}

	return nil
}

func (r *Receiver) finalize() error {
	r.resetCurrentFile()
	fmt.Println(`{"type":"complete"}`)
	return nil
}

func (r *Receiver) verifyAndSendFinalAck() error {
	computed := fmt.Sprintf("sha256:%x", r.hasher.Sum(nil))

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
		go func() {
			rc := NewReceiver(r.targetDir)
			if err := rc.Receive(conn); err != nil {
				fmt.Fprintf(os.Stderr, "receive error: %v\n", err)
			}
		}()
	}
}

func (r *Receiver) sendAck(status string, code int, msg string) error {
	ack := protocol.Ack{Status: status, Code: code, Msg: msg}
	return protocol.SendJSON(r.conn, protocol.TypeAck, ack)
}
