package protocol

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"
)

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

	if len(data) != HeaderSize {
		t.Fatalf("expected header size %d, got %d", HeaderSize, len(data))
	}

	decoded, err := DecodeHeader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("DecodeHeader failed: %v", err)
	}

	if decoded.Magic != Magic {
		t.Errorf("Magic mismatch: got 0x%X, want 0x%X", decoded.Magic, Magic)
	}
	if decoded.Version != Version {
		t.Errorf("Version mismatch: got %d, want %d", decoded.Version, Version)
	}
	if decoded.Type != TypeFileHeader {
		t.Errorf("Type mismatch: got %d, want %d", decoded.Type, TypeFileHeader)
	}
	if decoded.Length != 42 {
		t.Errorf("Length mismatch: got %d, want %d", decoded.Length, 42)
	}
}

func TestEncodeDecodeHeader_Endianness(t *testing.T) {
	// Ensure big-endian encoding is correct
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

	// Verify magic bytes in big-endian order
	expectedMagic := []byte{0x58, 0x53, 0x48, 0x52}
	for i, b := range expectedMagic {
		if data[i] != b {
			t.Errorf("Magic byte %d: got 0x%02X, want 0x%02X", i, data[i], b)
		}
	}

	// Verify version byte
	if data[4] != 0x01 {
		t.Errorf("Version byte: got 0x%02X, want 0x01", data[4])
	}

	// Verify type byte
	if data[5] != 0x03 {
		t.Errorf("Type byte: got 0x%02X, want 0x03", data[5])
	}

	// Verify length in big-endian order
	var length uint64
	binary.Read(bytes.NewReader(data[6:14]), binary.BigEndian, &length)
	if length != 0x0102030405060708 {
		t.Errorf("Length: got 0x%016X, want 0x0102030405060708", length)
	}

	// Decode and verify
	decoded, err := DecodeHeader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("DecodeHeader failed: %v", err)
	}

	if decoded.Length != 0x0102030405060708 {
		t.Errorf("Round-trip Length mismatch: got 0x%016X", decoded.Length)
	}
}

func TestDecodeHeader_InvalidMagic(t *testing.T) {
	// Header with wrong magic
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(0xDEADBEEF))
	binary.Write(buf, binary.BigEndian, Version)
	binary.Write(buf, binary.BigEndian, TypeAck)
	binary.Write(buf, binary.BigEndian, uint64(0))

	_, err := DecodeHeader(bytes.NewReader(buf.Bytes()))
	if err == nil {
		t.Fatal("expected error for invalid magic, got nil")
	}
}

func TestDecodeHeader_VersionMismatch(t *testing.T) {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, Magic)
	binary.Write(buf, binary.BigEndian, uint8(0x99))
	binary.Write(buf, binary.BigEndian, TypeAck)
	binary.Write(buf, binary.BigEndian, uint64(0))

	_, err := DecodeHeader(bytes.NewReader(buf.Bytes()))
	if err == nil {
		t.Fatal("expected error for version mismatch, got nil")
	}
}

func TestEncodeMessage(t *testing.T) {
	payload := []byte("hello world")
	data, err := EncodeMessage(TypeChunk, payload)
	if err != nil {
		t.Fatalf("EncodeMessage failed: %v", err)
	}

	expectedLen := HeaderSize + len(payload)
	if len(data) != expectedLen {
		t.Fatalf("expected %d bytes, got %d", expectedLen, len(data))
	}

	// Decode header
	decodedHeader, err := DecodeHeader(bytes.NewReader(data[:HeaderSize]))
	if err != nil {
		t.Fatalf("DecodeHeader failed: %v", err)
	}

	if decodedHeader.Type != TypeChunk {
		t.Errorf("Type mismatch: got %d, want %d", decodedHeader.Type, TypeChunk)
	}
	if decodedHeader.Length != uint64(len(payload)) {
		t.Errorf("Length mismatch: got %d, want %d", decodedHeader.Length, len(payload))
	}

	// Verify payload
	decodedPayload := data[HeaderSize:]
	if string(decodedPayload) != string(payload) {
		t.Errorf("Payload mismatch: got %q, want %q", decodedPayload, payload)
	}
}

func TestEncodeDecodeJSON(t *testing.T) {
	mkdir := Mkdir{Path: "docs/subdir"}
	data, err := EncodeJSON(TypeMkdir, mkdir)
	if err != nil {
		t.Fatalf("EncodeJSON failed: %v", err)
	}

	header, err := DecodeHeader(bytes.NewReader(data[:HeaderSize]))
	if err != nil {
		t.Fatalf("DecodeHeader failed: %v", err)
	}

	if header.Type != TypeMkdir {
		t.Errorf("Type mismatch: got %d, want %d", header.Type, TypeMkdir)
	}

	payload, err := ReadPayload(bytes.NewReader(data[HeaderSize:]), header.Length)
	if err != nil {
		t.Fatalf("ReadPayload failed: %v", err)
	}

	var decoded Mkdir
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if decoded.Path != "docs/subdir" {
		t.Errorf("Path mismatch: got %q, want %q", decoded.Path, "docs/subdir")
	}
}

func TestAllMessageTypes(t *testing.T) {
	tests := []struct {
		name    string
		msgType uint8
		payload interface{}
	}{
		{"TaskInfo", TypeTaskInfo, TaskInfo{ID: "t1", TotalSize: 1024, ItemCount: 5}},
		{"FileHeader", TypeFileHeader, FileHeader{Path: "a/b.txt", Size: 512, Mode: 0644, Hash: "sha256:abc"}},
		{"Ack", TypeAck, Ack{Status: "ok", Code: 0, Msg: ""}},
		{"AckError", TypeAck, Ack{Status: "error", Code: 3, Msg: "disk full"}},
		{"Mkdir", TypeMkdir, Mkdir{Path: "deep/nested/dir"}},
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

			if header.Type != tt.msgType {
				t.Errorf("Type mismatch: got 0x%02X, want 0x%02X", header.Type, tt.msgType)
			}
		})
	}
}

func TestDoneMessage(t *testing.T) {
	// Done message has no payload
	data, err := EncodeMessage(TypeDone, nil)
	if err != nil {
		t.Fatalf("EncodeMessage for Done failed: %v", err)
	}

	if len(data) != HeaderSize {
		t.Errorf("Done message length: got %d, want %d", len(data), HeaderSize)
	}

	header, err := DecodeHeader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("DecodeHeader failed: %v", err)
	}

	if header.Type != TypeDone {
		t.Errorf("Type mismatch: got 0x%02X, want 0x%02X", header.Type, TypeDone)
	}
	if header.Length != 0 {
		t.Errorf("Done payload length: got %d, want 0", header.Length)
	}
}

func TestReadPayload_PartialRead(t *testing.T) {
	// Simulate TCP fragmentation: provide fewer bytes than header indicates
	shortData := []byte{0x01, 0x02, 0x03} // Only 3 bytes, header says 10
	_, err := ReadPayload(bytes.NewReader(shortData), 10)
	if err == nil {
		t.Fatal("expected error for short read, got nil")
	}
}
