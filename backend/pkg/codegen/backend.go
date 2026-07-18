package codegen

import (
	"fmt"
	"strings"

	"github.com/liujitcn/go-utils/stringcase"
)

// --- Go 业务层与服务层模板渲染 ---

// renderBackendBizFile 渲染后端业务文件内容。
func (c *renderer) renderBackendBizFile(table *Table, columns []*CodeGenColumn, methods []*Proto) string {
	entity := table.EntityName
	columns = codeGenRequestColumns(table, columns)
	methods = filterProtoMethods(methods, c.defaultProtoPath(table))
	entityVar := stringcase.ToCamelCase(entity)
	repoField := entity + "Repository"
	queryName := entity
	modelType := "models." + entity
	dtoType := "adminv1." + entity
	formType := "adminv1." + entity + "Form"
	pageMethod := "Page" + entity
	pageResponse := "adminv1.Page" + entity + "Response"
	listField := stringcase.ToPascalCase(pluralize(entity))
	treeMethod := firstMethodByKind(methods, APIKindTree, TriggerPageTree)
	optionMethods := methodsByKinds(methods, APIKindOption, APIKindTree)
	defaultOrderOption := renderDefaultOrderOption(columns)
	orderOptionCount := 0
	if defaultOrderOption != "" {
		orderOptionCount = 1
	}
	var methodsBuilder strings.Builder
	if treeMethod != nil {
		methodsBuilder.WriteString(fmt.Sprintf(`// %s 查询%s树形列表。
func (c *%sCase) %s(ctx context.Context, req *adminv1.%sRequest) (*adminv1.%sResponse, error) {
	query := c.Query(ctx).%s
	opts := make([]repository.QueryOption, 0, %d)
%s%s
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &adminv1.%sResponse{%s: c.build%sTree(list, 0)}, nil
}

`, treeMethod.MethodName, table.BusinessName, entity, treeMethod.MethodName, treeMethod.MethodName, treeMethod.MethodName, queryName, countQueryColumns(columns)+orderOptionCount, defaultOrderOption, c.renderQueryOptions(columns), treeMethod.MethodName, listField, entity))
	}
	for _, method := range optionMethods {
		if method.APIKind == APIKindOption {
			methodsBuilder.WriteString(c.renderOptionBizMethod(table, columns, method))
			continue
		}
		if method.TriggerType == TriggerEntityOption || method.TriggerType == TriggerFieldOption || method.TriggerType == TriggerLeftTree {
			methodsBuilder.WriteString(c.renderTreeOptionBizMethod(table, columns, method))
		}
	}
	mainMethods := fmt.Sprintf(`// %s 查询%s分页列表。
func (c *%sCase) %s(ctx context.Context, req *adminv1.%sRequest) (*%s, error) {
	query := c.Query(ctx).%s
	opts := make([]repository.QueryOption, 0, %d)
%s%s
	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.%s, 0, len(list))
	for _, item := range list {
		res := c.mapper.ToDTO(item)
		resList = append(resList, res)
	}
	return &%s{%s: resList, Total: int32(total)}, nil
}

// Get%s 查询%s详情。
func (c *%sCase) Get%s(ctx context.Context, id int64) (*adminv1.%sForm, error) {
	%s, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.formMapper.ToDTO(%s), nil
}

// Create%s 创建%s。
func (c *%sCase) Create%s(ctx context.Context, req *adminv1.%sForm) error {
	%s := c.formMapper.ToEntity(req)
	return c.Create(ctx, %s)
}

// Update%s 更新%s。
func (c *%sCase) Update%s(ctx context.Context, id int64, req *adminv1.%sForm) error {
	%s := c.formMapper.ToEntity(req)
	%s.ID = id
	return c.UpdateByID(ctx, %s)
}

// Delete%s 删除%s。
func (c *%sCase) Delete%s(ctx context.Context, ids string) error {
	return c.DeleteByIDs(ctx, _string.ConvertStringToInt64Array(ids))
}
	`, pageMethod, table.BusinessName, entity, pageMethod, pageMethod, pageResponse, queryName, countQueryColumns(columns)+orderOptionCount, defaultOrderOption, c.renderQueryOptions(columns), entity, pageResponse, listField, entity, table.BusinessName, entity, entity, entity, entityVar, entityVar, entity, table.BusinessName, entity, entity, entity, entityVar, entityVar, entity, table.BusinessName, entity, entity, entity, entityVar, entityVar, entityVar, entity, table.BusinessName, entity, entity)
	if treeMethod != nil {
		getMethodIndex := strings.Index(mainMethods, "// Get"+entity)
		if getMethodIndex >= 0 {
			mainMethods = mainMethods[getMethodIndex:]
		}
	}
	methodsBuilder.WriteString(mainMethods)
	for _, statusMethod := range methodsByKinds(methods, APIKindStatus) {
		statusColumn := findStatusColumn(columns, statusMethod.ColumnName)
		if statusColumn == nil {
			continue
		}
		methodsBuilder.WriteString(fmt.Sprintf(`
// %s 设置%s状态。
func (c *%sCase) %s(ctx context.Context, req *adminv1.%sRequest) error {
	return c.UpdateByID(ctx, &models.%s{
		ID: req.GetId(),
		%s: req.GetStatus(),
	})
}
`, statusMethod.MethodName, DefaultString(statusColumn.ColumnComment, statusColumn.ColumnName), entity, statusMethod.MethodName, statusMethod.MethodName, entity, modelFieldName(statusColumn.ColumnName)))
	}
	if treeMethod != nil {
		parentField := DefaultString(table.ParentColumn, "parent_id")
		methodsBuilder.WriteString(fmt.Sprintf(`
// build%sTree 构建%s树。
func (c *%sCase) build%sTree(list []*models.%s, parentID int64) []*adminv1.%s {
	res := make([]*adminv1.%s, 0)
	for _, item := range list {
		if item.%s != parentID {
			continue
		}
		%s := c.mapper.ToDTO(item)
		%s.Children = c.build%sTree(list, item.ID)
		res = append(res, %s)
	}
	return res
}
`, entity, table.BusinessName, entity, entity, entity, entity, entity, modelFieldName(parentField), entityVar, entityVar, entity, entityVar))
	}
	content := renderTemplate("backend_biz.tmpl", backendBizTemplateData{
		Entity:       entity,
		EntityVar:    entityVar,
		BusinessName: table.BusinessName,
		Repository:   repoField,
		FormType:     formType,
		ModelType:    modelType,
		DTOType:      dtoType,
		CommonImport: renderGoCommonImport(optionMethods),
		Methods:      methodsBuilder.String(),
	})
	content = removeGoReceiverMethods(content, missingCoreMethodNames(table, methods))
	// 没有时间区间查询时移除模板中不再使用的时间工具依赖。
	if !strings.Contains(content, "_time.") {
		content = strings.Replace(content, "\t_time \"github.com/liujitcn/go-utils/time\"\n", "", 1)
	}
	return reorderGoReceiverMethods(content, entity+"Case")
}

// renderBackendServiceFile 渲染后端服务文件内容。
func (c *renderer) renderBackendServiceFile(table *Table, columns []*CodeGenColumn, methods []*Proto) string {
	entity := table.EntityName
	methods = filterProtoMethods(methods, c.defaultProtoPath(table))
	entityVar := stringcase.ToCamelCase(entity)
	var methodsBuilder strings.Builder
	if treeMethod := firstMethodByKind(methods, APIKindTree, TriggerPageTree); treeMethod != nil {
		methodsBuilder.WriteString(c.renderServiceMethod(table, treeMethod, entityVar, "tree", "查询"+table.BusinessName+"树形列表失败"))
	} else {
		methodsBuilder.WriteString(c.renderServiceMethod(table, &Proto{APIKind: APIKindList, MethodName: "Page" + entity}, entityVar, "page", "查询"+table.BusinessName+"分页列表失败"))
	}
	for _, method := range methodsByKinds(methods, APIKindOption, APIKindTree) {
		if method.TriggerType == TriggerEntityOption || method.TriggerType == TriggerFieldOption || method.TriggerType == TriggerLeftTree || method.APIKind == APIKindOption {
			methodsBuilder.WriteString(c.renderServiceMethod(table, method, entityVar, "option", "查询选项失败"))
		}
	}
	methodsBuilder.WriteString(c.renderServiceMethod(table, &Proto{APIKind: APIKindCRUD, MethodName: "Get" + entity}, entityVar, "get", "查询"+table.BusinessName+"失败"))
	methodsBuilder.WriteString(c.renderServiceMethod(table, &Proto{APIKind: APIKindCRUD, MethodName: "Create" + entity}, entityVar, "empty", "创建"+table.BusinessName+"失败"))
	methodsBuilder.WriteString(c.renderServiceMethod(table, &Proto{APIKind: APIKindCRUD, MethodName: "Update" + entity}, entityVar, "empty", "更新"+table.BusinessName+"失败"))
	methodsBuilder.WriteString(c.renderServiceMethod(table, &Proto{APIKind: APIKindCRUD, MethodName: "Delete" + entity}, entityVar, "empty", "删除"+table.BusinessName+"失败"))
	for _, statusMethod := range methodsByKinds(methods, APIKindStatus) {
		methodsBuilder.WriteString(c.renderServiceMethod(table, statusMethod, entityVar, "empty", "设置状态失败"))
	}
	content := renderTemplate("backend_service.tmpl", backendServiceTemplateData{
		Entity:       entity,
		EntityVar:    entityVar,
		BusinessName: table.BusinessName,
		CommonImport: renderGoCommonImport(methodsByKinds(methods, APIKindOption, APIKindTree)),
		Methods:      methodsBuilder.String(),
	})
	return reorderGoReceiverMethods(removeGoReceiverMethods(content, missingCoreMethodNames(table, methods)), entity+"Service")
}

// appendExternalTargetBizMethods 向已有业务文件追加外部目标选项方法。
func (c *renderer) appendExternalTargetBizMethods(content string, table *Table, methods []*Proto) string {
	candidate := c.renderExternalTargetBizFile(table, methods)
	generatedReceiver := table.EntityName + "Case"
	existingReceiver := goReceiverType(content, "Case")
	if existingReceiver == "" {
		return content
	}
	methodNames := missingGoReceiverMethodNames(candidate, content, generatedReceiver, existingReceiver)
	methodContent := strings.Join(extractGoMethods(candidate, methodNames), "\n\n")
	if methodContent == "" {
		return content
	}
	_, repositoryType := goStructDependency(content, existingReceiver, "data", "Repository")
	repositoryType = strings.TrimSuffix(repositoryType, "Repository")
	if repositoryType == "" {
		return content
	}
	methodContent = strings.ReplaceAll(methodContent, "*"+generatedReceiver, "*"+existingReceiver)
	methodContent = strings.ReplaceAll(methodContent, "models."+table.EntityName, "models."+repositoryType)
	methodContent = strings.ReplaceAll(methodContent, "c.Query(ctx)."+table.EntityName, "c.Query(ctx)."+repositoryType)
	return reorderGoReceiverMethods(appendGeneratedGoMethods(content, methodContent), existingReceiver)
}

// renderExternalTargetBizFile 渲染外部目标实体的最小业务文件。
func (c *renderer) renderExternalTargetBizFile(table *Table, methods []*Proto) string {
	entity := table.EntityName
	entityVar := stringcase.ToCamelCase(entity)
	repoField := entity + "Repository"
	return renderTemplate("backend_external_biz.tmpl", backendBizTemplateData{
		Entity:       entity,
		EntityVar:    entityVar,
		BusinessName: table.BusinessName,
		Repository:   repoField,
		Methods:      c.renderExternalTargetBizMethods(table, methods),
	})
}

// renderExternalTargetBizMethods 渲染外部目标实体选项业务方法。
func (c *renderer) renderExternalTargetBizMethods(table *Table, methods []*Proto) string {
	var builder strings.Builder
	for _, method := range methods {
		if method.APIKind == APIKindOption {
			builder.WriteString(c.renderOptionBizMethod(table, nil, method))
			continue
		}
		if method.APIKind == APIKindTree {
			builder.WriteString(c.renderTreeOptionBizMethod(table, nil, method))
		}
	}
	return builder.String()
}

// appendExternalTargetServiceMethods 向已有服务文件追加外部目标选项方法。
func (c *renderer) appendExternalTargetServiceMethods(content string, table *Table, methods []*Proto) string {
	candidate := c.renderExternalTargetServiceFile(table, methods)
	generatedReceiver := table.EntityName + "Service"
	existingReceiver := goReceiverType(content, "Service")
	if existingReceiver == "" {
		return content
	}
	methodNames := missingGoReceiverMethodNames(candidate, content, generatedReceiver, existingReceiver)
	methodContent := strings.Join(extractGoMethods(candidate, methodNames), "\n\n")
	if methodContent == "" {
		return content
	}
	existingCaseField, _ := goStructDependency(content, existingReceiver, "biz", "Case")
	generatedCaseField := stringcase.ToCamelCase(table.EntityName) + "Case"
	if existingCaseField == "" {
		return content
	}
	methodContent = strings.ReplaceAll(methodContent, "*"+generatedReceiver, "*"+existingReceiver)
	methodContent = strings.ReplaceAll(methodContent, "s."+generatedCaseField, "s."+existingCaseField)
	return reorderGoReceiverMethods(appendGeneratedGoMethods(content, methodContent), existingReceiver)
}

// renderExternalTargetServiceFile 渲染外部目标实体的最小服务文件。
func (c *renderer) renderExternalTargetServiceFile(table *Table, methods []*Proto) string {
	entity := table.EntityName
	entityVar := stringcase.ToCamelCase(entity)
	var methodsBuilder strings.Builder
	for _, method := range methods {
		methodsBuilder.WriteString(c.renderServiceMethod(table, method, entityVar, "option", "查询选项失败"))
	}
	return renderTemplate("backend_external_service.tmpl", backendServiceTemplateData{
		Entity:       entity,
		EntityVar:    entityVar,
		BusinessName: table.BusinessName,
		Methods:      methodsBuilder.String(),
	})
}

// renderGoCommonImport 按需渲染 commonv1 导入。
func renderGoCommonImport(methods []*Proto) string {
	for _, method := range methods {
		if method.APIKind == APIKindOption {
			return "\tcommonv1 \"shop/api/gen/go/common/v1\""
		}
		if method.APIKind == APIKindTree && (method.TriggerType == TriggerEntityOption || method.TriggerType == TriggerFieldOption || method.TriggerType == TriggerLeftTree) {
			return "\tcommonv1 \"shop/api/gen/go/common/v1\""
		}
	}
	return ""
}
