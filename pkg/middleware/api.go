package middleware

import (
	"TMA/pkg/common"
	"TMA/pkg/nucl"
	"TMA/pkg/sdlog"
	"TMA/pkg/web/webapi"
	"TMA/service"
	"TMA/service/db/model"
	"net"
	"net/http"
	"strings"

	"github.com/ip2location/ip2location-go"
	"github.com/labstack/echo/v4"
	initdata "github.com/telegram-mini-apps/init-data-golang"
)

var (
	IpLocationDB *ip2location.DB
)

// AppRequestContext Context封装，每次请求创建实例
type AppRequestContext struct {
	echo.Context
	// 全局资源
	Nu        *nucl.Nucleus
	AuthData  initdata.InitData `json:"auth_data"`
	UserAgent string            `json:"user_agent"`
}

var DontLoginUrls = map[string]bool{
	"/api/v1/wallet/captcha":           true,
	"/api/v1/wallet/login":             true,
	"/api/v1/auth/google/login":        true,
	"/api/v1/auth/apple/login":         true,
	"/api/v1/auth/sms/captcha":         true,
	"/api/v1/auth/sms/login":           true,
	"/api/v1/auth/password/login":      true,
	"/api/v1/ue/notify":                true,
	"/api/v1/ue/secret/open":           true,
	"/api/v1/ue/secret/close":          true,
	"/api/v1/ue/secret/status":         true,
	"/api/v1/spot/category":            true,
	"/api/v1/spot/list_v2":             true,
	"/api/v1/spot/list_v4":             true,
	"/api/v1/spot/list_pithy":          true,
	"/api/v1/product/list":             true,
	"/api/v1/complaint/type/list":      true,
	"/api/v1/email/captcha":            true,
	"/api/v1/email/login":              true,
	"/api/v2/auth/google/login":        true,
	"/api/v2/auth/apple/login":         true,
	"/api/v2/auth/password/login":      true,
	"/api/v2/email/login":              true,
	"/api/v1/token/refresh":            true,
	"/api/v1/ota/latest":               true,
	"/api/v1/spot/detail":              true,
	"/api/v1/spot/address":             true,
	"/api/v1/spot/list_grid":           true,
	"/api/v1/spot/list_by_distance":    true,
	"/api/v1/spot/list_by_distance_v2": true,
	"/api/v1/product/search":           true,
	"/api/v1/product/detail":           true,
	"/api/v1/product/tag/list":         true,
	"/api/v1/checkin/tags":             true,
	"/api/v1/checkin/list":             true,
	"/api/v1/checkin/detail":           true,
	"/api/v1/user/info/other":          true,
	"/api/v1/checkin/comment/list":     true,
	"/api/v1/spot/checkin/latest":      true,
	"/api/v1/spot/checkin/recommended": true,
	"/api/v1/user/checkin/list":        true,
}

// AppContext 通用参数中间件
func AppContext(nu *nucl.Nucleus) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ec echo.Context) error {
			requestContext := &AppRequestContext{
				Context:   ec,
				Nu:        nu,
				UserAgent: strings.ToLower(ec.Request().UserAgent()),
			}
			language := ec.Request().Header.Get("Language")
			common.SetRequestLanguage(language)

			// 判断是否属于无需登录的URL
			requestPath := ec.Request().URL.Path
			if DontLoginUrls[requestPath] && ec.Request().Header.Get("authorization") == "" {
				return next(requestContext)
			}
			if requestPath == "/api/v1/ota/latest" || requestPath == "/api/v1/token/refresh" {
				return next(requestContext)
			}
			// 检查 APP 版本
			appVersion := ec.Request().Header.Get("App-Version")
			platform := ec.Request().Header.Get("Platform")
			if appVersion != "" && platform != "" {
				// 查询最新的升级信息
				var ota model.Otum
				err := nu.DB.WithContext(ec.Request().Context()).
					Where("platform = ? AND is_active = ?", platform, true).
					Order("created_at DESC").
					First(&ota).Error
				if err == nil && ota.ForceUpdate == "force" {
					// 检查版本是否需要强制更新
					if service.NeedUpdate(appVersion, ota.MinVersion) {
						return webapi.Error(common.ErrVersionTooLow).Render(ec)
					}
				}
			}
			authParts := strings.Split(ec.Request().Header.Get("authorization"), " ")
			if len(authParts) != 2 {
				return webapi.Error(common.ErrUnauthorized).Render(ec)
			}
			authType := authParts[0]
			authData := authParts[1]
			switch authType {
			case "supermap":
				uInfo, err := webapi.ParseToken(nu.Config.WalletJwtSecret, authData)
				if err != nil {
					return webapi.Error(common.ErrAuthLogin).Render(ec)
				}
				requestContext.AuthData.User.ID = uInfo.UserID
			default:
				return webapi.Error(common.ErrUnauthorized).Render(ec)
			}
			return next(requestContext)
		}
	}
}

func InitBlockIPLimitDB(nu *nucl.Nucleus) {
	// 初始化 IP2Location 数据库
	var err error
	ipDB, err := ip2location.OpenDB(nu.Config.IpLocationDB)
	if err != nil {
		panic("无法打开 IP2Location 数据库: " + err.Error())
	}
	IpLocationDB = ipDB
}

func BlockIPFromCountryMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 获取客户端 IP 地址
			ipAddress := c.Request().Header.Get("X-Forwarded-For")
			if ipAddress == "" {
				ipAddress, _, _ = net.SplitHostPort(c.Request().RemoteAddr)
			}

			// 查询 IP 地址的地理位置信息
			result, err := IpLocationDB.Get_all(ipAddress)
			if err != nil {
				sdlog.Info("无法查询地理位置信息,", ipAddress)
				return webapi.Error(common.ErrAccessDenied).Render(c)
			}

			// 获取需要阻止的国家列表
			blockList := strings.Split(getIPBlockList(), ",")

			// 检查 IP 所属国家是否在阻止列表中
			countryCode := strings.ToLower(result.Country_short)
			for _, blockCountry := range blockList {
				if countryCode == strings.ToLower(strings.TrimSpace(blockCountry)) {
					return webapi.Error(common.ErrAccessDenied).Render(c)
				}
			}
			return next(c)
		}
	}
}

func getIPBlockList() string {
	return ""
}

type AppApiHandler func(ctx *AppRequestContext) error

// AppApiHandlerWrapper apiHandler包装
func AppApiHandlerWrapper(handler AppApiHandler) echo.HandlerFunc {
	return func(ec echo.Context) error {
		language := ec.Request().Header.Get("Language")
		common.SetRequestLanguage(language)

		// 检查类型转换
		ctx, ok := ec.(*AppRequestContext)
		if !ok {
			return echo.NewHTTPError(http.StatusInternalServerError, "invalid context type")
		}

		return handler(ctx)
	}
}
