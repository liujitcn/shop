package codegen

import systemcodegen "shop/service/system/admin/codegen"

// Target 返回商城管理端代码生成目标。
func Target() systemcodegen.ProtoTarget {
	return systemcodegen.ProtoTarget{
		Directory:              "shop/admin/v1",
		PackageName:            "shop.admin.v1",
		GoAlias:                "shopadminv1",
		GoImportPath:           "shop/api/gen/go/shop/admin/v1",
		ServiceImportAlias:     "shopadmin",
		BackendModuleDirectory: "backend/service/shop/admin",
		ModuleRegisterPath:     "backend/server/shop/admin/register.go",
		FrontendAPIDirectory:   "frontend/admin/src/api/shop/admin",
		FrontendPageDirectory:  "frontend/admin/src/views/shop/admin",
	}
}

// Registration 表示商城管理端代码生成目标已在组合根注册。
type Registration struct{}

// NewRegistration 显式注册商城管理端代码生成目标。
func NewRegistration() (Registration, error) {
	_, err := systemcodegen.RegisterProtoTarget(Target())
	if err != nil {
		return Registration{}, err
	}
	return Registration{}, nil
}
