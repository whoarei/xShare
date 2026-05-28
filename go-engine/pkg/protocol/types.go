// xShare协议类型定义
package protocol

// Header 协议头，14字节固定长度
type Header struct {
	Magic   uint32 // 魔数 0x58534852
	Version uint8  // 协议版本
	Type    uint8  // 消息类型
	Length  uint64 // 载荷长度
}

// TaskInfo 传输任务信息
type TaskInfo struct {
	ID        string `json:"id"`         // 任务ID
	TotalSize int64  `json:"total_size"` // 总字节数
	ItemCount int    `json:"item_count"` // 文件数量
}

// FileHeader 文件元数据头
type FileHeader struct {
	Path string `json:"path"` // 相对路径
	Size int64  `json:"size"` // 文件大小
	Mode int    `json:"mode"` // 文件权限
	Hash string `json:"hash"` // SHA256校验和
}

// Ack 确认消息
type Ack struct {
	Status string `json:"status"` // ok/error
	Code   int    `json:"code"`   // 错误码
	Msg    string `json:"msg"`    // 描述信息
}
