package sdencoding

import (
	"encoding/hex"

	"TMA/pkg/sderr"
)

type hexEncoding struct{}

var (
	Hex Encoding = hexEncoding{}
)

func (e hexEncoding) EncodeBytes(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}
	buff := make([]byte, hex.EncodedLen(len(data)))
	hex.Encode(buff, data)
	return buff, nil
}

func (e hexEncoding) EncodeStr(data []byte) string {
	return hex.EncodeToString(data)
}

func (e hexEncoding) DecodeBytes(encoded []byte) ([]byte, error) {
	if len(encoded) == 0 {
		return []byte{}, nil
	}
	buff := make([]byte, hex.DecodedLen(len(encoded)))
	n, err := hex.Decode(buff, encoded)
	if err != nil {
		return nil, sderr.WithStack(err)
	}
	return buff[:n], nil
}

func (e hexEncoding) DecodeStr(encoded string) ([]byte, error) {
	buff, err := hex.DecodeString(encoded)
	if err != nil {
		return nil, sderr.WithStack(err)
	}
	return buff, nil
}

func (e hexEncoding) MustDecodeBytes(encoded []byte) []byte {
	r, err := e.DecodeBytes(encoded)
	if err != nil {
		panic(sderr.WithStack(err))
	}
	return r
}

func (e hexEncoding) MustDecodeStr(encoded string) []byte {
	r, err := e.DecodeStr(encoded)
	if err != nil {
		panic(sderr.WithStack(err))
	}
	return r
}
