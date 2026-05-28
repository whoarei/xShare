// protocol包的单元测试
// 测试协议头编解码、字节序、消息类型等
package protocol

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"
)

// TestEncodeDecodeHeader 测试协议头编码和解码的正确性
// 验证Magic、Version、Type、Length字段的往返一致性
func TestEncodeDecodeHeader(t *testing.T) {
	h := &Header{
		Magic:   Magic,
		Version: Version,
		Type:    TypeFileHeader,
		Length:  42,
	}

	data, err := EncodeHeader(h)
	if err != nil {
		t.Fatalf("EncodeHeader failed: %v", err)
	}

	// 验证点：编码后的字节长度必须等于协议头固定大小(14字节)
	if len(data) != HeaderSize {
		t.Fatalf("expected header size %d, got %d", HeaderSize, len(data))
	}

	decoded, err := DecodeHeader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("DecodeHeader failed: %v", err)
	}

	// 验证点：解码后的Magic字段必须等于协议魔数0x58534852
	if decoded.Magic != Magic {
		t.Errorf("Magic mismatch: got 0x%X, want 0x%X", decoded.Magic, Magic)
	}
	// 验证点：解码后的Version字段必须等于协议版本号1
	if decoded.Version != Version {
		t.Errorf("Version mismatch: got %d, want %d", decoded.Version, Version)
	}
	// 验证点：解码后的Type字段必须等于编码时的消息类型
	if decoded.Type != TypeFileHeader {
		t.Errorf("Type mismatch: got %d, want %d", decoded.Type, TypeFileHeader)
	}
	// 验证点：解码后的Length字段必须等于编码时的载荷长度
	if decoded.Length != 42 {
		t.Errorf("Length mismatch: got %d, want %d", decoded.Length, 42)
	}
}

// TestEncodeDecodeHeader_Endianness 测试大端字节序编码
// 验证多字节字段按网络字节序(大端)正确编码
func TestEncodeDecodeHeader_Endianness(t *testing.T) {
	// 测试大端字节序编码
	h := &Header{
		Magic:   0x58534852,
		Version: 0x01,
		Type:    0x03,
		Length:  0x0102030405060708,
	}

	data, err := EncodeHeader(h)
	if err != nil {
		t.Fatalf("EncodeHeader failed: %v", err)
	}

	// 验证点：Magic字段必须按大端序排列为 0x58, 0x53, 0x48, 0x52
	expectedMagic := []byte{0x58, 0x53, 0x48, 0x52}
	for i, b := range expectedMagic {
		if data[i] != b {
			t.Errorf("Magic byte %d: got 0x%02X, want 0x%02X", i, data[i], b)
		}
	}

	// 验证点：Version字节必须为0x01
	if data[4] != 0x01 {
		t.Errorf("Version byte: got 0x%02X, want 0x01", data[4])
	}

	// 验证点：Type字节必须为0x03
	if data[5] != 0x03 {
		t.Errorf("Type byte: got 0x%02X, want 0x03", data[5])
	}

	// 验证点：Length字段必须按大端序编码，高位在前
	var length uint64
	binary.Read(bytes.NewReader(data[6:14]), binary.BigEndian, &length)
	if length != 0x0102030405060708 {
		t.Errorf("Length: got 0x%016X, want 0x0102030405060708", length)
	}

	// 验证点：解码往返后Length值必须保持一致
	decoded, err := DecodeHeader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("DecodeHeader failed: %v", err)
	}

	if decoded.Length != 0x0102030405060708 {
		t.Errorf("Round-trip Length mismatch: got 0x%016X", decoded.Length)
	}
}

// TestDecodeHeader_InvalidMagic 测试无效魔数的错误处理
// 验证解码器能正确识别并拒绝非法魔数
func TestDecodeHeader_InvalidMagic(t *testing.T) {
	// 构造一个魔数错误的协议头
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(0xDEADBEEF)) // 错误的魔数
	binary.Write(buf, binary.BigEndian, Version)
	binary.Write(buf, binary.BigEndian, TypeAck)
	binary.Write(buf, binary.BigEndian, uint64(0))

	_, err := DecodeHeader(bytes.NewReader(buf.Bytes()))
	// 验证点：当魔数不是0x58534852时，解码器必须返回错误
	if err == nil {
		t.Fatal("expected error for invalid magic, got nil")
	}
}

// TestDecodeHeader_VersionMismatch 测试版本不匹配的错误处理
// 验证解码器能正确识别并拒绝不支持的协议版本
func TestDecodeHeader_VersionMismatch(t *testing.T) {
	// 构造一个版本号不匹配的协议头
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, Magic)
	binary.Write(buf, binary.BigEndian, uint8(0x99)) // 不支持的版本号
	binary.Write(buf, binary.BigEndian, TypeAck)
	binary.Write(buf, binary.BigEndian, uint64(0))

	_, err := DecodeHeader(bytes.NewReader(buf.Bytes()))
	// 验证点：当版本号不是0x01时，解码器必须返回错误
	if err == nil {
		t.Fatal("expected error for version mismatch, got nil")
	}
}

// TestEncodeMessage 测试完整消息编码
// 验证消息头和载荷的正确组合
func TestEncodeMessage(t *testing.T) {
	payload := []byte("hello world")
	data, err := EncodeMessage(TypeChunk, payload)
	if err != nil {
		t.Fatalf("EncodeMessage failed: %v", err)
	}

	expectedLen := HeaderSize + len(payload)
	// 验证点：编码后的总长度必须等于头部大小(14) + 载荷长度
	if len(data) != expectedLen {
		t.Fatalf("expected %d bytes, got %d", expectedLen, len(data))
	}

	// 解码头部进行验证
	decodedHeader, err := DecodeHeader(bytes.NewReader(data[:HeaderSize]))
	if err != nil {
		t.Fatalf("DecodeHeader failed: %v", err)
	}

	// 验证点：解码后的消息类型必须与编码时指定的TypeChunk一致
	if decodedHeader.Type != TypeChunk {
		t.Errorf("Type mismatch: got %d, want %d", decodedHeader.Type, TypeChunk)
	}
	// 验证点：解码后的载荷长度必须等于原始载荷的字节数
	if decodedHeader.Length != uint64(len(payload)) {
		t.Errorf("Length mismatch: got %d, want %d", decodedHeader.Length, len(payload))
	}

	// 验证载荷内容
	decodedPayload := data[HeaderSize:]
	// 验证点：载荷内容必须与原始数据完全一致
	if string(decodedPayload) != string(payload) {
		t.Errorf("Payload mismatch: got %q, want %q", decodedPayload, payload)
	}
}

// TestEncodeDecodeJSON 测试JSON消息的编码和解码
// 验证TaskInfo结构体的JSON序列化往返一致性
func TestEncodeDecodeJSON(t *testing.T) {
	ti := TaskInfo{ID: "task-1", TotalSize: 2048, ItemCount: 3}
	data, err := EncodeJSON(TypeTaskInfo, ti)
	if err != nil {
		t.Fatalf("EncodeJSON failed: %v", err)
	}

	header, err := DecodeHeader(bytes.NewReader(data[:HeaderSize]))
	if err != nil {
		t.Fatalf("DecodeHeader failed: %v", err)
	}

	// 验证点：消息类型必须为TypeTaskInfo(0x01)
	if header.Type != TypeTaskInfo {
		t.Errorf("Type mismatch: got %d, want %d", header.Type, TypeTaskInfo)
	}

	payload, err := ReadPayload(bytes.NewReader(data[HeaderSize:]), header.Length)
	if err != nil {
		t.Fatalf("ReadPayload failed: %v", err)
	}

	var decoded TaskInfo
	// 验证点：载荷必须能正确反序列化为TaskInfo结构体
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	// 验证点：反序列化后的ID字段必须保持原始值"task-1"
	if decoded.ID != "task-1" {
		t.Errorf("ID mismatch: got %q, want %q", decoded.ID, "task-1")
	}
}

// TestAllMessageTypes 测试所有消息类型的编码
// 验证TaskInfo、FileHeader、Ack等消息类型的正确处理
func TestAllMessageTypes(t *testing.T) {
	tests := []struct {
		name    string
		msgType uint8
		payload interface{}
	}{
		// 测试TaskInfo消息：任务元数据（ID、总大小、文件数）
		{"TaskInfo", TypeTaskInfo, TaskInfo{ID: "t1", TotalSize: 1024, ItemCount: 5}},
		// 测试FileHeader消息：文件元数据（路径、大小、权限、哈希）
		{"FileHeader", TypeFileHeader, FileHeader{Path: "a/b.txt", Size: 512, Mode: 0644, Hash: "sha256:abc"}},
		// 测试Ack成功消息：确认状态为ok
		{"Ack", TypeAck, Ack{Status: "ok", Code: 0, Msg: ""}},
		// 测试Ack错误消息：确认状态为error，包含错误码和描述
		{"AckError", TypeAck, Ack{Status: "error", Code: 3, Msg: "disk full"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := EncodeJSON(tt.msgType, tt.payload)
			if err != nil {
				t.Fatalf("EncodeJSON failed: %v", err)
			}

			header, err := DecodeHeader(bytes.NewReader(data[:HeaderSize]))
			if err != nil {
				t.Fatalf("DecodeHeader failed: %v", err)
			}

			// 验证点：解码后的消息类型必须与编码时指定的类型一致
			if header.Type != tt.msgType {
				t.Errorf("Type mismatch: got 0x%02X, want 0x%02X", header.Type, tt.msgType)
			}
		})
	}
}

// TestDoneMessage 测试完成消息的编码
// 验证Done消息无载荷的正确处理
func TestDoneMessage(t *testing.T) {
	// Done消息无载荷，仅包含头部
	data, err := EncodeMessage(TypeDone, nil)
	if err != nil {
		t.Fatalf("EncodeMessage for Done failed: %v", err)
	}

	// 验证点：Done消息总长度必须等于头部大小(14字节)，无载荷
	if len(data) != HeaderSize {
		t.Errorf("Done message length: got %d, want %d", len(data), HeaderSize)
	}

	header, err := DecodeHeader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("DecodeHeader failed: %v", err)
	}

	// 验证点：消息类型必须为TypeDone(0x05)
	if header.Type != TypeDone {
		t.Errorf("Type mismatch: got 0x%02X, want 0x%02X", header.Type, TypeDone)
	}
	// 验证点：载荷长度必须为0
	if header.Length != 0 {
		t.Errorf("Done payload length: got %d, want 0", header.Length)
	}
}

// TestReadPayload_PartialRead 测试部分读取的错误处理
// 模拟TCP分片场景，验证数据不完整时的错误检测
func TestReadPayload_PartialRead(t *testing.T) {
	// 模拟TCP分片场景：实际数据只有3字节，但头部指示需要10字节
	shortData := []byte{0x01, 0x02, 0x03}
	_, err := ReadPayload(bytes.NewReader(shortData), 10)
	// 验证点：当读取的字节数少于请求的长度时，必须返回错误（防止TCP短读）
	if err == nil {
		t.Fatal("expected error for short read, got nil")
	}
}
