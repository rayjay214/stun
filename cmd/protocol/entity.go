package protocol

// 消息体
type Entity interface {
	MsgID() MsgID
	Encode() ([]byte, error)
	Decode([]byte) (int, error)
}
