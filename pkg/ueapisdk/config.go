package ueapisdk

// 支付配置参数
var (
	// 码库接口地址
	// 测试环境：http://client2.ue.shumaoheng.com
	// 生产环境：https://uev2.dtwy.xyz
	UeApiUrl = "http://client2.ue.shumaoheng.com"

	// 码库WEB支付页面地址
	// 测试环境：http://res.ue.shumaoheng.com/ueres/pay.html
	// 生产环境：https://ue-res.dtwy.xyz/ueres/pay.html
	UePayUrl = "http://res.ue.shumaoheng.com/ueres/pay.html"

	// 凭证信息（测试环境默认值）
	AccessKeyId    = "GEECBO9JY8IPI9CMCCJU4AB8PHOP41VA"
	AccessSecret   = "38EN519B5IHM30EUALP9ZL25V32GP5U2"
	PayCode        = "HXCNALXIV0H7AP3Z3EY4NXGYBG3U63DM"
	BusinessTypeId = 70003
	HardId         = "8e22f632bc6341bdb63647a2fe12e60c"
	MyDomain       = "http://localhost:5004"
)
