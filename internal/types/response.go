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
	ResponseSuccess        = newResponseCode("0000", "成功")
	ResponseUnError        = newResponseCode("0001", "未知失败")
	ResponseIllegalParam   = newResponseCode("0002", "非法参数")
	ResponseIndexException = newResponseCode("0003", "唯一索引冲突")
	ResponseUpdateZero     = newResponseCode("0004", "更新记录为0")
	ResponseHTTPException  = newResponseCode("0005", "HTTP接口调用异常")
	ResponseRateLimiter    = newResponseCode("0006", "接口限流")

	ResponseE0001 = newResponseCode("E0001", "不存在对应的折扣计算服务")
	ResponseE0002 = newResponseCode("E0002", "无拼团营销配置")
	ResponseE0003 = newResponseCode("E0003", "拼团活动降级拦截")
	ResponseE0004 = newResponseCode("E0004", "拼团活动切量拦截")
	ResponseE0005 = newResponseCode("E0005", "拼团组队失败，记录更新为0")
	ResponseE0006 = newResponseCode("E0006", "拼团组队完结，锁单量已达成")
	ResponseE0007 = newResponseCode("E0007", "拼团人群限定，不可参与")
	ResponseE0008 = newResponseCode("E0008", "拼团组队失败，缓存库存不足")

	ResponseE0101 = newResponseCode("E0101", "拼团活动未生效")
	ResponseE0102 = newResponseCode("E0102", "不在拼团活动有效时间内")
	ResponseE0103 = newResponseCode("E0103", "当前用户参与此拼团次数已达上限")
	ResponseE0104 = newResponseCode("E0104", "不存在的外部交易单号或用户已退单")
	ResponseE0105 = newResponseCode("E0105", "SC渠道黑名单拦截")
	ResponseE0106 = newResponseCode("E0106", "订单交易时间不在拼团有效时间范围内")
	ResponseE0107 = newResponseCode("E0107", "用户已退单，请勿重复退单")
)
