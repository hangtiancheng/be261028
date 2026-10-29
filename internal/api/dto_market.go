package api

import (
	"fmt"
	"time"

	"github.com/hangtiancheng/be261028/internal/types"
)

// GoodsMarketRequestDTO mirrors com.buy.group.easy.api.dto.GoodsMarketRequestDTO.
type GoodsMarketRequestDTO struct {
	UserId  string `json:"userId"`
	Source  string `json:"source"`
	Channel string `json:"channel"`
	GoodsId string `json:"goodsId"`
}

// GoodsMarketResponseDTO mirrors com.buy.group.easy.api.dto.GoodsMarketResponseDTO.
type GoodsMarketResponseDTO struct {
	Goods         *Goods         `json:"goods"`
	TeamList      []*Team        `json:"teamList"`
	TeamStatistic *TeamStatistic `json:"teamStatistic"`
}

// Goods mirrors GoodsMarketResponseDTO.Goods. Note activityId is a string here,
// matching the legacy DTO.
type Goods struct {
	GoodsId        string      `json:"goodsId"`
	ActivityId     string      `json:"activityId"`
	OriginalPrice  types.Money `json:"originalPrice"`
	DeductionPrice types.Money `json:"deductionPrice"`
	PayPrice       types.Money `json:"payPrice"`
}

// Team mirrors GoodsMarketResponseDTO.Team.
type Team struct {
	UserId             string         `json:"userId"`
	TeamId             string         `json:"teamId"`
	ActivityId         int64          `json:"activityId"`
	TargetCount        int            `json:"targetCount"`
	CompleteCount      int            `json:"completeCount"`
	LockCount          int            `json:"lockCount"`
	ValidStartTime     types.JSONTime `json:"validStartTime"`
	ValidEndTime       types.JSONTime `json:"validEndTime"`
	ValidTimeCountdown string         `json:"validTimeCountdown"`
	OutTradeNo         string         `json:"outTradeNo"`
}

// TeamStatistic mirrors GoodsMarketResponseDTO.TeamStatistic.
type TeamStatistic struct {
	AllTeamCount         int `json:"allTeamCount"`
	AllTeamCompleteCount int `json:"allTeamCompleteCount"`
	AllTeamUserCount     int `json:"allTeamUserCount"`
}

// DifferenceDateTime2Str mirrors GoodsMarketResponseDTO.Team.differenceDateTime2Str:
// it renders the remaining countdown between start and end as HH:MM:SS, returning
// "Invalid time" for missing values and "Ended" once the window has passed.
//
// Defect fix: the legacy format applied %24 to the hours, silently dropping whole
// days from windows longer than 24h (a 25h countdown showed as 01:00:00 and a 24h
// one as 00:00:00). Windows of a day or more are now prefixed with "Nd ".
func DifferenceDateTime2Str(validStartTime, validEndTime time.Time) string {
	if validStartTime.IsZero() || validEndTime.IsZero() {
		return "Invalid time"
	}
	diff := validEndTime.Sub(validStartTime)
	if diff < 0 {
		return "Ended"
	}
	totalSeconds := int64(diff.Seconds())
	days := totalSeconds / 86400
	seconds := totalSeconds % 60
	minutes := (totalSeconds / 60) % 60
	hours := (totalSeconds / 3600) % 24
	if days > 0 {
		return fmt.Sprintf("%dd %02d:%02d:%02d", days, hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}
