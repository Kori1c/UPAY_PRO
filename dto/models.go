package dto

// 定义一个异步通知请求参数的结构体

type PaymentNotification_request struct {
	TradeID            string  `json:"trade_id"`
	OrderID            string  `json:"order_id"`
	Amount             float64 `json:"amount"`
	ActualAmount       float64 `json:"actual_amount"`
	Token              string  `json:"token"`
	BlockTransactionID string  `json:"block_transaction_id"`
	Signature          string  `json:"signature"`
	Status             int     `json:"status"`
}

type Data struct {
	TradeID        string  `json:"trade_id"`
	OrderID        string  `json:"order_id"`
	Amount         float64 `json:"amount"`
	ActualAmount   float64 `json:"actual_amount"`
	Token          string  `json:"token"`
	ExpirationTime int64   `json:"expiration_time"`
	PaymentURL     string  `json:"payment_url"`
	Type           string  `json:"type"`
	Network        string  `json:"network"`
	Chain          string  `json:"chain"`
	PayAmount      float64 `json:"pay_amount"`
	PayAddress     string  `json:"pay_address"`
	QRContent      string  `json:"qr_content"`
	QRCodeURL      string  `json:"qr_code_url,omitempty"`
	StatusURL      string  `json:"status_url"`
	StatusPath     string  `json:"status_path"`
}

// 定义返回的结构体|创建订单后返回的数据
type Response struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
	Data       Data   `json:"data"`
	// RequestID  string
}

// 定义模版所需数据视图模型
// 模版所需数据视图模型
type PaymentViewModel struct {
	Currency               string  `json:"currency"`
	TradeId                string  `json:"tradeId"`
	ActualAmount           float64 `json:"actualAmount"`
	Token                  string  `json:"token"`
	ExpirationTime         int64   `json:"expirationTime"`
	RedirectUrl            string  `json:"redirectUrl"`            // 添加重定向URL
	AppName                string  `json:"appName"`                //应用名称
	CustomerServiceContact string  `json:"customerServiceContact"` //客户服务联系方式
	Logo                   string  `json:"logo"`                   // 币种图标
}

// RequestParams 用于存储请求参数
type RequestParams struct {
	Type        string  `json:"type" validate:"required"`
	OrderID     string  `json:"order_id" validate:"required"`
	Amount      float64 `json:"amount" validate:"required,gte=0.01"`
	NotifyURL   string  `json:"notify_url" validate:"required,url"`
	RedirectURL string  `json:"redirect_url" validate:"required,url"`
	Signature   string  `json:"signature" validate:"required"`
}
