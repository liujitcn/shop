package leveldb

import "google.golang.org/protobuf/proto"

// EncodeMessage 将 proto 消息编码为字节切片。
func EncodeMessage(message proto.Message) ([]byte, error) {
	if message == nil {
		return nil, nil
	}
	return proto.Marshal(message)
}

// DecodeMessage 将字节切片解码到 proto 消息。
func DecodeMessage(rawValue []byte, message proto.Message) error {
	if len(rawValue) == 0 || message == nil {
		return nil
	}
	return proto.Unmarshal(rawValue, message)
}
