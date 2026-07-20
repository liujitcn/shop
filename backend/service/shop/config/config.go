package config

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	configv1 "shop/api/gen/go/shop/config/v1"
	"shop/pkg/errorsx"

	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/sdk"
)

// WRAPPER_CONFIG_KEY 表示商城业务配置在自定义配置中的包装键。
const WRAPPER_CONFIG_KEY = "Shop"

var payTimeoutMinutes = 30

// CACHE_KEY_PAY_TIMEOUT 表示支付超时时间缓存键。
const CACHE_KEY_PAY_TIMEOUT = "payTimeout"

// NewShopConfig 获取商城业务配置。
func NewShopConfig(ctx *bootstrap.Context) *configv1.ShopConfig {
	cfg, ok := ctx.GetCustomConfig(WRAPPER_CONFIG_KEY)
	// 自定义包装配置存在时，优先返回包装中的商城配置。
	if ok {
		wrapperCfg := cfg.(*configv1.ShopConfigWrapper)
		return wrapperCfg.GetShop()
	}
	return &configv1.ShopConfig{}
}

// ParseWxPay 解析微信支付配置。
func ParseWxPay(cfg *configv1.ShopConfig) (*configv1.WxPay, error) {
	wxPay := cfg.GetWxPay()
	// 缺少微信支付配置时，直接返回配置错误。
	if wxPay == nil {
		return nil, errorsx.Internal("支付配置信息错误")
	}
	appID := wxPay.GetAppid()
	mchID := wxPay.GetMchId()
	mchCertSn := wxPay.GetMchCertSn()
	mchCertPath := wxPay.GetMchCertPath()
	mchAPIV3Key := wxPay.GetMchApiV3Key()
	// 微信支付关键字段缺失时，视为配置不可用。
	if appID == "" || mchID == "" || mchCertSn == "" || mchCertPath == "" || mchAPIV3Key == "" {
		return nil, errorsx.Internal("支付配置信息错误")
	}
	// 兼容不同工作目录启动（GoLand/命令行）导致的相对路径差异。
	if resolvedPath, ok := resolveFilePath(mchCertPath); ok {
		wxPay.MchCertPath = resolvedPath
	}
	return wxPay, nil
}

// ParseRecommend 解析推荐配置。
func ParseRecommend(cfg *configv1.ShopConfig) (*configv1.Recommend, error) {
	// 商城配置缺失时，回退到空推荐配置，后续由业务侧自动走本地兜底。
	if cfg == nil {
		return &configv1.Recommend{}, nil
	}
	recommend := cfg.GetRecommend()
	// 缺少推荐配置时，回退到空推荐配置，避免因未启用推荐系统导致服务启动失败。
	if recommend == nil {
		return &configv1.Recommend{}, nil
	}
	return recommend, nil
}

// ParsePayTimeout 解析支付超时时间。
func ParsePayTimeout() time.Duration {
	cache := sdk.Runtime.GetCache()
	// 未启用缓存时，回退到默认支付超时时间。
	if cache == nil {
		return time.Duration(payTimeoutMinutes) * time.Minute
	}

	cacheValue, err := cache.Get(CACHE_KEY_PAY_TIMEOUT)
	if err != nil {
		return time.Duration(payTimeoutMinutes) * time.Minute
	}
	var parsedPayTimeoutMinutes int
	parsedPayTimeoutMinutes, err = strconv.Atoi(cacheValue)
	if err != nil {
		return time.Duration(payTimeoutMinutes) * time.Minute
	}
	payTimeoutMinutes = parsedPayTimeoutMinutes
	return time.Duration(payTimeoutMinutes) * time.Minute
}

// resolveFilePath 解析配置中的证书文件路径。
func resolveFilePath(path string) (string, bool) {
	var err error
	// 绝对路径存在时直接返回原路径。
	if filepath.IsAbs(path) {
		// 绝对路径对应文件存在时，直接返回原路径。
		_, err = os.Stat(path)
		if err == nil {
			return path, true
		}
		return path, false
	}

	candidates := []string{
		path,
		filepath.Join("server", path),
		filepath.Join("..", path),
		filepath.Join("..", "..", path),
		filepath.Join("..", "..", "..", path),
		filepath.Join("..", "server", path),
		filepath.Join(filepath.Dir(os.Args[0]), "..", path),
		filepath.Join(filepath.Dir(os.Args[0]), "..", "..", path),
	}

	for _, p := range candidates {
		cleaned := filepath.Clean(p)
		// 命中可用文件后，立即返回标准化路径。
		_, err = os.Stat(cleaned)
		if err == nil {
			return cleaned, true
		}
	}
	return path, false
}
