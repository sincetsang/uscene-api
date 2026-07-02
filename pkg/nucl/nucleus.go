package nucl

import (
	"TMA/pkg/sderr"
	"TMA/pkg/sdlog"
	"TMA/pkg/sdredis"

	"github.com/sulink/ueapisdk/models"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"

	"github.com/go-redis/redis_rate/v9"
	"gorm.io/gorm"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dm "github.com/alibabacloud-go/dm-20151123/v2/client"
	"github.com/alibabacloud-go/dysmsapi-20170525/v4/client"
	"github.com/alibabacloud-go/tea/tea"
)

var (
	// globalNucleus 全局变量
	globalNucleus *Nucleus
)

// GetGlobalNucleus 仅用于测试
func GetGlobalNucleus() *Nucleus {
	return globalNucleus
}

type Env string

func (e Env) IsStaging() bool {
	return e == EnvStaging
}

func (e Env) IsDevelop() bool {
	return e == EnvDevelop
}

func (e Env) IsOnline() bool {
	return e == EnvProduction
}

func (e Env) IsPreview() bool {
	return e == EnvPreview
}

const (
	// EnvDevelop 开发环境
	EnvDevelop = "develop"
	// EnvStaging 测试环境
	EnvStaging Env = "staging"
	// EnvPreview 线上预览环境
	EnvPreview Env = "preview"
	// EnvProduction 生产环境
	EnvProduction Env = "production"
)

// Nucleus 全局使用的资源
type Nucleus struct {
	Env              Env                      // 执行环境
	Options          Options                  // 各模块是否已经初始化
	Config           *Config                  // 配置
	DB               *gorm.DB                 // Mysql Postgres
	RedisClient      *sdredis.Client          // redis client
	HwObsClient      *obs.ObsClient           // obs client
	AwsClient        *rekognition.Rekognition // aws client
	AwsSession       *session.Session         // aws session
	RedisRateLimiter *redis_rate.Limiter      // redis的频控器
	Snowflake        *Snowflake               // ID生成器
	GoogleMapsKey    string
	OpenaiApiKey     string
	WalletJwtSecret  string
	ChanTask         chan string
	AliSMSClient     *client.Client
	UeParam          map[string]interface{} // UE API配置参数
	UeConfigParam    models.UeConfigParam2  // UE API配置参数
	EmailClient      *dm.Client
	BelieveClient    *BelieveClient
}

type Options struct {
	DB             bool
	Redis          bool
	DisableInitLog bool
	HwObs          bool
	Aws            bool
	Snowflake      bool
	GormLogger     string
	ServiceName    string
	UE             bool // 是否初始化UE API
	Email          bool `toml:"email"`
	Believe        bool `toml:"believe"`
}

func (o *Options) SetDB(enable bool) *Options {
	if o == nil {
		return o
	}

	o.DB = enable

	return o
}

func (o *Options) SetRedis(enable bool) *Options {
	if o == nil {
		return o
	}

	o.Redis = enable

	return o
}

func InitGlobalNucleusByConfig(config Config, opts Options) (*Nucleus, error) {
	nu, err := NewByConfig(config, opts)
	if err != nil {
		return nil, sderr.WithStack(err)
	}

	globalNucleus = nu
	return nu, nil
}

func InitGlobalNucleusByConfigFile(loc string, opts Options) (*Nucleus, error) {

	config, err := LoadConfig(loc, opts.ServiceName)
	if err != nil {
		return nil, sderr.WithStack(err)
	}

	return InitGlobalNucleusByConfig(*config, opts)
}

func NewByConfig(config Config, opts Options) (*Nucleus, error) {
	var err error
	nu := Nucleus{
		Env:             config.Env,
		Options:         opts,
		Config:          &config,
		GoogleMapsKey:   config.GoogleMapsKey,
		WalletJwtSecret: config.WalletJwtSecret,
		OpenaiApiKey:    config.OpenaiApiKey,
	}

	// Logging
	config.setupLogging()

	// DB
	err = nu.initDB()
	if err != nil {
		return nil, sderr.WithStack(err)
	}

	// Redis
	err = nu.initRedis()
	if err != nil {
		return nil, sderr.WithStack(err)
	}

	err = nu.initHwObs()
	if err != nil {
		return nil, sderr.WithStack(err)
	}

	err = nu.initAws()
	if err != nil {
		return nil, sderr.WithStack(err)
	}

	err = nu.initSnowflake()
	if err != nil {
		return nil, sderr.WithStack(err)
	}

	err = nu.initAliSMS()
	if err != nil {
		return nil, sderr.WithStack(err)
	}

	err = nu.initBelieve()
	if err != nil {
		return nil, sderr.WithStack(err)
	}

	// Email
	if opts.Email {
		err = nu.initEmail()
		if err != nil {
			return nil, sderr.WithStack(err)
		}
	}

	// UE API
	if opts.UE {
		err = nu.InitUE()
		if err != nil {
			return nil, sderr.WithStack(err)
		}
	}

	// All OK
	return &nu, nil
}

func (nu *Nucleus) withLog(stage string, action func() error) error {
	err := action()
	if err != nil {
		if !nu.Options.DisableInitLog {
			sdlog.Errorf("init %s error :-(", stage)
		}
		return sderr.WithStack(err)
	}
	if !nu.Options.DisableInitLog {
		sdlog.Infof("init %s done", stage)
	}
	return nil
}

// Close 程序停止时关闭
func (nu *Nucleus) Close() {
	if nu == nil {
		return
	}

	if nu.Options.DB {
		conn, err := nu.DB.DB()
		if err != nil {
			sdlog.WithError(err).Error("get nu.DB mysql conn failed")
		} else {
			if err := conn.Close(); err != nil {
				sdlog.WithError(err).Error("close nu.DB mysql conn failed")
			}
		}
	}
}

func (nu *Nucleus) initAliSMS() error {
	return nu.withLog("AliSMS", func() error {
		config := &openapi.Config{
			AccessKeyId:     tea.String(nu.Config.AliSMS.AccessKeyID),
			AccessKeySecret: tea.String(nu.Config.AliSMS.AccessKeySecret),
			Endpoint:        tea.String("dysmsapi.aliyuncs.com"),
		}
		client, err := client.NewClient(config)
		if err != nil {
			return err
		}
		nu.AliSMSClient = client
		return nil
	})
}
