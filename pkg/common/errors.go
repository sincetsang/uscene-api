package common

import (
	"TMA/pkg/sderr"
)

const (
	DefaultErrCode = 500
)

var (
	ErrParam                      = sderr.Sentinel("Param Error")
	ErrService                    = sderr.Sentinel("service Error")
	ErrUnauthorized               = sderr.Sentinel("Unauthorized")
	ErrUnPower                    = sderr.Sentinel("No permission")
	ErrSendLikeLimit              = sderr.Sentinel("You can continue to give him/her a thumbs up tomorrow.")
	ErrUnknown                    = sderr.Sentinel("unknown error")
	ErrNotPrimary                 = sderr.Sentinel("Please upgrade your Pass to Premium")
	ErrAllreadyCheckedIn          = sderr.Sentinel("Allready checked in")
	ErrAllreadyLiked              = sderr.Sentinel("Allready liked")
	ErrAllreadyClaimed            = sderr.Sentinel("Already claimed")
	ErrCheckInFailed              = sderr.Sentinel("Check in failed")
	ErrSpotLikeFailed             = sderr.Sentinel("Spot like failed")
	ErrCheckInPointNotExist       = sderr.Sentinel("Spot does not exist")
	ErrCheckInOverLimit           = sderr.Sentinel("Check in reaches the upper limit")
	ErrCheckInOverLimitDaily      = sderr.Sentinel("Check in reaches the upper limit daily")
	ErrLikeInOverLimit            = sderr.Sentinel("Like in reaches the upper limit")
	ErrUserNotVerifyLocation      = sderr.Sentinel("Please update your location")
	ErrCheckPointNotOpen          = sderr.Sentinel("Spot not open now")
	ErrRequestTooFast             = sderr.Sentinel("Request too fast")
	ErrCheckInExistLock           = sderr.Sentinel("The current check-in point has been locked by another user. Please change it or try again later")
	ErrCanOnlyClaimOnce           = sderr.Sentinel("You can only claim once a day")
	ErrLocationLatLonErr          = sderr.Sentinel("Latitude or longitude err")
	ErrUserNotExist               = sderr.Sentinel("User does not exist")
	ErrUserInfoNicknameTooLong    = sderr.Sentinel("User nickname too long")
	ErrUserInfoSelectProperGender = sderr.Sentinel("Please select proper gender")
	ErrUserInfoMustNotEmpoty      = sderr.Sentinel("User nickname must not empty")
	ErrRankingTypeErr             = sderr.Sentinel("Ranking type error")
	ErrBalance                    = sderr.Sentinel("Insufficient Balance")
	ErrLastedLevel                = sderr.Sentinel("The current level is already the latest level")
	ErrCreateCity                 = sderr.Sentinel("Creating Spot is not yet open.")
	ErrWalletAddress              = sderr.Sentinel("Wallet address format error")
	ErrRequestExpire              = sderr.Sentinel("Request has expired")
	ErrNotFoundProduct            = sderr.Sentinel("Product does not exist")
	ErrSpotNotExist               = sderr.Sentinel("Landmarks are under review")
	ErrUploadLimit                = sderr.Sentinel("Daily upload limit reached")
	ErrOrderNotExist              = sderr.Sentinel("Order does not exist")
	ErrOrderStatusChanged         = sderr.Sentinel("Order status has changed")
	ErrInsufficientBalance        = sderr.Sentinel("余额不足")
	ErrWithdrawInProgress         = sderr.Sentinel("已有进行中的提现订单")

	ErrUserUnAuthorizedLocation = sderr.Sentinel("Please authorize the current location first")

	ErrTaskAlreadyCompleted = sderr.Sentinel("Task already completed")
	ErrTaskNotFound         = sderr.Sentinel("Task not found")

	ErrSpotAlreadyHighestLevel = sderr.Sentinel("It's already the highest level")

	ErrSpotOrderLimit               = sderr.Sentinel("Pending Spots limit reached.")
	ErrSpotFinishOrderLimit         = sderr.Sentinel("Owned Spots limit reached.")
	ErrSpotOverlap                  = sderr.Sentinel("The current location overlaps with other check-in points, please change the location.")
	ErrSpotOrderDoing               = sderr.Sentinel("The order is currently being purchased, please check later.")
	ErrSpotOrderNotBuy              = sderr.Sentinel("The current spot is locked and cannot be purchased")
	ErrSpotLandmarkProtectionPeriod = sderr.Sentinel("The current spot cannot be purchased during the protection period. Please try again later")
	ErrNotSpotWaitClaim             = sderr.Sentinel("No income to be claimed")
	ErrSpotNameLength               = sderr.Sentinel("The length of the spot name cannot exceed 50 characters")
	ErrSpotTypeLimitByEvent         = sderr.Sentinel("Not open for purchase")

	ErrSpotOrderStatus        = sderr.Sentinel("Please check the order status")
	ErrSpotOrderNotExist      = sderr.Sentinel("Spot order not exists")
	ErrSpotOrderRefundNotAble = sderr.Sentinel("Refunding or refunded, Can't refund")

	ErrSpotNoRefundInfo           = sderr.Sentinel("Spot has no withdrawal")
	ErrSpotCannotSubmitRepeatedly = sderr.Sentinel("Cannot submit orders repeatedly")
	ErrSpotCannotBuySelfSpot      = sderr.Sentinel("Can't purchase the spot that belongs to youself")

	ErrFileSizeLimit = sderr.Sentinel("file size exceeds the limit of 2MB")

	ErrUserTransferToYourself  = sderr.Sentinel("No need to transfer $LAND to yourself")
	ErrUserTransferAmountLimit = sderr.Sentinel("The transfer amount cannot be less than or equal to 0")

	ErrActivityNotExists        = sderr.Sentinel("Activity not exists")
	ErrInviteSelf               = sderr.Sentinel("Cannot invite self")
	ErrSign                     = sderr.Sentinel("Sign in failed")
	ErrAlreadySignIn            = sderr.Sentinel("Already signed in today")
	ErrFetchStatus              = sderr.Sentinel("Fetch status error")
	ErrRepeatOperation          = sderr.Sentinel("Do not repeat the operation")
	ErrAlreadyAddFriend         = sderr.Sentinel("This friend has already been added, no need to repeat")
	ErrAuthLogin                = sderr.Sentinel("Login authorization error")
	ErrAuthLoginExpired         = sderr.Sentinel("Login authorization expired")
	ErrAccessDenied             = sderr.Sentinel("Access Denied")
	ErrInviteExists             = sderr.Sentinel("Recommended person already bound")
	ErrAddInviteExpired         = sderr.Sentinel("The registration has exceeded 24 hours and cannot be set")
	ErrInviteEachOther          = sderr.Sentinel("Cannot recommend each other")
	ErrCaptcha                  = sderr.Sentinel("Captcha error")
	ErrSpotBalanceNotCanCheckIn = sderr.Sentinel("Landmark balance insufficient, unable to clock in")

	ErrPasswordTooWeak      = sderr.Sentinel("Password too weak")
	ErrPasswordNotMatch     = sderr.Sentinel("Passwords do not match")
	ErrOldPasswordRequired  = sderr.Sentinel("Old password is required")
	ErrOldPasswordIncorrect = sderr.Sentinel("Old password is incorrect")
	ErrPasswordNotSet       = sderr.Sentinel("Password not set")
	ErrPasswordIncorrect    = sderr.Sentinel("Password incorrect")

	ErrUnsupportedFileType       = sderr.New("不支持的文件类型")
	ErrImageContentInappropriate = sderr.New("图片内容不适合，请上传合适的图片")
	ErrVersionTooLow             = sderr.New("版本过低，请升级")
	ErrTooManyRequests           = sderr.New("今日消息发送已达上限，请明天再继续")
	ErrPayMethodNotExist         = sderr.New("支付方式不存在")

	ErrProductNotExist         = sderr.Sentinel("Product does not exist")
	ErrProductFavoriteExist    = sderr.Sentinel("Product favorite already exists")
	ErrProductFavoriteNotExist = sderr.Sentinel("Product favorite does not exist")
	ErrBelieveChatLinkNotFound = sderr.New("请检查Believe id是否正确")

	// comment
	ErrCommentCanOnlyAddOnce = sderr.Sentinel("Comment can only add once")
	ErrCommentNotExist       = sderr.Sentinel("Comment does not exist")

	// address
	ErrAddressNotExist = sderr.Sentinel("Address does not exist")

	// product order
	ErrProductNotSupportThisCurrency = sderr.Sentinel("Product does not support this currency")
	ErrProductOrderNotExist          = sderr.Sentinel("Product order does not exist")
	ErrOrderStatusCannotApplyRefund  = sderr.Sentinel("Order status cannot apply refund")
	// 不能重复申请退款
	ErrOrderAlreadyAppliedRefund = sderr.Sentinel("Order already applied refund")
	// 退款中不能确认收货，有待退款
	ErrOrderRefundingCannotConfirmReceipt = sderr.Sentinel("Cannot confirm receipt, there is a refund to be processed")

	// FEC 支付不支持修改价格
	ErrFECCannotUpdatePrice = sderr.Sentinel("FEC payment cannot update price")

	ErrCheckInNotExists        = sderr.Sentinel("Check in does not exist")
	ErrUserFollowYourself      = sderr.Sentinel("Cannot follow yourself")
	ErrCheckInCommentNotExists = sderr.Sentinel("Check in comment does not exist")

	ErrNoNearSpotFound = sderr.Sentinel("No nearby spot found")

	//member
	ErrMemberOrderAlreadyExists = sderr.Sentinel("There is a pending payment order, please cancel it first")
	ErrMemberOrderNotExist      = sderr.Sentinel("Member order does not exist")

	ErrCheckinUpgradeMemberToPublish = sderr.Sentinel("Please upgrade to VIP to publish check-in")
)

var ErrCodeMap map[error]int
var ErrMsgMap map[string]map[error]error

func init() {
	ErrCodeMap = make(map[error]int)
	ErrCodeMap = map[error]int{
		ErrUnauthorized:     401,
		ErrAuthLoginExpired: 401,
		ErrAuthLogin:        401,
		ErrParam:            400,
		ErrAccessDenied:     403,
		ErrService:          503,
		ErrVersionTooLow:    4002,
		ErrTooManyRequests:  4010,
	}

	// 初始化多语言错误消息映射
	ErrMsgMap = map[string]map[error]error{
		"en": {
			ErrParam:                        sderr.Sentinel("Param Error"),
			ErrService:                      sderr.Sentinel("service Error"),
			ErrUnauthorized:                 sderr.Sentinel("Unauthorized"),
			ErrUnPower:                      sderr.Sentinel("No permission"),
			ErrSendLikeLimit:                sderr.Sentinel("You can continue to give him/her a thumbs up tomorrow."),
			ErrUnknown:                      sderr.Sentinel("unknown error"),
			ErrNotPrimary:                   sderr.Sentinel("Please upgrade your Pass to Premium"),
			ErrAllreadyCheckedIn:            sderr.Sentinel("Allready checked in"),
			ErrAllreadyLiked:                sderr.Sentinel("Allready liked"),
			ErrAllreadyClaimed:              sderr.Sentinel("Already claimed"),
			ErrCheckInFailed:                sderr.Sentinel("Check in failed"),
			ErrSpotLikeFailed:               sderr.Sentinel("Spot like failed"),
			ErrCheckInPointNotExist:         sderr.Sentinel("Spot does not exist"),
			ErrCheckInOverLimit:             sderr.Sentinel("Check in reaches the upper limit"),
			ErrCheckInOverLimitDaily:        sderr.Sentinel("Check in reaches the upper limit daily"),
			ErrLikeInOverLimit:              sderr.Sentinel("Like in reaches the upper limit"),
			ErrUserNotVerifyLocation:        sderr.Sentinel("Please update your location"),
			ErrCheckPointNotOpen:            sderr.Sentinel("Spot not open now"),
			ErrRequestTooFast:               sderr.Sentinel("Request too fast"),
			ErrCheckInExistLock:             sderr.Sentinel("The current check-in point has been locked by another user. Please change it or try again later"),
			ErrCanOnlyClaimOnce:             sderr.Sentinel("You can only claim once a day"),
			ErrLocationLatLonErr:            sderr.Sentinel("Latitude or longitude err"),
			ErrUserNotExist:                 sderr.Sentinel("User does not exist"),
			ErrUserInfoNicknameTooLong:      sderr.Sentinel("User nickname too long"),
			ErrUserInfoSelectProperGender:   sderr.Sentinel("Please select proper gender"),
			ErrUserInfoMustNotEmpoty:        sderr.Sentinel("User nickname must not empty"),
			ErrRankingTypeErr:               sderr.Sentinel("Ranking type error"),
			ErrBalance:                      sderr.Sentinel("Insufficient Balance"),
			ErrLastedLevel:                  sderr.Sentinel("The current level is already the latest level"),
			ErrCreateCity:                   sderr.Sentinel("Creating Spot is not yet open."),
			ErrUserUnAuthorizedLocation:     sderr.Sentinel("Please authorize the current location first"),
			ErrTaskAlreadyCompleted:         sderr.Sentinel("Task already completed"),
			ErrTaskNotFound:                 sderr.Sentinel("Task not found"),
			ErrSpotAlreadyHighestLevel:      sderr.Sentinel("It's already the highest level"),
			ErrSpotOrderLimit:               sderr.Sentinel("Pending Spots limit reached."),
			ErrSpotFinishOrderLimit:         sderr.Sentinel("Owned Spots limit reached."),
			ErrSpotOverlap:                  sderr.Sentinel("The current location overlaps with other check-in points, please change the location."),
			ErrSpotOrderDoing:               sderr.Sentinel("The order is currently being purchased, please check later."),
			ErrSpotOrderNotBuy:              sderr.Sentinel("The current spot is locked and cannot be purchased"),
			ErrSpotLandmarkProtectionPeriod: sderr.Sentinel("The current spot cannot be purchased during the protection period. Please try again later"),
			ErrNotSpotWaitClaim:             sderr.Sentinel("No income to be claimed"),
			ErrSpotNameLength:               sderr.Sentinel("The length of the spot name cannot exceed 50 characters"),
			ErrSpotTypeLimitByEvent:         sderr.Sentinel("Not open for purchase"),
			ErrSpotOrderStatus:              sderr.Sentinel("Please check the order status"),
			ErrSpotOrderNotExist:            sderr.Sentinel("Spot order not exists"),
			ErrSpotOrderRefundNotAble:       sderr.Sentinel("Refunding or refunded, Can't refund"),
			ErrSpotNoRefundInfo:             sderr.Sentinel("Spot has no withdrawal"),
			ErrSpotCannotSubmitRepeatedly:   sderr.Sentinel("Cannot submit orders repeatedly"),
			ErrSpotCannotBuySelfSpot:        sderr.Sentinel("Can't purchase the spot that belongs to yourself"),
			ErrFileSizeLimit:                sderr.Sentinel("file size exceeds the limit of 2MB"),
			ErrUserTransferToYourself:       sderr.Sentinel("No need to transfer $LAND to yourself"),
			ErrUserTransferAmountLimit:      sderr.Sentinel("The transfer amount cannot be less than or equal to 0"),
			ErrActivityNotExists:            sderr.Sentinel("Activity not exists"),
			ErrInviteSelf:                   sderr.Sentinel("Cannot invite self"),
			ErrSign:                         sderr.Sentinel("Sign in failed"),
			ErrAlreadySignIn:                sderr.Sentinel("Already signed in today"),
			ErrFetchStatus:                  sderr.Sentinel("Fetch status error"),
			ErrRepeatOperation:              sderr.Sentinel("Do not repeat the operation"),
			ErrAlreadyAddFriend:             sderr.Sentinel("This friend has already been added, no need to repeat"),
			ErrAuthLogin:                    sderr.Sentinel("Login authorization error"),
			ErrAuthLoginExpired:             sderr.Sentinel("Login authorization expired"),
			ErrAccessDenied:                 sderr.Sentinel("Access Denied"),
			ErrInviteExists:                 sderr.Sentinel("Recommended person already bound"),
			ErrAddInviteExpired:             sderr.Sentinel("The registration has exceeded 24 hours and cannot be set"),
			ErrInviteEachOther:              sderr.Sentinel("Cannot recommend each other"),
			ErrWalletAddress:                sderr.Sentinel("Wallet address format error"),
			ErrRequestExpire:                sderr.Sentinel("Request has expired"),
			ErrCaptcha:                      sderr.Sentinel("Captcha error"),
			ErrSpotBalanceNotCanCheckIn:     sderr.Sentinel("Landmark balance insufficient, unable to clock in"),
			ErrPasswordTooWeak:              sderr.Sentinel("Password too weak"),
			ErrPasswordNotMatch:             sderr.Sentinel("Passwords do not match"),
			ErrOldPasswordRequired:          sderr.Sentinel("Old password is required"),
			ErrOldPasswordIncorrect:         sderr.Sentinel("Old password is incorrect"),
			ErrPasswordNotSet:               sderr.Sentinel("Password not set"),
			ErrPasswordIncorrect:            sderr.Sentinel("Password incorrect"),
			ErrUnsupportedFileType:          sderr.Sentinel("Unsupported file type"),
			ErrImageContentInappropriate:    sderr.Sentinel("Inappropriate image content, please upload a suitable image"),
			ErrNotFoundProduct:              sderr.Sentinel("Product does not exist"),
			ErrSpotNotExist:                 sderr.Sentinel("Landmarks are under review"),
			ErrUploadLimit:                  sderr.Sentinel("Daily upload limit reached"),
			ErrOrderNotExist:                sderr.Sentinel("Order does not exist"),
			ErrOrderStatusChanged:           sderr.Sentinel("Order status has changed"),
			ErrInsufficientBalance:          sderr.Sentinel("Insufficient Balance"),
			ErrWithdrawInProgress:           sderr.Sentinel("Withdrawal in progress"),
			ErrVersionTooLow:                sderr.Sentinel("Version too low, please upgrade"),
			ErrTooManyRequests:              sderr.Sentinel("Daily message limit reached, please try again tomorrow"),
			ErrPayMethodNotExist:            sderr.Sentinel("Payment method does not exist"),
			ErrBelieveChatLinkNotFound:      sderr.Sentinel("please check the believe id"),

			// comment
			ErrCommentCanOnlyAddOnce: sderr.Sentinel("Comment can only add once"),
			ErrCommentNotExist:       sderr.Sentinel("Comment does not exist"),

			// product order
			ErrProductNotSupportThisCurrency:      sderr.Sentinel("Product does not support this currency"),
			ErrProductOrderNotExist:               sderr.Sentinel("Product order does not exist"),
			ErrOrderStatusCannotApplyRefund:       sderr.Sentinel("Order status cannot apply refund"),
			ErrOrderAlreadyAppliedRefund:          sderr.Sentinel("Order already applied refund"),
			ErrOrderRefundingCannotConfirmReceipt: sderr.Sentinel("Cannot confirm receipt, there is a refund to be processed"),
			ErrFECCannotUpdatePrice:               sderr.Sentinel("FEC payment cannot update price"),

			ErrCheckInNotExists:              sderr.Sentinel("Check in does not exist"),
			ErrUserFollowYourself:            sderr.Sentinel("Cannot follow yourself"),
			ErrCheckInCommentNotExists:       sderr.Sentinel("Check in comment does not exist"),
			ErrNoNearSpotFound:               sderr.Sentinel("No nearby spot found"),
			ErrMemberOrderAlreadyExists:      sderr.Sentinel("There is a pending payment order, please cancel it first"),
			ErrMemberOrderNotExist:           sderr.Sentinel("Member order does not exist"),
			ErrCheckinUpgradeMemberToPublish: sderr.Sentinel("Please upgrade to VIP to publish check-in"),
		},
		"zh": {
			ErrUnauthorized:                 sderr.Sentinel("未授权"),
			ErrParam:                        sderr.Sentinel("参数错误"),
			ErrService:                      sderr.Sentinel("服务不可用"),
			ErrUnPower:                      sderr.Sentinel("无权限"),
			ErrSendLikeLimit:                sderr.Sentinel("明天再继续给他/她点赞吧"),
			ErrUnknown:                      sderr.Sentinel("服务器错误"),
			ErrNotPrimary:                   sderr.Sentinel("请先成为会员"),
			ErrAllreadyCheckedIn:            sderr.Sentinel("已打卡"),
			ErrAllreadyLiked:                sderr.Sentinel("已点赞"),
			ErrAllreadyClaimed:              sderr.Sentinel("已领取"),
			ErrCheckInFailed:                sderr.Sentinel("打卡失败"),
			ErrSpotLikeFailed:               sderr.Sentinel("打卡点Like失败"),
			ErrCheckInPointNotExist:         sderr.Sentinel("打卡点不存在"),
			ErrCheckInOverLimit:             sderr.Sentinel("打卡次数已达上限"),
			ErrCheckInOverLimitDaily:        sderr.Sentinel("每日打卡次数已达上限"),
			ErrLikeInOverLimit:              sderr.Sentinel("Like次数已达上限"),
			ErrUserNotVerifyLocation:        sderr.Sentinel("请更新你的位置"),
			ErrCheckPointNotOpen:            sderr.Sentinel("打卡点暂未开放"),
			ErrRequestTooFast:               sderr.Sentinel("请求过快，请稍后再试"),
			ErrCheckInExistLock:             sderr.Sentinel("当前地标已被其他用户锁定，请更改或稍后再试"),
			ErrCanOnlyClaimOnce:             sderr.Sentinel("每天只能领取一次"),
			ErrLocationLatLonErr:            sderr.Sentinel("经度或纬度错误"),
			ErrUserNotExist:                 sderr.Sentinel("用户不存在"),
			ErrUserInfoNicknameTooLong:      sderr.Sentinel("用户名超过限制长度"),
			ErrUserInfoSelectProperGender:   sderr.Sentinel("请选择正确的性别"),
			ErrUserInfoMustNotEmpoty:        sderr.Sentinel("用户名不能为空"),
			ErrRankingTypeErr:               sderr.Sentinel("排行类型错误"),
			ErrBalance:                      sderr.Sentinel("余额不足"),
			ErrLastedLevel:                  sderr.Sentinel("已经是最高等级"),
			ErrCreateCity:                   sderr.Sentinel("创建地标服务暂未开放"),
			ErrUserUnAuthorizedLocation:     sderr.Sentinel("请先授权位置信息"),
			ErrTaskAlreadyCompleted:         sderr.Sentinel("任务已完成"),
			ErrTaskNotFound:                 sderr.Sentinel("找不到对应任务，请检查参数"),
			ErrSpotAlreadyHighestLevel:      sderr.Sentinel("已经是最高等级"),
			ErrSpotOrderLimit:               sderr.Sentinel("支付中订单过多，请稍后再试"),
			ErrSpotFinishOrderLimit:         sderr.Sentinel("持有地标数量已达上限"),
			ErrSpotOverlap:                  sderr.Sentinel("当前位置与已创建地标重叠，请更改位置"),
			ErrSpotOrderDoing:               sderr.Sentinel("订单正在购买中"),
			ErrSpotOrderNotBuy:              sderr.Sentinel("当前地标被其他用户锁定，无法购买"),
			ErrSpotLandmarkProtectionPeriod: sderr.Sentinel("当前地标处在保护期内，无法购买"),
			ErrNotSpotWaitClaim:             sderr.Sentinel("没有待领取收益"),
			ErrSpotNameLength:               sderr.Sentinel("坐标点名称不能超过50个字符"),
			ErrSpotTypeLimitByEvent:         sderr.Sentinel("未开放购买"),
			ErrSpotOrderStatus:              sderr.Sentinel("请检查订单状态"),
			ErrSpotOrderNotExist:            sderr.Sentinel("订单不存在"),
			ErrSpotOrderRefundNotAble:       sderr.Sentinel("退款中或已退款，请检查到账信息"),
			ErrSpotNoRefundInfo:             sderr.Sentinel("没有退款信息"),
			ErrSpotCannotSubmitRepeatedly:   sderr.Sentinel("请勿重复提交订单"),
			ErrSpotCannotBuySelfSpot:        sderr.Sentinel("已经是地标持有人"),
			ErrFileSizeLimit:                sderr.Sentinel("图片限制为2MB"),
			ErrUserTransferToYourself:       sderr.Sentinel("不能转账给自己"),
			ErrUserTransferAmountLimit:      sderr.Sentinel("转账金额不能小于等于0"),
			ErrActivityNotExists:            sderr.Sentinel("坐标记录不存在"),
			ErrInviteSelf:                   sderr.Sentinel("不能邀请自己"),
			ErrSign:                         sderr.Sentinel("签名失败"),
			ErrAlreadySignIn:                sderr.Sentinel("今日已签到"),
			ErrFetchStatus:                  sderr.Sentinel("获取状态错误"),
			ErrRepeatOperation:              sderr.Sentinel("请勿重复操作"),
			ErrAlreadyAddFriend:             sderr.Sentinel("此好友已添加，无需重复"),
			ErrAuthLogin:                    sderr.Sentinel("登录授权错误"),
			ErrAuthLoginExpired:             sderr.Sentinel("登录授权过期"),
			ErrAccessDenied:                 sderr.Sentinel("拒绝访问"),
			ErrInviteExists:                 sderr.Sentinel("已绑定推荐人"),
			ErrAddInviteExpired:             sderr.Sentinel("注册已超过24小时，无法设置"),
			ErrInviteEachOther:              sderr.Sentinel("不能互相为推荐人"),
			ErrWalletAddress:                sderr.Sentinel("钱包地址格式错误"),
			ErrRequestExpire:                sderr.Sentinel("请求已过期"),
			ErrCaptcha:                      sderr.Sentinel("验证码错误"),
			ErrSpotBalanceNotCanCheckIn:     sderr.Sentinel("地标余额不足不能打卡"),
			ErrPasswordTooWeak:              sderr.Sentinel("密码强度太弱"),
			ErrPasswordNotMatch:             sderr.Sentinel("两次输入的密码不一致"),
			ErrOldPasswordRequired:          sderr.Sentinel("需要输入旧密码"),
			ErrOldPasswordIncorrect:         sderr.Sentinel("旧密码不正确"),
			ErrPasswordNotSet:               sderr.Sentinel("未设置密码，请使用验证码登录"),
			ErrPasswordIncorrect:            sderr.Sentinel("密码错误"),
			ErrUnsupportedFileType:          sderr.Sentinel("不支持的文件类型"),
			ErrImageContentInappropriate:    sderr.Sentinel("图片内容不适合，请上传合适的图片"),
			ErrNotFoundProduct:              sderr.Sentinel("产品不存在"),
			ErrSpotNotExist:                 sderr.Sentinel("地标正在审核中"),
			ErrUploadLimit:                  sderr.Sentinel("今日上传图片数量已达上限"),
			ErrOrderNotExist:                sderr.Sentinel("订单不存在"),
			ErrOrderStatusChanged:           sderr.Sentinel("订单状态已变更，请检查"),
			ErrInsufficientBalance:          sderr.Sentinel("余额不足"),
			ErrWithdrawInProgress:           sderr.Sentinel("已有进行中的提现订单"),
			ErrVersionTooLow:                sderr.Sentinel("版本过低，请升级"),
			ErrTooManyRequests:              sderr.Sentinel("今日消息发送已达上限，请明天再继续"),
			ErrPayMethodNotExist:            sderr.Sentinel("支付方式不存在"),
			ErrBelieveChatLinkNotFound:      sderr.Sentinel("请检查Believe id是否正确"),

			// comment
			ErrCommentCanOnlyAddOnce: sderr.Sentinel("只能添加一次评论"),
			ErrCommentNotExist:       sderr.Sentinel("评论不存在"),

			// address
			ErrAddressNotExist: sderr.Sentinel("地址不存在"),

			// product order
			ErrProductNotSupportThisCurrency:      sderr.Sentinel("不支持该货币"),
			ErrProductOrderNotExist:               sderr.Sentinel("订单不存在"),
			ErrOrderStatusCannotApplyRefund:       sderr.Sentinel("订单状态不能申请退款"),
			ErrOrderAlreadyAppliedRefund:          sderr.Sentinel("已申请退款"),
			ErrOrderRefundingCannotConfirmReceipt: sderr.Sentinel("退款中不能确认收货，有待退款"),
			ErrFECCannotUpdatePrice:               sderr.Sentinel("FEC支付不支持修改价格"),

			ErrCheckInNotExists:              sderr.Sentinel("打卡不存在"),
			ErrUserFollowYourself:            sderr.Sentinel("不能关注自己"),
			ErrCheckInCommentNotExists:       sderr.Sentinel("打卡评论不存在"),
			ErrNoNearSpotFound:               sderr.Sentinel("附近没有打卡点"),
			ErrMemberOrderAlreadyExists:      sderr.Sentinel("有待支付的订单，请先取消"),
			ErrMemberOrderNotExist:           sderr.Sentinel("会员订单不存在"),
			ErrCheckinUpgradeMemberToPublish: sderr.Sentinel("请升级为会员以发布打卡"),
		},
		"zh_tw": {
			ErrUnauthorized:                 sderr.Sentinel("未授權"),
			ErrParam:                        sderr.Sentinel("參數錯誤"),
			ErrService:                      sderr.Sentinel("服務不可用"),
			ErrUnPower:                      sderr.Sentinel("無權限"),
			ErrSendLikeLimit:                sderr.Sentinel("明天再繼續給他/她點贊吧"),
			ErrUnknown:                      sderr.Sentinel("伺服器錯誤"),
			ErrNotPrimary:                   sderr.Sentinel("請先成為會員"),
			ErrAllreadyCheckedIn:            sderr.Sentinel("已打卡"),
			ErrAllreadyLiked:                sderr.Sentinel("已點贊"),
			ErrAllreadyClaimed:              sderr.Sentinel("已領取"),
			ErrCheckInFailed:                sderr.Sentinel("打卡失敗"),
			ErrSpotLikeFailed:               sderr.Sentinel("打卡點Like失敗"),
			ErrCheckInPointNotExist:         sderr.Sentinel("打卡點不存在"),
			ErrCheckInOverLimit:             sderr.Sentinel("打卡次數已達上限"),
			ErrCheckInOverLimitDaily:        sderr.Sentinel("每日打卡次數已達上限"),
			ErrLikeInOverLimit:              sderr.Sentinel("Like次數已達上限"),
			ErrUserNotVerifyLocation:        sderr.Sentinel("請更新你的位置"),
			ErrCheckPointNotOpen:            sderr.Sentinel("打卡點暫未開放"),
			ErrRequestTooFast:               sderr.Sentinel("請求過快，請稍後再試"),
			ErrCheckInExistLock:             sderr.Sentinel("當前地標已被其他使用者鎖定，請更改或稍後再試"),
			ErrCanOnlyClaimOnce:             sderr.Sentinel("每天只能領取一次"),
			ErrLocationLatLonErr:            sderr.Sentinel("經度或緯度錯誤"),
			ErrUserNotExist:                 sderr.Sentinel("使用者不存在"),
			ErrUserInfoNicknameTooLong:      sderr.Sentinel("使用者名稱超過限制長度"),
			ErrUserInfoSelectProperGender:   sderr.Sentinel("請選擇正確的性別"),
			ErrUserInfoMustNotEmpoty:        sderr.Sentinel("使用者名稱不能為空"),
			ErrRankingTypeErr:               sderr.Sentinel("排行類型錯誤"),
			ErrBalance:                      sderr.Sentinel("餘額不足"),
			ErrLastedLevel:                  sderr.Sentinel("已經是最高等級"),
			ErrCreateCity:                   sderr.Sentinel("創建地標服務暫未開放"),
			ErrUserUnAuthorizedLocation:     sderr.Sentinel("請先授權位置信息"),
			ErrTaskAlreadyCompleted:         sderr.Sentinel("任務已完成"),
			ErrTaskNotFound:                 sderr.Sentinel("找不到對應任務，請檢查參數"),
			ErrSpotAlreadyHighestLevel:      sderr.Sentinel("已經是最高等級"),
			ErrSpotOrderLimit:               sderr.Sentinel("支付中訂單過多，請稍後再試"),
			ErrSpotFinishOrderLimit:         sderr.Sentinel("持有地標數量已達上限"),
			ErrSpotOverlap:                  sderr.Sentinel("當前位置與已創建地標重疊，請更改位置"),
			ErrSpotOrderDoing:               sderr.Sentinel("訂單正在購買中"),
			ErrSpotOrderNotBuy:              sderr.Sentinel("當前地標被其他使用者鎖定，無法購買"),
			ErrSpotLandmarkProtectionPeriod: sderr.Sentinel("當前地標處在保護期內，無法購買"),
			ErrNotSpotWaitClaim:             sderr.Sentinel("沒有待領取收益"),
			ErrSpotNameLength:               sderr.Sentinel("座標點名稱不能超過50個字符"),
			ErrSpotTypeLimitByEvent:         sderr.Sentinel("未開放購買"),
			ErrSpotOrderStatus:              sderr.Sentinel("請檢查訂單狀態"),
			ErrSpotOrderNotExist:            sderr.Sentinel("訂單不存在"),
			ErrSpotOrderRefundNotAble:       sderr.Sentinel("退款中或已退款，請檢查到賬信息"),
			ErrSpotNoRefundInfo:             sderr.Sentinel("沒有退款信息"),
			ErrSpotCannotSubmitRepeatedly:   sderr.Sentinel("請勿重複提交訂單"),
			ErrSpotCannotBuySelfSpot:        sderr.Sentinel("已經是地標持有人"),
			ErrFileSizeLimit:                sderr.Sentinel("圖片限制為2MB"),
			ErrUserTransferToYourself:       sderr.Sentinel("不能轉帳給自己"),
			ErrUserTransferAmountLimit:      sderr.Sentinel("轉帳金額不能小於等於0"),
			ErrActivityNotExists:            sderr.Sentinel("座標記錄不存在"),
			ErrInviteSelf:                   sderr.Sentinel("不能邀請自己"),
			ErrSign:                         sderr.Sentinel("簽名失敗"),
			ErrAlreadySignIn:                sderr.Sentinel("今日已簽到"),
			ErrFetchStatus:                  sderr.Sentinel("獲取狀態錯誤"),
			ErrRepeatOperation:              sderr.Sentinel("請勿重複操作"),
			ErrAlreadyAddFriend:             sderr.Sentinel("此好友已添加，無需重複"),
			ErrAuthLogin:                    sderr.Sentinel("登入授權錯誤"),
			ErrAuthLoginExpired:             sderr.Sentinel("登入授權已過期"),
			ErrAccessDenied:                 sderr.Sentinel("拒絕訪問"),
			ErrInviteExists:                 sderr.Sentinel("已綁定推薦人"),
			ErrAddInviteExpired:             sderr.Sentinel("註冊已超過24小時，無法設置"),
			ErrInviteEachOther:              sderr.Sentinel("不能互相為推薦人"),
			ErrWalletAddress:                sderr.Sentinel("錢包地址格式錯誤"),
			ErrRequestExpire:                sderr.Sentinel("請求已過期"),
			ErrCaptcha:                      sderr.Sentinel("验证码错误"),
			ErrSpotBalanceNotCanCheckIn:     sderr.Sentinel("地標餘額不足不能打卡"),
			ErrPasswordTooWeak:              sderr.Sentinel("密碼強度太弱"),
			ErrPasswordNotMatch:             sderr.Sentinel("兩次輸入的密碼不一致"),
			ErrOldPasswordRequired:          sderr.Sentinel("需要輸入舊密碼"),
			ErrOldPasswordIncorrect:         sderr.Sentinel("舊密碼不正確"),
			ErrPasswordNotSet:               sderr.Sentinel("未設置密碼，請使用驗證碼登錄"),
			ErrPasswordIncorrect:            sderr.Sentinel("密碼錯誤"),
			ErrUnsupportedFileType:          sderr.Sentinel("不支持的文件類型"),
			ErrImageContentInappropriate:    sderr.Sentinel("圖片內容不適合，請上傳合適的圖片"),
			ErrNotFoundProduct:              sderr.Sentinel("產品不存在"),
			ErrSpotNotExist:                 sderr.Sentinel("地標正在審核中"),
			ErrUploadLimit:                  sderr.Sentinel("今日上傳圖片數量已達上限"),
			ErrOrderNotExist:                sderr.Sentinel("訂單不存在"),
			ErrOrderStatusChanged:           sderr.Sentinel("訂單狀態已變更，請檢查"),
			ErrInsufficientBalance:          sderr.Sentinel("餘額不足"),
			ErrWithdrawInProgress:           sderr.Sentinel("已有進行中的提現訂單"),
			ErrVersionTooLow:                sderr.Sentinel("版本過低，請升級"),
			ErrTooManyRequests:              sderr.Sentinel("今日消息發送已達上限，請明天再繼續"),
			ErrPayMethodNotExist:            sderr.Sentinel("支付方式不存在"),
			ErrBelieveChatLinkNotFound:      sderr.Sentinel("请检查Believe id是否正确"),

			// comment
			ErrCommentCanOnlyAddOnce: sderr.Sentinel("只能添加一次评论"),
			ErrCommentNotExist:       sderr.Sentinel("评论不存在"),

			// address
			ErrAddressNotExist: sderr.Sentinel("地址不存在"),

			// product order
			ErrProductNotSupportThisCurrency:      sderr.Sentinel("不支持该货币"),
			ErrProductOrderNotExist:               sderr.Sentinel("订单不存在"),
			ErrOrderStatusCannotApplyRefund:       sderr.Sentinel("订单状态不能申请退款"),
			ErrOrderAlreadyAppliedRefund:          sderr.Sentinel("已申请退款"),
			ErrOrderRefundingCannotConfirmReceipt: sderr.Sentinel("退款中不能确认收货，有待退款"),
			ErrFECCannotUpdatePrice:               sderr.Sentinel("FEC支付不支持修改价格"),

			ErrCheckInNotExists:        sderr.Sentinel("打卡不存在"),
			ErrUserFollowYourself:      sderr.Sentinel("不能关注自己"),
			ErrCheckInCommentNotExists: sderr.Sentinel("打卡评论不存在"),

			ErrNoNearSpotFound: sderr.Sentinel("附近没有打卡点"),

			//member
			ErrMemberOrderAlreadyExists: sderr.Sentinel("有待支付的订单，请先取消"),
			ErrMemberOrderNotExist:      sderr.Sentinel("会员订单不存在"),

			ErrCheckinUpgradeMemberToPublish: sderr.Sentinel("请升级为会员以发布打卡"),
		},
		"ru": {
			ErrUnauthorized:                 sderr.Sentinel("Неавторизованно"),
			ErrParam:                        sderr.Sentinel("Ошибка параметров"),
			ErrService:                      sderr.Sentinel("Сервис недоступен"),
			ErrUnPower:                      sderr.Sentinel("Нет прав доступа"),
			ErrSendLikeLimit:                sderr.Sentinel("Попробуйте снова поставить лайк завтра"),
			ErrUnknown:                      sderr.Sentinel("Ошибка сервера"),
			ErrNotPrimary:                   sderr.Sentinel("Пожалуйста, сначала станьте участником"),
			ErrAllreadyCheckedIn:            sderr.Sentinel("Уже зарегистрировано"),
			ErrAllreadyLiked:                sderr.Sentinel("Уже поставлен лайк"),
			ErrAllreadyClaimed:              sderr.Sentinel("Уже получен лайк"),
			ErrCheckInFailed:                sderr.Sentinel("Ошибка регистрации"),
			ErrSpotLikeFailed:               sderr.Sentinel("Ошибка лайка на точке регистрации"),
			ErrCheckInPointNotExist:         sderr.Sentinel("Точка регистрации не существует"),
			ErrCheckInOverLimit:             sderr.Sentinel("Превышено количество регистраций"),
			ErrCheckInOverLimitDaily:        sderr.Sentinel("Превышено количество регистраций в день"),
			ErrLikeInOverLimit:              sderr.Sentinel("Превышено количество лайков"),
			ErrUserNotVerifyLocation:        sderr.Sentinel("Обновите свое местоположение"),
			ErrCheckPointNotOpen:            sderr.Sentinel("Точка регистрации временно недоступна"),
			ErrRequestTooFast:               sderr.Sentinel("Слишком много запросов, повторите попытку позже"),
			ErrCheckInExistLock:             sderr.Sentinel("Текущая точка заблокирована другим пользователем, измените или повторите позже"),
			ErrCanOnlyClaimOnce:             sderr.Sentinel("Можно получать только один раз в день"),
			ErrLocationLatLonErr:            sderr.Sentinel("Ошибка широты или долготы"),
			ErrUserNotExist:                 sderr.Sentinel("Пользователь не существует"),
			ErrUserInfoNicknameTooLong:      sderr.Sentinel("Имя пользователя превышает максимальную длину"),
			ErrUserInfoSelectProperGender:   sderr.Sentinel("Выберите правильный пол"),
			ErrUserInfoMustNotEmpoty:        sderr.Sentinel("Имя пользователя не может быть пустым"),
			ErrRankingTypeErr:               sderr.Sentinel("Ошибка типа рейтинга"),
			ErrBalance:                      sderr.Sentinel("Недостаточно средств"),
			ErrLastedLevel:                  sderr.Sentinel("Уже достигнут максимальный уровень"),
			ErrCreateCity:                   sderr.Sentinel("Создание достопримечательностей временно недоступно"),
			ErrUserUnAuthorizedLocation:     sderr.Sentinel("Пожалуйста, сначала авторизуйте данные о местоположении"),
			ErrTaskAlreadyCompleted:         sderr.Sentinel("Задание выполнено"),
			ErrTaskNotFound:                 sderr.Sentinel("Не найдено соответствующее задание, проверьте параметры"),
			ErrSpotAlreadyHighestLevel:      sderr.Sentinel("Уже максимальный уровень"),
			ErrSpotOrderLimit:               sderr.Sentinel("Слишком много неоплаченных заказов, повторите позже"),
			ErrSpotFinishOrderLimit:         sderr.Sentinel("Достигнуто максимальное количество достопримечательностей"),
			ErrSpotOverlap:                  sderr.Sentinel("Текущая позиция перекрывается с уже созданной достопримечательностью, измените местоположение"),
			ErrSpotOrderDoing:               sderr.Sentinel("Заказ в процессе покупки"),
			ErrSpotOrderNotBuy:              sderr.Sentinel("Текущая точка заблокирована другим пользователем, покупка невозможна"),
			ErrSpotLandmarkProtectionPeriod: sderr.Sentinel("Текущая точка находится в периоде защиты, покупка невозможна"),
			ErrNotSpotWaitClaim:             sderr.Sentinel("Нет доходов для получения"),
			ErrSpotNameLength:               sderr.Sentinel("Название точки не должно превышать 50 символов"),
			ErrSpotTypeLimitByEvent:         sderr.Sentinel("Покупка недоступна"),
			ErrSpotOrderStatus:              sderr.Sentinel("Проверьте статус заказа"),
			ErrSpotOrderNotExist:            sderr.Sentinel("Заказ не существует"),
			ErrSpotOrderRefundNotAble:       sderr.Sentinel("Возврат в процессе или уже выполнен, проверьте данные"),
			ErrSpotNoRefundInfo:             sderr.Sentinel("Информация о возврате отсутствует"),
			ErrSpotCannotSubmitRepeatedly:   sderr.Sentinel("Не отправляйте заказ повторно"),
			ErrSpotCannotBuySelfSpot:        sderr.Sentinel("Вы уже являетесь владельцем достопримечательности"),
			ErrFileSizeLimit:                sderr.Sentinel("Лимит на изображение: 2 МБ"),
			ErrUserTransferToYourself:       sderr.Sentinel("Нельзя переводить себе"),
			ErrUserTransferAmountLimit:      sderr.Sentinel("Сумма перевода должна быть больше нуля"),
			ErrActivityNotExists:            sderr.Sentinel("Запись координат не существует"),
			ErrInviteSelf:                   sderr.Sentinel("Нельзя приглашать самого себя"),
			ErrSign:                         sderr.Sentinel("Ошибка подписи"),
			ErrAlreadySignIn:                sderr.Sentinel("Сегодня уже зарегистрировано"),
			ErrFetchStatus:                  sderr.Sentinel("Ошибка получения статуса"),
			ErrRepeatOperation:              sderr.Sentinel("Пожалуйста, не повторяйте операцию"),
			ErrAlreadyAddFriend:             sderr.Sentinel("Этот друг уже добавлен, повторное добавление не требуется"),
			ErrAuthLogin:                    sderr.Sentinel("Ошибка авторизации при входе"),
			ErrAuthLoginExpired:             sderr.Sentinel("Срок действия авторизации истёк"),
			ErrAccessDenied:                 sderr.Sentinel("Доступ запрещён"),
			ErrInviteExists:                 sderr.Sentinel("Уже привязан рекомендатель"),
			ErrAddInviteExpired:             sderr.Sentinel("Регистрация прошла более 24 часов, настройка невозможна"),
			ErrInviteEachOther:              sderr.Sentinel("Нельзя быть друг другу рекомендателями"),
			ErrWalletAddress:                sderr.Sentinel("Неверный формат адреса кошелька"),
			ErrRequestExpire:                sderr.Sentinel("Запрос истек"),
			ErrCaptcha:                      sderr.Sentinel("Captcha error"),
			ErrSpotBalanceNotCanCheckIn:     sderr.Sentinel("地标余额不足不能打卡"),
			ErrPasswordTooWeak:              sderr.Sentinel("Слишком слабый пароль"),
			ErrPasswordNotMatch:             sderr.Sentinel("Пароли не совпадают"),
			ErrOldPasswordRequired:          sderr.Sentinel("Требуется старый пароль"),
			ErrOldPasswordIncorrect:         sderr.Sentinel("Неверный старый пароль"),
			ErrPasswordNotSet:               sderr.Sentinel("Пароль не установлен, используйте код подтверждения"),
			ErrPasswordIncorrect:            sderr.Sentinel("Неверный пароль"),
			ErrUnsupportedFileType:          sderr.Sentinel("Неподдерживаемый тип файла"),
			ErrImageContentInappropriate:    sderr.Sentinel("Недопустимое содержание изображения, загрузите подходящее изображение"),
			ErrNotFoundProduct:              sderr.Sentinel("Продукт не существует"),
			ErrSpotNotExist:                 sderr.Sentinel("Место находится на рассмотрении"),
			ErrUploadLimit:                  sderr.Sentinel("Достигнут дневной лимит загрузки"),
			ErrOrderNotExist:                sderr.Sentinel("Заказ не существует"),
			ErrOrderStatusChanged:           sderr.Sentinel("Статус заказа изменился, проверьте"),
			ErrInsufficientBalance:          sderr.Sentinel("Недостаточно средств"),
			ErrWithdrawInProgress:           sderr.Sentinel("Вывод средств в процессе"),
			ErrVersionTooLow:                sderr.Sentinel("Версия слишком низкая, пожалуйста, обновите"),
			ErrTooManyRequests:              sderr.Sentinel("Сегодня сообщений отправлено слишком много, пожалуйста, попробуйте завтра"),
			ErrPayMethodNotExist:            sderr.Sentinel("支付方式不存在"),
			ErrBelieveChatLinkNotFound:      sderr.Sentinel("请检查Believe id是否正确"),
		},
	}
}

func GetErrCode(err error) int {
	code, exists := ErrCodeMap[err]
	if !exists {
		code = DefaultErrCode
	}
	return code
}

func GetErrMessage(err error) string {
	language := GetRequestLanguage()
	if langMap, exists := ErrMsgMap[language]; exists {
		if msg, ok := langMap[err]; ok {
			return msg.Error()
		}
	}
	// 如果指定语言中没有找到，使用默认语言
	if msg, ok := ErrMsgMap[DefaultLanguage][err]; ok {
		return msg.Error()
	}
	// 默认返回错误的原始消息
	return err.Error()
}
