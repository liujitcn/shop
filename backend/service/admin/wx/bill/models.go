package bill

// TradeBillRequest 交易账单请求
type TradeBillRequest struct {
	// 格式YYYY-MM-DD。仅支持三个月内的账单下载申请
	BillDate *string `json:"bill_date"`
	// 账单类型，不填则默认是ALL
	BillType *string `json:"bill_type"`
}

// TradeBillResponse 交易账单响应
type TradeBillResponse struct {
	HashType    *string `json:"hash_type"`    // 哈希类型，固定为SHA1
	HashValue   *string `json:"hash_value"`   // 账单文件的SHA1摘要值，用于商户侧校验文件的一致性
	DownloadUrl *string `json:"download_url"` // 供下一步请求账单文件的下载地址，该地址5min内有效
}

const (
	BILL_TYPE_SUCCESS string = "SUCCESS"
	BILL_TYPE_REFUND  string = "REFUND"
)
