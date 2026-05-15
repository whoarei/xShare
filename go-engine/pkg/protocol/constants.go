package protocol

const (
	Magic   uint32 = 0x58534852
	Version uint8  = 0x01
)

const HeaderSize = 14

const (
	TypeTaskInfo   uint8 = 0x01
	TypeFileHeader uint8 = 0x02
	TypeChunk      uint8 = 0x03
	TypeAck        uint8 = 0x04
	TypeDone       uint8 = 0x05
)

var TypeNames = map[uint8]string{
	TypeTaskInfo:   "TaskInfo",
	TypeFileHeader: "FileHeader",
	TypeChunk:      "Chunk",
	TypeAck:        "Ack",
	TypeDone:       "Done",
}
