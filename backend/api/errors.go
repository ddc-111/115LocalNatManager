package api

var errorMessages = map[int]string{
	40100000: "参数缺失",
	40101017: "用户验证失败",
	40110000: "请求异常需要重试",
	40140100: "client_id 错误",
	40140101: "code_challenge 必填",
	40140102: "code_challenge_method 必须是 sha256、sha1、md5 之一",
	40140103: "sign 必填",
	40140104: "sign 签名失败",
	40140105: "生成二维码失败",
	40140106: "APP ID 无效",
	40140107: "应用不存在",
	40140108: "应用未审核通过",
	40140109: "应用已被停用",
	40140110: "应用已过期",
	40140111: "APP Secret 错误",
	40140112: "code_verifier 长度要求43~128位",
	40140113: "code_verifier 验证失败",
	40140114: "refresh_token 格式错误（防篡改）",
	40140115: "refresh_token 签名校验失败（防篡改）",
	40140116: "refresh_token 无效（已解除授权）",
	40140117: "access_token 刷新太频繁",
	40140118: "开发者认证已过期",
	40140119: "refresh_token 已过期",
	40140120: "refresh_token 检验失败（防篡改）",
	40140121: "access_token 刷新失败",
	40140122: "超出授权应用个数上限",
	40140123: "access_token 格式错误（防篡改）",
	40140124: "access_token 签名校验失败（防篡改）",
	40140125: "access_token 无效（已过期或者已解除授权）",
	40140126: "access_token 校验失败（防篡改）",
	40140127: "response_type 错误",
	40140128: "redirect_uri 缺少协议",
	40140129: "redirect_uri 缺少域名",
	40140130: "没有配置重定向域名",
	40140131: "redirect_uri 非法域名",
	40140132: "grant_type 错误",
	40140133: "client_secret 验证失败",
	40140134: "授权码 code 验证失败",
	40140135: "client_id 验证失败",
	40140136: "redirect_uri 验证失败（防MITM）",
}

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     string `json:"error"`
}

func (e *APIError) Error() string {
	if desc, ok := errorMessages[e.Code]; ok {
		return desc + ": " + e.Message
	}
	return e.Message
}

func GetErrorMessage(code int) string {
	if msg, ok := errorMessages[code]; ok {
		return msg
	}
	return ""
}

func IsAuthError(code int) bool {
	return code >= 40140100 && code <= 40140136
}

func IsTokenExpired(code int) bool {
	return code == 40140119 || code == 40140125
}

func IsTokenInvalid(code int) bool {
	return code == 40140114 || code == 40140115 || code == 40140116 || 
	       code == 40140120 || code == 40140123 || code == 40140124 || 
	       code == 40140126
}

func NeedReAuth(code int) bool {
	return code == 40140116 || code == 40140119
}
