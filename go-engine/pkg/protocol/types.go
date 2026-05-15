package protocol

type Header struct {
	Magic   uint32
	Version uint8
	Type    uint8
	Length  uint64
}

type TaskInfo struct {
	ID        string `json:"id"`
	TotalSize int64  `json:"total_size"`
	ItemCount int    `json:"item_count"`
}

type FileHeader struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
	Mode int    `json:"mode"`
	Hash string `json:"hash"`
}

type Ack struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
}
