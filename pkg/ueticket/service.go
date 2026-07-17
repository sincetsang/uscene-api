package ueticket

import (
	"TMA/pkg/common"
	"TMA/pkg/nucl"
	"TMA/pkg/sderr"
	"TMA/service/db"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/sulink/ueapisdk/models"
	"github.com/sulink/ueapisdk/ue_api_v2"
	"gorm.io/gorm"
)

const (
	defaultTicketTTL = 120 * time.Second
	defaultFlowTTL   = 5 * time.Minute
	flowStatusKey    = "ue:tmp-ticket:flow:"
	ticketKeyPrefix  = "ue:tmp-ticket:"
	requestKeyPrefix = "ue:tmp-ticket:request:"
	lockKeyPrefix    = "ue:tmp-ticket:lock:"
)

type TicketRequest struct {
	RequestID    string `json:"request_id"`
	ProjectID    string `json:"project_id"`
	ClientHardID string `json:"client_hardid"`
	ClientIP     string `json:"-"`
	UserID       int64  `json:"-"`
}

type RegisterRequest struct {
	RequestID string `json:"request_id"`
	FlowID    string `json:"flow_id"`
	ProjectID string `json:"project_id"`
	MobileNo  string `json:"mobileno"`
	AreaCode  string `json:"area_code"`
	UserID    int64  `json:"-"`
}

type TicketOutcome struct {
	FlowID    string `json:"flow_id"`
	Result    int    `json:"result"`
	Message   string `json:"message,omitempty"`
	Action    string `json:"action,omitempty"`
	DeviceID  string `json:"device_id,omitempty"`
	ExpiresIn int    `json:"expires_in,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
}

type BindStatusOutcome struct {
	FlowID      string `json:"flow_id"`
	Status      string `json:"status"`
	RetryTicket bool   `json:"retry_ticket"`
}

type ticketRecord struct {
	FlowID    string    `json:"flow_id"`
	DeviceID  string    `json:"device_id"`
	ProjectID string    `json:"project_id"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type flowRecord struct {
	FlowID           string `json:"flow_id"`
	UserID           int64  `json:"user_id"`
	ProjectID        string `json:"project_id"`
	ClientHardIDHash string `json:"client_hardid_hash"`
	Status           string `json:"status"`
	Result           int    `json:"result"`
	RequestID        string `json:"request_id"`
}

type Service struct {
	nu *nucl.Nucleus
}

func New(nu *nucl.Nucleus) *Service {
	return &Service{nu: nu}
}

func (s *Service) GetTmpTicket(ctx context.Context, req TicketRequest) (TicketOutcome, error) {
	if err := s.ensureRedis(); err != nil {
		return TicketOutcome{}, err
	}
	if err := s.validateTicketRequest(req); err != nil {
		return TicketOutcome{}, err
	}
	project, err := s.project(req.ProjectID)
	if err != nil {
		return TicketOutcome{}, err
	}
	if err := validateEndpoint(s.nu.Config.UE.TmpTicket.GetTicketURL, "UE 临时票据接口"); err != nil {
		return TicketOutcome{}, err
	}
	if err := s.validateUEConfig(); err != nil {
		return TicketOutcome{}, err
	}

	hardIDHash := hashClientHardID(req.ClientHardID)
	ticketKey := s.ticketKey(req.UserID, req.ProjectID, hardIDHash)
	if outcome, ok := s.cachedTicket(ctx, ticketKey); ok {
		return outcome, nil
	}

	lockKey := lockKeyPrefix + strconv.FormatInt(req.UserID, 10) + ":" + req.ProjectID + ":" + hardIDHash
	lockValue := uuid.NewString()
	lockTTL := 30 * time.Second
	locked, err := s.nu.RedisClient.SetNX(ctx, lockKey, lockValue, lockTTL).Result()
	if err != nil {
		return TicketOutcome{}, err
	}
	if !locked {
		return TicketOutcome{}, common.ErrRequestTooFast
	}
	defer s.releaseLock(lockKey, lockValue)

	if outcome, ok := s.cachedTicket(ctx, ticketKey); ok {
		return outcome, nil
	}

	binding, err := db.GetUserUeSecret(s.nu.DB, req.UserID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return TicketOutcome{}, err
	}

	flowID := uuid.NewString()
	if project.RequiredBindUEAccount && err == gorm.ErrRecordNotFound {
		if req.RequestID != "" {
			if outcome, ok := s.cachedRequest(ctx, s.requestKey(req.UserID, req.ProjectID, req.RequestID)); ok {
				return outcome, nil
			}
		}
		result := -13
		action := "bind"
		if project.AutoRegUEAccount {
			result = -14
			action = "bind_or_register"
		}
		outcome := TicketOutcome{FlowID: flowID, Result: result, Action: action, ProjectID: req.ProjectID}
		if err := s.saveFlow(ctx, flowRecord{
			FlowID: flowID, UserID: req.UserID, ProjectID: req.ProjectID,
			ClientHardIDHash: hardIDHash, Status: "NEED_BIND", Result: result,
			RequestID: req.RequestID,
		}); err != nil {
			return TicketOutcome{}, err
		}
		if req.RequestID != "" {
			if err := s.saveRequest(ctx, req.UserID, req.ProjectID, req.RequestID, outcome); err != nil {
				return TicketOutcome{}, err
			}
		}
		return outcome, nil
	}

	nodecode := strconv.FormatInt(req.UserID, 10)
	if binding != nil && binding.Opennodecode != "" {
		nodecode = binding.Opennodecode
	}

	var deviceID string
	expiresIn := s.ticketTTL()
	if project.RequiredBindUEAccount {
		resp, callErr := ue_api_v2.GetTmpTicket(
			s.nu.Config.UE.TmpTicket.GetTicketURL,
			models.GetTmpTicketReq{
				Opentype: 10, Nodecode: nodecode, ClientHardID: req.ClientHardID,
				ClientIP: req.ClientIP, ProjectID: req.ProjectID,
			},
			s.nu.UeConfigParam,
		)
		if callErr != nil {
			return TicketOutcome{}, callErr
		}
		if resp.Result <= 0 {
			outcome := TicketOutcome{FlowID: flowID, Result: resp.Result, Message: resp.Message, ProjectID: req.ProjectID}
			status := "FAILED"
			if resp.Result == -13 || resp.Result == -14 {
				status = "NEED_BIND"
			}
			if err := s.saveFlow(ctx, flowRecord{
				FlowID: flowID, UserID: req.UserID, ProjectID: req.ProjectID,
				ClientHardIDHash: hardIDHash, Status: status, Result: resp.Result,
				RequestID: req.RequestID,
			}); err != nil {
				return TicketOutcome{}, err
			}
			if req.RequestID != "" {
				if err := s.saveRequest(ctx, req.UserID, req.ProjectID, req.RequestID, outcome); err != nil {
					return TicketOutcome{}, err
				}
			}
			return outcome, nil
		}
		deviceID = resp.Data.DeviceID
		if resp.Data.ExpiresIn > 0 && time.Duration(resp.Data.ExpiresIn)*time.Second < expiresIn {
			expiresIn = time.Duration(resp.Data.ExpiresIn) * time.Second
		}
	} else {
		user, userErr := db.UserInfoById(s.nu.DB, req.UserID)
		if userErr != nil {
			return TicketOutcome{}, userErr
		}
		if err := validateEndpoint(s.nu.Config.UE.TmpTicket.RegisterZsURL, "UE ZS 注册接口"); err != nil {
			return TicketOutcome{}, err
		}
		resp, callErr := ue_api_v2.RegisterZsUser(
			s.nu.Config.UE.TmpTicket.RegisterZsURL,
			models.RegZsUserReq{
				ClientIP: req.ClientIP, ClientHardID: req.ClientHardID,
				Token: project.TokenType, Addr: nodecode, Nodename: user.Nickname,
				AvatarURL: nullableString(user.AvatarURL),
			},
			s.nu.UeConfigParam,
		)
		if callErr != nil {
			return TicketOutcome{}, callErr
		}
		if resp.Result <= 0 {
			outcome := TicketOutcome{FlowID: flowID, Result: resp.Result, Message: resp.Message, ProjectID: req.ProjectID}
			status := "FAILED"
			if resp.Result == -13 || resp.Result == -14 {
				status = "NEED_BIND"
			}
			if err := s.saveFlow(ctx, flowRecord{
				FlowID: flowID, UserID: req.UserID, ProjectID: req.ProjectID,
				ClientHardIDHash: hardIDHash, Status: status, Result: resp.Result,
				RequestID: req.RequestID,
			}); err != nil {
				return TicketOutcome{}, err
			}
			if req.RequestID != "" {
				if err := s.saveRequest(ctx, req.UserID, req.ProjectID, req.RequestID, outcome); err != nil {
					return TicketOutcome{}, err
				}
			}
			return outcome, nil
		}
		deviceID = resp.Data.DeviceID
		if resp.Data.ExpiresIn > 0 && time.Duration(resp.Data.ExpiresIn)*time.Second < expiresIn {
			expiresIn = time.Duration(resp.Data.ExpiresIn) * time.Second
		}
	}

	if strings.TrimSpace(deviceID) == "" {
		return TicketOutcome{}, sderr.New("UE 临时票据响应缺少 deviceId")
	}

	issuedAt := time.Now().UTC()
	flow := flowRecord{
		FlowID: flowID, UserID: req.UserID, ProjectID: req.ProjectID,
		ClientHardIDHash: hardIDHash, Status: "TICKET_ISSUED", Result: 1,
		RequestID: req.RequestID,
	}
	record := ticketRecord{
		FlowID: flowID, DeviceID: deviceID, ProjectID: req.ProjectID,
		IssuedAt: issuedAt, ExpiresAt: issuedAt.Add(expiresIn),
	}
	if err := s.saveTicket(ctx, ticketKey, record, expiresIn); err != nil {
		return TicketOutcome{}, err
	}
	if err := s.saveFlow(ctx, flow); err != nil {
		return TicketOutcome{}, err
	}

	return TicketOutcome{
		FlowID: flowID, Result: 1, DeviceID: deviceID,
		ExpiresIn: int(expiresIn / time.Second), ProjectID: req.ProjectID,
	}, nil
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (TicketOutcome, error) {
	if err := s.ensureRedis(); err != nil {
		return TicketOutcome{}, err
	}
	if req.UserID <= 0 || strings.TrimSpace(req.RequestID) == "" || strings.TrimSpace(req.MobileNo) == "" {
		return TicketOutcome{}, common.ErrParam
	}
	if strings.TrimSpace(req.FlowID) == "" && strings.TrimSpace(req.ProjectID) == "" {
		return TicketOutcome{}, common.ErrParam
	}
	if err := validateEndpoint(s.nu.Config.UE.TmpTicket.RegisterURL, "UE 注册接口"); err != nil {
		return TicketOutcome{}, err
	}
	if err := s.validateUEConfig(); err != nil {
		return TicketOutcome{}, err
	}

	projectID := strings.TrimSpace(req.ProjectID)
	flowID := strings.TrimSpace(req.FlowID)
	if flowID != "" {
		flow, err := s.loadFlow(ctx, flowID)
		if err != nil {
			return TicketOutcome{}, err
		}
		if flow.UserID != req.UserID {
			return TicketOutcome{}, common.ErrUnauthorized
		}
		if strings.TrimSpace(req.ProjectID) != "" && req.ProjectID != flow.ProjectID {
			return TicketOutcome{}, common.ErrParam
		}
		projectID = flow.ProjectID
	}
	project, err := s.project(projectID)
	if err != nil {
		return TicketOutcome{}, err
	}
	if !project.AutoRegUEAccount {
		return TicketOutcome{}, sderr.New("当前项目未开启 UE 自动注册")
	}
	registerRequestKey := s.requestKey(req.UserID, projectID, "register:"+req.RequestID)
	if outcome, ok := s.cachedRequest(ctx, registerRequestKey); ok {
		return outcome, nil
	}

	hardIDHash := "unknown"
	if flowID != "" {
		flow, loadErr := s.loadFlow(ctx, flowID)
		if loadErr != nil {
			return TicketOutcome{}, loadErr
		}
		hardIDHash = flow.ClientHardIDHash
	}
	lockKey := lockKeyPrefix + strconv.FormatInt(req.UserID, 10) + ":" + projectID + ":register"
	lockValue := uuid.NewString()
	locked, err := s.nu.RedisClient.SetNX(ctx, lockKey, lockValue, 30*time.Second).Result()
	if err != nil {
		return TicketOutcome{}, err
	}
	if !locked {
		return TicketOutcome{}, common.ErrRequestTooFast
	}
	defer s.releaseLock(lockKey, lockValue)

	nodecode := strconv.FormatInt(req.UserID, 10)
	if binding, bindingErr := db.GetUserUeSecret(s.nu.DB, req.UserID); bindingErr == nil && binding.Opennodecode != "" {
		nodecode = binding.Opennodecode
	}
	resp, callErr := ue_api_v2.RegisterUE(
		s.nu.Config.UE.TmpTicket.RegisterURL,
		models.RegReq{Opentype: 10, OpenID: nodecode, MobileNo: req.MobileNo},
		s.nu.UeConfigParam,
	)
	if callErr != nil {
		return TicketOutcome{}, callErr
	}
	if resp.Result <= 0 {
		outcome := TicketOutcome{FlowID: flowID, Result: resp.Result, Message: resp.Message, ProjectID: projectID}
		if err := s.saveRequest(ctx, req.UserID, projectID, "register:"+req.RequestID, outcome); err != nil {
			return TicketOutcome{}, err
		}
		return outcome, nil
	}
	if strings.TrimSpace(resp.Data.Nodecode) == "" {
		return TicketOutcome{}, sderr.New("UE 注册响应缺少 nodecode")
	}
	if err := db.BindUserUeSecret(s.nu.DB, req.UserID, nodecode, resp.Data.Nodecode); err != nil {
		return TicketOutcome{}, err
	}
	if flowID != "" {
		if err := s.saveFlow(ctx, flowRecord{
			FlowID: flowID, UserID: req.UserID, ProjectID: projectID,
			ClientHardIDHash: hardIDHash, Status: "SUCCEEDED", Result: 1,
			RequestID: req.RequestID,
		}); err != nil {
			return TicketOutcome{}, err
		}
	}
	outcome := TicketOutcome{FlowID: flowID, Result: 1, ProjectID: projectID}
	if err := s.saveRequest(ctx, req.UserID, projectID, "register:"+req.RequestID, outcome); err != nil {
		return TicketOutcome{}, err
	}
	return outcome, nil
}

func (s *Service) BindStatus(ctx context.Context, flowID string, userID int64) (BindStatusOutcome, error) {
	if err := s.ensureRedis(); err != nil {
		return BindStatusOutcome{}, err
	}
	if strings.TrimSpace(flowID) == "" || userID <= 0 {
		return BindStatusOutcome{}, common.ErrParam
	}
	flow, err := s.loadFlow(ctx, flowID)
	if err != nil {
		if errors.Is(err, common.ErrRequestExpire) {
			return BindStatusOutcome{FlowID: flowID, Status: "expired", RetryTicket: false}, nil
		}
		return BindStatusOutcome{}, err
	}
	if flow.UserID != userID {
		return BindStatusOutcome{}, common.ErrUnauthorized
	}
	if _, err := db.GetUserUeSecret(s.nu.DB, userID); err == nil {
		return BindStatusOutcome{FlowID: flowID, Status: "bound", RetryTicket: true}, nil
	} else if err != gorm.ErrRecordNotFound {
		return BindStatusOutcome{}, err
	}
	status := strings.ToLower(flow.Status)
	if flow.Status == "NEED_BIND" {
		status = "pending"
	}
	return BindStatusOutcome{FlowID: flowID, Status: status, RetryTicket: false}, nil
}

func (s *Service) validateTicketRequest(req TicketRequest) error {
	if req.UserID <= 0 || strings.TrimSpace(req.RequestID) == "" || strings.TrimSpace(req.ProjectID) == "" || strings.TrimSpace(req.ClientHardID) == "" {
		return common.ErrParam
	}
	if len(req.ClientHardID) > 256 || len(req.RequestID) > 128 {
		return common.ErrParam
	}
	return nil
}

func (s *Service) ensureRedis() error {
	if s == nil || s.nu == nil || s.nu.RedisClient == nil {
		return sderr.New("Redis 未初始化")
	}
	return nil
}

func (s *Service) project(projectID string) (*nucl.UETmpTicketProjectConfig, error) {
	if s.nu == nil || s.nu.Config == nil || !s.nu.Config.UE.TmpTicket.Enabled {
		return nil, sderr.New("UE 临时票据功能未启用")
	}
	for i := range s.nu.Config.UE.TmpTicket.Projects {
		project := &s.nu.Config.UE.TmpTicket.Projects[i]
		if project.ProjectID == projectID {
			return project, nil
		}
	}
	return nil, sderr.New("不支持的 UE 临时票据项目")
}

func (s *Service) validateUEConfig() error {
	if strings.TrimSpace(s.nu.UeConfigParam.Aeskey) == "" ||
		strings.TrimSpace(s.nu.UeConfigParam.Privatekey) == "" ||
		strings.TrimSpace(s.nu.UeConfigParam.Accesstoken) == "" {
		return sderr.New("UE 登录配置未完成")
	}
	return nil
}

func validateEndpoint(endpoint, name string) error {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return sderr.New(name + "地址配置无效")
	}
	return nil
}

func (s *Service) ticketTTL() time.Duration {
	if seconds := s.nu.Config.UE.TmpTicket.TicketTTLSeconds; seconds > 0 {
		ttl := time.Duration(seconds) * time.Second
		if ttl < defaultTicketTTL {
			return ttl
		}
	}
	return defaultTicketTTL
}

func (s *Service) flowTTL() time.Duration {
	if seconds := s.nu.Config.UE.TmpTicket.FlowTTLSeconds; seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return defaultFlowTTL
}

func (s *Service) ticketKey(userID int64, projectID, hardIDHash string) string {
	return ticketKeyPrefix + strconv.FormatInt(userID, 10) + ":" + projectID + ":" + hardIDHash
}

func (s *Service) requestKey(userID int64, projectID, requestID string) string {
	return requestKeyPrefix + strconv.FormatInt(userID, 10) + ":" + projectID + ":" + hashClientHardID(requestID)
}

func hashClientHardID(value string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(digest[:])
}

func (s *Service) cachedTicket(ctx context.Context, key string) (TicketOutcome, bool) {
	value, err := s.nu.RedisClient.Get(ctx, key).Result()
	if err != nil {
		return TicketOutcome{}, false
	}
	var record ticketRecord
	if json.Unmarshal([]byte(value), &record) != nil || record.DeviceID == "" {
		return TicketOutcome{}, false
	}
	ttl, err := s.nu.RedisClient.TTL(ctx, key).Result()
	if err != nil || ttl <= 0 {
		return TicketOutcome{}, false
	}
	return TicketOutcome{
		FlowID: record.FlowID, Result: 1, DeviceID: record.DeviceID,
		ExpiresIn: int(ttl / time.Second), ProjectID: record.ProjectID,
	}, true
}

func (s *Service) cachedRequest(ctx context.Context, key string) (TicketOutcome, bool) {
	value, err := s.nu.RedisClient.Get(ctx, key).Result()
	if err != nil {
		return TicketOutcome{}, false
	}
	var outcome TicketOutcome
	if json.Unmarshal([]byte(value), &outcome) != nil || outcome.FlowID == "" {
		return TicketOutcome{}, false
	}
	return outcome, true
}

func (s *Service) saveTicket(ctx context.Context, key string, record ticketRecord, ttl time.Duration) error {
	value, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return s.nu.RedisClient.Set(ctx, key, value, ttl).Err()
}

func (s *Service) saveRequest(ctx context.Context, userID int64, projectID, requestID string, outcome TicketOutcome) error {
	value, err := json.Marshal(outcome)
	if err != nil {
		return err
	}
	return s.nu.RedisClient.Set(ctx, s.requestKey(userID, projectID, requestID), value, s.flowTTL()).Err()
}

func (s *Service) saveFlow(ctx context.Context, flow flowRecord) error {
	value, err := json.Marshal(flow)
	if err != nil {
		return err
	}
	return s.nu.RedisClient.Set(ctx, flowStatusKey+flow.FlowID, value, s.flowTTL()).Err()
}

func (s *Service) loadFlow(ctx context.Context, flowID string) (flowRecord, error) {
	value, err := s.nu.RedisClient.Get(ctx, flowStatusKey+flowID).Result()
	if err != nil {
		if err == redis.Nil {
			return flowRecord{}, common.ErrRequestExpire
		}
		return flowRecord{}, err
	}
	var flow flowRecord
	if err := json.Unmarshal([]byte(value), &flow); err != nil {
		return flowRecord{}, err
	}
	return flow, nil
}

func (s *Service) releaseLock(key, value string) {
	const script = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`
	_, _ = s.nu.RedisClient.Eval(context.Background(), script, []string{key}, value).Result()
}

func nullableString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
