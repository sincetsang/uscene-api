package main

import (
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/sdlog"
	"TMA/pkg/ueticket"
	"TMA/pkg/web/webapi"
	"errors"
)

type ueTmpTicketRequest struct {
	RequestID    string `json:"request_id"`
	ProjectID    string `json:"project_id"`
	ClientHardID string `json:"client_hardid"`
}

type ueRegisterRequest struct {
	RequestID string `json:"request_id"`
	FlowID    string `json:"flow_id"`
	ProjectID string `json:"project_id"`
	MobileNo  string `json:"mobileno"`
	AreaCode  string `json:"area_code"`
}

func getUETmpTicket(ec *middleware.AppRequestContext) error {
	var req ueTmpTicketRequest
	if err := ec.Bind(&req); err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	outcome, err := ueticket.New(ec.Nu).GetTmpTicket(ec.Request().Context(), ueticket.TicketRequest{
		RequestID: req.RequestID, ProjectID: req.ProjectID, ClientHardID: req.ClientHardID,
		ClientIP: ec.RealIP(), UserID: ec.AuthData.User.ID,
	})
	if err != nil {
		return renderUETicketError(ec, err)
	}
	return webapi.OK(outcome).Render(ec)
}

func registerUEAccount(ec *middleware.AppRequestContext) error {
	var req ueRegisterRequest
	if err := ec.Bind(&req); err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	outcome, err := ueticket.New(ec.Nu).Register(ec.Request().Context(), ueticket.RegisterRequest{
		RequestID: req.RequestID, FlowID: req.FlowID, ProjectID: req.ProjectID,
		MobileNo: req.MobileNo, AreaCode: req.AreaCode, UserID: ec.AuthData.User.ID,
	})
	if err != nil {
		return renderUETicketError(ec, err)
	}
	return webapi.OK(outcome).Render(ec)
}

func getUETmpTicketBindStatus(ec *middleware.AppRequestContext) error {
	outcome, err := ueticket.New(ec.Nu).BindStatus(
		ec.Request().Context(), ec.QueryParam("flow_id"), ec.AuthData.User.ID,
	)
	if err != nil {
		return renderUETicketError(ec, err)
	}
	return webapi.OK(outcome).Render(ec)
}

func renderUETicketError(ec *middleware.AppRequestContext, err error) error {
	switch {
	case errors.Is(err, common.ErrParam), errors.Is(err, common.ErrUnauthorized), errors.Is(err, common.ErrRequestTooFast), errors.Is(err, common.ErrRequestExpire):
		return webapi.Error(err).Render(ec)
	default:
		sdlog.WithError(err).Error("UE临时票据处理失败")
		return webapi.Error(common.ErrService).Render(ec)
	}
}
