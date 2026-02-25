package codec

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
)

// Codec 定义序列化接口
type Codec interface {
	Marshal(any) ([]byte, error)
	Unmarshal([]byte, any) error
}

// JSONCodec 默认的 JSON 序列化实现
type JSONCodec struct{}

func (j JSONCodec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (j JSONCodec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

type GobCodec struct{}

func (g GobCodec) Marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (g GobCodec) Unmarshal(data []byte, v any) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(v)
}
