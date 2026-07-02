package sdencoding

import (
	"strings"
)

type Bytes []byte

func (b Bytes) ToArray() []byte {
	return []byte(b)
}

func (b Bytes) HexL() string {
	if len(b) <= 0 {
		return ""
	}
	return Hex.EncodeStr(b)
}

func (b Bytes) HexU() string {
	if len(b) <= 0 {
		return ""
	}
	return strings.ToUpper(Hex.EncodeStr(b))
}

func (b Bytes) Base64Std() string {
	if len(b) <= 0 {
		return ""
	}
	return Base64Std.EncodeStr(b)
}

func (b Bytes) Base64Url() string {
	if len(b) <= 0 {
		return ""
	}
	return Base64Url.EncodeStr(b)
}
