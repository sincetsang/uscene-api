package common

var ComplaintTypeMap = map[string]map[string]string{
	"zh": {
		"info_error":      "地标信息有误(如名称，描述，图片)",
		"illegal_content": "地标内容涉及违法违规内容",
		"other_issues":    "地标有其他问题，我要举报",
	},
	"en": {
		"info_error":      "Landmark information is incorrect (name, description, images)",
		"illegal_content": "Landmark content involves illegal or inappropriate content",
		"other_issues":    "Other issues with the landmark, I want to report",
	},
	"ru": {
		"info_error":      "Информация о достопримечательности неверна (название, описание, изображения)",
		"illegal_content": "Содержимое достопримечательности содержит незаконный или неприемлемый контент",
		"other_issues":    "Другие проблемы с достопримечательностью, хочу сообщить",
	},
}

func GetComplaintType(lang string, typeID string) string {
	if types, ok := ComplaintTypeMap[lang]; ok {
		if text, ok := types[typeID]; ok {
			return text
		}
	}
	// 默认返回中文
	return ComplaintTypeMap["zh"][typeID]
}
