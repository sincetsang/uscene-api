package nucl

import (
	"TMA/pkg/sderr"
	"TMA/pkg/sdgorm"
	"TMA/pkg/sdload"
	"TMA/pkg/sdlog"
	"TMA/pkg/sdredis"
	"TMA/service/obs"
	"time"
)

type Config struct {
	Env             Env             `toml:"env"`
	ServiceName     string          `toml:"service_name"`
	Log             sdlog.Options   `toml:"log"`
	DB              sdgorm.Address  `toml:"db"`
	Redis           sdredis.Address `toml:"redis"`
	HwObs           obs.Address     `toml:"hw_obs"`
	Aws             AwsConfig       `toml:"aws"`
	GoogleMapsKey   string          `toml:"google_maps_key"`
	OpenaiApiKey    string          `toml:"openai_api_key"`
	WalletJwtSecret string          `toml:"wallet_jwt_secret"`
	IpLocationDB    string          `toml:"ip_location_db"`
	Email           EmailConfig     `toml:"email"`
	UE              struct {
		UeApiUrl       string            `toml:"ue_api_url"`
		AccessKeyId    string            `toml:"access_key_id"`
		AccessSecret   string            `toml:"access_secret"`
		PayCode        string            `toml:"pay_code"`
		BusinessTypeid int               `toml:"business_typeid"`
		MyDomain       string            `toml:"my_domain"`
		TmpTicket      UETmpTicketConfig `toml:"tmp_ticket"`
	} `toml:"ue"`
	GoogleOAuth struct {
		Android struct {
			ClientID string `toml:"client_id"`
		} `toml:"android"`
		IOS struct {
			ClientID string `toml:"client_id"`
		} `toml:"ios"`
		Web struct {
			ClientID string `toml:"client_id"`
		} `toml:"web"`
		JwtSecret string `toml:"jwt_secret"`
	} `toml:"google_oauth"`

	AppleOAuth struct {
		BundleID string `toml:"bundle_id"`
		TeamID   string `toml:"team_id"`
		KeyID    string `toml:"key_id"`
	} `toml:"apple_oauth"`
	AliSMS struct {
		AccessKeyID     string `toml:"access_key_id"`
		AccessKeySecret string `toml:"access_key_secret"`
		SignName        string `toml:"sign_name"`
		TemplateCode    string `toml:"template_code"`
	} `toml:"ali_sms"`
	Believe struct {
		Domain string `toml:"domain"`
		Secret string `toml:"secret"`
	} `toml:"believe"`
}

// UETmpTicketConfig configures the temporary-ticket integration.
// Upstream URLs are intentionally explicit because the current Go SDK does
// not contain the temporary-ticket endpoint definitions.
type UETmpTicketConfig struct {
	Enabled          bool                       `toml:"enabled"`
	TicketTTLSeconds int                        `toml:"ticket_ttl_seconds"`
	FlowTTLSeconds   int                        `toml:"flow_ttl_seconds"`
	GetTicketURL     string                     `toml:"get_ticket_url"`
	RegisterURL      string                     `toml:"register_url"`
	RegisterZsURL    string                     `toml:"register_zs_url"`
	Projects         []UETmpTicketProjectConfig `toml:"projects"`
}

// UETmpTicketProjectConfig mirrors the project policy in TmpTicketProjectOpt.cs.
type UETmpTicketProjectConfig struct {
	ProjectID             string `toml:"project_id"`
	TokenType             string `toml:"token_type"`
	AutoRegUEAccount      bool   `toml:"auto_reg_ue_account"`
	RequiredBindUEAccount bool   `toml:"required_bind_ue_account"`
}

func LoadConfig(loc string, serviceName string) (*Config, error) {
	var config Config
	config.ServiceName = serviceName
	err := sdload.Toml(loc, &config)
	if err != nil {
		return nil, sderr.WithStack(err)
	}
	return &config, nil
}

func (c *Config) setupLogging() {
	c.Log.Filename = c.Log.Filename + time.Now().Format("2006-01-02") + ".log"
	c.Log.ServiceName = c.ServiceName
	sdlog.Init(c.Log)
}
