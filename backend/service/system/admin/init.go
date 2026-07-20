package admin

import (
	"shop/service/system/admin/biz"
	"shop/service/system/admin/codegen"

	"github.com/google/wire"
)

// ProviderSet 汇总系统管理端服务依赖注入提供者。
var ProviderSet = wire.NewSet(
	codegen.NewManager,
	biz.NewAuthCase,
	biz.NewBaseAPICase,
	biz.NewBaseConfigCase,
	biz.NewBaseDeptCase,
	biz.NewBaseDictCase,
	biz.NewBaseDictItemCase,
	biz.NewBaseJobCase,
	biz.NewBaseJobLogCase,
	biz.NewBaseLogCase,
	biz.NewBaseMenuCase,
	biz.NewBaseRoleCase,
	biz.NewBaseTenantCase,
	biz.NewBaseUserCase,
	biz.NewCasbinRuleCase,
	biz.NewCodeGenCase,
	biz.NewCodeGenColumnCase,
	biz.NewCodeGenProtoCase,
	biz.NewCodeGenTableCase,
	NewAuthService,
	NewBaseApiService,
	NewBaseConfigService,
	NewBaseDeptService,
	NewBaseDictService,
	NewBaseJobService,
	NewBaseLogService,
	NewBaseMenuService,
	NewBaseRoleService,
	NewBaseTenantService,
	NewBaseUserService,
	NewCodeGenService,
	NewCodeGenColumnService,
	NewCodeGenProtoService,
	NewCodeGenTableService,
)
