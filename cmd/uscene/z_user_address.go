package main

import (
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/sdlog"
	"TMA/pkg/web/webapi"
	"TMA/service/db/model"
)

type AddAddressRequest struct {
	Phone     string `json:"phone"`
	Name      string `json:"name"`
	Area      string `json:"area"`
	Detail    string `json:"address"`
	IsDefault bool   `json:"is_default"`
}

func addAddress(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	if userId == 0 {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}

	var req AddAddressRequest
	err := ec.Bind(&req)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	sdlog.Info("addAddress: %d, %s, %s, %s, %s, %t", userId, req.Phone, req.Name, req.Area, req.Detail, req.IsDefault)
	if req.Phone == "" || req.Area == "" || req.Detail == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	if req.IsDefault {
		err := ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.Address{}).
			Where("user_id = ?", userId).
			Where("is_default = ?", true).
			Update("is_default", false).Error
		if err != nil {
			sdlog.Error("update address is default error: %s", err)
			return webapi.Error(common.ErrService).Render(ec)
		}
	}

	address := &model.Address{
		UserID:    userId,
		Phone:     req.Phone,
		Name:      req.Name,
		Area:      req.Area,
		Detail:    req.Detail,
		IsDefault: req.IsDefault,
	}
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.Address{}).Create(address).Error
	if err != nil {
		sdlog.Error("create address error: %s", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(true).Render(ec)
}

type UpdateAddressRequest struct {
	AddressID int64  `json:"address_id"`
	Phone     string `json:"phone"`
	Name      string `json:"name"`
	Area      string `json:"area"`
	Detail    string `json:"address"`
	IsDefault bool   `json:"is_default"`
}

func updateAddress(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	if userId == 0 {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}
	var req UpdateAddressRequest
	err := ec.Bind(&req)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	if req.AddressID == 0 {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	if req.IsDefault {
		err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.Address{}).
			Where("user_id = ?", userId).
			Where("is_default = ?", true).
			Update("is_default", false).Error
		if err != nil {
			return webapi.Error(common.ErrService).Render(ec)
		}
	}

	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.Address{}).
		Where("id = ?", req.AddressID).
		Where("user_id = ?", userId).
		Updates(map[string]interface{}{
			"phone":      req.Phone,
			"name":       req.Name,
			"area":       req.Area,
			"detail":     req.Detail,
			"is_default": req.IsDefault, // 即使是 false 也会更新
		}).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(true).Render(ec)
}

type AddressListResponse struct {
	AddressID int64  `json:"address_id"`
	Phone     string `json:"phone"`
	Name      string `json:"name"`
	Area      string `json:"area"`
	Detail    string `json:"address"`
	IsDefault bool   `json:"is_default"`
}

func getAddressList(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	if userId == 0 {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}

	addresses := []model.Address{}

	err := ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.Address{}).
		Where("user_id = ?", userId).
		Find(&addresses).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	addressList := []AddressListResponse{}
	for _, address := range addresses {
		addressList = append(addressList, AddressListResponse{
			AddressID: address.ID,
			Phone:     address.Phone,
			Name:      address.Name,
			Area:      address.Area,
			Detail:    address.Detail,
			IsDefault: address.IsDefault,
		})
	}
	return webapi.OK(addressList).Render(ec)
}

type DeleteAddressRequest struct {
	AddressID int64 `json:"address_id"`
}

func deleteAddress(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	if userId == 0 {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}

	var req DeleteAddressRequest
	err := ec.Bind(&req)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	var count int64
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.Address{}).
		Where("id = ?", req.AddressID).
		Where("user_id = ?", userId).
		Count(&count).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	if count == 0 {
		return webapi.Error(common.ErrAddressNotExist).Render(ec)
	}

	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.Address{}).
		Where("id = ?", req.AddressID).
		Where("user_id = ?", userId).
		Delete(&model.Address{}).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(true).Render(ec)
}
