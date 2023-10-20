package protocol

// 消息ID枚举
type MsgID uint8

const PrefixID = byte(0x68)

const (
	// 终端向服务器查询是否有升级
	Msg_0x11 MsgID = 0x11
	// 服务器向终端返回是否有升级
	Msg_0x91 MsgID = 0x91
	// 终端向服务器上报升级结果
	Msg_0x12 MsgID = 0x12
	// 升级结果应答
	Msg_0x92 MsgID = 0x92
	// 上传本机IP地址
	Msg_0x13 MsgID = 0x13
	// 请求PeerIP地址
	Msg_0x14 MsgID = 0x14
	// 返回PeerIP地址
	Msg_0x94 MsgID = 0x94
)

// 消息实体映射
var entityMapper = map[uint8]func() Entity{
	uint8(Msg_0x11): func() Entity {
		return new(Body_0x11)
	},
	uint8(Msg_0x91): func() Entity {
		return new(Body_0x91)
	},
	uint8(Msg_0x12): func() Entity {
		return new(Body_0x12)
	},
	uint8(Msg_0x92): func() Entity {
		return new(Body_0x92)
	},
	uint8(Msg_0x13): func() Entity {
		return new(Body_0x13)
	},
	uint8(Msg_0x14): func() Entity {
		return new(Body_0x14)
	},
	uint8(Msg_0x94): func() Entity {
		return new(Body_0x94)
	},
}
