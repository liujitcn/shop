package codegen

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/errorsx"

	"github.com/liujitcn/go-utils/stringcase"
)

// LeftTreeConfigFromTable 从生成对象模型读取左树配置。
func LeftTreeConfigFromTable(table *Table) CodeGenLeftTreeConfig {
	if table == nil {
		return CodeGenLeftTreeConfig{}
	}
	var config CodeGenLeftTreeConfig
	if table.LeftTreeConfig != "" {
		_ = json.Unmarshal([]byte(table.LeftTreeConfig), &config)
	}
	if table.PageType == PageTypeLeftTree {
		config.Enabled = true
	}
	return config
}

// EntityOptionColumns 推导当前实体 Option 接口使用的父级、显示和值字段。
func EntityOptionColumns(table *Table, columns []*CodeGenColumn) (string, string, string) {
	parentColumn := DefaultString(table.ParentColumn, "parent_id")
	labelColumn := table.TreeLabelColumn
	valueColumn := "id"
	for _, column := range columns {
		if column.IsPrimary == 1 {
			valueColumn = column.ColumnName
			break
		}
	}
	if labelColumn != "" {
		return parentColumn, labelColumn, valueColumn
	}
	for _, candidate := range []string{"name", "title", "label", "code"} {
		if column := FindColumnByName(columns, candidate); column != nil {
			return parentColumn, column.ColumnName, valueColumn
		}
	}
	for _, column := range columns {
		if IsStringColumn(column) && !isManagedAuditColumn(column.ColumnName) {
			return parentColumn, column.ColumnName, valueColumn
		}
	}
	return parentColumn, valueColumn, valueColumn
}

// FindColumnByName 按数据库字段名查找生成字段配置。
func FindColumnByName(columns []*CodeGenColumn, columnName string) *CodeGenColumn {
	for _, column := range columns {
		if column.ColumnName == columnName {
			return column
		}
	}
	return nil
}

// IsStringColumn 判断字段是否可安全赋给 string。
func IsStringColumn(column *CodeGenColumn) bool {
	goType := DefaultString(column.GoType, InferGoType(column.DbType))
	return goType == "string"
}

// ProtoMethodExists 检查指定 Proto 文件的目标服务中是否存在 RPC 方法。
func ProtoMethodExists(protoPath string, targetEntity string, methodName string) (bool, string) {
	return (&renderer{}).protoMethodExists(protoPath, targetEntity, methodName)
}

func (c *renderer) buildExpectedProtoChecks(table *Table, columns []*CodeGenColumn) []*ProtoCheck {
	protoPath := c.defaultProtoPath(table)
	entity := table.EntityName
	leftTreeConfig := LeftTreeConfigFromTable(table)
	parentColumn, labelColumn, valueColumn := EntityOptionColumns(table, columns)
	statusColumnList := statusAPIColumns(columns)
	statusColumnCount := len(statusColumnList)
	checks := make([]*ProtoCheck, 0, 8)
	if table.PageType == PageTypeTree {
		checks = append(checks,
			c.newProtoCheck(table.ID, "", TriggerPageTree, APIKindTree, entity, "Tree"+entity, protoPath, parentColumn, labelColumn, valueColumn, true),
			c.newProtoCheck(table.ID, "", TriggerEntityOption, APIKindTree, entity, "Option"+entity, protoPath, parentColumn, labelColumn, valueColumn, true),
		)
	} else {
		checks = append(checks,
			c.newProtoCheck(table.ID, "", TriggerCRUD, APIKindList, entity, "Page"+entity, protoPath, "", "", "", true),
			c.newProtoCheck(table.ID, "", TriggerEntityOption, APIKindOption, entity, "Option"+entity, protoPath, "", labelColumn, valueColumn, true),
		)
	}
	checks = append(checks,
		c.newProtoCheck(table.ID, "", TriggerCRUD, APIKindCRUD, entity, "Get"+entity, protoPath, "", "", "", true),
		c.newProtoCheck(table.ID, "", TriggerCRUD, APIKindCRUD, entity, "Create"+entity, protoPath, "", "", "", true),
		c.newProtoCheck(table.ID, "", TriggerCRUD, APIKindCRUD, entity, "Update"+entity, protoPath, "", "", "", true),
		c.newProtoCheck(table.ID, "", TriggerCRUD, APIKindCRUD, entity, "Delete"+entity, protoPath, "", "", "", true),
	)
	if (table.PageType == PageTypeLeftTree || leftTreeConfig.Enabled) && leftTreeConfig.SourceType == OptionSourceTable {
		target := DefaultString(stringcase.ToPascalCase(leftTreeConfig.SourceValue), entity)
		targetProtoPath := c.defaultTargetProtoPath(table, target)
		checks = append(checks,
			c.newProtoCheck(table.ID, "", TriggerLeftTree, APIKindTree, target, "Option"+target, targetProtoPath, leftTreeConfig.ParentColumn, leftTreeConfig.LabelColumn, leftTreeConfig.ValueColumn, false),
		)
	}

	for _, column := range columns {
		if column.IsStatusField == 1 && column.StatusGenerateAPI == 1 {
			checks = append(checks,
				c.newProtoCheck(
					table.ID,
					column.ColumnName,
					TriggerFieldStatus,
					APIKindStatus,
					entity,
					statusMethodNameForColumn(table, column, statusColumnCount),
					protoPath,
					"",
					"",
					column.ColumnName,
					true,
				),
			)
		}
		for _, option := range enabledCodeGenColumnOptions(column) {
			if option.SourceType != OptionSourceTable {
				continue
			}
			target := DefaultString(stringcase.ToPascalCase(option.SourceValue), entity)
			targetProtoPath := c.defaultTargetProtoPath(table, target)
			if option.Kind == APIKindTree {
				checks = append(checks,
					c.newProtoCheck(table.ID, column.ColumnName, TriggerFieldOption, APIKindTree, target, "Option"+target, targetProtoPath, option.ParentField, option.LabelField, option.ValueField, false),
				)
				continue
			}
			checks = append(checks,
				c.newProtoCheck(table.ID, column.ColumnName, TriggerFieldOption, APIKindOption, target, "Option"+target, targetProtoPath, "", option.LabelField, option.ValueField, false),
			)
		}
	}
	return dedupeProtoChecks(checks)
}

// enabledCodeGenColumnOptions 返回当前字段各作用域实际启用的选项配置。
func enabledCodeGenColumnOptions(column *CodeGenColumn) []CodeGenColumnOptionConfig {
	options := make([]CodeGenColumnOptionConfig, 0, 3)
	if column.IsQuery == 1 && column.QueryOption.Kind != "" {
		options = append(options, column.QueryOption)
	}
	if column.IsList == 1 && column.ListOption.Kind != "" {
		options = append(options, column.ListOption)
	}
	if column.IsForm == 1 && column.FormOption.Kind != "" {
		options = append(options, column.FormOption)
	}
	return options
}

// newProtoCheck 创建 Proto 检查项。
func (c *renderer) newProtoCheck(tableID int64, columnName string, triggerType, apiKind, targetEntity, methodName, protoPath, parentColumn, labelColumn, valueColumn string, generate bool) *ProtoCheck {
	return &ProtoCheck{
		TableID:             tableID,
		ColumnName:          columnName,
		TriggerType:         triggerType,
		APIKind:             apiKind,
		TargetEntityName:    targetEntity,
		MethodName:          methodName,
		ProtoFilePath:       protoPath,
		GenerateWhenMissing: generate,
		ParentColumn:        parentColumn,
		LabelColumn:         labelColumn,
		ValueColumn:         valueColumn,
	}
}

// checkProtoMessageExists 检查指定 Proto 文件中是否存在 message。
func (c *renderer) checkProtoMessageExists(protoPath string, messageName string) bool {
	content, err := c.readRepoFile(protoPath)
	if err != nil {
		return false
	}
	matches := protoMessagePattern.FindAllStringSubmatch(string(content), -1)
	for _, match := range matches {
		if len(match) > 1 && match[1] == messageName {
			return true
		}
	}
	return false
}

// resolveCodeGenOutputPaths 合并本次请求路径和默认路径，并校验启用的生成目标。
func (c *renderer) resolveCodeGenOutputPaths(table *Table, requested *adminv1.CodeGenOutputPaths) (*adminv1.CodeGenOutputPaths, error) {
	snakeEntity := stringcase.ToSnakeCase(table.EntityName)
	// 默认路径统一由实体名和资源路径推导，保证首次进入预览时即可直接生成。
	paths := &adminv1.CodeGenOutputPaths{
		ProtoFilePath:          "backend/api/protos/admin/v1/" + snakeEntity + ".proto",
		BackendBizFilePath:     "backend/service/admin/biz/" + snakeEntity + ".go",
		BackendServiceFilePath: "backend/service/admin/" + snakeEntity + "_service.go",
		FrontendApiFilePath:    "frontend/admin/src/api/admin/" + snakeEntity + ".ts",
		FrontendPageFilePath:   "frontend/admin/src/views/" + c.frontendResourcePath(table) + "/index.vue",
		SqlFilePath:            "sql/generated/" + table.TableName_ + ".sql",
	}
	// 请求只覆盖显式填写的字段，空值继续使用默认路径。
	if requested != nil {
		if requested.GetProtoFilePath() != "" {
			paths.ProtoFilePath = requested.GetProtoFilePath()
		}
		if requested.GetBackendBizFilePath() != "" {
			paths.BackendBizFilePath = requested.GetBackendBizFilePath()
		}
		if requested.GetBackendServiceFilePath() != "" {
			paths.BackendServiceFilePath = requested.GetBackendServiceFilePath()
		}
		if requested.GetFrontendApiFilePath() != "" {
			paths.FrontendApiFilePath = requested.GetFrontendApiFilePath()
		}
		if requested.GetFrontendPageFilePath() != "" {
			paths.FrontendPageFilePath = requested.GetFrontendPageFilePath()
		}
		if requested.GetSqlFilePath() != "" {
			paths.SqlFilePath = requested.GetSqlFilePath()
		}
	}

	targets := []struct {
		label   string
		path    *string
		enabled bool
	}{
		{label: "Proto文件路径", path: &paths.ProtoFilePath, enabled: table.GenBackend == 1},
		{label: "后端Biz文件路径", path: &paths.BackendBizFilePath, enabled: table.GenBackend == 1},
		{label: "后端Service文件路径", path: &paths.BackendServiceFilePath, enabled: table.GenBackend == 1},
		{label: "前端API文件路径", path: &paths.FrontendApiFilePath, enabled: table.GenFrontend == 1},
		{label: "前端页面文件路径", path: &paths.FrontendPageFilePath, enabled: table.GenFrontend == 1},
		{label: "SQL文件路径", path: &paths.SqlFilePath, enabled: table.GenSql == 1},
	}
	seen := make(map[string]string, len(targets))
	// 只校验本轮启用的目标；未启用模块的路径允许暂时为空或保留旧配置。
	for _, target := range targets {
		*target.path = filepath.ToSlash(filepath.Clean(*target.path))
		if !target.enabled {
			continue
		}
		var pathErr error
		_, pathErr = SafeRepoFilePath(*target.path)
		if pathErr != nil {
			return nil, errorsx.InvalidArgument(target.label + "无效").WithCause(pathErr)
		}
		if previousLabel, ok := seen[*target.path]; ok {
			return nil, errorsx.InvalidArgument(fmt.Sprintf("%s不能与%s使用相同路径", target.label, previousLabel))
		}
		seen[*target.path] = target.label
	}
	// 模块边界校验用于阻止 Proto、Go、Vue 和 SQL 文件落入错误目录。
	layoutErr := validateCodeGenOutputPathLayout(paths)
	if layoutErr != nil {
		return nil, layoutErr
	}
	return paths, nil
}

// applyCodeGenOutputPaths 创建仅供本次预览或生成使用的配置副本。
func (c *renderer) applyCodeGenOutputPaths(table *Table, methods []*Proto, paths *adminv1.CodeGenOutputPaths) (*Table, []*Proto) {
	generationTable := *table
	generationTable.APIPath = paths.GetProtoFilePath()
	generationMethods := make([]*Proto, 0, len(methods))
	for _, method := range methods {
		generationMethod := *method
		// 当前实体的方法跟随本次临时 Proto 路径，外部选项实体仍使用其自身路径。
		if DefaultString(generationMethod.TargetEntityName, table.EntityName) == table.EntityName {
			generationMethod.ProtoFilePath = paths.GetProtoFilePath()
		}
		generationMethods = append(generationMethods, &generationMethod)
	}
	return &generationTable, generationMethods
}

// buildPreviewFiles 按后端、前端顺序构建本轮全部预览文件。
func (c *renderer) buildPreviewFiles(table *Table, columns []*CodeGenColumn, methods []*Proto, paths *adminv1.CodeGenOutputPaths) []*adminv1.CodeGenPreviewFile {
	generatedMethods := c.generatedProtoMethods(table, columns, methods)
	frontendMethods := c.frontendProtoMethods(table, columns, methods)
	files := make([]*adminv1.CodeGenPreviewFile, 0, 5)
	// 生成对象来源于已存在的数据表，菜单权限由生成流程直接同步到数据库，不生成 SQL 文件。
	if table.GenBackend == 1 {
		// 主 Proto 先生成，随后按路径去重补齐外部选项目标的 Proto 文件。
		files = append(files, c.newTargetProtoPreviewFile(table, columns, generatedMethods, c.defaultProtoPath(table)))
		seenProtoPaths := map[string]struct{}{c.defaultProtoPath(table): {}}
		for _, method := range generatedMethods {
			if method.GenerateWhenMissing != 1 || method.ProtoFilePath == "" {
				continue
			}
			if _, ok := seenProtoPaths[method.ProtoFilePath]; ok {
				continue
			}
			seenProtoPaths[method.ProtoFilePath] = struct{}{}
			files = append(files, c.newTargetProtoPreviewFile(table, columns, generatedMethods, method.ProtoFilePath))
		}
		// 主实体文件存在时只追加缺失方法，不覆盖已有业务实现。
		files = append(files,
			c.newPatchedPreviewFile(paths.GetBackendBizFilePath(), c.renderBackendBizFile(table, columns, generatedMethods), func(content string) string {
				return c.appendMainBizMethods(content, table, columns, generatedMethods)
			}),
			c.newPatchedPreviewFile(paths.GetBackendServiceFilePath(), c.renderBackendServiceFile(table, columns, generatedMethods), func(content string) string {
				return c.appendMainServiceMethods(content, table, columns, generatedMethods)
			}),
		)
		files = append(files, c.newExternalTargetBackendPreviewFiles(table, generatedMethods)...)
		files = append(files, c.newAdminRegistrationPreviewFiles(table, generatedMethods)...)
	}
	if table.GenFrontend == 1 {
		// 前端 API 同样只追加缺失方法，页面则要求查询和 CRUD 契约完整后才生成。
		files = append(files, c.newPatchedPreviewFile(paths.GetFrontendApiFilePath(), c.renderFrontendAPIFile(table, columns, frontendMethods), func(content string) string {
			return c.appendMainFrontendAPIMethods(content, table, columns, frontendMethods)
		}))
		pagePath := paths.GetFrontendPageFilePath()
		if frontendPageMethodsComplete(table, frontendMethods) {
			files = append(files, c.newPreviewFile(pagePath, c.renderFrontendPageFile(table, columns, frontendMethods, paths)))
		} else {
			pageFile := c.newPreviewFile(pagePath, "")
			pageFile.Action = "skip"
			pageFile.Message = "缺少页面所需的查询或 CRUD 接口，已跳过页面生成"
			files = append(files, pageFile)
		}
		files = append(files, c.newExternalTargetFrontendPreviewFiles(table, frontendMethods)...)
	}
	return files
}

// appendMainBizMethods 根据业务文件自身内容追加尚未实现的方法及其树形辅助方法。
func (c *renderer) appendMainBizMethods(content string, table *Table, columns []*CodeGenColumn, methods []*Proto) string {
	// 候选文件只用于提取当前配置下应存在的方法，已有文件仍是最终合并基准。
	candidate := c.renderBackendBizFile(table, columns, methods)
	generatedReceiver := table.EntityName + "Case"
	existingReceiver := goReceiverType(content, "Case")
	if existingReceiver == "" {
		return content
	}
	// Mapper 已统一处理时间字段，清理旧模板生成的重复赋值。
	content = redundantTimePattern.ReplaceAllString(content, "")
	if !strings.Contains(content, "_time.") {
		content = strings.Replace(content, "\t_time \"github.com/liujitcn/go-utils/time\"\n", "", 1)
	}
	methodNames := missingGoReceiverMethodNamesFold(candidate, content, generatedReceiver, existingReceiver)
	if len(methodNames) == 0 {
		return reorderGoReceiverMethods(content, existingReceiver)
	}
	// 已有业务文件可能采用不同实体名，先从接收者依赖中识别真实 Repository 类型。
	_, repositoryType := goStructDependency(content, existingReceiver, "data", "Repository")
	repositoryType = strings.TrimSuffix(repositoryType, "Repository")
	if repositoryType == "" {
		return reorderGoReceiverMethods(content, existingReceiver)
	}

	methodContent := strings.Join(extractGoMethods(candidate, methodNames), "\n\n")
	if methodContent == "" {
		return reorderGoReceiverMethods(content, existingReceiver)
	}
	// 仅改写新追加的方法片段，避免替换已有业务代码中的同名标识符。
	entityVar := stringcase.ToCamelCase(table.EntityName)
	methodContent = strings.ReplaceAll(methodContent, "*"+generatedReceiver, "*"+existingReceiver)
	methodContent = strings.ReplaceAll(methodContent, "models."+table.EntityName, "models."+repositoryType)
	methodContent = strings.ReplaceAll(methodContent, "c.Query(ctx)."+table.EntityName, "c.Query(ctx)."+repositoryType)
	methodContent = strings.ReplaceAll(methodContent, "c.formMapper.ToEntity(req)", "mapper.NewCopierMapper[adminv1."+table.EntityName+"Form, models."+repositoryType+"]().ToEntity(req)")
	methodContent = strings.ReplaceAll(methodContent, "c.formMapper.ToDTO("+entityVar+")", "mapper.NewCopierMapper[adminv1."+table.EntityName+"Form, models."+repositoryType+"]().ToDTO("+entityVar+")")
	methodContent = strings.ReplaceAll(methodContent, "c.mapper.ToDTO(item)", "mapper.NewCopierMapper[adminv1."+table.EntityName+", models."+repositoryType+"]().ToDTO(item)")
	return reorderGoReceiverMethods(appendGeneratedGoMethods(content, methodContent), existingReceiver)
}

// appendMainServiceMethods 向已有服务文件追加未实现的 RPC 方法。
func (c *renderer) appendMainServiceMethods(content string, table *Table, columns []*CodeGenColumn, methods []*Proto) string {
	// 服务层同样从完整候选文件中提取缺失方法，不重写已有实现。
	candidate := c.renderBackendServiceFile(table, columns, methods)
	generatedReceiver := table.EntityName + "Service"
	existingReceiver := goReceiverType(content, "Service")
	if existingReceiver == "" {
		return content
	}
	methodNames := missingGoReceiverMethodNames(candidate, content, generatedReceiver, existingReceiver)
	methodContent := strings.Join(extractGoMethods(candidate, methodNames), "\n\n")
	if methodContent == "" {
		return reorderGoReceiverMethods(content, existingReceiver)
	}
	// 根据已有 Service 的依赖字段改写候选接收者，兼容人工命名的 Case 字段。
	existingCaseField, _ := goStructDependency(content, existingReceiver, "biz", "Case")
	generatedCaseField := stringcase.ToCamelCase(table.EntityName) + "Case"
	if existingCaseField == "" {
		return reorderGoReceiverMethods(content, existingReceiver)
	}
	methodContent = strings.ReplaceAll(methodContent, "*"+generatedReceiver, "*"+existingReceiver)
	methodContent = strings.ReplaceAll(methodContent, "s."+generatedCaseField, "s."+existingCaseField)
	return reorderGoReceiverMethods(appendGeneratedGoMethods(content, methodContent), existingReceiver)
}

// appendMainFrontendAPIMethods 向已有前端服务类追加缺失请求方法。
func (c *renderer) appendMainFrontendAPIMethods(content string, table *Table, columns []*CodeGenColumn, methods []*Proto) string {
	candidate := c.renderFrontendAPIFile(table, columns, methods)
	className := table.EntityName + "ServiceImpl"
	if findTSClassEndIndex(content, className) < 0 {
		return content
	}
	methodNames := goReceiverMethodNames(c.renderBackendServiceFile(table, columns, methods), table.EntityName+"Service")
	methodContents := make([]string, 0, len(methodNames))
	typeNames := make([]string, 0, len(methodNames)*2)
	// 方法和请求响应类型同步收集，保证追加实现后 import 仍然完整。
	for methodName := range methodNames {
		if tsClassMethodExists(content, className, methodName) {
			continue
		}
		methodContent := extractTSClassMethod(candidate, methodName)
		if methodContent == "" {
			continue
		}
		methodContents = append(methodContents, methodContent)
		typeNames = append(typeNames, methodName+"Request")
		if strings.Contains(methodContent, table.EntityName+"Form") {
			typeNames = append(typeNames, table.EntityName+"Form")
		}
		if strings.Contains(methodContent, methodName+"Response") {
			typeNames = append(typeNames, methodName+"Response")
		}
	}
	if len(methodContents) == 0 {
		return reorderTSClassMethods(content, className)
	}

	// 先补类型导入再定位类结尾，因为新增 import 会改变类在源码中的偏移量。
	rpcPath := frontendRPCImportPath(c.defaultProtoPath(table))
	content = ensureTSNamedTypeNames(content, rpcPath, typeNames)
	joinedMethods := strings.Join(methodContents, "\n\n")
	if strings.Contains(joinedMethods, "Empty") {
		content = ensureTSNamedTypeNames(content, "@/rpc/google/protobuf/empty", []string{"Empty"})
	}
	commonTypes := make([]string, 0, 2)
	if strings.Contains(joinedMethods, "SelectOptionResponse") {
		commonTypes = append(commonTypes, "SelectOptionResponse")
	}
	if strings.Contains(joinedMethods, "TreeOptionResponse") {
		commonTypes = append(commonTypes, "TreeOptionResponse")
	}
	content = ensureTSNamedTypeNames(content, "@/rpc/common/v1/common", commonTypes)
	classEnd := findTSClassEndIndex(content, className)
	if classEnd < 0 {
		return content
	}
	content = content[:classEnd] + "\n" + joinedMethods + "\n" + content[classEnd:]
	return reorderTSClassMethods(content, className)
}
