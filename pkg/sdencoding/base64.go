package sdencoding

import (
	"encoding/base64"

	"TMA/pkg/sderr"
)

type base64Encoding struct {
	encoding *base64.Encoding
}

var (
	Base64Std Encoding = base64Encoding{base64.StdEncoding}
	Base64Url Encoding = base64Encoding{base64.URLEncoding}
)

func (e base64Encoding) EncodeBytes(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}
	enc := e.encoding
	buff := make([]byte, enc.EncodedLen(len(data)))
	enc.Encode(buff, data)
	return buff, nil
}

func (e base64Encoding) EncodeStr(data []byte) string {
	return e.encoding.EncodeToString(data)
}

func (e base64Encoding) DecodeBytes(encoded []byte) ([]byte, error) {
	if len(encoded) == 0 {
		return []byte{}, nil
	}
	enc := e.encoding
	buff := make([]byte, enc.DecodedLen(len(encoded)))
	n, err := enc.Decode(buff, encoded)
	if err != nil {
		return nil, sderr.WithStack(err)
	}
	return buff[:n], nil
}

func (e base64Encoding) DecodeStr(encoded string) ([]byte, error) {
	buff, err := e.encoding.DecodeString(encoded)
	if err != nil {
		return nil, sderr.WithStack(err)
	}
	return buff, nil
}

func (e base64Encoding) MustDecodeBytes(encoded []byte) []byte {
	r, err := e.DecodeBytes(encoded)
	if err != nil {
		panic(sderr.WithStack(err))
	}
	return r
}

func (e base64Encoding) MustDecodeStr(encoded string) []byte {
	r, err := e.DecodeStr(encoded)
	if err != nil {
		panic(sderr.WithStack(err))
	}
	return r
}
