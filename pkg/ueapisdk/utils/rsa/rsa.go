package rsa

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"github.com/sulink/ueapisdk/utils/constants"
)

const KeySize = 2048

// 生成RSA密钥对
// func GenerateKeyPair() (map[string]string, error) {
// 	privateKey, err := rsa.GenerateKey(rand.Reader, KeySize)
// 	if err != nil {
// 		return nil, err
// 	}

// 	publicKey := &privateKey.PublicKey

// 	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
// 	pubBytes, err := x509.MarshalPKIXPublicKey(publicKey)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return map[string]string{
// 		constants.RSA_PUBLICKEY:  base64.StdEncoding.EncodeToString(pubBytes),
// 		constants.RSA_PRIVATEKEY: base64.StdEncoding.EncodeToString(privBytes),
// 	}, nil
// }

func GenerateKeyPair() (map[string]string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, KeySize)
	if err != nil {
		return nil, err
	}

	// 生成 PEM 格式的私钥
	privPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}))

	// 生成 PEM 格式的公钥
	pubKey := &privateKey.PublicKey
	pubBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, err
	}
	pubPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}))

	return map[string]string{
		constants.RSA_PUBLICKEY:  pubPEM,
		constants.RSA_PRIVATEKEY: privPEM,
	}, nil
}

// 公钥加密
func Encrypt(plaintext string, publicKey string) (string, error) {
	block, _ := pem.Decode([]byte(publicKey))
	if block == nil {
		return "", errors.New("invalid public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", err
	}

	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, pub.(*rsa.PublicKey), []byte(plaintext))
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// 私钥解密
func Decrypt(ciphertext string, privateKey string) (string, error) {
	block, _ := pem.Decode([]byte(privateKey))
	if block == nil {
		return "", errors.New("invalid private key")
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	plaintext, err := rsa.DecryptPKCS1v15(rand.Reader, priv, decoded)
	return string(plaintext), err
}

// 签名验证功能实现
func Sign(content string, privateKey string) (string, error) {
	block, _ := pem.Decode([]byte(privateKey))
	if block == nil {
		return "", errors.New("invalid private key")
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	hashed := sha256.Sum256([]byte(content))
	signature, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, hashed[:])
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}

func CheckSign(content string, sign string, publicKey string) bool {
	block, _ := pem.Decode([]byte(publicKey))
	if block == nil {
		return false
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return false
	}

	decodedSign, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return false
	}

	hashed := sha256.Sum256([]byte(content))
	err = rsa.VerifyPKCS1v15(pub.(*rsa.PublicKey), crypto.SHA256, hashed[:], decodedSign)
	return err == nil
}

// Base64转PEM格式
func Base64ToPEM(base64Str string, isPublicKey bool) string {
	keyType := "PUBLIC KEY"
	if !isPublicKey {
		keyType = "PRIVATE KEY"
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("-----BEGIN %s-----\n", keyType))

	// 每64字符插入换行
	for i := 0; i < len(base64Str); i += 64 {
		end := i + 64
		if end > len(base64Str) {
			end = len(base64Str)
		}
		builder.WriteString(base64Str[i:end] + "\n")
	}

	builder.WriteString(fmt.Sprintf("-----END %s-----", keyType))
	return builder.String()
}

// PEM格式转Base64
func PEMToBase64(pemKey string) string {
	lines := strings.Split(pemKey, "\n")
	var base64Lines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 过滤头尾标识行和空行
		if !strings.HasPrefix(trimmed, "-----") && trimmed != "" {
			base64Lines = append(base64Lines, trimmed)
		}
	}

	return strings.Join(base64Lines, "")
}
