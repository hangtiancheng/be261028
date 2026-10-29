package trade

import (
	"crypto/rand"
	"math/big"
	"strconv"
)

// Redis key prefixes, renamed from the legacy easy_group_buy_* to be261028_*.
const (
	teamStockKeyPrefix   = "be261028_team_stock_key_"
	refundLockKeyPrefix  = "be261028_refund_lock_"
	notifyJobExecLockKey = "be261028_notify_job_exec"
)

// GenerateTeamStockKey mirrors TradeLockRuleFilterFactory.generateTeamStockKey.
func GenerateTeamStockKey(activityId int64, teamId string) string {
	return teamStockKeyPrefix + strconv.FormatInt(activityId, 10) + "_" + teamId
}

// GenerateRecoveryTeamStockKey mirrors TradeLockRuleFilterFactory.generateRecoveryTeamStockKey.
func GenerateRecoveryTeamStockKey(activityId int64, teamId string) string {
	return GenerateTeamStockKey(activityId, teamId) + "_recovery"
}

// refundLockKey mirrors the per-order recovery lock in TradeRepository.refund2AddRecovery.
func refundLockKey(orderId string) string {
	return refundLockKeyPrefix + orderId
}

// NotifyJobExecLockKey is the distributed lock guarding the scheduled notify job.
func NotifyJobExecLockKey() string { return notifyJobExecLockKey }

// randomNumeric returns a string of n random digits, replacing the legacy
// RandomStringUtils.randomNumeric used for teamId (8) and orderId (12).
func randomNumeric(n int) string {
	if n <= 0 {
		return ""
	}
	const digits = "0123456789"
	buf := make([]byte, n)
	max := big.NewInt(int64(len(digits)))
	for i := 0; i < n; i++ {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			// Extremely unlikely; fall back to a zero digit.
			buf[i] = '0'
			continue
		}
		buf[i] = digits[idx.Int64()]
	}
	return string(buf)
}
