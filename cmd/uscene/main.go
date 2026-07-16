package main

import (
	apiMiddleware "TMA/pkg/middleware"
	"TMA/pkg/nucl"
	"TMA/pkg/process"
	"TMA/pkg/sderr"
	"TMA/pkg/sdlog"
	"TMA/pkg/sdrand"
	"TMA/pkg/web"
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/urfave/cli/v2"
)

func main() {
	sdrand.InitSeed()

	app := &cli.App{
		Name:  "supermap",
		Usage: "APP服务器",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   os.Getenv("TMA_CONFIG"),
				Usage:   "配置文件路径",
			},
			&cli.IntFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Value:   8050, // 默认端口
				Usage:   "服务器端口",
			},
		},
		Action: main0,
	}
	err := app.Run(os.Args)
	if err != nil {
		sdlog.Fatal(err)
	}
}

func main0(cc *cli.Context) error {
	time.Local = time.UTC
	configLoc, port := cc.String("config"), cc.Int("port")
	nu, err := nucl.InitGlobalNucleusByConfigFile(configLoc, nucl.Options{
		DB:          true,
		Redis:       true,
		Aws:         true,
		Snowflake:   true,
		GormLogger:  "supermap_api",
		ServiceName: "supermap_api",
		UE:          true,
		Email:       true,
		Believe:     true,
	})
	if err != nil {
		return sderr.WithStack(err)
	}

	// 初始化内存
	InitMemoryCache(nu)

	app := web.NewEcho(web.EchoOptions{})

	// 启动静态文件服务器，指向 "static" 目录
	app.Static("/static", "static")

	go func() {
		process.GracefullyExit(func() {
			err = app.Shutdown(context.Background())
			if err != nil {
				sdlog.WithError(err).Error("shut down failed")
			}

			nu.Close()
		})
	}()

	route(nu, app)

	listenAddr := fmt.Sprintf(":%d", port)
	sdlog.Infof("server listen on %s", listenAddr)
	err = app.Start(listenAddr)
	if err != nil && err != http.ErrServerClosed {
		return sderr.WithStack(err)
	}
	return nil
}

func route(nu *nucl.Nucleus, app *echo.Echo) {
	// 设置 CORS 中间件
	app.Use(middleware.CORS())
	app.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			echo.GET,
			echo.POST,
			echo.PUT,
			echo.DELETE,
			echo.OPTIONS,
		},
		AllowHeaders: []string{
			echo.HeaderContentType,
			echo.HeaderAuthorization,
			"X-Requested-With",
		},
	}))
	publicGroup := app.Group("/api/v1")
	publicGroup.Use(apiMiddleware.AppContext(nu))
	publicGroup.GET("/banner", apiMiddleware.AppApiHandlerWrapper(banner))
	publicGroup.POST("/wallet/captcha", apiMiddleware.AppApiHandlerWrapper(captcha))
	publicGroup.POST("/wallet/login", apiMiddleware.AppApiHandlerWrapper(login))
	publicGroup.GET("/spot/list", apiMiddleware.AppApiHandlerWrapper(spotList))
	publicGroup.GET("/spot/list_v2", apiMiddleware.AppApiHandlerWrapper(spotListV2))
	// publicGroup.GET("/spot/list_v4", apiMiddleware.AppApiHandlerWrapper(spotListV4))
	publicGroup.GET("/spot/list_pithy", apiMiddleware.AppApiHandlerWrapper(spotListPithy))
	publicGroup.GET("/spot/list_grid", apiMiddleware.AppApiHandlerWrapper(spotListGrid))
	publicGroup.POST("/spot/check_in", apiMiddleware.AppApiHandlerWrapper(spotCheckIn))
	publicGroup.GET("/spot/category", apiMiddleware.AppApiHandlerWrapper(spotCategory))
	publicGroup.POST("/spot/create", apiMiddleware.AppApiHandlerWrapper(spotCreate))
	publicGroup.POST("/spot/update", apiMiddleware.AppApiHandlerWrapper(spotUpdate))
	publicGroup.GET("/spot/detail", apiMiddleware.AppApiHandlerWrapper(spotDetail))
	publicGroup.GET("/spot/address", apiMiddleware.AppApiHandlerWrapper(spotAddress))
	publicGroup.GET("/spot/list_by_distance", apiMiddleware.AppApiHandlerWrapper(spotListByDistance))
	publicGroup.GET("/spot/list_by_distance_v2", apiMiddleware.AppApiHandlerWrapper(spotListByDistanceV2))

	// user
	publicGroup.GET("/user/spot", apiMiddleware.AppApiHandlerWrapper(mySpotList))
	publicGroup.POST("/user/latLng", apiMiddleware.AppApiHandlerWrapper(updateUserLatLng))
	publicGroup.GET("/user/info", apiMiddleware.AppApiHandlerWrapper(userInfo))
	publicGroup.POST("/user/upgrade", apiMiddleware.AppApiHandlerWrapper(userUpgrade))
	publicGroup.GET("/user/invite", apiMiddleware.AppApiHandlerWrapper(userInvite))
	// publicGroup.POST("/user/move", apiMiddleware.AppApiHandlerWrapper(move))
	// publicGroup.GET("/user/income", apiMiddleware.AppApiHandlerWrapper(userIncome))
	// user transfer
	publicGroup.POST("/user/search", apiMiddleware.AppApiHandlerWrapper(userSearch))
	publicGroup.POST("/user/transfer", apiMiddleware.AppApiHandlerWrapper(userTransfer))
	publicGroup.GET("/user/transfer/record", apiMiddleware.AppApiHandlerWrapper(userTransferRecord))

	// config
	publicGroup.GET("/level/config", apiMiddleware.AppApiHandlerWrapper(userLevelInfo))

	// 在 route 函数中添加
	publicGroup.POST("/auth/google/login", apiMiddleware.AppApiHandlerWrapper(googleLogin))
	publicGroup.POST("/auth/apple/login", apiMiddleware.AppApiHandlerWrapper(appleLogin))
	publicGroup.POST("/auth/sms/captcha", apiMiddleware.AppApiHandlerWrapper(smsCaptcha))
	publicGroup.POST("/auth/sms/login", apiMiddleware.AppApiHandlerWrapper(smsLogin))
	publicGroup.POST("/user/set_password", apiMiddleware.AppApiHandlerWrapper(setPassword))
	publicGroup.POST("/auth/password/login", apiMiddleware.AppApiHandlerWrapper(passwordLogin))
	publicGroup.POST("/user/upload_image", apiMiddleware.AppApiHandlerWrapper(uploadImage))
	publicGroup.POST("/user/profile", apiMiddleware.AppApiHandlerWrapper(updateUserProfile))

	publicGroup.POST("/user/description", apiMiddleware.AppApiHandlerWrapper(updateUserDescription))

	publicGroup.POST("/ue/notify", apiMiddleware.AppApiHandlerWrapper(notify))
	publicGroup.POST("/ue/secret/open", apiMiddleware.AppApiHandlerWrapper(openSecret))
	publicGroup.POST("/ue/secret/close", apiMiddleware.AppApiHandlerWrapper(closeSecret))
	// 免密状态查询兼容客户端历史接口：统一复用同一个查询处理器。
	publicGroup.GET("/ue/secret/status", apiMiddleware.AppApiHandlerWrapper(getSecretStatus))
	publicGroup.GET("/ue/bind/status", apiMiddleware.AppApiHandlerWrapper(getSecretStatus))
	publicGroup.POST("/ue/secret/status", apiMiddleware.AppApiHandlerWrapper(getSecretStatus))
	publicGroup.POST("/order/create", apiMiddleware.AppApiHandlerWrapper(createOrder))
	publicGroup.GET("/order/pay_info", apiMiddleware.AppApiHandlerWrapper(getPayInfo))
	publicGroup.GET("/order/list", apiMiddleware.AppApiHandlerWrapper(getOrderList))
	publicGroup.GET("/product/list", apiMiddleware.AppApiHandlerWrapper(getProductList))

	// product create
	publicGroup.POST("/product/create", apiMiddleware.AppApiHandlerWrapper(productCreate))
	publicGroup.POST("/product/update", apiMiddleware.AppApiHandlerWrapper(productUpdate))
	// product search
	publicGroup.GET("/product/search", apiMiddleware.AppApiHandlerWrapper(getProductListBySearch))
	// product detail
	publicGroup.GET("/product/detail", apiMiddleware.AppApiHandlerWrapper(getProductDetail))
	publicGroup.POST("/product/favorite/add", apiMiddleware.AppApiHandlerWrapper(productFavoriteAdd))
	publicGroup.POST("/product/favorite/delete", apiMiddleware.AppApiHandlerWrapper(productFavoriteDelete))

	// 上架 下架商品
	publicGroup.POST("/product/publish", apiMiddleware.AppApiHandlerWrapper(productPublish))
	publicGroup.POST("/product/unpublish", apiMiddleware.AppApiHandlerWrapper(productUnpublish))

	publicGroup.GET("/user/product/favorites", apiMiddleware.AppApiHandlerWrapper(getUserProductFavorites))
	// user product
	publicGroup.GET("/user/product", apiMiddleware.AppApiHandlerWrapper(getUserProduct))
	publicGroup.GET("/user/product/visit_history", apiMiddleware.AppApiHandlerWrapper(getUserProductVisitHistory))

	//我买到的
	publicGroup.GET("/user/product/bought", apiMiddleware.AppApiHandlerWrapper(getUserProductBought))
	// 我卖出的
	publicGroup.GET("/user/product/sold", apiMiddleware.AppApiHandlerWrapper(getUserProductSold))

	publicGroup.GET("/user/statistics/data", apiMiddleware.AppApiHandlerWrapper(getUserStatisticsData))
	// product list tag config
	publicGroup.GET("/product/tag/list", apiMiddleware.AppApiHandlerWrapper(getProductListTagConfig))

	// user verification code
	publicGroup.GET("/user/verification_code/list", apiMiddleware.AppApiHandlerWrapper(myVerificationCodeList))
	publicGroup.POST("/spot/like", apiMiddleware.AppApiHandlerWrapper(spotLike))
	publicGroup.GET("/spot/like/list", apiMiddleware.AppApiHandlerWrapper(spotLikeList))
	publicGroup.POST("/order/cancel", apiMiddleware.AppApiHandlerWrapper(CancelOrder))
	publicGroup.GET("/pay/method/list", apiMiddleware.AppApiHandlerWrapper(getPayMethodList))

	publicGroup.POST("/spot/claim", apiMiddleware.AppApiHandlerWrapper(spotClaim))
	publicGroup.GET("/user/rewards", apiMiddleware.AppApiHandlerWrapper(userRewards))
	publicGroup.POST("/user/claim", apiMiddleware.AppApiHandlerWrapper(userClaim))

	publicGroup.POST("/image/get_presigned_url", apiMiddleware.AppApiHandlerWrapper(getPresignedURL))
	publicGroup.POST("/advertisement/claim", apiMiddleware.AppApiHandlerWrapper(advertisementClaim))
	publicGroup.GET("/advertisement/list", apiMiddleware.AppApiHandlerWrapper(advertisementListByCategory))
	publicGroup.GET("/complaint/type/list", apiMiddleware.AppApiHandlerWrapper(complaintTypeList))
	publicGroup.POST("/complaint/create", apiMiddleware.AppApiHandlerWrapper(createComplaint))
	publicGroup.GET("/complaint/list", apiMiddleware.AppApiHandlerWrapper(complaintList))

	publicGroup.POST("/email/captcha", apiMiddleware.AppApiHandlerWrapper(emailCaptcha))
	publicGroup.POST("/email/login", apiMiddleware.AppApiHandlerWrapper(emailLogin))

	publicGroup.POST("/token/refresh", apiMiddleware.AppApiHandlerWrapper(refreshToken))

	publicGroup.GET("/user/bill", apiMiddleware.AppApiHandlerWrapper(userBill))
	publicGroup.POST("/user/withdraw", apiMiddleware.AppApiHandlerWrapper(userWithdraw))
	publicGroup.GET("/user/withdraw/latest", apiMiddleware.AppApiHandlerWrapper(userLatestWithdrawOrder))

	publicGroup.GET("/ota/latest", apiMiddleware.AppApiHandlerWrapper(otaLatest))
	publicGroup.GET("/believe/chat_link", apiMiddleware.AppApiHandlerWrapper(getBelieveChatLink))

	publicGroup.GET("/chat/conversations", apiMiddleware.AppApiHandlerWrapper(getConversations))
	publicGroup.GET("/chat/conversation/:id", apiMiddleware.AppApiHandlerWrapper(getConversation))
	publicGroup.POST("/chat/message", apiMiddleware.AppApiHandlerWrapper(sendMessage))
	publicGroup.DELETE("/chat/conversation/:id", apiMiddleware.AppApiHandlerWrapper(deleteConversation))
	publicGroup.GET("/chat/recommended", apiMiddleware.AppApiHandlerWrapper(getRecommendedConversations))

	// user address
	publicGroup.POST("/user/address/add", apiMiddleware.AppApiHandlerWrapper(addAddress))
	publicGroup.POST("/user/address/update", apiMiddleware.AppApiHandlerWrapper(updateAddress))
	publicGroup.GET("/user/address/list", apiMiddleware.AppApiHandlerWrapper(getAddressList))
	publicGroup.POST("/user/address/delete", apiMiddleware.AppApiHandlerWrapper(deleteAddress))

	// user asset
	// publicGroup.GET("/user/asset/list", apiMiddleware.AppApiHandlerWrapper(getUserAssetList))

	// comment
	publicGroup.POST("/product/comment/add", apiMiddleware.AppApiHandlerWrapper(addComment))
	publicGroup.POST("/product/comment/delete", apiMiddleware.AppApiHandlerWrapper(deleteComment))
	publicGroup.POST("/product/comment/update", apiMiddleware.AppApiHandlerWrapper(updateComment))
	publicGroup.GET("/spot/comment/list", apiMiddleware.AppApiHandlerWrapper(getSpotComments))
	publicGroup.GET("/product/comment/list", apiMiddleware.AppApiHandlerWrapper(getProductComments))

	// product order
	publicGroup.POST("/product/order/create", apiMiddleware.AppApiHandlerWrapper(createProductOrder))
	publicGroup.GET("/product/order/detail", apiMiddleware.AppApiHandlerWrapper(getProductOrderDetail))
	publicGroup.GET("/product/order/pay_status", apiMiddleware.AppApiHandlerWrapper(getProductOrderPayStatus))
	publicGroup.POST("/product/order/cancel", apiMiddleware.AppApiHandlerWrapper(cancelProductOrder))
	publicGroup.GET("/product/order/pay_info", apiMiddleware.AppApiHandlerWrapper(getProductPayInfo))

	// 支付上报
	publicGroup.POST("/product/order/pay_report", apiMiddleware.AppApiHandlerWrapper(payReport))
	// 用户删除订单
	publicGroup.POST("/product/order/delete", apiMiddleware.AppApiHandlerWrapper(deleteProductOrder))

	// 商户修改订单价格
	publicGroup.POST("/product/order/update_price", apiMiddleware.AppApiHandlerWrapper(updateProductOrderPrice))

	// 用户修改订单地址
	publicGroup.POST("/product/order/update_address", apiMiddleware.AppApiHandlerWrapper(updateProductOrderAddress))

	// product order confirm shipment
	publicGroup.POST("/product/order/confirm_shipment", apiMiddleware.AppApiHandlerWrapper(confirmProductOrderShipment))

	// product order confirm receipt
	publicGroup.POST("/product/order/confirm_receipt", apiMiddleware.AppApiHandlerWrapper(confirmProductOrderReceipt))

	// product order cancel by merchant
	publicGroup.POST("/product/order/cancel_by_merchant", apiMiddleware.AppApiHandlerWrapper(cancelProductOrderByMerchant))

	// apply refund
	publicGroup.POST("/product/order/apply_refund", apiMiddleware.AppApiHandlerWrapper(applyRefund))
	// 商户处理退款
	publicGroup.POST("/product/order/handle_refund", apiMiddleware.AppApiHandlerWrapper(handleRefund))

	// checkin
	publicGroup.POST("/checkin/create", apiMiddleware.AppApiHandlerWrapper(CreateCheckIn))
	publicGroup.GET("/checkin/detail", apiMiddleware.AppApiHandlerWrapper(getCheckin))
	publicGroup.GET("/checkin/tags", apiMiddleware.AppApiHandlerWrapper(getCheckinTags))
	publicGroup.GET("/user/checkin/list", apiMiddleware.AppApiHandlerWrapper(userCheckinList))
	publicGroup.POST("/checkin/comment/add", apiMiddleware.AppApiHandlerWrapper(addCheckinComment))
	publicGroup.POST("/checkin/comment/delete", apiMiddleware.AppApiHandlerWrapper(deleteCheckinComment))
	publicGroup.POST("/checkin/comment/reply", apiMiddleware.AppApiHandlerWrapper(replyCheckinComment))
	publicGroup.GET("/checkin/comment/list", apiMiddleware.AppApiHandlerWrapper(getCheckinCommentList))
	// comment like
	publicGroup.POST("/checkin/comment/like", apiMiddleware.AppApiHandlerWrapper(likeCheckinComment))
	// checkin delete
	publicGroup.POST("/checkin/delete", apiMiddleware.AppApiHandlerWrapper(deleteCheckin))
	// checkin update private
	publicGroup.POST("/checkin/private/update", apiMiddleware.AppApiHandlerWrapper(updateCheckinPrivate))

	// checkin 非会员最多只能发布5个打卡
	publicGroup.GET("/checkin/publish/restrict", apiMiddleware.AppApiHandlerWrapper(getCheckinPublishRestrict))

	// 获取打卡message未读数量
	publicGroup.GET("/checkin/message/unread_count", apiMiddleware.AppApiHandlerWrapper(checkinMessageUnreadCount))
	publicGroup.GET("/checkin/message/list", apiMiddleware.AppApiHandlerWrapper(checkinMessageList))

	// 打卡列表，根据用户所在城市推荐打卡点
	publicGroup.GET("/checkin/list", apiMiddleware.AppApiHandlerWrapper(getCheckinList))

	// 打卡点赞 取消
	publicGroup.POST("/checkin/like", apiMiddleware.AppApiHandlerWrapper(checkinLike))
	// 地标下的打卡
	publicGroup.GET("/spot/checkin/latest", apiMiddleware.AppApiHandlerWrapper(getSpotLatestCheckinList))
	publicGroup.GET("/spot/checkin/recommended", apiMiddleware.AppApiHandlerWrapper(getSpotRecommendedCheckinList))
	// user follow
	publicGroup.POST("/user/follow", apiMiddleware.AppApiHandlerWrapper(userFollow))

	// 其他用户的信息和统计信息，区别于user/info
	publicGroup.GET("/user/info/other", apiMiddleware.AppApiHandlerWrapper(getUserInfoOther))

	// member config
	publicGroup.GET("/member/config", apiMiddleware.AppApiHandlerWrapper(getMemberConfig))
	// member order
	publicGroup.POST("/member/order/create", apiMiddleware.AppApiHandlerWrapper(createMemberOrder))
	// member order detail
	publicGroup.GET("/member/order/detail", apiMiddleware.AppApiHandlerWrapper(getMemberOrderDetail))
	publicGroup.POST("/member/order/pay_report", apiMiddleware.AppApiHandlerWrapper(memberPayReport))
	// member order pay status
	publicGroup.GET("/member/order/pay_status", apiMiddleware.AppApiHandlerWrapper(getMemberOrderPayStatus))
	// member order cancel
	publicGroup.POST("/member/order/cancel", apiMiddleware.AppApiHandlerWrapper(cancelMemberOrder))
	// member payinfo
	publicGroup.GET("/member/order/pay_info", apiMiddleware.AppApiHandlerWrapper(getMemberOrderPayInfo))
	// user member info
	publicGroup.GET("/user/member/info", apiMiddleware.AppApiHandlerWrapper(getUserMemberInfo))
	// user member order list
	publicGroup.GET("/user/member/order/list", apiMiddleware.AppApiHandlerWrapper(getUserMemberOrderList))

	v2Group := app.Group("/api/v2")
	v2Group.Use(apiMiddleware.AppContext(nu))
	v2Group.POST("/auth/google/login", apiMiddleware.AppApiHandlerWrapper(googleLoginV2))
	v2Group.POST("/auth/apple/login", apiMiddleware.AppApiHandlerWrapper(appleLoginV2))
	v2Group.POST("/auth/password/login", apiMiddleware.AppApiHandlerWrapper(passwordLoginV2))
	v2Group.POST("/email/login", apiMiddleware.AppApiHandlerWrapper(emailLoginV2))

	v2Group.POST("/order/create", apiMiddleware.AppApiHandlerWrapper(createOrderV2))
	v2Group.POST("/order/cancel", apiMiddleware.AppApiHandlerWrapper(CancelOrderV2))
	v2Group.GET("/order/pay_info", apiMiddleware.AppApiHandlerWrapper(getPayInfoV2))
	v2Group.GET("/order/pay_status", apiMiddleware.AppApiHandlerWrapper(getOrderPayStatus))
	v2Group.GET("/order/detail", apiMiddleware.AppApiHandlerWrapper(getOrderDetailV2))

	v2Group.GET("/user/bill", apiMiddleware.AppApiHandlerWrapper(userBillV2))

}
