package sdencoding

type Encoding interface {
	EncodeBytes(data []byte) ([]byte, error)
	EncodeStr(data []byte) string
	DecodeBytes(encoded []byte) ([]byte, error)
	DecodeStr(encoded string) ([]byte, error)
	MustDecodeBytes(encoded []byte) []byte
	MustDecodeStr(encoded string) []byte
}

type CharsetEncoding interface {
	EncodeBytes(s string) ([]byte, error)
	DecodeBytes(encoded []byte) (string, error)
	MustDecodeBytes(encoded []byte) string
}
