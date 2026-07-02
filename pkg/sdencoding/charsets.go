package sdencoding

import (
	"TMA/pkg/sderr"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
)

var (
	Gbk     CharsetEncoding = charsetEncoding{simplifiedchinese.GBK}
	Gb2312  CharsetEncoding = charsetEncoding{simplifiedchinese.HZGB2312}
	Gb18030 CharsetEncoding = charsetEncoding{simplifiedchinese.GB18030}
)

type charsetEncoding struct {
	encoding encoding.Encoding
}

func (e charsetEncoding) EncodeBytes(s string) ([]byte, error) {
	b, err := e.encoding.NewEncoder().Bytes([]byte(s))
	if err != nil {
		return nil, sderr.WithStack(err)
	}
	return b, nil
}

func (e charsetEncoding) DecodeBytes(encoded []byte) (string, error) {
	if encoded == nil {
		return "", sderr.New("encoded is nil")
	}
	b, err := e.encoding.NewDecoder().Bytes(encoded)
	if err != nil {
		return "", sderr.WithStack(err)
	}
	return string(b), nil
}

func (e charsetEncoding) MustDecodeBytes(encoded []byte) string {
	r, err := e.DecodeBytes(encoded)
	if err != nil {
		panic(sderr.WithStack(err))
	}
	return r
}
