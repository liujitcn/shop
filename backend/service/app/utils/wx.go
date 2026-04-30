package utils

import (
	_http "github.com/liujitcn/go-utils/http"
)

// WxSessionKey 表示微信登录接口返回的会话信息
type WxSessionKey struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

// WxAccessToken 表示微信接口访问令牌返回结果
type WxAccessToken struct {
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// PhoneNumber 表示微信手机号授权接口返回结果
type PhoneNumber struct {
	ErrCode   int    `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
	PhoneInfo struct {
		PhoneNumber     string `json:"phoneNumber"`
		PurePhoneNumber string `json:"purePhoneNumber"`
		CountryCode     string `json:"countryCode"`
		Watermark       struct {
			Timestamp int    `json:"timestamp"`
			AppID     string `json:"appid"`
		} `json:"watermark"`
	} `json:"phone_info"`
}

// GetPhoneNumber 调用微信接口换取手机号信息
func GetPhoneNumber(accessToken, code string) (*PhoneNumber, error) {
	// 微信接口要求请求体为 JSON，对外只传授权码即可
	query := make(map[string]string)
	query["access_token"] = accessToken

	body := make(map[string]string)
	body["code"] = code
	var res PhoneNumber
	err := _http.Post(
		"https://api.weixin.qq.com/wxa/business/getuserphonenumber",
		&res,
		_http.WithQueries(query),
		_http.WithJSONBody(body),
	)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

// GetAccessToken 获取微信接口访问令牌
func GetAccessToken(appID, secret string) (*WxAccessToken, error) {
	query := make(map[string]string)
	query["grant_type"] = "client_credential"
	query["appid"] = appID
	query["secret"] = secret
	var res WxAccessToken
	err := _http.Post(
		"https://api.weixin.qq.com/cgi-bin/token",
		&res,
		_http.WithQueries(query),
	)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

// GetSessionKey 使用登录授权码换取微信会话信息
func GetSessionKey(appID, secret, code string) (*WxSessionKey, error) {
	query := make(map[string]string)
	query["grant_type"] = "authorization_code"
	query["appid"] = appID
	query["secret"] = secret
	query["js_code"] = code
	var res WxSessionKey
	err := _http.Post(
		"https://api.weixin.qq.com/sns/jscode2session",
		&res,
		_http.WithQueries(query),
	)
	if err != nil {
		return nil, err
	}

	return &res, nil
}
