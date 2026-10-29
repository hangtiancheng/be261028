package types

// ResponseCode mirrors com.buy.group.easy.types.enums.ResponseCode. Each code
// carries a stable identifier plus a human readable message.
type ResponseCode struct {
	Code string
	Info string
}

func newResponseCode(code, info string) ResponseCode {
	return ResponseCode{Code: code, Info: info}
}

var (
	ResponseSuccess        = newResponseCode("0000", "success")
	ResponseUnError        = newResponseCode("0001", "unknown failure")
	ResponseIllegalParam   = newResponseCode("0002", "illegal parameter")
	ResponseIndexException = newResponseCode("0003", "unique index conflict")
	ResponseUpdateZero     = newResponseCode("0004", "zero rows updated")
	ResponseHTTPException  = newResponseCode("0005", "HTTP call exception")
	ResponseRateLimiter    = newResponseCode("0006", "rate limited")

	ResponseE0001 = newResponseCode("E0001", "no matching discount calculation service")
	ResponseE0002 = newResponseCode("E0002", "no group-buy marketing config")
	ResponseE0003 = newResponseCode("E0003", "group-buy activity intercepted by downgrade switch")
	ResponseE0004 = newResponseCode("E0004", "group-buy activity intercepted by traffic cut-over")
	ResponseE0005 = newResponseCode("E0005", "team-up failed, zero rows updated")
	ResponseE0006 = newResponseCode("E0006", "team already complete, lock quota reached")
	ResponseE0007 = newResponseCode("E0007", "crowd-restricted group-buy, participation denied")
	ResponseE0008 = newResponseCode("E0008", "team-up failed, insufficient cached stock")

	ResponseE0101 = newResponseCode("E0101", "group-buy activity not effective")
	ResponseE0102 = newResponseCode("E0102", "outside the group-buy activity validity window")
	ResponseE0103 = newResponseCode("E0103", "user reached the participation limit for this group-buy")
	ResponseE0104 = newResponseCode("E0104", "unknown external trade number or order already refunded")
	ResponseE0105 = newResponseCode("E0105", "intercepted by SC channel blacklist")
	ResponseE0106 = newResponseCode("E0106", "order trade time outside the group-buy validity window")
	ResponseE0107 = newResponseCode("E0107", "order already refunded, duplicate refund rejected")
)
