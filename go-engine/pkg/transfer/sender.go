package transfer

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"

	"go-engine/pkg/protocol"
)

type Sender struct {
	conn net.Conn
}

func NewSender(addr string) (*Sender, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	return &Sender{conn: conn}, nil
}

func (s *Sender) Close() error {
	return s.conn.Close()
}

func (s *Sender) SendDirectory(dirPath string) error {
	absDir, err := filepath.Abs(dirPath)
	if err != nil {
		return fmt.Errorf("abs path: %w", err)
	}
	baseName := filepath.Base(absDir)

	var itemCount int
	var totalSize int64

	err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == absDir {
			return nil
		}
		itemCount++
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk count: %w", err)
	}

	taskInfo := protocol.TaskInfo{
		ID:        baseName,
		TotalSize: totalSize,
		ItemCount: itemCount,
	}
	if err := protocol.SendJSON(s.conn, protocol.TypeTaskInfo, taskInfo); err != nil {
		return fmt.Errorf("send task info: %w", err)
	}

	var sentFiles int

	err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == absDir {
			return nil
		}

		relPath, _ := filepath.Rel(absDir, path)
		relPath = filepath.ToSlash(relPath)

		if info.IsDir() {
			mkdir := protocol.Mkdir{Path: relPath}
			if err := protocol.SendJSON(s.conn, protocol.TypeMkdir, mkdir); err != nil {
				return fmt.Errorf("send mkdir: %w", err)
			}
			fmt.Printf(`{"type":"progress","path":"%s","kind":"mkdir"}`+"\n", relPath)
		} else {
			if err := s.sendFile(path, relPath, info); err != nil {
				return fmt.Errorf("send file %s: %w", relPath, err)
			}
			sentFiles++
			fmt.Printf(`{"type":"progress","path":"%s","kind":"file","done":true}`+"\n", relPath)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if err := protocol.Send(s.conn, protocol.TypeDone, nil); err != nil {
		return fmt.Errorf("send done: %w", err)
	}

	fmt.Printf(`{"type":"complete","total_files":%d}`+"\n", sentFiles)
	return nil
}

func (s *Sender) sendFile(absPath, relPath string, info os.FileInfo) error {
	f, err := os.Open(absPath)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return fmt.Errorf("hash: %w", err)
	}
	fileHash := fmt.Sprintf("sha256:%x", hasher.Sum(nil))

	if _, err := f.Seek(0, 0); err != nil {
		return fmt.Errorf("seek: %w", err)
	}

	fh := protocol.FileHeader{
		Path: relPath,
		Size: info.Size(),
		Mode: int(info.Mode()),
		Hash: fileHash,
	}
	if err := protocol.SendJSON(s.conn, protocol.TypeFileHeader, fh); err != nil {
		return fmt.Errorf("send file header: %w", err)
	}

	ack, err := s.readAck()
	if err != nil {
		return fmt.Errorf("read pre-check ack: %w", err)
	}
	if ack.Status != "ok" {
		return fmt.Errorf("receiver rejected: code=%d msg=%s", ack.Code, ack.Msg)
	}

	buf := make([]byte, 64*1024)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			if err := protocol.Send(s.conn, protocol.TypeChunk, buf[:n]); err != nil {
				return fmt.Errorf("send chunk: %w", err)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}
	}

	ack, err = s.readAck()
	if err != nil {
		return fmt.Errorf("read final ack: %w", err)
	}
	if ack.Status != "ok" {
		return fmt.Errorf("hash mismatch: %s", ack.Msg)
	}

	return nil
}

func (s *Sender) readAck() (*protocol.Ack, error) {
	header, err := protocol.DecodeHeader(s.conn)
	if err != nil {
		return nil, fmt.Errorf("decode header: %w", err)
	}
	if header.Type != protocol.TypeAck {
		return nil, fmt.Errorf("expected Ack, got type 0x%02X", header.Type)
	}
	payload, err := protocol.ReadPayload(s.conn, header.Length)
	if err != nil {
		return nil, fmt.Errorf("read payload: %w", err)
	}
	ack := &protocol.Ack{}
	if err := json.Unmarshal(payload, ack); err != nil {
		return nil, fmt.Errorf("unmarshal ack: %w", err)
	}
	return ack, nil
}
