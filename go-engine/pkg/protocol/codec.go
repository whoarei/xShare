// xShare协议编解码器
package protocol

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// EncodeHeader 将协议头编码为字节序列
func EncodeHeader(h *Header) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, h.Magic); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, h.Version); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, h.Type); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, h.Length); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DecodeHeader 从字节流解码协议头
func DecodeHeader(r io.Reader) (*Header, error) {
	data := make([]byte, HeaderSize)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	buf := bytes.NewReader(data)
	h := &Header{}
	if err := binary.Read(buf, binary.BigEndian, &h.Magic); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.BigEndian, &h.Version); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.BigEndian, &h.Type); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.BigEndian, &h.Length); err != nil {
		return nil, err
	}
	if h.Magic != Magic {
		return nil, fmt.Errorf("invalid magic: 0x%X", h.Magic)
	}
	if h.Version != Version {
		return nil, fmt.Errorf("version mismatch: got %d, want %d", h.Version, Version)
	}
	return h, nil
}

// ReadPayload 读取指定长度的载荷数据
func ReadPayload(r io.Reader, length uint64) ([]byte, error) {
	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, fmt.Errorf("read payload: %w", err)
	}
	return data, nil
}

// EncodeMessage 编码完整消息（头+载荷）
func EncodeMessage(msgType uint8, payload []byte) ([]byte, error) {
	h := &Header{
		Magic:   Magic,
		Version: Version,
		Type:    msgType,
		Length:  uint64(len(payload)),
	}
	headerBytes, err := EncodeHeader(h)
	if err != nil {
		return nil, err
	}
	return append(headerBytes, payload...), nil
}

// EncodeJSON 将JSON对象编码为协议消息
func EncodeJSON(msgType uint8, v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return EncodeMessage(msgType, data)
}

// DecodeJSON 将协议载荷解码为JSON对象
func DecodeJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// SendJSON 发送JSON消息到写入器
func SendJSON(w io.Writer, msgType uint8, v interface{}) error {
	data, err := EncodeJSON(msgType, v)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// Send 发送原始载荷消息到写入器
func Send(w io.Writer, msgType uint8, payload []byte) error {
	data, err := EncodeMessage(msgType, payload)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
