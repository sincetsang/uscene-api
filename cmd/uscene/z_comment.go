package main

import (
	"TMA/pkg/common"
	"TMA/pkg/middleware"
	"TMA/pkg/sdlog"
	"TMA/pkg/sdparse"
	"TMA/pkg/web/webapi"
	"TMA/service/db/model"

	"time"

	"gorm.io/gorm"
)

type AddCommentRequest struct {
	Star    int64  `json:"star"`
	OrderId int64  `json:"order_id"`
	Comment string `json:"comment"`
}

func addComment(ec *middleware.AppRequestContext) error {
	var req AddCommentRequest
	err := ec.Bind(&req)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	userId := ec.AuthData.User.ID
	if userId == 0 {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}
	if req.Comment == "" || req.Star < 1 || req.Star > 5 || req.OrderId == 0 {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	// check product's spot id
	var productOrder model.ProductOrder
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.ProductOrder{}).Where("order_id = ?", req.OrderId).
		Where("user_id = ?", userId).
		First(&productOrder).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrProductNotExist).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}

	// check if product order is completed
	if productOrder.Status != common.ProductOrderStatusCompleted {
		return webapi.Error(common.ErrOrderStatusChanged).Render(ec)
	}

	if productOrder.IsCommented {
		return webapi.Error(common.ErrCommentCanOnlyAddOnce).Render(ec)
	}

	commentModel := &model.Comment{
		ProductID:   productOrder.ProductID,
		OrderID:     productOrder.OrderID,
		UserID:      userId,
		Star:        int32(req.Star),
		SpotID:      productOrder.SpotID,
		Content:     req.Comment,
		AuditStatus: common.SpotAuditStatusApproved,
		Timestamp:   int32(time.Now().Unix()),
	}
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.Comment{}).Create(commentModel).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 更新订单的评论状态
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.ProductOrder{}).Where("order_id = ?", req.OrderId).Update("is_commented", true).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 更新商品的评论数量
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.Product{}).Where("id = ?", productOrder.ProductID).Update("comment_count", gorm.Expr("comment_count + 1")).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(true).Render(ec)

}

type UpdateCommentRequest struct {
	CommentID int64  `json:"comment_id"`
	Star      int64  `json:"star"`
	Comment   string `json:"comment"`
}

func updateComment(ec *middleware.AppRequestContext) error {
	var req UpdateCommentRequest
	err := ec.Bind(&req)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	userId := ec.AuthData.User.ID
	if userId == 0 {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}
	if req.CommentID == 0 || req.Star == 0 || req.Comment == "" {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.Comment{}).Where("id = ?", req.CommentID).Where("user_id = ?", userId).Update("star", req.Star).Update("content", req.Comment).Error
	if err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}
	return webapi.OK(true).Render(ec)
}

type DeleteCommentRequest struct {
	CommentID int64 `json:"comment_id"`
}

func deleteComment(ec *middleware.AppRequestContext) error {
	var req DeleteCommentRequest
	err := ec.Bind(&req)
	if err != nil {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	userId := ec.AuthData.User.ID
	commentId := req.CommentID
	if commentId == 0 {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	// 获取评论
	var comment model.Comment
	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.Comment{}).Where("id = ?", commentId).First(&comment).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrCommentNotExist).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}

	err = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.Comment{}).Where("id = ?", commentId).Where("user_id = ?", userId).Delete(&model.Comment{}).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return webapi.Error(common.ErrCommentNotExist).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 获取商品订单
	var productOrder model.ProductOrder
	_ = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.ProductOrder{}).Where("order_id = ?", comment.OrderID).First(&productOrder).Error

	if productOrder.ProductID >= 0 {
		// 更新商品的评论数量 comment_count不能小于0
		_ = ec.Nu.DB.WithContext(ec.Request().Context()).Model(&model.Product{}).Where("id = ?", productOrder.ProductID).Where("comment_count > 0").Update("comment_count", gorm.Expr("comment_count - 1")).Error
	}

	return webapi.OK(true).Render(ec)
}

// 分页获取spot的评论， get query params, 同时查询user的nickname和avatar_url
type GetSpotCommentsRequest struct {
	SpotID int64 `json:"spot_id"`
	Page   int64 `json:"page"`
	Size   int64 `json:"size"`
}

type CommentWithUser struct {
	ID            int64  `json:"id"`
	SpotID        int64  `json:"spot_id"`
	ProductID     int64  `json:"product_id"`
	UserID        int64  `json:"user_id"`
	Content       string `json:"content"`
	Star          int32  `json:"star"`
	Timestamp     int32  `json:"timestamp"`
	UserNickname  string `json:"user_nickname"`
	UserAvatarURL string `json:"user_avatar_url"`
}

func getSpotComments(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	if userId == 0 {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}
	spotId := sdparse.Int64Def(ec.QueryParams().Get("spot_id"), 0)
	page := sdparse.Int64Def(ec.QueryParams().Get("page"), 1)
	pageSize := sdparse.Int64Def(ec.QueryParams().Get("page_size"), 10)
	if spotId == 0 || page == 0 || pageSize == 0 {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	var results []CommentWithUser = []CommentWithUser{}
	var total int64
	db := ec.Nu.DB.WithContext(ec.Request().Context()).
		Table("comment c").
		Where("c.spot_id = ?", spotId).Where("c.audit_status = ?", common.SpotAuditStatusApproved)

	// 统计总数
	if err := db.Count(&total).Error; err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	offset := (page - 1) * pageSize

	err := ec.Nu.DB.WithContext(ec.Request().Context()).
		Table("comment c").
		Joins("LEFT JOIN user u ON c.user_id = u.user_id").
		Where("c.spot_id = ?", spotId).
		Where("c.audit_status = ?", common.SpotAuditStatusApproved).
		Select("c.id, c.spot_id, c.product_id, c.user_id, c.content, c.timestamp, c.star, u.nickname as user_nickname, u.avatar_url as user_avatar_url").
		Order("c.timestamp DESC").
		Offset(int(offset)).
		Limit(int(pageSize)).
		Scan(&results).Error
	if err != nil {
		sdlog.Errorf("getSpotComments error: %v", err)
		if err == gorm.ErrRecordNotFound {
			return webapi.OK(map[string]interface{}{
				"list":      results,
				"total":     total,
				"page":      page,
				"page_size": pageSize,
			}).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}
	// 对user nickname 脱敏，中间4个字符用*代替， 需要考虑nickname长度小于2的情况
	for item := range results {
		if len(results[item].UserNickname) < 2 {
			results[item].UserNickname = "****"
		} else {
			results[item].UserNickname = results[item].UserNickname[:2] + "****" + results[item].UserNickname[len(results[item].UserNickname)-2:]
		}
	}

	resp := map[string]interface{}{
		"list":      results,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}

	return webapi.OK(resp).Render(ec)

}

func getProductComments(ec *middleware.AppRequestContext) error {
	userId := ec.AuthData.User.ID
	if userId == 0 {
		return webapi.Error(common.ErrUnauthorized).Render(ec)
	}
	productId := sdparse.Int64Def(ec.QueryParams().Get("product_id"), 0)
	if productId == 0 {
		return webapi.Error(common.ErrParam).Render(ec)
	}
	page := sdparse.Int64Def(ec.QueryParams().Get("page"), 1)
	pageSize := sdparse.Int64Def(ec.QueryParams().Get("page_size"), 10)
	if page == 0 || pageSize == 0 {
		return webapi.Error(common.ErrParam).Render(ec)
	}

	var results []CommentWithUser = []CommentWithUser{}
	var total int64
	db := ec.Nu.DB.WithContext(ec.Request().Context()).
		Table("comment c").
		Where("c.product_id = ?", productId).Where("c.audit_status = ?", common.SpotAuditStatusApproved)

	// 统计总数
	if err := db.Count(&total).Error; err != nil {
		return webapi.Error(common.ErrService).Render(ec)
	}

	offset := (page - 1) * pageSize

	err := ec.Nu.DB.WithContext(ec.Request().Context()).
		Table("comment c").
		Joins("LEFT JOIN user u ON c.user_id = u.user_id").
		Where("c.product_id = ?", productId).
		Where("c.audit_status = ?", common.SpotAuditStatusApproved).
		Select("c.id, c.spot_id, c.product_id, c.user_id, c.content, c.timestamp, c.star, u.nickname as user_nickname, u.avatar_url as user_avatar_url").
		Order("c.timestamp DESC").
		Offset(int(offset)).
		Limit(int(pageSize)).
		Scan(&results).Error
	if err != nil {
		sdlog.Errorf("getProductComments error: %v", err)
		if err == gorm.ErrRecordNotFound {
			sdlog.Infof("getProductComments no data")
			return webapi.OK(map[string]interface{}{
				"list":      []CommentWithUser{},
				"total":     total,
				"page":      page,
				"page_size": pageSize,
			}).Render(ec)
		}
		return webapi.Error(common.ErrService).Render(ec)
	}

	// 对user nickname 脱敏，中间4个字符用*代替， 需要考虑nickname长度小于2的情况
	for item := range results {
		if len(results[item].UserNickname) < 2 {
			results[item].UserNickname = "****"
		} else {
			results[item].UserNickname = results[item].UserNickname[:2] + "****" + results[item].UserNickname[len(results[item].UserNickname)-2:]
		}
	}

	resp := map[string]interface{}{
		"list":      results,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}
	return webapi.OK(resp).Render(ec)

}
