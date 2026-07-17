package models

import "encoding/json"

// GetTmpTicketReq is the request used by the UE temporary-ticket business API.
// The exact upstream endpoint is configured by uscene because it is not part
// of the current Go SDK's published methods.
type GetTmpTicketReq struct {
	Opentype     int    `json:"opentype"`
	Nodecode     string `json:"nodecode"`
	ClientHardID string `json:"clienthardid"`
	ClientIP     string `json:"clientip"`
	ProjectID    string `json:"projectid,omitempty"`
}

// TmpTicketDto is intentionally tolerant of the fields commonly returned by
// the UE SDK. DeviceId is the temporary ticket consumed by LoginTmpTicketAsync.
type TmpTicketDto struct {
	DeviceID  string `json:"deviceid"`
	ExpiresIn int    `json:"expiresin"`
}

// UnmarshalJSON accepts the casing used by the different UE SDK serializers.
func (d *TmpTicketDto) UnmarshalJSON(data []byte) error {
	var value struct {
		DeviceIDLower string `json:"deviceid"`
		DeviceIDCamel string `json:"deviceId"`
		DeviceIDUpper string `json:"DeviceId"`
		Ticket        string `json:"ticket"`
		ExpiresLower  int    `json:"expiresin"`
		ExpiresCamel  int    `json:"expiresIn"`
		ExpiresUpper  int    `json:"ExpiresIn"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	d.DeviceID = value.DeviceIDLower
	if d.DeviceID == "" {
		d.DeviceID = value.DeviceIDCamel
	}
	if d.DeviceID == "" {
		d.DeviceID = value.DeviceIDUpper
	}
	if d.DeviceID == "" {
		d.DeviceID = value.Ticket
	}
	d.ExpiresIn = value.ExpiresLower
	if d.ExpiresIn == 0 {
		d.ExpiresIn = value.ExpiresCamel
	}
	if d.ExpiresIn == 0 {
		d.ExpiresIn = value.ExpiresUpper
	}
	return nil
}

// RegReq is the phone-based registration request shown in UeService.cs.
type RegReq struct {
	Opentype int    `json:"opentype"`
	OpenID   string `json:"openid"`
	MobileNo string `json:"mobileno"`
}

// RegDto contains the UE node code returned after registration.
type RegDto struct {
	Nodecode string `json:"nodecode"`
}

func (d *RegDto) UnmarshalJSON(data []byte) error {
	var value struct {
		NodecodeLower string `json:"nodecode"`
		NodecodeCamel string `json:"nodeCode"`
		NodecodeUpper string `json:"Nodecode"`
		NodeCodeUpper string `json:"NodeCode"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	d.Nodecode = value.NodecodeLower
	if d.Nodecode == "" {
		d.Nodecode = value.NodecodeCamel
	}
	if d.Nodecode == "" {
		d.Nodecode = value.NodecodeUpper
	}
	if d.Nodecode == "" {
		d.Nodecode = value.NodeCodeUpper
	}
	return nil
}

// RegZsUserReq is the project-specific registration request used when a
// project does not require an existing UE binding.
type RegZsUserReq struct {
	ClientIP     string  `json:"clientip"`
	ClientHardID string  `json:"clienthardid"`
	Token        string  `json:"token"`
	Addr         string  `json:"addr"`
	Nodename     string  `json:"nodename"`
	AvatarURL    *string `json:"avatarurl"`
}
