// 文件传输发送端实现
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

// Sender 文件发送器
type Sender struct {
	conn net.Conn
}

// NewSender 创建新的发送器并连接到目标地址
func NewSender(addr string) (*Sender, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	return &Sender{conn: conn}, nil
}

// Close 关闭发送器连接
func (s *Sender) Close() error {
	return s.conn.Close()
}

// SendFile 发送单个文件
func (s *Sender) SendFile(filePath string) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("abs path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("expected file, got directory: %s", filePath)
	}

	relPath := filepath.Base(absPath)
	relPath = filepath.ToSlash(relPath)

	if err := s.sendFile(absPath, relPath, info); err != nil {
		return fmt.Errorf("send file %s: %w", relPath, err)
	}

	fmt.Printf(`{"type":"progress","path":"%s","kind":"file","done":true}`+"\n", relPath)
	fmt.Printf(`{"type":"complete","total_files":1}`+"\n")
	return nil
}

// SendDirectory 发送整个目录
func (s *Sender) SendDirectory(dirPath string) error {
	absDir, err := filepath.Abs(dirPath)
	if err != nil {
		return fmt.Errorf("abs path: %w", err)
	}
	baseName := filepath.Base(absDir)

	var itemCount int
	var totalSize int64

	// 第一次遍历：统计文件数量和总大小
	err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == absDir {
			return nil
		}
		if !info.IsDir() {
			itemCount++
			totalSize += info.Size()
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk count: %w", err)
	}

	// 发送任务信息
	taskInfo := protocol.TaskInfo{
		ID:        baseName,
		TotalSize: totalSize,
		ItemCount: itemCount,
	}
	if err := protocol.SendJSON(s.conn, protocol.TypeTaskInfo, taskInfo); err != nil {
		return fmt.Errorf("send task info: %w", err)
	}

	var sentFiles int

	// 第二次遍历：发送文件
	err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == absDir {
			return nil
		}

		relPath, _ := filepath.Rel(absDir, path)
		relPath = filepath.ToSlash(filepath.Join(baseName, relPath))

		if info.IsDir() {
			return nil
		}
		if err := s.sendFile(path, relPath, info); err != nil {
			return fmt.Errorf("send file %s: %w", relPath, err)
		}
		sentFiles++
		fmt.Printf(`{"type":"progress","path":"%s","kind":"file","done":true}`+"\n", relPath)
		return nil
	})
	if err != nil {
		return err
	}

	// 发送完成标记
	if err := protocol.Send(s.conn, protocol.TypeDone, nil); err != nil {
		return fmt.Errorf("send done: %w", err)
	}

	fmt.Printf(`{"type":"complete","total_files":%d}`+"\n", sentFiles)
	return nil
}

// sendFile 发送单个文件的内部实现
func (s *Sender) sendFile(absPath, relPath string, info os.FileInfo) error {
	f, err := os.Open(absPath)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	// 计算文件SHA256校验和
	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return fmt.Errorf("hash: %w", err)
	}
	fileHash := fmt.Sprintf("sha256:%x", hasher.Sum(nil))

	// 重置文件指针到开头
	if _, err := f.Seek(0, 0); err != nil {
		return fmt.Errorf("seek: %w", err)
	}

	// 发送文件头信息
	fh := protocol.FileHeader{
		Path: relPath,
		Size: info.Size(),
		Mode: int(info.Mode()),
		Hash: fileHash,
	}
	if err := protocol.SendJSON(s.conn, protocol.TypeFileHeader, fh); err != nil {
		return fmt.Errorf("send file header: %w", err)
	}

	// 等待接收端确认
	ack, err := s.readAck()
	if err != nil {
		return fmt.Errorf("read pre-check ack: %w", err)
	}
	if ack.Status != "ok" {
		return fmt.Errorf("receiver rejected: code=%d msg=%s", ack.Code, ack.Msg)
	}

	// 分块发送文件数据
	buf := make([]byte, 64*1024)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			if err := protocol.Send(s.conn, protocol.TypeChunk, buf[:n]); err != nil {
				return fmt.Errorf("send chunk: %w", err)
			}
		}
		if err == io.EOF {
			if n == 0 && info.Size() == 0 {
				if err := protocol.Send(s.conn, protocol.TypeChunk, nil); err != nil {
					return fmt.Errorf("send empty chunk: %w", err)
				}
			}
			break
		}
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}
	}

	// 等待最终校验确认
	ack, err = s.readAck()
	if err != nil {
		return fmt.Errorf("read final ack: %w", err)
	}
	if ack.Status != "ok" {
		return fmt.Errorf("hash mismatch: %s", ack.Msg)
	}

	return nil
}

// readAck 读取接收端的确认消息
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
