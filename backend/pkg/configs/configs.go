package configs

import (
	"errors"
	"os"
	"path/filepath"
	"shop/api/gen/go/conf"
	"strconv"
	"time"

	bootstrapConf "github.com/liujitcn/kratos-kit/api/gen/go/conf"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/sdk"
)

const WrapperConfigKey = "Shop"

var payTimeout = 30

const cacheKeyPayTimeout = "payTimeout"

func NewShopConfig(ctx *bootstrap.Context) *conf.ShopConfig {
	cfg, ok := ctx.GetCustomConfig(WrapperConfigKey)
	if ok {
		wrapperCfg := cfg.(*conf.ShopConfigWrapper)
		return wrapperCfg.GetShop()
	}
	return &conf.ShopConfig{}
}

func ParseWxMiniApp(cfg *conf.ShopConfig) (*conf.WxMiniApp, error) {
	wxMiniApp := cfg.GetWxMiniApp()
	if wxMiniApp == nil {
		return nil, errors.New("微信登录配置信息错误")
	}
	appid := wxMiniApp.GetAppid()
	secret := wxMiniApp.GetSecret()
	if appid == "" || secret == "" {
		return nil, errors.New("微信登录配置信息错误")
	}
	return wxMiniApp, nil
}

func ParseWxPay(cfg *conf.ShopConfig) (*conf.WxPay, error) {
	wxPay := cfg.GetWxPay()
	if wxPay == nil {
		return nil, errors.New("支付配置信息错误")
	}
	appid := wxPay.GetAppid()
	mchId := wxPay.GetMchId()
	mchCertSn := wxPay.GetMchCertSn()
	mchCertPath := wxPay.GetMchCertPath()
	mchAPIv3Key := wxPay.GetMchAPIv3Key()
	if appid == "" || mchId == "" || mchCertSn == "" || mchCertPath == "" || mchAPIv3Key == "" {
		return nil, errors.New("支付配置信息错误")
	}
	// 兼容不同工作目录启动（GoLand/命令行）导致的相对路径差异。
	if resolvedPath, ok := resolveFilePath(mchCertPath); ok {
		wxPay.MchCertPath = resolvedPath
	}
	return wxPay, nil
}

func resolveFilePath(path string) (string, bool) {
	if filepath.IsAbs(path) {
		if _, err := os.Stat(path); err == nil {
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
		if _, err := os.Stat(cleaned); err == nil {
			return cleaned, true
		}
	}
	return path, false
}

func ParsePayTimeout() time.Duration {
	cache := sdk.Runtime.GetCache()
	if cache == nil {
		// 默认30分钟
		return time.Duration(payTimeout) * time.Minute
	}

	v, err := cache.Get(cacheKeyPayTimeout)
	if err != nil {
		// 默认30分钟
		return time.Duration(payTimeout) * time.Minute
	}
	payTimeout, err = strconv.Atoi(v)
	if err != nil {
		// 默认30分钟
		return time.Duration(payTimeout) * time.Minute
	}
	// 默认30分钟
	return time.Duration(payTimeout) * time.Minute
}

func ParseOss(ctx *bootstrap.Context) (*bootstrapConf.OSS, error) {
	cfg := ctx.GetConfig()
	if cfg == nil || cfg.GetOss() == nil {
		return nil, errors.New("config oss is nil")
	}
	return cfg.GetOss(), nil
}

func ParseData(ctx *bootstrap.Context) (*bootstrapConf.Data, error) {
	cfg := ctx.GetConfig()
	if cfg == nil || cfg.GetData() == nil {
		return nil, errors.New("config data is nil")
	}
	return cfg.GetData(), nil
}

func ParseDatabase(cfg *bootstrapConf.Data) *bootstrapConf.Data_Database {
	return cfg.GetDatabase()
}

func ParseQueue(cfg *bootstrapConf.Data) *bootstrapConf.Data_Queue {
	return cfg.GetQueue()
}

func ParseRedis(cfg *bootstrapConf.Data) *bootstrapConf.Data_Redis {
	return cfg.GetRedis()
}

func ParsePprof(ctx *bootstrap.Context) (*bootstrapConf.Pprof, error) {
	cfg := ctx.GetConfig()
	if cfg == nil || cfg.GetPprof() == nil {
		return nil, errors.New("config pprof is nil")
	}
	return cfg.GetPprof(), nil
}

func ParseAuthnJwt(ctx *bootstrap.Context) *bootstrapConf.Authentication_Jwt {
	cfg := ctx.GetConfig()
	if cfg == nil || cfg.GetAuthn() == nil || cfg.GetAuthn().GetJwt() == nil {
		return &bootstrapConf.Authentication_Jwt{
			Method: "HS256",
			Secret: "shop-base",
		}
	}
	return cfg.GetAuthn().GetJwt()
}
