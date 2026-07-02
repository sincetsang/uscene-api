package main

import (
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/sdlog"
	"TMA/pkg/web/webapi"
	"TMA/service/db/model"
	"strconv"
)

// ComplaintType 投诉类型结构体
type ComplaintType struct {
	Type  string `json:"type"`  // 类型ID
	Label string `json:"label"` // 类型标签
}

// CreateComplaintRequest 创建投诉请求结构体
type CreateComplaintRequest struct {
	SpotID    int64  `json:"spot_id"`    // 地标ID
	Type      string `json:"type"`       // 投诉类型
	Content   string `json:"content"`    // 投诉内容
	ImageURLs string `json:"image_urls"` // 图片URL，多个用逗号分隔
}

// ComplaintResponse 投诉响应结构体
type ComplaintResponse struct {
	ID          int64  `json:"id"`           // 投诉ID
	Type        string `json:"type"`         // 投诉类型
	TypeLabel   string `json:"type_label"`   // 投诉类型标签
	SpotID      int64  `json:"spot_id"`      // 地标ID
	SpotName    string `json:"spot_name"`    // 地标名称
	SpotImage   string `json:"spot_image"`   // 地标图片
	Content     string `json:"content"`      // 投诉内容
	Images      string `json:"images"`       // 图片URL，多个用逗号分隔
	Status      string `json:"status"`       // 投诉状态
	CreatedAt   string `json:"created_at"`   // 创建时间
	StatusLabel string `json:"status_label"` // 状态标签
}

// complaintTypeList 获取投诉类型列表
func complaintTypeList(ec *middleware.AppRequestContext) error {
	// 获取请求语言
	lang := common.GetRequestLanguage()

	// 获取对应语言的投诉类型映射
	types, ok := common.ComplaintTypeMap[lang]
	if !ok {
		// 如果语言不存在，使用中文
		types = common.ComplaintTypeMap["zh"]
	}

	// 构建响应数据
	var result []ComplaintType
	for typeID, label := range types {
		result = append(result, ComplaintType{
			Type:  typeID,
			Label: label,
		})
	}

	return webapi.OK(result).Render(ec)
}

// createComplaint 创建投诉
func createComplaint(ec *middleware.AppRequestContext) error {
	// 解析请求参数
	var req CreateComplaintRequest
	if err := ec.Bind(&req); err != nil {
		sdlog.Errorf("解析请求参数失败: %v", err)
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 参数校验
	if req.SpotID <= 0 {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	if req.Type == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	if req.Content == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 检查投诉类型是否有效
	if _, ok := common.ComplaintTypeMap["zh"][req.Type]; !ok {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 检查地标是否存在
	var spot model.CheckInPoint
	err := ec.Nu.DB.Model(&model.CheckInPoint{}).
		Where("id = ? AND status = 1", req.SpotID).
		First(&spot).Error
	if err != nil {
		sdlog.Errorf("查询地标失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 创建投诉记录
	complaint := model.Complaint{
		UserID:         ec.AuthData.User.ID,
		CheckInPointID: req.SpotID,
		Type:           req.Type,
		Description:    req.Content,
		Images:         req.ImageURLs,
		Status:         "pending",
	}

	err = ec.Nu.DB.Create(&complaint).Error
	if err != nil {
		sdlog.Errorf("创建投诉记录失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	return webapi.OK(true).Render(ec)
}

// complaintList 获取投诉列表
func complaintList(ec *middleware.AppRequestContext) error {
	// 获取分页参数
	page := ec.QueryParams().Get("page")
	pageSize := ec.QueryParams().Get("page_size")
	if page == "" {
		page = "1"
	}
	if pageSize == "" {
		pageSize = "10"
	}

	pageNum, err := strconv.ParseInt(page, 10, 64)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	pageSizeNum, err := strconv.ParseInt(pageSize, 10, 64)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 查询总数
	var total int64
	err = ec.Nu.DB.Model(&model.Complaint{}).
		Where("user_id = ?", ec.AuthData.User.ID).
		Count(&total).Error
	if err != nil {
		sdlog.Errorf("查询投诉总数失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 查询投诉列表
	var complaints []model.Complaint
	err = ec.Nu.DB.Model(&model.Complaint{}).
		Where("user_id = ?", ec.AuthData.User.ID).
		Order("created_at DESC").
		Offset(int((pageNum - 1) * pageSizeNum)).
		Limit(int(pageSizeNum)).
		Find(&complaints).Error
	if err != nil {
		sdlog.Errorf("查询投诉列表失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 获取所有地标ID
	var spotIDs []int64
	for _, c := range complaints {
		spotIDs = append(spotIDs, c.CheckInPointID)
	}

	// 批量查询地标信息
	var spots []model.CheckInPoint
	err = ec.Nu.DB.Model(&model.CheckInPoint{}).
		Where("id IN ?", spotIDs).
		Find(&spots).Error
	if err != nil {
		sdlog.Errorf("查询地标信息失败: %v", err)
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 创建地标映射
	spotMap := make(map[int64]model.CheckInPoint)
	for _, spot := range spots {
		spotMap[spot.ID] = spot
	}

	// 构建响应数据
	result := make([]ComplaintResponse, 0)
	for _, c := range complaints {
		spot := spotMap[c.CheckInPointID]
		// 获取投诉类型标签
		typeLabel := common.GetComplaintType(common.GetRequestLanguage(), c.Type)
		// 获取状态标签

		result = append(result, ComplaintResponse{
			ID:          c.ID,
			Type:        c.Type,
			TypeLabel:   typeLabel,
			SpotID:      c.CheckInPointID,
			SpotName:    spot.Title,
			SpotImage:   spot.LandmarkImage,
			Content:     c.Description,
			Images:      c.Images,
			Status:      c.Status,
			CreatedAt:   c.CreatedAt.Format("2006-01-02 15:04:05"),
			StatusLabel: c.Status,
		})
	}

	return webapi.OK(map[string]interface{}{
		"total": total,
		"list":  result,
	}).Render(ec)
}
