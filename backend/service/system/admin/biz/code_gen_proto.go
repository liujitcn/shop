package biz

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	systemadminv1 "shop/api/gen/go/system/admin/v1"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/service/system/admin/codegen"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/go-utils/stringcase"
	"github.com/liujitcn/gorm-kit/repository"
)

const (
	codeGenTriggerCRUD         = "crud"
	codeGenTriggerPageTree     = "page_tree"
	codeGenTriggerLeftTree     = "left_tree"
	codeGenTriggerEntityOption = "entity_option"
	codeGenTriggerFieldOption  = "field_option"
	codeGenTriggerFieldStatus  = "field_status"
	codeGenAPIKindCRUD         = "crud"
	codeGenAPIKindList         = "list"
	codeGenAPIKindOption       = "option"
	codeGenAPIKindTree         = "tree"
	codeGenAPIKindStatus       = "status"
)

var codeGenProtoRPCPattern = regexp.MustCompile(`(?m)^\s*rpc\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)

// codeGenProtoConfigField 表示生成模板依赖的配置字段。
type codeGenProtoConfigField struct {
	label string
	value string
}

// CodeGenProtoCase 管理代码生成 Proto 接口配置。
type CodeGenProtoCase struct {
	*data.CodeGenProtoRepository
	tx                data.Transaction
	codeGenTableRepo  *data.CodeGenTableRepository
	codeGenColumnCase *CodeGenColumnCase
	mapper            *mapper.CopierMapper[systemadminv1.CodeGenProto, models.CodeGenProto]
	tableMapper       *mapper.CopierMapper[systemadminv1.CodeGenTableForm, models.CodeGenTable]
}

// NewCodeGenProtoCase 创建代码生成 Proto 接口业务实例。
func NewCodeGenProtoCase(
	codeGenProtoRepo *data.CodeGenProtoRepository,
	tx data.Transaction,
	codeGenTableRepo *data.CodeGenTableRepository,
	codeGenColumnCase *CodeGenColumnCase,
) *CodeGenProtoCase {
	protoMapper := mapper.NewCopierMapper[systemadminv1.CodeGenProto, models.CodeGenProto]()
	protoMapper.AppendConverters(mapper.NewJSONTypeConverter[*systemadminv1.CodeGenProtoConfig]().NewConverterPair())
	protoMapper.AppendConverters(mapper.NewGenericTypeConverterPair(
		false,
		int32(0),
		func(value bool) int32 {
			// 数据库存储使用 1 表示启用状态。
			if value {
				return 1
			}
			return 0
		},
		func(value int32) bool {
			return value == 1
		},
	))
	tableMapper := mapper.NewCopierMapper[systemadminv1.CodeGenTableForm, models.CodeGenTable]()
	tableMapper.AppendConverters(mapper.NewJSONTypeConverter[*systemadminv1.CodeGenLeftTreeConfig]().NewConverterPair())
	return &CodeGenProtoCase{
		CodeGenProtoRepository: codeGenProtoRepo,
		tx:                     tx,
		codeGenTableRepo:       codeGenTableRepo,
		codeGenColumnCase:      codeGenColumnCase,
		mapper:                 protoMapper,
		tableMapper:            tableMapper,
	}
}

// ListCodeGenProto 查询当前生成配置需要的 Proto 接口。
func (c *CodeGenProtoCase) ListCodeGenProto(ctx context.Context, tableID int64) (*systemadminv1.ListCodeGenProtoResponse, error) {
	if tableID <= 0 {
		return nil, errorsx.InvalidArgument("代码生成表配置ID不能为空")
	}
	table, err := c.codeGenTableRepo.FindByID(ctx, tableID)
	if err != nil {
		return nil, err
	}
	var columns []*systemadminv1.CodeGenColumn
	columns, err = c.codeGenColumnCase.listCodeGenColumns(ctx, tableID)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).CodeGenProto
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.TableID.Eq(tableID)))
	opts = append(opts, repository.Order(query.Sort.Asc()))
	var savedProtos []*models.CodeGenProto
	savedProtos, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	var checks []*systemadminv1.CodeGenProtoCheck
	checks, err = c.inspectCodeGenProtos(ctx, table, c.tableMapper.ToDTO(table), columns, savedProtos)
	if err != nil {
		return nil, err
	}
	return &systemadminv1.ListCodeGenProtoResponse{
		CodeGenProtos: checks,
	}, nil
}

// SaveCodeGenProto 保存代码生成 Proto 接口配置快照。
func (c *CodeGenProtoCase) SaveCodeGenProto(ctx context.Context, req *systemadminv1.SaveCodeGenProtoRequest) error {
	table, err := c.codeGenTableRepo.FindByID(ctx, req.GetTableId())
	if err != nil {
		return err
	}
	var checks *systemadminv1.ListCodeGenProtoResponse
	checks, err = c.ListCodeGenProto(ctx, req.GetTableId())
	if err != nil {
		return err
	}
	expectedByKey := make(map[string]*systemadminv1.CodeGenProtoCheck, len(checks.GetCodeGenProtos()))
	for _, check := range checks.GetCodeGenProtos() {
		expectedByKey[codeGenProtoKey(check.GetProtoFilePath(), check.GetTargetEntityName(), check.GetMethodName())] = check
	}
	protos := make([]*models.CodeGenProto, 0, len(req.GetCodeGenProtos()))
	seen := make(map[string]struct{}, len(req.GetCodeGenProtos()))
	columnNamesByTable := make(map[string]map[string]struct{})
	for index, input := range req.GetCodeGenProtos() {
		if input == nil {
			return errorsx.InvalidArgument("Proto接口配置不能为空")
		}
		key := codeGenProtoKey(input.GetProtoFilePath(), input.GetTargetEntityName(), input.GetMethodName())
		check := expectedByKey[key]
		if check == nil {
			return errorsx.InvalidArgument("Proto接口" + input.GetMethodName() + "不在当前检查结果中")
		}
		if _, exists := seen[key]; exists {
			return errorsx.InvalidArgument("Proto接口" + input.GetMethodName() + "配置重复")
		}
		seen[key] = struct{}{}
		// 已存在的接口不保存补齐选择，避免后续生成流程重复处理。
		if check.GetExists() {
			continue
		}
		proto := &systemadminv1.CodeGenProto{
			TableId:             req.GetTableId(),
			TriggerType:         check.GetTriggerType(),
			ApiKind:             check.GetApiKind(),
			TargetEntityName:    check.GetTargetEntityName(),
			MethodName:          check.GetMethodName(),
			ProtoFilePath:       check.GetProtoFilePath(),
			Config:              mergeCodeGenProtoConfig(check.GetApiKind(), input.GetConfig(), check.GetConfig()),
			GenerateWhenMissing: input.GetGenerateWhenMissing(),
			Sort:                input.GetSort(),
		}
		if proto.Sort == 0 {
			proto.Sort = int32(index + 1)
		}
		targetTableName := codeGenProtoTargetTableName(table, check.GetTargetEntityName())
		if err = c.validateCodeGenProtoColumns(ctx, targetTableName, proto, columnNamesByTable); err != nil {
			return err
		}
		item := c.mapper.ToEntity(proto)
		item.ID = 0
		protos = append(protos, item)
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		query := c.Query(ctx).CodeGenProto
		if err = c.Delete(ctx, repository.Where(query.TableID.Eq(req.GetTableId()))); err != nil {
			return err
		}
		// 没有待生成接口时仅清理旧快照。
		if len(protos) == 0 {
			return nil
		}
		return c.BatchCreate(ctx, protos)
	})
}

// DeleteByTableIDs 删除多个代码生成表配置关联的 Proto 接口配置。
func (c *CodeGenProtoCase) DeleteByTableIDs(ctx context.Context, tableIDs []int64) error {
	if len(tableIDs) == 0 {
		return nil
	}
	query := c.Query(ctx).CodeGenProto
	return c.Delete(ctx, repository.Where(query.TableID.In(tableIDs...)))
}

// validateCodeGenProtoColumns 校验待生成 Proto 接口的类型配置和数据库字段。
func (c *CodeGenProtoCase) validateCodeGenProtoColumns(ctx context.Context, tableName string, proto *systemadminv1.CodeGenProto, columnNamesByTable map[string]map[string]struct{}) error {
	// 未勾选生成的接口不消费类型配置，无需校验字段。
	if !proto.GetGenerateWhenMissing() {
		return nil
	}
	fields, supported := codeGenProtoConfigFields(proto.GetApiKind(), proto.GetConfig())
	// 未知接口类型没有可用的生成模板。
	if !supported {
		return errorsx.InvalidArgument("Proto接口" + proto.GetMethodName() + "的接口类型不支持")
	}
	// 基础接口没有类型配置字段，无需读取数据库元数据。
	if len(fields) == 0 {
		return nil
	}
	columnNames, err := c.loadCodeGenProtoColumnNames(ctx, tableName, columnNamesByTable)
	if err != nil {
		return err
	}
	for _, field := range fields {
		if field.value == "" {
			return errorsx.InvalidArgument("请选择Proto接口" + proto.GetMethodName() + "的" + field.label)
		}
		if _, exists := columnNames[field.value]; !exists {
			return errorsx.InvalidArgument("Proto接口" + proto.GetMethodName() + "的" + field.label + field.value + "不属于目标表" + tableName)
		}
	}
	return nil
}

// inspectCodeGenProtos 推导当前配置需要的 Proto 接口并检查仓库与目标表状态。
func (c *CodeGenProtoCase) inspectCodeGenProtos(ctx context.Context, table *models.CodeGenTable, form *systemadminv1.CodeGenTableForm, columns []*systemadminv1.CodeGenColumn, savedProtos []*models.CodeGenProto) ([]*systemadminv1.CodeGenProtoCheck, error) {
	checks := buildExpectedCodeGenProtos(table, form, columns)
	columnNamesByTable := make(map[string]map[string]struct{})
	for _, check := range checks {
		for _, saved := range savedProtos {
			// 兼容读取旧版复数方法名保存的选择，但检查结果始终输出当前单数契约名。
			if !savedCodeGenProtoMatches(saved, check) {
				continue
			}
			check.GenerateWhenMissing = saved.GenerateWhenMissing == 1
			savedProto := c.mapper.ToDTO(saved)
			check.Config = mergeCodeGenProtoConfig(check.GetApiKind(), savedProto.GetConfig(), check.GetConfig())
			break
		}
		check.Exists, check.Message = codeGenProtoExists(check.GetProtoFilePath(), check.GetTargetEntityName(), check.GetMethodName())
		// 仓库中已存在接口时不再进入后续生成流程。
		if check.Exists {
			check.GenerateWhenMissing = false
			continue
		}
		// 用户已取消选择时保留保存结果，不再做默认字段判断。
		if !check.GenerateWhenMissing {
			continue
		}
		targetTableName := codeGenProtoTargetTableName(table, check.GetTargetEntityName())
		// 默认字段只用于判断新接口能否默认勾选，不能回填到未保存的配置中。
		selectionConfig := mergeCodeGenProtoConfig(
			check.GetApiKind(),
			check.GetConfig(),
			defaultCodeGenProtoConfig(check.GetApiKind()),
		)
		fields, supported := codeGenProtoConfigFields(check.GetApiKind(), selectionConfig)
		// 未知接口类型没有可用生成模板，不能默认勾选。
		if !supported {
			check.GenerateWhenMissing = false
			continue
		}
		// 基础接口不依赖类型配置字段，保持默认勾选。
		if len(fields) == 0 {
			continue
		}
		columnNames, err := c.loadCodeGenProtoColumnNames(ctx, targetTableName, columnNamesByTable)
		if err != nil {
			return nil, err
		}
		// 目标表缺少当前配置需要的任一字段时取消默认勾选。
		if !hasCodeGenProtoConfigColumns(check.GetApiKind(), selectionConfig, columnNames) {
			check.GenerateWhenMissing = false
		}
	}
	return checks, nil
}

// loadCodeGenProtoColumnNames 加载并缓存目标表字段名。
func (c *CodeGenProtoCase) loadCodeGenProtoColumnNames(ctx context.Context, tableName string, columnNamesByTable map[string]map[string]struct{}) (map[string]struct{}, error) {
	columnNames := columnNamesByTable[tableName]
	// 同一目标表已加载时直接复用字段集合。
	if columnNames != nil {
		return columnNames, nil
	}
	databaseColumns, err := c.codeGenColumnCase.listDatabaseColumns(ctx, tableName)
	if err != nil {
		return nil, err
	}
	columnNames = make(map[string]struct{}, len(databaseColumns))
	for _, column := range databaseColumns {
		columnNames[column.ColumnName] = struct{}{}
	}
	columnNamesByTable[tableName] = columnNames
	return columnNames, nil
}

// buildExpectedCodeGenProtos 根据表与字段配置推导所需 Proto 接口。
func buildExpectedCodeGenProtos(table *models.CodeGenTable, form *systemadminv1.CodeGenTableForm, columns []*systemadminv1.CodeGenColumn) []*systemadminv1.CodeGenProtoCheck {
	protoPath := defaultCodeGenProtoPath(table)
	entity := table.EntityName
	checks := make([]*systemadminv1.CodeGenProtoCheck, 0, 10)
	// 树形页面使用树接口，普通页面使用分页与平铺选项接口。
	if table.PageType == "tree" {
		checks = append(checks,
			newCodeGenProtoCheck(table.ID, codeGenTriggerPageTree, codeGenAPIKindTree, entity, "Tree"+entity, protoPath),
			newCodeGenProtoCheck(table.ID, codeGenTriggerEntityOption, codeGenAPIKindOption, entity, "Option"+entity, protoPath),
		)
	} else {
		checks = append(checks,
			newCodeGenProtoCheck(table.ID, codeGenTriggerCRUD, codeGenAPIKindList, entity, "Page"+entity, protoPath),
			newCodeGenProtoCheck(table.ID, codeGenTriggerEntityOption, codeGenAPIKindOption, entity, "Option"+entity, protoPath),
		)
	}
	checks = append(checks,
		newCodeGenProtoCheck(table.ID, codeGenTriggerCRUD, codeGenAPIKindCRUD, entity, "Get"+entity, protoPath),
		newCodeGenProtoCheck(table.ID, codeGenTriggerCRUD, codeGenAPIKindCRUD, entity, "Create"+entity, protoPath),
		newCodeGenProtoCheck(table.ID, codeGenTriggerCRUD, codeGenAPIKindCRUD, entity, "Update"+entity, protoPath),
		newCodeGenProtoCheck(table.ID, codeGenTriggerCRUD, codeGenAPIKindCRUD, entity, "Delete"+entity, protoPath),
	)
	leftTree := form.GetLeftTreeConfig()
	// 左树页面的数据表来源需要在目标实体上提供树形选项接口。
	if table.PageType == "left_tree" && leftTree.GetTableName() != "" {
		target := stringcase.ToPascalCase(leftTree.GetTableName())
		checks = append(checks, newCodeGenProtoCheck(
			table.ID,
			codeGenTriggerLeftTree,
			codeGenAPIKindTree,
			target,
			"Option"+target,
			defaultTargetCodeGenProtoPath(table, target),
		))
	}
	statusCount := 0
	// 状态接口数量决定方法名是否需要带字段名以避免冲突。
	for _, column := range columns {
		listConfig := column.GetListConfig()
		if listConfig.GetEnabled() && listConfig.GetComponent() == "switch" {
			statusCount++
		}
	}
	for _, column := range columns {
		listConfig := column.GetListConfig()
		// 列表使用开关时为当前字段补充状态变更 RPC。
		if listConfig.GetEnabled() && listConfig.GetComponent() == "switch" {
			checks = append(checks, newCodeGenProtoCheck(
				table.ID,
				codeGenTriggerFieldStatus,
				codeGenAPIKindStatus,
				entity,
				codeGenStatusMethodName(entity, column.GetColumnName(), statusCount),
				protoPath,
			))
		}
		var options []*systemadminv1.CodeGenColumnOptionConfig
		// 只检查已启用配置中的数据表选项，查询、列表和表单彼此独立。
		if column.GetQueryConfig().GetEnabled() {
			options = append(options, column.GetQueryConfig().GetOption())
		}
		if column.GetListConfig().GetEnabled() {
			options = append(options, column.GetListConfig().GetOption())
		}
		if column.GetFormConfig().GetEnabled() {
			options = append(options, column.GetFormConfig().GetOption())
		}
		for _, option := range options {
			// 只有数据表来源的选项字段才需要检查目标实体接口。
			if option.GetKind() == "" || option.GetSourceType() != "table" {
				continue
			}
			target := stringcase.ToPascalCase(option.GetSourceValue())
			// 来源未声明实体名时回退到当前实体。
			if target == "" {
				target = entity
			}
			apiKind := codeGenAPIKindOption
			// 树形选项需要使用树接口类型。
			if option.GetKind() == "tree" {
				apiKind = codeGenAPIKindTree
			}
			checks = append(checks, newCodeGenProtoCheck(
				table.ID,
				codeGenTriggerFieldOption,
				apiKind,
				target,
				"Option"+target,
				defaultTargetCodeGenProtoPath(table, target),
			))
		}
	}
	list := make([]*systemadminv1.CodeGenProtoCheck, 0, len(checks))
	seen := make(map[string]struct{}, len(checks))
	for _, check := range checks {
		key := codeGenProtoKey(check.GetProtoFilePath(), check.GetTargetEntityName(), check.GetMethodName())
		// 同一目标服务的同名 RPC 只保留一份检查项。
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		list = append(list, check)
	}
	return list
}

// newCodeGenProtoCheck 创建默认勾选生成的 Proto 接口检查项。
func newCodeGenProtoCheck(tableID int64, triggerType string, apiKind string, targetEntity string, methodName string, protoPath string) *systemadminv1.CodeGenProtoCheck {
	return &systemadminv1.CodeGenProtoCheck{
		TableId:             tableID,
		TriggerType:         triggerType,
		ApiKind:             apiKind,
		TargetEntityName:    targetEntity,
		MethodName:          methodName,
		ProtoFilePath:       protoPath,
		Config:              &systemadminv1.CodeGenProtoConfig{},
		GenerateWhenMissing: true,
	}
}

// defaultCodeGenProtoConfig 返回接口类型用于默认选择判断的约定字段。
func defaultCodeGenProtoConfig(apiKind string) *systemadminv1.CodeGenProtoConfig {
	// 不同生成模板只保留自身消费的配置字段。
	switch apiKind {
	case codeGenAPIKindOption:
		return &systemadminv1.CodeGenProtoConfig{LabelColumn: "name", ValueColumn: "id"}
	case codeGenAPIKindTree:
		return &systemadminv1.CodeGenProtoConfig{ParentColumn: "parent_id", LabelColumn: "name", ValueColumn: "id"}
	case codeGenAPIKindStatus:
		return &systemadminv1.CodeGenProtoConfig{StatusColumn: "status"}
	default:
		return &systemadminv1.CodeGenProtoConfig{}
	}
}

// codeGenProtoConfigFields 返回接口类型生成模板依赖的配置字段。
func codeGenProtoConfigFields(apiKind string, config *systemadminv1.CodeGenProtoConfig) ([]codeGenProtoConfigField, bool) {
	// 每种接口类型只返回生成模板实际消费的固定字段。
	switch apiKind {
	case codeGenAPIKindCRUD, codeGenAPIKindList:
		return nil, true
	case codeGenAPIKindOption:
		return []codeGenProtoConfigField{
			{label: "显示字段", value: config.GetLabelColumn()},
			{label: "值字段", value: config.GetValueColumn()},
		}, true
	case codeGenAPIKindTree:
		return []codeGenProtoConfigField{
			{label: "父节点字段", value: config.GetParentColumn()},
			{label: "显示字段", value: config.GetLabelColumn()},
			{label: "值字段", value: config.GetValueColumn()},
		}, true
	case codeGenAPIKindStatus:
		return []codeGenProtoConfigField{
			{label: "状态字段", value: config.GetStatusColumn()},
		}, true
	default:
		return nil, false
	}
}

// hasCodeGenProtoConfigColumns 判断目标表是否包含接口配置所需的全部字段。
func hasCodeGenProtoConfigColumns(apiKind string, config *systemadminv1.CodeGenProtoConfig, columnNames map[string]struct{}) bool {
	fields, supported := codeGenProtoConfigFields(apiKind, config)
	// 未知类型不能进入默认生成选择。
	if !supported {
		return false
	}
	for _, field := range fields {
		if field.value == "" {
			return false
		}
		if _, exists := columnNames[field.value]; !exists {
			return false
		}
	}
	return true
}

// mergeCodeGenProtoConfig 合并用户配置与推导默认值，并按接口类型裁剪无关字段。
func mergeCodeGenProtoConfig(apiKind string, preferred *systemadminv1.CodeGenProtoConfig, fallback *systemadminv1.CodeGenProtoConfig) *systemadminv1.CodeGenProtoConfig {
	config := &systemadminv1.CodeGenProtoConfig{}
	// 每种接口类型只合并生成模板需要的字段。
	switch apiKind {
	case codeGenAPIKindOption:
		config.LabelColumn = preferred.GetLabelColumn()
		config.ValueColumn = preferred.GetValueColumn()
		if config.LabelColumn == "" {
			config.LabelColumn = fallback.GetLabelColumn()
		}
		if config.ValueColumn == "" {
			config.ValueColumn = fallback.GetValueColumn()
		}
	case codeGenAPIKindTree:
		config.ParentColumn = preferred.GetParentColumn()
		config.LabelColumn = preferred.GetLabelColumn()
		config.ValueColumn = preferred.GetValueColumn()
		if config.ParentColumn == "" {
			config.ParentColumn = fallback.GetParentColumn()
		}
		if config.LabelColumn == "" {
			config.LabelColumn = fallback.GetLabelColumn()
		}
		if config.ValueColumn == "" {
			config.ValueColumn = fallback.GetValueColumn()
		}
	case codeGenAPIKindStatus:
		config.StatusColumn = preferred.GetStatusColumn()
		if config.StatusColumn == "" {
			config.StatusColumn = fallback.GetStatusColumn()
		}
	}
	return config
}

// defaultCodeGenProtoPath 返回当前实体默认 Proto 文件路径。
func defaultCodeGenProtoPath(table *models.CodeGenTable) string {
	return codegen.ProtoFilePath(table.APIPath, table.EntityName)
}

// defaultTargetCodeGenProtoPath 返回关联实体默认 Proto 文件路径。
func defaultTargetCodeGenProtoPath(table *models.CodeGenTable, target string) string {
	// 关联目标就是当前实体时复用当前实体默认路径。
	if target == table.EntityName {
		return defaultCodeGenProtoPath(table)
	}
	return codegen.ProtoFilePath(table.APIPath, target)
}

// codeGenProtoTargetTableName 返回接口目标实体对应的数据库表名。
func codeGenProtoTargetTableName(table *models.CodeGenTable, targetEntity string) string {
	// 当前实体保留代码生成配置中的原始表名。
	if targetEntity == table.EntityName {
		return table.Name
	}
	return stringcase.ToSnakeCase(targetEntity)
}

// savedCodeGenProtoMatches 判断已保存配置是否对应当前检查项，并兼容旧版复数契约名。
func savedCodeGenProtoMatches(saved *models.CodeGenProto, check *systemadminv1.CodeGenProtoCheck) bool {
	if saved.ProtoFilePath != check.GetProtoFilePath() || saved.TargetEntityName != check.GetTargetEntityName() {
		return false
	}
	if saved.MethodName == check.GetMethodName() {
		return true
	}
	legacyMethodName := legacyPluralCodeGenMethodName(check.GetTargetEntityName(), check.GetMethodName())
	return legacyMethodName != "" && saved.MethodName == legacyMethodName
}

// legacyPluralCodeGenMethodName 返回旧版 Page、Tree、Option 复数契约名。
func legacyPluralCodeGenMethodName(entity string, methodName string) string {
	for _, prefix := range []string{"Page", "Tree", "Option"} {
		if methodName == prefix+entity {
			return prefix + pluralizeCodeGenEntity(entity)
		}
	}
	return ""
}

// pluralizeCodeGenEntity 返回兼容旧版配置所需的实体复数形式。
func pluralizeCodeGenEntity(value string) string {
	// 以 s 结尾的实体名追加 es。
	if strings.HasSuffix(value, "s") {
		return value + "es"
	}
	// 以 y 结尾的实体名使用 ies 形式。
	if strings.HasSuffix(value, "y") {
		return strings.TrimSuffix(value, "y") + "ies"
	}
	return value + "s"
}

// codeGenStatusMethodName 返回状态字段对应的方法名。
func codeGenStatusMethodName(entity string, columnName string, statusCount int) string {
	// 只有一个状态字段时沿用简洁的统一方法名。
	if statusCount <= 1 {
		return "Set" + entity + "Status"
	}
	fieldName := stringcase.ToPascalCase(columnName)
	// 字段名已带 Status 后缀时避免重复拼接。
	if strings.HasSuffix(fieldName, "Status") {
		return "Set" + entity + fieldName
	}
	return "Set" + entity + fieldName + "Status"
}

// codeGenProtoKey 返回 Proto 接口配置稳定键。
func codeGenProtoKey(protoPath string, targetEntity string, methodName string) string {
	return protoPath + ":" + targetEntity + ":" + methodName
}

// codeGenProtoExists 检查目标 Proto 服务中是否已有指定 RPC。
func codeGenProtoExists(protoPath string, targetEntity string, methodName string) (bool, string) {
	fullPath, err := safeCodeGenProtoPath(protoPath)
	if err != nil {
		return false, err.Error()
	}
	var content []byte
	content, err = os.ReadFile(fullPath)
	if err != nil {
		return false, "Proto文件不存在"
	}
	start, end := findCodeGenProtoServiceBounds(string(content), targetEntity+"Service")
	// 目标 service 不存在时无法继续检查 RPC。
	if start < 0 || end < 0 {
		return false, "Proto服务不存在"
	}
	for _, match := range codeGenProtoRPCPattern.FindAllStringSubmatch(string(content)[start:end], -1) {
		// 正则捕获组中的方法名与目标一致即视为已存在。
		if len(match) > 1 && match[1] == methodName {
			return true, "已存在"
		}
	}
	return false, "缺少，可选择生成"
}

// safeCodeGenProtoPath 返回仓库内安全的 Proto 文件路径。
func safeCodeGenProtoPath(protoPath string) (string, error) {
	// 只允许仓库相对路径，拒绝空路径和绝对路径。
	if protoPath == "" || filepath.IsAbs(protoPath) {
		return "", os.ErrInvalid
	}
	cleanPath := filepath.Clean(protoPath)
	// 清理后仍越过仓库根目录的路径视为非法输入。
	if cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return "", os.ErrInvalid
	}
	root, err := os.Getwd()
	if err != nil {
		return "", err
	}
	// 服务从 backend 目录启动时向上定位仓库根目录。
	if filepath.Base(root) == "backend" {
		root = filepath.Dir(root)
	}
	fullPath := filepath.Join(root, cleanPath)
	var relativePath string
	relativePath, err = filepath.Rel(root, fullPath)
	// 再次校验相对路径，避免不同平台路径规则绕过边界检查。
	if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", os.ErrInvalid
	}
	return fullPath, nil
}

// findCodeGenProtoServiceBounds 返回目标 Proto service 的大括号范围。
func findCodeGenProtoServiceBounds(content string, serviceName string) (int, int) {
	pattern := regexp.MustCompile(`(?m)^[\t ]*service[\t ]+` + regexp.QuoteMeta(serviceName) + `[\t ]*\{`)
	location := pattern.FindStringIndex(content)
	// 未找到 service 声明时返回无效边界。
	if location == nil {
		return -1, -1
	}
	openIndex := strings.LastIndex(content[location[0]:location[1]], "{") + location[0]
	depth := 0
	for index := openIndex; index < len(content); index++ {
		// 通过大括号深度定位 service 的闭合位置。
		switch content[index] {
		case '{':
			depth++
		case '}':
			depth--
			// 深度归零表示当前 service 已结束。
			if depth == 0 {
				return openIndex, index
			}
		}
	}
	return -1, -1
}
