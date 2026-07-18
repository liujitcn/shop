package codegen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"slices"
	"strings"

	adminv1 "shop/api/gen/go/admin/v1"

	"github.com/liujitcn/go-utils/stringcase"
)

// --- 已有源码的增量分析与补丁 ---

// newExternalTargetBackendPreviewFiles 创建外部选项目标的后端补齐文件。
func (c *renderer) newExternalTargetBackendPreviewFiles(table *Table, methods []*Proto) []*adminv1.CodeGenPreviewFile {
	targets := c.externalOptionTargets(table, methods)
	files := make([]*adminv1.CodeGenPreviewFile, 0, len(targets)*2)
	for _, target := range targets {
		// 外部实体只补选项查询所需的 Biz 和 Service，不生成该实体的完整 CRUD。
		bizPath := "backend/service/admin/biz/" + stringcase.ToSnakeCase(target.Table.EntityName) + ".go"
		servicePath := "backend/service/admin/" + stringcase.ToSnakeCase(target.Table.EntityName) + "_service.go"
		files = append(files,
			c.newPatchedPreviewFile(bizPath, c.renderExternalTargetBizFile(target.Table, target.Methods), func(content string) string {
				return c.appendExternalTargetBizMethods(content, target.Table, target.Methods)
			}),
			c.newPatchedPreviewFile(servicePath, c.renderExternalTargetServiceFile(target.Table, target.Methods), func(content string) string {
				return c.appendExternalTargetServiceMethods(content, target.Table, target.Methods)
			}),
		)
	}
	return files
}

// newExternalTargetFrontendPreviewFiles 创建外部选项目标的前端 API 补齐文件。
func (c *renderer) newExternalTargetFrontendPreviewFiles(table *Table, methods []*Proto) []*adminv1.CodeGenPreviewFile {
	targets := c.externalOptionTargets(table, methods)
	files := make([]*adminv1.CodeGenPreviewFile, 0, len(targets))
	for _, target := range targets {
		// 前端只需要补齐选项数据源对应的请求方法。
		path := "frontend/admin/src/api/admin/" + stringcase.ToSnakeCase(target.Table.EntityName) + ".ts"
		files = append(files, c.newPatchedPreviewFile(path, c.renderExternalTargetFrontendAPIFile(target.Table, target.Methods), func(content string) string {
			return c.appendExternalTargetFrontendAPIMethods(content, target.Table, target.Methods)
		}))
	}
	return files
}

// newPatchedPreviewFile 创建支持已有文件追加的预览文件。
func (c *renderer) newPatchedPreviewFile(path string, createContent string, patch func(string) string) *adminv1.CodeGenPreviewFile {
	// 所有预览文件先经过仓库边界校验，非法路径只返回 skip 结果，不触碰磁盘。
	_, pathErr := SafeRepoFilePath(path)
	if pathErr != nil {
		return &adminv1.CodeGenPreviewFile{Path: path, Action: "skip", Content: createContent, Exists: false, Message: pathErr.Error()}
	}
	content, err := c.readRepoFile(path)
	if err != nil {
		// 目标不存在时按完整模板创建；已有文件才进入增量补丁逻辑。
		return c.newPreviewFile(path, createContent)
	}
	patched := patch(string(content))
	if patched == string(content) {
		// 内容相同必须标记为 skip，避免生成任务无意义地更新文件时间。
		return &adminv1.CodeGenPreviewFile{Path: path, Action: "skip", Content: string(content), Exists: true, Message: "目标文件已存在，未发现需要追加的方法"}
	}
	return &adminv1.CodeGenPreviewFile{Path: path, Action: "update", Content: patched, Exists: true, Message: "目标文件已存在，将追加缺失接口实现"}
}

// newAdminRegistrationPreviewFiles 创建管理端服务依赖注入与传输层注册补丁。
func (c *renderer) newAdminRegistrationPreviewFiles(table *Table, methods []*Proto) []*adminv1.CodeGenPreviewFile {
	entities := []string{table.EntityName}
	for _, target := range c.externalOptionTargets(table, methods) {
		entities = append(entities, target.Table.EntityName)
	}
	patches := []struct {
		path  string
		patch func(string, string) string
	}{
		{path: "backend/service/admin/init.go", patch: appendAdminProviderRegistration},
		{path: "backend/server/services.go", patch: appendServerServicesRegistration},
		{path: "backend/server/http.go", patch: appendHTTPServiceRegistration},
		{path: "backend/server/grpc.go", patch: appendGRPCServiceRegistration},
		{path: "backend/server/mcp.go", patch: appendMCPServiceRegistration},
	}
	files := make([]*adminv1.CodeGenPreviewFile, 0, len(patches))
	for _, item := range patches {
		patch := item.patch
		files = append(files, c.newExistingPatchedPreviewFile(item.path, func(content string) string {
			for _, entity := range entities {
				content = patch(content, entity)
			}
			return content
		}))
	}
	return files
}

// externalOptionTargets 按外部目标实体分组选项接口。
func (c *renderer) externalOptionTargets(table *Table, methods []*Proto) []CodeGenExternalTarget {
	grouped := make(map[string][]*Proto)
	for _, method := range methods {
		if method.TargetEntityName == "" || method.TargetEntityName == table.EntityName {
			continue
		}
		if !IsOptionProtoMethod(method) {
			continue
		}
		grouped[method.TargetEntityName] = append(grouped[method.TargetEntityName], method)
	}
	targets := make([]CodeGenExternalTarget, 0, len(grouped))
	for target, targetMethods := range grouped {
		targets = append(targets, CodeGenExternalTarget{
			Table:   c.externalTargetTable(table, target, targetMethods),
			Methods: targetMethods,
		})
	}
	slices.SortFunc(targets, func(a CodeGenExternalTarget, b CodeGenExternalTarget) int {
		return strings.Compare(a.Table.EntityName, b.Table.EntityName)
	})
	return targets
}

// externalTargetTable 构造外部目标实体的最小生成上下文。
func (c *renderer) externalTargetTable(table *Table, target string, methods []*Proto) *Table {
	businessName := target
	for _, method := range methods {
		if method.TargetBusinessName != "" {
			businessName = method.TargetBusinessName
			break
		}
	}
	return &Table{
		TableName_:   stringcase.ToSnakeCase(target),
		TableComment: businessName,
		BusinessName: businessName,
		EntityName:   target,
		ModulePath:   resourcePathByEntity(target),
		APIPath:      c.defaultTargetProtoPath(table, target),
		PageType:     "normal",
		GenBackend:   1,
		GenFrontend:  1,
		GenSql:       0,
		Status:       StatusDraft,
		CreatedAt:    table.CreatedAt,
		UpdatedAt:    table.UpdatedAt,
	}
}

// newExistingPatchedPreviewFile 创建只允许追加已有文件的预览补丁。
func (c *renderer) newExistingPatchedPreviewFile(path string, patch func(string) string) *adminv1.CodeGenPreviewFile {
	_, err := SafeRepoFilePath(path)
	if err != nil {
		return &adminv1.CodeGenPreviewFile{Path: path, Action: "skip", Message: err.Error()}
	}
	var content []byte
	content, err = c.readRepoFile(path)
	if err != nil {
		return &adminv1.CodeGenPreviewFile{Path: path, Action: "skip", Message: "目标文件不存在，无法追加服务注册"}
	}
	patched := patch(string(content))
	if patched == string(content) {
		return &adminv1.CodeGenPreviewFile{Path: path, Action: "skip", Content: string(content), Exists: true, Message: "目标文件已存在，未发现需要追加的服务注册"}
	}
	return &adminv1.CodeGenPreviewFile{Path: path, Action: "update", Content: patched, Exists: true, Message: "目标文件已存在，将追加缺失服务注册"}
}

// generatedProtoMethods 合并基础能力与用户勾选的缺失接口。
func (c *renderer) generatedProtoMethods(table *Table, columns []*CodeGenColumn, methods []*Proto) []*Proto {
	checks := c.buildExpectedProtoChecks(table, columns)
	list := make([]*Proto, 0, len(checks))
	for _, check := range checks {
		saved := findSavedProtoMethod(methods, check)
		if saved != nil {
			applySavedProtoMethod(check, saved)
		}
		exists, _ := c.protoMethodExists(check.ProtoFilePath, check.TargetEntityName, check.MethodName)
		if exists || table.GenBackend == 1 && check.GenerateWhenMissing {
			list = append(list, c.protoCheckToModel(check))
		}
	}
	return sortCodeGenProtoMethods(list)
}

// frontendProtoMethods 合并前端页面可使用的已存在或将生成接口。
func (c *renderer) frontendProtoMethods(table *Table, columns []*CodeGenColumn, methods []*Proto) []*Proto {
	checks := c.buildExpectedProtoChecks(table, columns)
	list := make([]*Proto, 0, len(checks))
	for _, check := range checks {
		saved := findSavedProtoMethod(methods, check)
		if saved != nil {
			applySavedProtoMethod(check, saved)
		}
		exists, _ := c.protoMethodExists(check.ProtoFilePath, check.TargetEntityName, check.MethodName)
		if exists || table.GenBackend == 1 && check.GenerateWhenMissing {
			list = append(list, c.protoCheckToModel(check))
		}
	}
	return sortCodeGenProtoMethods(list)
}

// protoCheckToModel 将检查项转换为生成方法配置。
func (c *renderer) protoCheckToModel(check *ProtoCheck) *Proto {
	return &Proto{
		TableID:             check.TableID,
		ColumnName:          check.ColumnName,
		TriggerType:         check.TriggerType,
		APIKind:             check.APIKind,
		TargetEntityName:    check.TargetEntityName,
		TargetBusinessName:  check.TargetBusinessName,
		MethodName:          check.MethodName,
		ProtoFilePath:       check.ProtoFilePath,
		ParentColumn:        check.ParentColumn,
		LabelColumn:         check.LabelColumn,
		ValueColumn:         check.ValueColumn,
		GenerateWhenMissing: BoolToInt32(check.GenerateWhenMissing),
	}
}

// newTargetProtoPreviewFile 创建指定 Proto 文件预览内容。
func (c *renderer) newTargetProtoPreviewFile(table *Table, columns []*CodeGenColumn, methods []*Proto, path string) *adminv1.CodeGenPreviewFile {
	_, err := SafeRepoFilePath(path)
	if err != nil {
		return &adminv1.CodeGenPreviewFile{
			Path:    path,
			Action:  "skip",
			Content: "",
			Exists:  false,
			Message: err.Error(),
		}
	}
	var content []byte
	content, err = c.readRepoFile(path)
	if err != nil {
		if path != c.defaultProtoPath(table) {
			return c.newPreviewFile(path, c.renderTargetProtoFile(table, columns, methods, path))
		}
		return c.newPreviewFile(path, c.renderProtoFile(table, columns, methods))
	}

	originalContent := string(content)
	patch := c.renderProtoPatch(table, columns, methods, path)
	normalizedContent := normalizeProtoMessageOrder(normalizeProtoRPCOrder(dedupeProtoMessageBlocks(originalContent), methods, path))
	if patch.Empty() {
		if normalizedContent != originalContent {
			return &adminv1.CodeGenPreviewFile{
				Path:    path,
				Action:  "update",
				Content: normalizedContent,
				Exists:  true,
				Message: "Proto文件结构将按固定方法槽位整理",
			}
		}
		return &adminv1.CodeGenPreviewFile{
			Path:    path,
			Action:  "skip",
			Content: originalContent,
			Exists:  true,
			Message: "Proto文件已存在，未选择缺失接口",
		}
	}
	patched := normalizeProtoMessageOrder(normalizeProtoRPCOrder(dedupeProtoMessageBlocks(c.appendProtoPatch(normalizedContent, patch)), methods, path))
	if patched == originalContent {
		return &adminv1.CodeGenPreviewFile{
			Path:    path,
			Action:  "skip",
			Content: originalContent,
			Exists:  true,
			Message: "Proto文件已存在，未找到目标service，已跳过追加",
		}
	}
	return &adminv1.CodeGenPreviewFile{
		Path:    path,
		Action:  "update",
		Content: patched,
		Exists:  true,
		Message: "Proto文件已存在，将追加缺失接口",
	}
}

// newPreviewFile 创建预览文件并标记处理动作。
func (c *renderer) newPreviewFile(path string, content string) *adminv1.CodeGenPreviewFile {
	_, pathErr := SafeRepoFilePath(path)
	if pathErr != nil {
		return &adminv1.CodeGenPreviewFile{
			Path:    path,
			Action:  "skip",
			Content: content,
			Exists:  false,
			Message: pathErr.Error(),
		}
	}
	exists, err := c.repoFileExists(path)
	if err != nil {
		exists = false
	}
	action := "create"
	message := "将新增"
	if exists {
		action = "skip"
		message = "已存在，不覆盖"
	}
	return &adminv1.CodeGenPreviewFile{
		Path:    path,
		Action:  action,
		Content: content,
		Exists:  exists,
		Message: message,
	}
}

// goReceiverType 返回 Go 文件中指定后缀的首个方法接收者或结构体类型。
func goReceiverType(content string, suffix string) string {
	file, _, err := parseGoSource(content)
	if err != nil {
		return ""
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv == nil || len(function.Recv.List) == 0 {
			continue
		}
		receiver := goReceiverName(function.Recv.List[0].Type)
		if strings.HasSuffix(receiver, suffix) {
			return receiver
		}
	}
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, specification := range general.Specs {
			typeSpec, ok := specification.(*ast.TypeSpec)
			if !ok || !strings.HasSuffix(typeSpec.Name.Name, suffix) {
				continue
			}
			if _, ok = typeSpec.Type.(*ast.StructType); ok {
				return typeSpec.Name.Name
			}
		}
	}
	return ""
}

// missingGoReceiverMethodNames 返回目标 Go 文件中尚未实现的候选方法名。
func missingGoReceiverMethodNames(candidateContent string, existingContent string, candidateReceiver string, existingReceiver string) map[string]struct{} {
	methods := goReceiverMethodNames(candidateContent, candidateReceiver)
	for methodName := range goReceiverMethodNames(existingContent, existingReceiver) {
		delete(methods, methodName)
	}
	return methods
}

// missingGoReceiverMethodNamesFold 返回目标 Go 文件中未实现的方法名，并兼容 API 等缩写大小写差异。
func missingGoReceiverMethodNamesFold(candidateContent string, existingContent string, candidateReceiver string, existingReceiver string) map[string]struct{} {
	methods := goReceiverMethodNames(candidateContent, candidateReceiver)
	existingMethods := goReceiverMethodNames(existingContent, existingReceiver)
	for candidateName := range methods {
		for existingName := range existingMethods {
			if strings.EqualFold(candidateName, existingName) {
				delete(methods, candidateName)
				break
			}
		}
	}
	return methods
}

// goReceiverMethodNames 返回指定接收者类型的全部方法名。
func goReceiverMethodNames(content string, receiverName string) map[string]struct{} {
	methods := make(map[string]struct{})
	file, _, err := parseGoSource(content)
	if err != nil {
		return methods
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv == nil || len(function.Recv.List) == 0 {
			continue
		}
		if goReceiverName(function.Recv.List[0].Type) == receiverName {
			methods[function.Name.Name] = struct{}{}
		}
	}
	return methods
}

// goStructDependency 查找结构体中指定包和类型后缀的依赖字段。
func goStructDependency(content string, structName string, packageName string, typeSuffix string) (string, string) {
	file, _, err := parseGoSource(content)
	if err != nil {
		return "", ""
	}
	for _, declaration := range file.Decls {
		general, isGeneral := declaration.(*ast.GenDecl)
		if !isGeneral {
			continue
		}
		for _, specification := range general.Specs {
			typeSpec, isTypeSpec := specification.(*ast.TypeSpec)
			if !isTypeSpec || typeSpec.Name.Name != structName {
				continue
			}
			structType, isStructType := typeSpec.Type.(*ast.StructType)
			if !isStructType {
				return "", ""
			}
			for _, field := range structType.Fields.List {
				selector := goSelectorType(field.Type)
				if selector == nil || selector.Sel == nil || !strings.HasSuffix(selector.Sel.Name, typeSuffix) {
					continue
				}
				identifier, isIdentifier := selector.X.(*ast.Ident)
				if !isIdentifier || identifier.Name != packageName {
					continue
				}
				fieldName := selector.Sel.Name
				if len(field.Names) > 0 {
					fieldName = field.Names[0].Name
				}
				return fieldName, selector.Sel.Name
			}
		}
	}
	return "", ""
}

// extractGoMethods 从生成候选文件中提取指定方法源码。
func extractGoMethods(content string, methodNames map[string]struct{}) []string {
	methods := make([]string, 0, len(methodNames))
	file, fileSet, err := parseGoSource(content)
	if err != nil {
		return methods
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv == nil {
			continue
		}
		if _, ok = methodNames[function.Name.Name]; !ok {
			continue
		}
		start := fileSet.Position(function.Pos()).Offset
		if function.Doc != nil {
			start = fileSet.Position(function.Doc.Pos()).Offset
		}
		end := fileSet.Position(function.End()).Offset
		if start >= 0 && end <= len(content) && start < end {
			methods = append(methods, strings.TrimSpace(content[start:end]))
		}
	}
	return methods
}

// removeGoReceiverMethods 从 Go 源码中删除指定接收者方法。
func removeGoReceiverMethods(content string, methodNames map[string]struct{}) string {
	file, fileSet, err := parseGoSource(content)
	if err != nil || len(methodNames) == 0 {
		return content
	}
	type sourceRange struct {
		start int
		end   int
	}
	var ranges []sourceRange
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv == nil {
			continue
		}
		if _, ok = methodNames[function.Name.Name]; !ok {
			continue
		}
		start := fileSet.Position(function.Pos()).Offset
		if function.Doc != nil {
			start = fileSet.Position(function.Doc.Pos()).Offset
		}
		ranges = append(ranges, sourceRange{start: start, end: fileSet.Position(function.End()).Offset})
	}
	for i := len(ranges) - 1; i >= 0; i-- {
		item := ranges[i]
		if item.start >= 0 && item.end <= len(content) && item.start < item.end {
			content = content[:item.start] + content[item.end:]
		}
	}
	return content
}

// appendAdminProviderRegistration 向管理端 ProviderSet 追加业务与服务构造函数。
func appendAdminProviderRegistration(content string, entity string) string {
	additions := make([]string, 0, 2)
	if !goIdentifierExists(content, "New"+entity+"Case") {
		additions = append(additions, "biz.New"+entity+"Case,")
	}
	if !goIdentifierExists(content, "New"+entity+"Service") {
		additions = append(additions, "New"+entity+"Service,")
	}
	if len(additions) == 0 {
		return content
	}
	file, fileSet, err := parseGoSource(content)
	if err != nil {
		return content
	}
	offset := -1
	for _, declaration := range file.Decls {
		general, isGeneral := declaration.(*ast.GenDecl)
		if !isGeneral {
			continue
		}
		for _, specification := range general.Specs {
			valueSpec, isValueSpec := specification.(*ast.ValueSpec)
			if !isValueSpec || len(valueSpec.Names) != 1 || valueSpec.Names[0].Name != "ProviderSet" || len(valueSpec.Values) != 1 {
				continue
			}
			call, isCall := valueSpec.Values[0].(*ast.CallExpr)
			if isCall {
				offset = fileSet.Position(call.Rparen).Offset
			}
		}
	}
	if offset < 0 {
		return content
	}
	lines := make([]string, 0, len(additions))
	for _, addition := range additions {
		lines = append(lines, "\t"+addition)
	}
	return validGoPatch(content, insertGoLines(content, offset, strings.Join(lines, "\n")))
}

// appendServerServicesRegistration 向 ServerServices 注册表追加服务依赖。
func appendServerServicesRegistration(content string, entity string) string {
	fieldName := goStructSelectorFieldName(content, "ServerServices", "admin", entity+"Service")
	var err error
	if fieldName == "" {
		fieldName = stringcase.ToCamelCase("admin_" + stringcase.ToSnakeCase(entity))
		var file *ast.File
		var fileSet *token.FileSet
		file, fileSet, err = parseGoSource(content)
		if err != nil {
			return content
		}
		structType := findGoStructType(file, "ServerServices")
		if structType == nil {
			return content
		}
		content = validGoPatch(content, insertGoLines(content, fileSet.Position(structType.Fields.Closing).Offset, "\t"+fieldName+" *admin."+entity+"Service"))
	}

	parameterName := goFuncSelectorParamName(content, "NewServerServices", "admin", entity+"Service")
	if parameterName == "" {
		parameterName = fieldName
		var file *ast.File
		var fileSet *token.FileSet
		file, fileSet, err = parseGoSource(content)
		if err != nil {
			return content
		}
		function := findGoFuncDecl(file, "NewServerServices")
		if function == nil {
			return content
		}
		content = validGoPatch(content, insertGoLines(content, fileSet.Position(function.Type.Params.Closing).Offset, "\t"+parameterName+" *admin."+entity+"Service,"))
	}

	var file *ast.File
	var fileSet *token.FileSet
	file, fileSet, err = parseGoSource(content)
	if err != nil {
		return content
	}
	function := findGoFuncDecl(file, "NewServerServices")
	if function == nil {
		return content
	}
	var servicesLiteral *ast.CompositeLit
	ast.Inspect(function.Body, func(node ast.Node) bool {
		literal, isLiteral := node.(*ast.CompositeLit)
		if !isLiteral {
			return true
		}
		identifier, isIdentifier := literal.Type.(*ast.Ident)
		if isIdentifier && identifier.Name == "ServerServices" {
			servicesLiteral = literal
			return false
		}
		return true
	})
	if servicesLiteral == nil || goCompositeKeyExists(servicesLiteral, fieldName) {
		return content
	}
	patched := insertGoLines(content, fileSet.Position(servicesLiteral.Rbrace).Offset, "\t\t"+fieldName+": "+parameterName+",")
	return validGoPatch(content, patched)
}

// appendHTTPServiceRegistration 向 HTTP Server 追加服务注册。
func appendHTTPServiceRegistration(content string, entity string) string {
	registerName := "Register" + entity + "ServiceHTTPServer"
	if goCallNameExists(content, registerName) {
		return content
	}
	fieldName := goStructSelectorFieldName(content, "ServerServices", "admin", entity+"Service")
	if fieldName == "" {
		fieldName = stringcase.ToCamelCase("admin_" + stringcase.ToSnakeCase(entity))
	}
	return insertBeforeGoFuncReturn(content, "NewHTTPServer", "\tadminv1."+registerName+"(srv, services."+fieldName+")")
}

// appendGRPCServiceRegistration 向 gRPC Server 追加服务依赖与注册。
func appendGRPCServiceRegistration(content string, entity string) string {
	parameterName := goFuncSelectorParamName(content, "NewGRPCServer", "admin", entity+"Service")
	if parameterName == "" {
		parameterName = stringcase.ToCamelCase("admin_" + stringcase.ToSnakeCase(entity))
		file, fileSet, err := parseGoSource(content)
		if err != nil {
			return content
		}
		function := findGoFuncDecl(file, "NewGRPCServer")
		if function == nil {
			return content
		}
		content = validGoPatch(content, insertGoLines(content, fileSet.Position(function.Type.Params.Closing).Offset, "\t"+parameterName+" *admin."+entity+"Service,"))
	}
	registerName := "Register" + entity + "ServiceServer"
	if goCallNameExists(content, registerName) {
		return content
	}
	return insertBeforeGoFuncReturn(content, "NewGRPCServer", "\tadminv1."+registerName+"(srv, "+parameterName+")")
}

// appendMCPServiceRegistration 向 MCP Server 追加服务工具注册。
func appendMCPServiceRegistration(content string, entity string) string {
	registerName := "Register" + entity + "ServiceMCPTools"
	if goCallNameExists(content, registerName) {
		return content
	}
	fieldName := goStructSelectorFieldName(content, "ServerServices", "admin", entity+"Service")
	if fieldName == "" {
		fieldName = stringcase.ToCamelCase("admin_" + stringcase.ToSnakeCase(entity))
	}
	file, fileSet, err := parseGoSource(content)
	if err != nil {
		return content
	}
	function := findGoFuncDecl(file, "registerMCPTools")
	if function == nil {
		return content
	}
	patched := insertGoLines(content, fileSet.Position(function.Body.Rbrace).Offset, "\tadminv1."+registerName+"(mcpServer, services."+fieldName+")")
	return validGoPatch(content, patched)
}

// goIdentifierExists 判断 Go 源码是否已包含指定标识符，兼容缩写大小写差异。
func goIdentifierExists(content string, name string) bool {
	file, _, err := parseGoSource(content)
	if err != nil {
		return false
	}
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if ok && strings.EqualFold(identifier.Name, name) {
			found = true
			return false
		}
		return !found
	})
	return found
}

// goCallNameExists 判断 Go 源码是否已调用指定函数。
func goCallNameExists(content string, name string) bool {
	file, _, err := parseGoSource(content)
	if err != nil {
		return false
	}
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		callName := ""
		switch function := call.Fun.(type) {
		case *ast.Ident:
			callName = function.Name
		case *ast.SelectorExpr:
			callName = function.Sel.Name
		}
		if strings.EqualFold(callName, name) {
			found = true
			return false
		}
		return !found
	})
	return found
}

// goStructSelectorFieldName 返回结构体中指定包类型对应的字段名。
func goStructSelectorFieldName(content string, structName string, packageName string, typeName string) string {
	file, _, err := parseGoSource(content)
	if err != nil {
		return ""
	}
	structType := findGoStructType(file, structName)
	if structType == nil {
		return ""
	}
	for _, field := range structType.Fields.List {
		selector := goSelectorType(field.Type)
		if selector == nil || !strings.EqualFold(selector.Sel.Name, typeName) {
			continue
		}
		identifier, ok := selector.X.(*ast.Ident)
		if !ok || identifier.Name != packageName || len(field.Names) == 0 {
			continue
		}
		return field.Names[0].Name
	}
	return ""
}

// findGoStructType 查找指定 Go 结构体。
func findGoStructType(file *ast.File, name string) *ast.StructType {
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, specification := range general.Specs {
			typeSpec, ok := specification.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != name {
				continue
			}
			structType, _ := typeSpec.Type.(*ast.StructType)
			return structType
		}
	}
	return nil
}

// goFuncSelectorParamName 返回函数参数中指定包类型对应的参数名。
func goFuncSelectorParamName(content string, functionName string, packageName string, typeName string) string {
	file, _, err := parseGoSource(content)
	if err != nil {
		return ""
	}
	function := findGoFuncDecl(file, functionName)
	if function == nil || function.Type.Params == nil {
		return ""
	}
	for _, field := range function.Type.Params.List {
		selector := goSelectorType(field.Type)
		if selector == nil || !strings.EqualFold(selector.Sel.Name, typeName) {
			continue
		}
		identifier, ok := selector.X.(*ast.Ident)
		if !ok || identifier.Name != packageName || len(field.Names) == 0 {
			continue
		}
		return field.Names[0].Name
	}
	return ""
}

// goCompositeKeyExists 判断结构体字面量是否已包含指定字段。
func goCompositeKeyExists(literal *ast.CompositeLit, fieldName string) bool {
	for _, element := range literal.Elts {
		keyValue, isKeyValue := element.(*ast.KeyValueExpr)
		if !isKeyValue {
			continue
		}
		identifier, isIdentifier := keyValue.Key.(*ast.Ident)
		if isIdentifier && strings.EqualFold(identifier.Name, fieldName) {
			return true
		}
	}
	return false
}

// insertBeforeGoFuncReturn 在函数最终返回前插入代码。
func insertBeforeGoFuncReturn(content string, functionName string, line string) string {
	file, fileSet, err := parseGoSource(content)
	if err != nil {
		return content
	}
	function := findGoFuncDecl(file, functionName)
	if function == nil {
		return content
	}
	for index := len(function.Body.List) - 1; index >= 0; index-- {
		statement, ok := function.Body.List[index].(*ast.ReturnStmt)
		if !ok {
			continue
		}
		patched := insertGoLines(content, fileSet.Position(statement.Pos()).Offset, line)
		return validGoPatch(content, patched)
	}
	return content
}

// findGoFuncDecl 查找指定 Go 函数。
func findGoFuncDecl(file *ast.File, name string) *ast.FuncDecl {
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == name {
			return function
		}
	}
	return nil
}

// insertGoLines 在源码偏移位置插入完整代码行。
func insertGoLines(content string, offset int, lines string) string {
	if offset < 0 || offset > len(content) || lines == "" {
		return content
	}
	prefix := ""
	if offset > 0 && content[offset-1] != '\n' {
		prefix = "\n"
	}
	return content[:offset] + prefix + lines + "\n" + content[offset:]
}

// validGoPatch 仅返回语法有效的 Go 源码补丁。
func validGoPatch(original string, patched string) string {
	if _, _, err := parseGoSource(patched); err != nil {
		return original
	}
	return patched
}

// goReceiverName 返回方法接收者类型名。
func goReceiverName(expression ast.Expr) string {
	if pointer, ok := expression.(*ast.StarExpr); ok {
		expression = pointer.X
	}
	identifier, _ := expression.(*ast.Ident)
	if identifier == nil {
		return ""
	}
	return identifier.Name
}

// goSelectorType 返回字段类型中的包选择器。
func goSelectorType(expression ast.Expr) *ast.SelectorExpr {
	if pointer, ok := expression.(*ast.StarExpr); ok {
		expression = pointer.X
	}
	selector, _ := expression.(*ast.SelectorExpr)
	return selector
}

// appendGeneratedGoMethods 补齐方法依赖的导入，且只在补丁保持 Go 语法有效时返回新内容。
func appendGeneratedGoMethods(content string, methodContent string) string {
	patched := ensureGeneratedGoImports(content, methodContent)
	patched = strings.TrimRight(patched, "\n") + "\n\n" + strings.TrimSpace(methodContent) + "\n"
	if _, _, err := parseGoSource(patched); err != nil {
		return content
	}
	return patched
}

// parseGoSource 解析 Go 源文件并返回位置映射。
func parseGoSource(content string) (*ast.File, *token.FileSet, error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "generated.go", content, parser.ParseComments)
	return file, fileSet, err
}

// ensureGeneratedGoImports 按追加的方法源码补齐必要导入。
func ensureGeneratedGoImports(content string, methodContent string) string {
	imports := []struct {
		marker     string
		importLine string
		importPath string
	}{
		{marker: "context.", importLine: `"context"`, importPath: "context"},
		{marker: "fmt.", importLine: `"fmt"`, importPath: "fmt"},
		{marker: "adminv1.", importLine: `adminv1 "shop/api/gen/go/admin/v1"`, importPath: "shop/api/gen/go/admin/v1"},
		{marker: "commonv1.", importLine: `commonv1 "shop/api/gen/go/common/v1"`, importPath: "shop/api/gen/go/common/v1"},
		{marker: "errorsx.", importLine: `"shop/pkg/errorsx"`, importPath: "shop/pkg/errorsx"},
		{marker: "models.", importLine: `"shop/pkg/gen/models"`, importPath: "shop/pkg/gen/models"},
		{marker: "mapper.", importLine: `"github.com/liujitcn/go-utils/mapper"`, importPath: "github.com/liujitcn/go-utils/mapper"},
		{marker: "_string.", importLine: `_string "github.com/liujitcn/go-utils/string"`, importPath: "github.com/liujitcn/go-utils/string"},
		{marker: "_time.", importLine: `_time "github.com/liujitcn/go-utils/time"`, importPath: "github.com/liujitcn/go-utils/time"},
		{marker: "repository.", importLine: `"github.com/liujitcn/gorm-kit/repository"`, importPath: "github.com/liujitcn/gorm-kit/repository"},
		{marker: "log.", importLine: `"github.com/go-kratos/kratos/v3/log"`, importPath: "github.com/go-kratos/kratos/v3/log"},
		{marker: "emptypb.", importLine: `"google.golang.org/protobuf/types/known/emptypb"`, importPath: "google.golang.org/protobuf/types/known/emptypb"},
	}
	for _, item := range imports {
		if strings.Contains(methodContent, item.marker) && !strings.Contains(content, `"`+item.importPath+`"`) {
			content = ensureGoImport(content, item.importLine)
		}
	}
	return content
}

// removeTSClassMethods 从 TypeScript 服务类中删除指定方法。
func removeTSClassMethods(content string, methodNames map[string]struct{}) string {
	for methodName := range methodNames {
		methodContent := extractTSClassMethod(content, methodName)
		if methodContent != "" {
			content = strings.Replace(content, methodContent, "", 1)
		}
	}
	return content
}

// extractTSClassMethod 从 TypeScript 服务类中提取指定方法。
func extractTSClassMethod(content string, methodName string) string {
	methodStart := strings.Index(content, "\n  "+methodName+"(")
	if methodStart < 0 {
		return ""
	}
	methodStart++
	blockStart := methodStart
	if docStart := strings.LastIndex(content[:methodStart], "\n  /**"); docStart >= 0 {
		docStart++
		docEnd := strings.Index(content[docStart:methodStart], "*/")
		if docEnd >= 0 && strings.TrimSpace(content[docStart+docEnd+2:methodStart]) == "" {
			blockStart = docStart
		}
	}
	openIndex := strings.Index(content[methodStart:], "{")
	if openIndex < 0 {
		return ""
	}
	openIndex += methodStart
	depth := 0
	for index := openIndex; index < len(content); index++ {
		switch content[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.TrimRight(content[blockStart:index+1], "\n")
			}
		}
	}
	return ""
}

// tsClassMethodExists 判断指定 TypeScript 类中是否已经实现目标方法。
func tsClassMethodExists(content string, className string, methodName string) bool {
	classIndex := strings.Index(content, "class "+className)
	classEnd := findTSClassEndIndex(content, className)
	if classIndex < 0 || classEnd < 0 {
		return false
	}
	pattern := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(methodName) + `\s*\(`)
	return pattern.MatchString(content[classIndex:classEnd])
}

// findTSClassEndIndex 查找 TypeScript 服务类的结束大括号。
func findTSClassEndIndex(content string, className string) int {
	classIndex := strings.Index(content, "class "+className)
	if classIndex < 0 {
		return -1
	}
	openIndex := strings.Index(content[classIndex:], "{")
	if openIndex < 0 {
		return -1
	}
	openIndex += classIndex
	depth := 0
	for index := openIndex; index < len(content); index++ {
		switch content[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}
