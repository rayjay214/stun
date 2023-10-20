package protocol

import (
	"strconv"
)

type Body_0x91 struct {
	UpgradeFlag byte
	UpgradeType byte
	TaskId      uint32
	SrcVersion  string
	DstVersion  string
	UrlLen      byte
	Url         string
	Url2Len     byte
	Url2        string
	Md5         string
	Md52        string
}

type Body_0x11 struct {
	DevType     string
	Sn          uint64
	CurrVersion string
	Lbsinfo     []byte
}

type Body_0x12 struct {
	TaskId      uint32
	UpgradeType byte
	Reason      byte
	CurrVersion string
}

type Body_0x92 struct {
	TaskId uint32
}

type Body_0x13 struct {
	Key  string
	Ip   string
	Port uint32
}

type Body_0x14 struct {
	PeerKey  string
	LocalKey string
}

type Body_0x94 struct {
	Key  string
	Ip   string
	Port uint32
}

func (entity *Body_0x11) Encode() ([]byte, error) {
	writer := NewWriter()

	writer.WriteString(entity.DevType, 10)

	writer.Write(stringToBCD(strconv.FormatUint(entity.Sn, 10), 6))

	writer.WriteString(entity.CurrVersion, 20)

	writer.Write(entity.Lbsinfo, 10)

	return writer.Bytes(), nil
}

func (entity *Body_0x11) Decode(data []byte) (int, error) {
	reader := NewReader(data)

	var err error
	entity.DevType, err = reader.ReadString(10)
	if err != nil {
		return 0, err
	}

	temp, err := reader.Read(6)
	if err != nil {
		return 0, ErrInvalidHeader
	}
	bcdStr := bcdToString(temp)
	if bcdStr == "" {
		entity.Sn = 0
	} else {
		entity.Sn, err = strconv.ParseUint(bcdToString(temp), 10, 64)
		if err != nil {
			return 0, err
		}
	}

	entity.CurrVersion, err = reader.ReadString(20)
	if err != nil {
		return 0, err
	}

	entity.Lbsinfo, err = reader.Read(10)
	if err != nil {
		return 0, err
	}

	return len(data) - reader.Len(), nil
}

func (entity *Body_0x91) MsgID() MsgID {
	return Msg_0x91
}

func (entity *Body_0x12) Encode() ([]byte, error) {
	writer := NewWriter()

	writer.WriteUint32(entity.TaskId)

	writer.WriteByte(entity.UpgradeType)

	writer.WriteByte(entity.Reason)

	writer.WriteString(entity.CurrVersion, 20)

	return writer.Bytes(), nil
}

func (entity *Body_0x12) Decode(data []byte) (int, error) {
	reader := NewReader(data)

	var err error
	entity.TaskId, err = reader.ReadUint32()
	if err != nil {
		return 0, err
	}

	entity.UpgradeType, err = reader.ReadByte()
	if err != nil {
		return 0, err
	}

	entity.Reason, err = reader.ReadByte()
	if err != nil {
		return 0, err
	}

	entity.CurrVersion, err = reader.ReadString(20)
	if err != nil {
		return 0, err
	}

	return len(data) - reader.Len(), nil
}

func (entity *Body_0x92) Encode() ([]byte, error) {
	writer := NewWriter()

	writer.WriteUint32(entity.TaskId)

	return writer.Bytes(), nil
}

func (entity *Body_0x92) Decode(data []byte) (int, error) {
	reader := NewReader(data)

	var err error
	entity.TaskId, err = reader.ReadUint32()
	if err != nil {
		return 0, err
	}

	return len(data) - reader.Len(), nil
}

func (entity *Body_0x12) MsgID() MsgID {
	return Msg_0x12
}

func (entity *Body_0x11) MsgID() MsgID {
	return Msg_0x11
}

func (entity *Body_0x92) MsgID() MsgID {
	return Msg_0x92
}

func (entity *Body_0x91) Encode() ([]byte, error) {
	writer := NewWriter()

	writer.WriteByte(entity.UpgradeFlag)

	writer.WriteByte(entity.UpgradeType)

	writer.WriteUint32(entity.TaskId)

	writer.WriteString(entity.SrcVersion, 20)

	writer.WriteString(entity.DstVersion, 20)

	writer.WriteByte(entity.UrlLen)

	writer.WriteString(entity.Url)

	if entity.UpgradeType == 2 {

		writer.WriteByte(entity.Url2Len)

		writer.WriteString(entity.Url2)
	}

	writer.WriteString(entity.Md5, 32)

	if entity.UpgradeType == 2 {
		writer.WriteString(entity.Md52, 32)
	}

	return writer.Bytes(), nil
}

func (entity *Body_0x91) Decode(data []byte) (int, error) {
	reader := NewReader(data)

	var err error
	entity.UpgradeFlag, err = reader.ReadByte()
	if err != nil {
		return 0, err
	}

	entity.UpgradeType, err = reader.ReadByte()
	if err != nil {
		return 0, err
	}

	entity.TaskId, err = reader.ReadUint32()
	if err != nil {
		return 0, err
	}

	entity.SrcVersion, err = reader.ReadString(20)
	if err != nil {
		return 0, err
	}

	entity.DstVersion, err = reader.ReadString(20)
	if err != nil {
		return 0, err
	}

	entity.UrlLen, err = reader.ReadByte()
	if err != nil {
		return 0, err
	}

	entity.Url, err = reader.ReadString(int(entity.UrlLen))
	if err != nil {
		return 0, err
	}

	if entity.UpgradeType == 2 {
		entity.Url2Len, err = reader.ReadByte()
		if err != nil {
			return 0, err
		}

		entity.Url2, err = reader.ReadString(int(entity.Url2Len))
		if err != nil {
			return 0, err
		}
	}

	entity.Md5, err = reader.ReadString(32)
	if err != nil {
		return 0, err
	}

	if entity.UpgradeType == 2 {

		entity.Md52, err = reader.ReadString(32)
		if err != nil {
			return 0, err
		}
	}

	return len(data) - reader.Len(), nil
}

func (entity *Body_0x13) MsgID() MsgID {
	return Msg_0x13
}

func (entity *Body_0x14) MsgID() MsgID {
	return Msg_0x14
}

func (entity *Body_0x94) MsgID() MsgID {
	return Msg_0x94
}

func (entity *Body_0x13) Encode() ([]byte, error) {
	writer := NewWriter()

	writer.WriteString(entity.Key, 10)

	writer.WriteString(entity.Ip, 20)

	writer.WriteUint32(entity.Port)

	return writer.Bytes(), nil
}

func (entity *Body_0x13) Decode(data []byte) (int, error) {
	reader := NewReader(data)

	var err error
	entity.Key, err = reader.ReadString(10)
	if err != nil {
		return 0, err
	}

	entity.Ip, err = reader.ReadString(20)
	if err != nil {
		return 0, err
	}

	entity.Port, err = reader.ReadUint32()
	if err != nil {
		return 0, err
	}

	return len(data) - reader.Len(), nil
}

func (entity *Body_0x14) Encode() ([]byte, error) {
	writer := NewWriter()

	writer.WriteString(entity.PeerKey, 10)

	writer.WriteString(entity.LocalKey, 10)

	return writer.Bytes(), nil
}

func (entity *Body_0x14) Decode(data []byte) (int, error) {
	reader := NewReader(data)

	var err error
	entity.PeerKey, err = reader.ReadString(10)
	if err != nil {
		return 0, err
	}

	entity.LocalKey, err = reader.ReadString(10)
	if err != nil {
		return 0, err
	}

	return len(data) - reader.Len(), nil
}

func (entity *Body_0x94) Encode() ([]byte, error) {
	writer := NewWriter()

	writer.WriteString(entity.Key, 10)

	writer.WriteString(entity.Ip, 20)

	writer.WriteUint32(entity.Port)

	return writer.Bytes(), nil
}

func (entity *Body_0x94) Decode(data []byte) (int, error) {
	reader := NewReader(data)

	var err error
	entity.Key, err = reader.ReadString(10)
	if err != nil {
		return 0, err
	}

	entity.Ip, err = reader.ReadString(20)
	if err != nil {
		return 0, err
	}

	entity.Port, err = reader.ReadUint32()
	if err != nil {
		return 0, err
	}

	return len(data) - reader.Len(), nil
}
