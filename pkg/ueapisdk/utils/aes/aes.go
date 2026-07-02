package aes

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

const CHAR_ENCODING = "UTF-8"

func Encrypt(data []byte, key []byte, iv []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 32 {
		return nil, errors.New("invalid AES key length")
	}
	if len(iv) != 16 {
		return nil, errors.New("invalid AES iv length")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	data = pkcs7Padding(data, block.BlockSize()) // 改为小写调用
	mode := cipher.NewCBCEncrypter(block, iv)
	ciphertext := make([]byte, len(data))
	mode.CryptBlocks(ciphertext, data)
	return ciphertext, nil
}

func Decrypt(data []byte, key []byte, iv []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 32 {
		return nil, errors.New("invalid AES key length")
	}
	if len(iv) != 16 {
		return nil, errors.New("invalid AES iv length")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(data))
	mode.CryptBlocks(plaintext, data)
	return pkcs7UnPadding(plaintext) // 改为小写调用
}

// Base64版本方法
func EncryptToBase64(data string, key string, iv string) (string, error) {
	encrypted, err := Encrypt([]byte(data), []byte(key), []byte(iv))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func DecryptFromBase64(data string, key string, iv string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}

	decrypted, err := Decrypt(decoded, []byte(key), []byte(iv))
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
}

// 填充方法
// 修改函数名为小写开头
func pkcs7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
}

func pkcs7UnPadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("empty data")
	}

	padding := int(data[length-1])
	if padding < 1 || padding > aes.BlockSize {
		return nil, errors.New("invalid padding")
	}

	return data[:length-padding], nil
}

// 生成随机密钥
func GenerateRandomKey() ([]byte, error) {
	key := make([]byte, 32) // 默认生成256位密钥
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

func GenerateRandomKeyWithBase64() (string, error) {
	key, err := GenerateRandomKey()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}
