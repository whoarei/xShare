// xShare协议常量定义
package protocol

const (
	Magic   uint32 = 0x58534852 // 协议魔数 "xSHR"
	Version uint8  = 0x01       // 协议版本
)

const HeaderSize = 14 // 协议头固定长度

// 消息类型常量
const (
	TypeTaskInfo   uint8 = 0x01 // 任务信息
	TypeFileHeader uint8 = 0x02 // 文件头
	TypeChunk      uint8 = 0x03 // 数据块
	TypeAck        uint8 = 0x04 // 确认消息
	TypeDone       uint8 = 0x05 // 传输完成
)

// TypeNames 消息类型名称映射
var TypeNames = map[uint8]string{
	TypeTaskInfo:   "TaskInfo",
	TypeFileHeader: "FileHeader",
	TypeChunk:      "Chunk",
	TypeAck:        "Ack",
	TypeDone:       "Done",
}
