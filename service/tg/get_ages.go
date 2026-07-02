package tg

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

// 直接初始化 ages 数据
var ages = map[string]int64{
	"2768409":    1383264000000,
	"7679610":    1388448000000,
	"11538514":   1391212000000,
	"15835244":   1392940000000,
	"23646077":   1393459000000,
	"38015510":   1393632000000,
	"44634663":   1399334000000,
	"46145305":   1400198000000,
	"54845238":   1411257000000,
	"63263518":   1414454000000,
	"101260938":  1425600000000,
	"101323197":  1426204000000,
	"111220210":  1429574000000,
	"103258382":  1432771000000,
	"103151531":  1433376000000,
	"116812045":  1437696000000,
	"122600695":  1437782000000,
	"109393468":  1439078000000,
	"112594714":  1439683000000,
	"124872445":  1439856000000,
	"130029930":  1441324000000,
	"125828524":  1444003000000,
	"133909606":  1444176000000,
	"157242073":  1446768000000,
	"143445125":  1448928000000,
	"148670295":  1452211000000,
	"152079341":  1453420000000,
	"171295414":  1457481000000,
	"181783990":  1460246000000,
	"222021233":  1465344000000,
	"225034354":  1466208000000,
	"278941742":  1473465000000,
	"285253072":  1476835000000,
	"294851037":  1479600000000,
	"297621225":  1481846000000,
	"328594461":  1482969000000,
	"337808429":  1487707000000,
	"341546272":  1487782000000,
	"352940995":  1487894000000,
	"369669043":  1490918000000,
	"400169472":  1501459000000,
	"805158066":  1563208000000,
	"1974255900": 1634000000000,
}

// 获取日期
func getDate(id int) (int, time.Time) {
	ids := make([]string, 0, len(ages))
	for k := range ages {
		ids = append(ids, k)
	}

	nids := make([]int, len(ids))
	for i, e := range ids {
		nids[i], _ = strconv.Atoi(e)
	}

	minId := nids[0]
	maxId := nids[len(nids)-1]

	if id < minId {
		return -1, time.Unix(ages[ids[0]]/1000, 0) // 将毫秒转换为秒
	} else if id > maxId {
		return 1, time.Unix(ages[ids[len(ids)-1]]/1000, 0) // 将毫秒转换为秒
	} else {
		lid := nids[0]
		for i := 0; i < len(ids); i++ {
			if id <= nids[i] {
				// 计算中间日期
				uid := nids[i]
				lage := ages[strconv.Itoa(lid)]
				uage := ages[strconv.Itoa(uid)]

				idratio := float64(id-lid) / float64(uid-lid)
				midDate := math.Floor(idratio*float64(uage-lage) + float64(lage))
				return 0, time.Unix(int64(midDate)/1000, 0) // 将毫秒转换为秒
			} else {
				lid = nids[i]
			}
		}
	}
	return 0, time.Time{}
}

func GetAge(id int) (string, int) {
	direction, date := getDate(id)
	ageDescriptor := ""
	switch direction {
	case -1:
		ageDescriptor = "older than"
	case 1:
		ageDescriptor = "newer than"
	default:
		ageDescriptor = "approximately"
	}

	ageString := fmt.Sprintf("%d/%d", date.Month(), date.Year())

	// 将 ageDescriptor 和 ageString 结合成一个字段
	combinedField := fmt.Sprintf("Your account registration date is %s %s.", ageDescriptor, ageString)

	// 定义时间范围：2015年1月1日到现在
	startDate := time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Now()

	// 计算从2015年到现在的总天数
	totalDays := endDate.Sub(startDate).Hours() / 24

	// 计算给定日期距离2015年1月1日的天数
	daysSinceStart := endDate.Sub(date).Hours() / 24

	var linearValue int
	if date.Before(startDate) {
		// 如果日期在2015年之前，返回最大值100
		linearValue = 100
	} else {
		// 计算线性比例值，日期越接近当前时间，值越低
		// 计算从2015年到现在的天数减去距离现在的天数得到的百分比
		linearValue = int(((totalDays - (totalDays - daysSinceStart)) / totalDays) * 100)
	}

	// 确保返回的值在0到100之间
	if linearValue < 0 {
		linearValue = 0
	} else if linearValue > 100 {
		linearValue = 100
	}

	return combinedField, linearValue
}
