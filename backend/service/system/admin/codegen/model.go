package codegen

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/stringcase"
)

const (
	// ProtoRootPath 表示仓库内 Proto 根目录。
	ProtoRootPath = "backend/api/proto"
	// DefaultProtoDirectory 表示代码生成器默认写入的 Proto 目录。
	DefaultProtoDirectory = "system/admin/v1"
)

// ProtoTarget 描述代码生成器支持的 Proto 与后端服务分组。
type ProtoTarget struct {
	Directory              string // Proto 根目录下的相对目录
	PackageName            string // Proto package 名称
	GoAlias                string // Go 协议包别名
	GoImportPath           string // Go 协议包导入路径
	ServiceImportAlias     string // 后端服务包别名
	BackendModuleDirectory string // 后端业务模块目录
	ModuleRegisterPath     string // 业务模块注册文件路径
	FrontendAPIDirectory   string // 前端 API 目录
	FrontendPageDirectory  string // 前端页面目录
}

// TargetRegistry 保存已由组合根启用的代码生成目标。
type TargetRegistry struct {
	mu      sync.RWMutex
	targets map[string]ProtoTarget
}

var defaultTargetRegistry = NewTargetRegistry()

// NewTargetRegistry 创建只包含基础系统目标的代码生成目标注册表。
func NewTargetRegistry() *TargetRegistry {
	registry := &TargetRegistry{targets: make(map[string]ProtoTarget)}
	registry.targets[DefaultProtoDirectory] = systemAdminProtoTarget()
	return registry
}

// Register 注册模块代码生成目标，并拒绝重复目录。
func (r *TargetRegistry) Register(target ProtoTarget) error {
	if r == nil {
		return fmt.Errorf("代码生成目标注册表未初始化")
	}
	target.Directory = filepath.ToSlash(filepath.Clean(target.Directory))
	if target.Directory == "." || target.Directory == "" {
		return fmt.Errorf("代码生成目标目录不能为空")
	}
	if target.PackageName == "" || target.GoAlias == "" || target.GoImportPath == "" {
		return fmt.Errorf("代码生成目标配置不完整: %s", target.Directory)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.targets[target.Directory]; exists {
		return fmt.Errorf("代码生成目标目录重复: %s", target.Directory)
	}
	r.targets[target.Directory] = target
	return nil
}

// TargetByDirectory 根据 Proto 目录查询已启用代码生成目标。
func (r *TargetRegistry) TargetByDirectory(directory string) (ProtoTarget, bool) {
	if r == nil {
		return ProtoTarget{}, false
	}
	directory = filepath.ToSlash(filepath.Clean(directory))
	r.mu.RLock()
	target, exists := r.targets[directory]
	r.mu.RUnlock()
	return target, exists
}

// Targets 返回已启用代码生成目标的稳定快照。
func (r *TargetRegistry) Targets() []ProtoTarget {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	targets := make([]ProtoTarget, 0, len(r.targets))
	for _, target := range r.targets {
		targets = append(targets, target)
	}
	r.mu.RUnlock()
	sort.Slice(targets, func(i int, j int) bool {
		return targets[i].Directory < targets[j].Directory
	})
	return targets
}

// TargetRegistration 表示模块代码生成目标已在组合根注册。
type TargetRegistration struct{}

// RegisterProtoTarget 将模块目标显式注册到当前进程的代码生成器。
func RegisterProtoTarget(target ProtoTarget) (TargetRegistration, error) {
	err := defaultTargetRegistry.Register(target)
	if err != nil {
		return TargetRegistration{}, err
	}
	return TargetRegistration{}, nil
}

// ProtoTargets 返回当前进程可用的全部代码生成目标。
func ProtoTargets() []ProtoTarget {
	return defaultTargetRegistry.Targets()
}

// ProtoTargetByDirectory 根据 Proto 目录返回生成器支持的目标分组。
func ProtoTargetByDirectory(directory string) (ProtoTarget, bool) {
	return defaultTargetRegistry.TargetByDirectory(directory)
}

// systemAdminProtoTarget 返回基础系统代码生成目标。
func systemAdminProtoTarget() ProtoTarget {
	return ProtoTarget{
		Directory:              DefaultProtoDirectory,
		PackageName:            "system.admin.v1",
		GoAlias:                "systemadminv1",
		GoImportPath:           "shop/api/gen/go/system/admin/v1",
		ServiceImportAlias:     "systemadmin",
		BackendModuleDirectory: "backend/service/system/admin",
		ModuleRegisterPath:     "backend/server/system/admin/register.go",
		FrontendAPIDirectory:   "frontend/admin/src/api/system/admin",
		FrontendPageDirectory:  "frontend/admin/src/views/system/admin",
	}
}

// ProtoTargetForTable 返回表配置对应的 Proto 目标，旧配置使用默认目录。
func ProtoTargetForTable(table *Table) ProtoTarget {
	directory := DefaultProtoDirectory
	if table != nil && table.APIPath != "" {
		directory = table.APIPath
	}
	if target, ok := ProtoTargetByDirectory(directory); ok {
		return target
	}
	target, _ := ProtoTargetByDirectory(DefaultProtoDirectory)
	return target
}

// BackendBizFilePath 返回目标分组内的 Biz 文件路径。
func (t ProtoTarget) BackendBizFilePath(entityName string) string {
	return filepath.ToSlash(filepath.Join(t.BackendModuleDirectory, "biz", stringcase.ToSnakeCase(entityName)+".go"))
}

// BackendServiceFilePath 返回目标分组内的 Service 文件路径。
func (t ProtoTarget) BackendServiceFilePath(entityName string) string {
	return filepath.ToSlash(filepath.Join(t.BackendModuleDirectory, stringcase.ToSnakeCase(entityName)+"_service.go"))
}

// FrontendAPIFilePath 返回目标分组内的前端 API 文件路径。
func (t ProtoTarget) FrontendAPIFilePath(entityName string) string {
	return filepath.ToSlash(filepath.Join(t.FrontendAPIDirectory, stringcase.ToSnakeCase(entityName)+".ts"))
}

// FrontendPageFilePath 返回目标分组内的前端页面文件路径。
func (t ProtoTarget) FrontendPageFilePath(resourcePath string) string {
	return filepath.ToSlash(filepath.Join(t.FrontendPageDirectory, resourcePath, "index.vue"))
}

// BackendBizImportPath 返回目标分组 Biz 包的 Go 导入路径。
func (t ProtoTarget) BackendBizImportPath() string {
	return "shop/" + strings.TrimPrefix(filepath.ToSlash(filepath.Join(t.BackendModuleDirectory, "biz")), "backend/")
}

// ProtoFilePath 根据 Proto 目录和实体名称生成仓库相对 Proto 文件路径。
func ProtoFilePath(directory string, entityName string) string {
	if directory == "" {
		directory = DefaultProtoDirectory
	}
	return filepath.ToSlash(filepath.Join(ProtoRootPath, directory, stringcase.ToSnakeCase(entityName)+".proto"))
}

// Table 描述一次代码生成所需的表配置快照。
type Table struct {
	ID               int64     // 代码生成表配置 ID
	TableName_       string    // 业务表名
	TableComment     string    // 业务表描述
	BusinessName     string    // 业务名称
	EntityName       string    // 实体名称
	ModulePath       string    // 模块路径
	APIPath          string    // Proto 目录
	ProtoFilePath    string    // 本次生成覆盖的 Proto 文件路径
	PermissionPrefix string    // 权限标识前缀
	ParentMenuID     int64     // 父级菜单 ID
	PageType         string    // 页面类型
	ParentColumn     string    // 树形页面父节点字段
	TreeLabelColumn  string    // 树形页面显示字段
	LeftTreeConfig   string    // 左树右表配置 JSON
	GenBackend       int32     // 是否生成后端
	GenFrontend      int32     // 是否生成前端
	GenSql           int32     // 是否同步菜单权限
	Status           int32     // 配置状态
	CreatedAt        time.Time // 配置创建时间
	UpdatedAt        time.Time // 配置更新时间
}

// Proto 描述一次代码生成所需的 Proto 接口配置快照。
type Proto struct {
	ID                  int64  // Proto 配置 ID
	TableID             int64  // 代码生成表配置 ID
	ColumnName          string // 触发字段名
	TriggerType         string // 触发来源
	APIKind             string // 接口类型
	TargetEntityName    string // 目标实体名
	TargetBusinessName  string // 目标数据库表描述
	MethodName          string // RPC 方法名
	ProtoFilePath       string // Proto 文件路径
	ParentColumn        string // 树接口父节点字段
	LabelColumn         string // 选项显示字段
	ValueColumn         string // 选项取值字段
	GenerateWhenMissing int32  // 缺失时是否生成
	Sort                int32  // 排序
}

// ProtoCheck 描述渲染阶段推导出的 Proto 接口检查项。
type ProtoCheck struct {
	TableID             int64  // 代码生成表配置 ID
	ColumnName          string // 触发字段名
	TriggerType         string // 触发来源
	APIKind             string // 接口类型
	TargetEntityName    string // 目标实体名
	TargetBusinessName  string // 目标数据库表描述
	MethodName          string // RPC 方法名
	ProtoFilePath       string // Proto 文件路径
	Exists              bool   // RPC 是否已经存在
	GenerateWhenMissing bool   // 缺失时是否生成
	ParentColumn        string // 树接口父节点字段
	LabelColumn         string // 选项显示字段
	ValueColumn         string // 选项取值字段
	Message             string // 检查说明
}

// CodeGenProtoPatch 描述向现有 Proto 文件追加的内容。
type CodeGenProtoPatch struct {
	// ServiceNames 需要追加 RPC 的服务名称。
	ServiceNames []string
	// RPCs 按服务名称分组的 RPC 定义。
	RPCs map[string][]string
	// Messages 需要补齐的消息定义。
	Messages []string
}

// Empty 判断补丁内容是否为空。
func (p CodeGenProtoPatch) Empty() bool {
	return len(p.ServiceNames) == 0 && len(p.Messages) == 0
}

// CommonImportRequired 判断追加内容是否依赖 common 响应类型。
func (p CodeGenProtoPatch) CommonImportRequired() bool {
	for _, serviceName := range p.ServiceNames {
		for _, rpc := range p.RPCs[serviceName] {
			if strings.Contains(rpc, ".common.v1.") {
				return true
			}
		}
	}
	return false
}

// CodeGenProtoRPCBlock 描述 Proto service 中可重排的单个 RPC 块。
type CodeGenProtoRPCBlock struct {
	// Name RPC 方法名。
	Name string
	// Content 包含相邻注释的完整 RPC 内容。
	Content string
	// OriginalIndex 原始位置，用于稳定保留未知方法顺序。
	OriginalIndex int
}

// CodeGenSourceMethodBlock 描述 Go 接收者或 TypeScript 类中的可重排方法块。
type CodeGenSourceMethodBlock struct {
	// Name 方法名。
	Name string
	// Content 包含方法注释的完整源码。
	Content string
	// Start 在当前源码中的起始偏移。
	Start int
	// End 在当前源码中的结束偏移。
	End int
	// OriginalIndex 原始位置，用于稳定保留扩展方法顺序。
	OriginalIndex int
}

// CodeGenExternalTarget 描述生成流程依赖的外部实体及其方法。
type CodeGenExternalTarget struct {
	// Table 外部实体对应的生成对象。
	Table *Table
	// Methods 外部实体需要补齐的方法。
	Methods []*Proto
}

// CodeGenMenuSpec 描述待同步的生成菜单。
type CodeGenMenuSpec struct {
	// Menu 待创建或更新的菜单。
	Menu *models.BaseMenu
}

// TableInfo 数据库表元数据查询结果。
type TableInfo struct {
	// TableName 数据库表名。
	TableName string `gorm:"column:table_name"`
	// TableComment 数据库表注释。
	TableComment string `gorm:"column:table_comment"`
}

// CodeGenColumn 汇总数据库字段与用户保存的生成配置。
type CodeGenColumn struct {
	// ID 字段配置 ID。
	ID int64
	// TableID 生成对象 ID。
	TableID int64
	// ColumnName 字段名称。
	ColumnName string
	// ColumnComment 字段注释。
	ColumnComment string
	// DbType 数据库基础类型。
	DbType string
	// ColumnType 数据库完整类型。
	ColumnType string
	// DbLength 字段长度。
	DbLength int32
	// DbScale 小数位数。
	DbScale int32
	// DefaultValue 默认值。
	DefaultValue string
	// HasDefault 是否声明默认值。
	HasDefault bool
	// Extra 数据库附加属性。
	Extra string
	// IsPrimary 是否为主键。
	IsPrimary int32
	// IsAutoIncrement 是否自增。
	IsAutoIncrement int32
	// IsNullable 是否允许为空。
	IsNullable int32
	// GoType Go 字段类型。
	GoType string
	// ProtoType Proto 字段类型。
	ProtoType string
	// TsType TypeScript 字段类型。
	TsType string
	// IsQuery 是否作为查询条件。
	IsQuery int32
	// QueryOperator 查询操作符。
	QueryOperator string
	// QueryComponent 查询组件。
	QueryComponent string
	// IsList 是否在列表展示。
	IsList int32
	// ListComponent 列表组件。
	ListComponent string
	// IsForm 是否在表单展示。
	IsForm int32
	// FormComponent 表单组件。
	FormComponent string
	// IsRequired 表单是否必填。
	IsRequired int32
	// FormMultiple 表单树形选择是否多选。
	FormMultiple bool
	// OptionKind 选项展示类型。
	OptionKind string
	// OptionSourceType 选项数据源类型。
	OptionSourceType string
	// OptionSourceValue 选项数据源值。
	OptionSourceValue string
	// OptionLabelField 选项标签字段。
	OptionLabelField string
	// OptionValueField 选项取值字段。
	OptionValueField string
	// OptionParentField 树形选项父级字段。
	OptionParentField string
	// QueryOption 查询条件独立使用的选项配置。
	QueryOption CodeGenColumnOptionConfig
	// ListOption 列表展示独立使用的选项配置。
	ListOption CodeGenColumnOptionConfig
	// FormOption 表单录入独立使用的选项配置。
	FormOption CodeGenColumnOptionConfig
	// IsStatusField 是否为状态字段。
	IsStatusField int32
	// StatusDataType 状态数据类型。
	StatusDataType string
	// StatusDictCode 状态字典编码。
	StatusDictCode string
	// StatusEnumName 状态枚举名称。
	StatusEnumName string
	// StatusEnabledValue 启用状态值。
	StatusEnabledValue string
	// StatusDisabledValue 禁用状态值。
	StatusDisabledValue string
	// StatusDefaultValue 状态默认值。
	StatusDefaultValue string
	// StatusGenerateAPI 是否生成状态接口。
	StatusGenerateAPI int32
	// StatusTableColumn 是否作为状态列表列。
	StatusTableColumn int32
	// StatusSearch 是否支持状态查询。
	StatusSearch int32
	// StatusSwitch 是否使用状态开关。
	StatusSwitch int32
	// StatusForm 是否在表单配置状态。
	StatusForm int32
	// Sort 字段排序。
	Sort int32
}

// CodeGenColumnQueryConfig 描述字段查询配置。
type CodeGenColumnQueryConfig struct {
	// Enabled 是否启用查询。
	Enabled bool `json:"enabled"`
	// Operator 查询操作符。
	Operator string `json:"operator"`
	// Component 查询组件。
	Component string `json:"component"`
}

// CodeGenColumnListConfig 描述字段列表配置。
type CodeGenColumnListConfig struct {
	// Enabled 是否在列表展示。
	Enabled bool `json:"enabled"`
	// Component 列表组件。
	Component string `json:"component"`
}

// CodeGenColumnFormConfig 描述字段表单配置。
type CodeGenColumnFormConfig struct {
	// Enabled 是否在表单展示。
	Enabled bool `json:"enabled"`
	// Component 表单组件。
	Component string `json:"component"`
	// Required 是否必填。
	Required bool `json:"required"`
	// Multiple 树形选择是否多选。
	Multiple bool `json:"multiple"`
}

// CodeGenColumnOptionConfig 描述字段选项配置。
type CodeGenColumnOptionConfig struct {
	// Kind 选项展示类型。
	Kind string `json:"kind"`
	// SourceType 数据源类型。
	SourceType string `json:"source_type"`
	// SourceValue 数据源值。
	SourceValue string `json:"source_value"`
	// LabelField 标签字段。
	LabelField string `json:"label_field"`
	// ValueField 取值字段。
	ValueField string `json:"value_field"`
	// ParentField 树形父级字段。
	ParentField string `json:"parent_field"`
	// ActiveValue 开关开启值。
	ActiveValue string `json:"active_value"`
	// InactiveValue 开关关闭值。
	InactiveValue string `json:"inactive_value"`
}

// CodeGenStaticOption 描述静态选择项。
type CodeGenStaticOption struct {
	// Label 选择项显示文案。
	Label string `json:"label"`
	// Value 选择项提交值，兼容历史数字和布尔类型。
	Value any `json:"value"`
}

// CodeGenColumnStatusConfig 描述字段状态能力配置。
type CodeGenColumnStatusConfig struct {
	// Enabled 是否启用状态能力。
	Enabled bool `json:"enabled"`
	// DataType 状态数据类型。
	DataType string `json:"data_type"`
	// DictCode 状态字典编码。
	DictCode string `json:"dict_code"`
	// EnumName 状态枚举名称。
	EnumName string `json:"enum_name"`
	// EnabledValue 启用状态值。
	EnabledValue string `json:"enabled_value"`
	// DisabledValue 禁用状态值。
	DisabledValue string `json:"disabled_value"`
	// DefaultValue 默认状态值。
	DefaultValue string `json:"default_value"`
	// GenerateAPI 是否生成状态接口。
	GenerateAPI bool `json:"generate_api"`
	// TableColumn 是否作为列表列。
	TableColumn bool `json:"table_column"`
	// Search 是否支持查询。
	Search bool `json:"search"`
	// Switch 是否使用开关组件。
	Switch bool `json:"switch"`
	// Form 是否在表单展示。
	Form bool `json:"form"`
}

// CodeGenColumnExtraConfig 汇总字段的扩展配置。
type CodeGenColumnExtraConfig struct {
	// Option 选项配置。
	Option CodeGenColumnOptionConfig `json:"option"`
	// Status 状态配置。
	Status CodeGenColumnStatusConfig `json:"status"`
}

// CodeGenLeftTreeConfig 描述左树右表页面配置。
type CodeGenLeftTreeConfig struct {
	// Enabled 是否启用左树布局。
	Enabled bool `json:"enabled"`
	// SourceType 左树数据源类型。
	SourceType string `json:"source_type"`
	// SourceValue 左树数据源值。
	SourceValue string `json:"source_value"`
	// FilterColumn 列表关联筛选字段。
	FilterColumn string `json:"filter_column"`
	// ParentColumn 树节点父级字段。
	ParentColumn string `json:"parent_column"`
	// LabelColumn 树节点标签字段。
	LabelColumn string `json:"label_column"`
	// ValueColumn 树节点取值字段。
	ValueColumn string `json:"value_column"`
}
