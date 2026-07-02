package main

import (
	"TMA/pkg/cache/redis"
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/sdrand"
	"TMA/pkg/web/webapi"
	"TMA/service"
	"TMA/service/db/model"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	solCommon "github.com/blocto/solana-go-sdk/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58"
	"gorm.io/gorm"
)

// 以太坊地址验证
func isValidEthereumAddress(address string) bool {
	re := regexp.MustCompile("^0x[a-fA-F0-9]{40}$")
	return re.MatchString(address)
}

// isValidSolanaAddress 验证 Solana 地址格式是否正确
func isValidSolanaAddress(address string) bool {
	pubKey := solCommon.PublicKeyFromString(address)
	return pubKey != (solCommon.PublicKey{}) // 确保地址是有效的 32 字节公钥
}

type CaptchaParam struct {
	Address  string `json:"address"`
	Currency string `json:"currency"`
}

func captcha(ec *middleware.AppRequestContext) error {
	req := CaptchaParam{}
	err := ec.Bind(&req)
	if err != nil || req.Address == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	switch req.Currency {
	case "eth":
		if !isValidEthereumAddress(req.Address) {
			return webapi.Error(common.ErrWalletAddress).Render(ec)
		}
	default:
		if !isValidSolanaAddress(req.Address) {
			return webapi.Error(common.ErrWalletAddress).Render(ec)
		}

	}
	code := sdrand.String(20, sdrand.LowerLetterNumbers)
	lockKey := common.Key(common.PrefixWalletAddressCode, req.Address)
	result, err := ec.Nu.RedisClient.SetNX(ec.Request().Context(), lockKey, code, 30*time.Second).Result()
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	if !result {
		var errGetCode error
		code, errGetCode = ec.Nu.RedisClient.Get(ec.Request().Context(), lockKey).Result()
		if errGetCode != nil {
			return webapi.Error(common.ErrService).Render(ec)
		}
	}
	code = "You are signing this message to sign in to uscene.\nThis is not a transaction and will not be broadcast or cost/use your fund. :)\n\nNonce: " + code
	return webapi.OK(code).Render(ec)
}

type LoginParam struct {
	InviteCode string `json:"invite_code"`
	Address    string `json:"address"`
	Captcha    string `json:"captcha"`
	Sign       string `json:"sign"`
	Currency   string `json:"currency"`
}

func login(ec *middleware.AppRequestContext) error {
	req := LoginParam{}
	err := ec.Bind(&req)
	if err != nil || req.Address == "" || req.Sign == "" || req.Captcha == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	if req.Currency != "sol" && req.Currency != "eth" {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	if req.Currency == "sol" {
		if !isValidSolanaAddress(req.Address) {
			return webapi.Error(common.ErrWalletAddress).Render(ec)
		}
	}
	if req.Currency == "eth" {
		if !isValidEthereumAddress(req.Address) {
			return webapi.Error(common.ErrWalletAddress).Render(ec)
		}
	}
	lockKey := common.Key(common.PrefixWalletAddressCode, req.Address)
	message, err := ec.Nu.RedisClient.Get(ec.Request().Context(), lockKey).Result()
	if err != nil {
		return webapi.Error(common.ErrRequestExpire).Render(ec)
	}
	if req.Captcha != message {
		return webapi.Error(common.ErrCaptcha).Render(ec)
	}
	//检查签名
	res := false
	if req.Currency == "sol" {
		res, err = verifySignSol(message, req.Sign, req.Address)
	}
	if req.Currency == "eth" {
		res, err = verifySignEth(message, req.Sign, req.Address)
	}
	if err != nil {
		return webapi.Error(err).Render(ec)
	}
	if !res {
		return webapi.Error(common.ErrSign).Render(ec)
	}
	ok := redis.AcquireUserBalanceLock(ec.Request().Context(), ec.Nu, common.ConfigProviderByWallet+req.Address)
	if !ok {
		return webapi.Error(common.ErrRequestTooFast).Render(ec)
	}
	//是否为新用户，不存在则写入
	uLoginInfo := model.UserAuth{}
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.UserAuth{}).Where("wallet_address = ?", req.Address).First(&uLoginInfo).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			newUid := ec.Nu.Snowflake.GenerateID()
			username := req.Address[:6] + "..." + req.Address[len(req.Address)-6:]
			errCreated := service.AddUser(ec.Request().Context(), ec.Nu, newUid, common.ConfigProviderByWallet, username, "", req.InviteCode, req.Address, "")
			if errCreated != nil {
				return webapi.Error(common.ErrService).Render(ec)
			}
			err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.UserAuth{}).Where("user_id = ? and provider = ?", newUid, common.ConfigProviderByWallet).First(&uLoginInfo).Error
			if err != nil {
				return webapi.Error(common.ErrService).Render(ec)
			}
		}
	}
	token, err := webapi.GenerateTokenString([]byte(ec.Nu.Config.WalletJwtSecret), uLoginInfo.UserID, uLoginInfo.WalletAddress, webapi.TokenTypeRefresh, webapi.RefreshTokenDuration)
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(token).Render(ec)

}

func verifySignSol(message, signatureBase64, address string) (bool, error) {
	var signature []byte
	var err error
	signature, err = hex.DecodeString(signatureBase64)
	if err != nil {
		return false, errors.New("签名解码失败")
	}

	// 2. 解析 Base58 公钥
	pubKey, err := base58.Decode(address)
	if err != nil || len(pubKey) != ed25519.PublicKeySize {
		return false, errors.New("无效的 Solana 地址")
	}

	// 3. 确保签名长度为 64 字节
	if len(signature) != ed25519.SignatureSize {
		return false, errors.New("无效的签名长度")
	}
	// 4. 添加 Solana 签名前缀
	//prefix := "\x16Solana Signed Message:\n"
	//messageBytes := append([]byte(prefix), []byte(message)...)

	// 5. 验证签名
	//if !ed25519.Verify(pubKey, messageBytes, signature) {
	if !ed25519.Verify(pubKey, []byte(message), signature) {
		return false, errors.New("签名错误")
	}

	return true, nil
}

func verifySignEth(message, signatureHex, address string) (bool, error) {
	// 计算消息哈希（EIP-191 标准）
	prefix := "\x19Ethereum Signed Message:\n" + fmt.Sprint(len(message)) + message
	messageHash := crypto.Keccak256([]byte(prefix))

	// 签名（去掉 "0x" 前缀）
	signatureHex = strings.TrimPrefix(signatureHex, "0x")
	signature, err := hex.DecodeString(signatureHex)
	if err != nil {

		return false, err
	}

	// 修正 `v` 值
	if len(signature) == 65 {
		v := signature[64]
		if v >= 27 {
			signature[64] = v - 27 // 让 `v` 变成 0 或 1
		}
	} else {
		return false, errors.New("签名长度错误，必须是 65 字节")
	}

	// 通过签名恢复公钥
	publicKey, err := crypto.SigToPub(messageHash, signature)
	if err != nil {
		return false, err
	}

	// 计算以太坊地址
	recoveredAddress := crypto.PubkeyToAddress(*publicKey).Hex()
	if recoveredAddress != address {
		return false, errors.New("签名错误")
	}

	return true, nil
}
