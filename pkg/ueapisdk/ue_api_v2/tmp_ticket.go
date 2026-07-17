package ue_api_v2

import (
	"github.com/sulink/ueapisdk/models"
	"github.com/sulink/ueapisdk/utils/constants"
)

// GetTmpTicket calls the configured UE temporary-ticket endpoint.
// The endpoint is supplied by the caller because the current Go SDK does not
// have a canonical public path for this API.
func GetTmpTicket(endpoint string, req models.GetTmpTicketReq, config models.UeConfigParam2) (models.UERespbaseV2[models.TmpTicketDto], error) {
	return post[models.GetTmpTicketReq, models.UERespbaseV2[models.TmpTicketDto]](
		endpoint,
		req,
		businessConfig(config),
	)
}

// RegisterUE calls the phone-based UE registration endpoint.
func RegisterUE(endpoint string, req models.RegReq, config models.UeConfigParam2) (models.UERespbaseV2[models.RegDto], error) {
	return post[models.RegReq, models.UERespbaseV2[models.RegDto]](
		endpoint,
		req,
		businessConfig(config),
	)
}

// RegisterZsUser calls the project-specific registration endpoint used by
// projects that do not require a pre-existing UE binding.
func RegisterZsUser(endpoint string, req models.RegZsUserReq, config models.UeConfigParam2) (models.UERespbaseV2[models.TmpTicketDto], error) {
	return post[models.RegZsUserReq, models.UERespbaseV2[models.TmpTicketDto]](
		endpoint,
		req,
		businessConfig(config),
	)
}

func businessConfig(config models.UeConfigParam2) models.UeConfigParam {
	return models.UeConfigParam{
		Aeskey:     config.Aeskey,
		Privatekey: config.Privatekey,
		Header:     map[string]string{constants.ACCESS_TOKEN: config.Accesstoken},
	}
}
