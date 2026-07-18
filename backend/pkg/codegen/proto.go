package codegen

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/liujitcn/go-utils/stringcase"
)

// --- Proto 契约渲染 ---

// renderProtoFile 渲染基础 Proto 文件。
func (c *renderer) renderProtoFile(table *Table, columns []*CodeGenColumn, methods []*Proto) string {
	entity := table.EntityName
	resourcePath := c.frontendResourcePath(table)
	methods = c.mainProtoMethods(table, methods)

	var rpcBuilder strings.Builder
	for _, method := range methods {
		rpcBuilder.WriteString(c.renderProtoRPC(table, method, resourcePath))
		rpcBuilder.WriteString("\n")
	}
	var messageBuilder strings.Builder
	messageNames := make(map[string]struct{})
	for _, method := range methods {
		for _, messageName := range c.protoMessageNamesForMethod(table, method) {
			if _, exists := messageNames[messageName]; exists {
				continue
			}
			messageNames[messageName] = struct{}{}
			messageBuilder.WriteString(c.renderProtoMessageByName(table, columns, method, messageName))
			messageBuilder.WriteString("\n")
		}
	}
	return renderTemplate("proto.tmpl", protoTemplateData{
		Entity:       entity,
		TableComment: table.TableComment,
		RPCs:         rpcBuilder.String(),
		Messages:     messageBuilder.String(),
	})
}

// mainProtoMethods 返回主实体已配置为缺失时生成的方法集合。
func (c *renderer) mainProtoMethods(table *Table, methods []*Proto) []*Proto {
	list := make([]*Proto, 0, len(methods))
	for _, method := range filterProtoMethods(methods, c.defaultProtoPath(table)) {
		if method.TargetEntityName != table.EntityName {
			continue
		}
		list = append(list, method)
	}
	return sortCodeGenProtoMethods(list)
}

// renderTargetProtoFile 渲染外部目标 Proto 的最小补齐文件。
func (c *renderer) renderTargetProtoFile(table *Table, columns []*CodeGenColumn, methods []*Proto, protoPath string) string {
	methods = sortCodeGenProtoMethods(filterProtoMethods(methods, protoPath))
	if len(methods) == 0 {
		return ""
	}
	target := methods[0].TargetEntityName
	var builder strings.Builder
	builder.WriteString("syntax = \"proto3\";\n\n")
	builder.WriteString("package admin.v1;\n\n")
	builder.WriteString("import \"common/v1/common.proto\";\n")
	builder.WriteString("import \"gnostic/openapi/v3/annotations.proto\";\n")
	builder.WriteString("import \"google/api/annotations.proto\";\n")
	builder.WriteString("import \"google/protobuf/empty.proto\";\n\n")
	builder.WriteString("// Admin" + methods[0].TargetBusinessName + "服务\n")
	builder.WriteString("service " + target + "Service {\n")
	for _, method := range methods {
		builder.WriteString(c.renderProtoRPC(table, method, resourcePathByEntity(target)))
	}
	builder.WriteString("}\n\n")
	builder.WriteString(c.renderGeneratedProtoMessages(table, columns, methods, protoPath))
	return builder.String()
}

// renderProtoPatch 渲染已有 Proto 文件需要追加的 RPC 与 message。
func (c *renderer) renderProtoPatch(table *Table, columns []*CodeGenColumn, methods []*Proto, protoPath string) CodeGenProtoPatch {
	filtered := filterProtoMethods(methods, protoPath)
	patch := CodeGenProtoPatch{
		ServiceNames: make([]string, 0, 1),
		RPCs:         make(map[string][]string),
		Messages:     make([]string, 0, len(filtered)),
	}
	messageNames := make(map[string]struct{})
	// 一个 Proto 文件可能包含多个 service，RPC 按目标 service 分组，message 则在文件级去重。
	for _, method := range filtered {
		exists, _ := c.protoMethodExists(method.ProtoFilePath, method.TargetEntityName, method.MethodName)
		if exists {
			continue
		}
		serviceName := method.TargetEntityName + "Service"
		if _, ok := patch.RPCs[serviceName]; !ok {
			patch.ServiceNames = append(patch.ServiceNames, serviceName)
		}
		patch.RPCs[serviceName] = append(patch.RPCs[serviceName], c.renderProtoRPC(table, method, resourcePathByEntity(method.TargetEntityName)))
		patch.Messages = append(patch.Messages, c.renderPatchProtoMessages(table, columns, method, protoPath, messageNames)...)
	}
	return patch
}

// appendProtoPatch 将 RPC 插入 service 内部，并将 message 追加到文件末尾。
func (c *renderer) appendProtoPatch(content string, patch CodeGenProtoPatch) string {
	// 任一目标 service 不存在时放弃整次补丁，避免把 RPC 插到错误位置。
	for _, serviceName := range patch.ServiceNames {
		serviceStart, serviceEnd := findProtoServiceBounds(content, serviceName)
		if serviceStart < 0 || serviceEnd < 0 {
			return content
		}
	}
	// 只有新增方法实际引用公共响应类型时才补 common.proto 导入。
	if patch.CommonImportRequired() && !strings.Contains(content, "common/v1/common.proto") {
		content = insertProtoImport(content, "import \"common/v1/common.proto\";\n")
	}
	// 动态 GET 路由必须排在 /{id} 之前，否则 HTTP 路由会把静态段误识别为 ID。
	for _, serviceName := range patch.ServiceNames {
		rpcContent := strings.Join(patch.RPCs[serviceName], "")
		_, serviceEnd := findProtoServiceBounds(content, serviceName)
		insertIndex := findProtoDynamicGETInsertIndex(content, serviceName)
		if insertIndex < 0 {
			insertIndex = serviceEnd
		}
		content = content[:insertIndex] + "\n" + strings.TrimRight(rpcContent, "\n") + "\n" + content[insertIndex:]
	}
	// message 是文件级声明，统一追加到末尾，后续再由去重与格式化流程整理。
	if len(patch.Messages) > 0 {
		content = strings.TrimRight(content, "\n") + "\n\n" + strings.TrimSpace(strings.Join(patch.Messages, "\n")) + "\n"
	}
	return content
}

// renderProtoRPC 渲染单个 RPC 契约。
func (c *renderer) renderProtoRPC(table *Table, method *Proto, resourcePath string) string {
	businessName := codeGenProtoMethodBusinessName(table, method)
	switch method.APIKind {
	case APIKindList:
		return fmt.Sprintf(`  // 查询%s分页列表
  rpc %s(%sRequest) returns (%sResponse) {
    option (google.api.http) = {
      get: "/api/v1/admin/%s"
    };
  }
`, table.BusinessName, method.MethodName, method.MethodName, method.MethodName, resourcePath)
	case APIKindTree:
		if method.TriggerType == TriggerEntityOption || method.TriggerType == TriggerFieldOption || method.TriggerType == TriggerLeftTree {
			return fmt.Sprintf(`  // 查询%s树形选择
  rpc %s(%sRequest) returns (.common.v1.TreeOptionResponse) {
    option (google.api.http) = {
      get: "/api/v1/admin/%s/option"
    };
  }
`, businessName, method.MethodName, method.MethodName, resourcePath)
		}
		return fmt.Sprintf(`  // 查询%s树形列表
  rpc %s(%sRequest) returns (%sResponse) {
    option (google.api.http) = {
      get: "/api/v1/admin/%s/tree"
    };
  }
`, table.BusinessName, method.MethodName, method.MethodName, method.MethodName, resourcePath)
	case APIKindOption:
		return fmt.Sprintf(`  // 查询%s下拉选择
  rpc %s(%sRequest) returns (.common.v1.SelectOptionResponse) {
    option (google.api.http) = {
      get: "/api/v1/admin/%s/option"
    };
  }
`, businessName, method.MethodName, method.MethodName, resourcePath)
	case APIKindStatus:
		return fmt.Sprintf(`  // 设置状态
  rpc %s(%sRequest) returns (google.protobuf.Empty) {
    option (google.api.http) = {
      put: "/api/v1/admin/%s/{id}/%s"
      body: "*"
    };
  }
`, method.MethodName, method.MethodName, resourcePath, statusResourcePath(table, method))
	}
	return c.renderCRUDProtoRPC(table, method, resourcePath)
}

// renderCRUDProtoRPC 渲染标准 CRUD RPC。
func (c *renderer) renderCRUDProtoRPC(table *Table, method *Proto, resourcePath string) string {
	entity := table.EntityName
	switch method.MethodName {
	case "Get" + entity:
		return fmt.Sprintf(`  // 查询%s详情
  rpc Get%s(Get%sRequest) returns (%sForm) {
    option (google.api.http) = {
      get: "/api/v1/admin/%s/{id}"
    };
  }
`, table.BusinessName, entity, entity, entity, resourcePath)
	case "Create" + entity:
		return fmt.Sprintf(`  // 创建%s
  rpc Create%s(Create%sRequest) returns (google.protobuf.Empty) {
    option (google.api.http) = {
      post: "/api/v1/admin/%s"
      body: "%s"
    };
  }
`, table.BusinessName, entity, entity, resourcePath, stringcase.ToSnakeCase(entity))
	case "Update" + entity:
		return fmt.Sprintf(`  // 更新%s
  rpc Update%s(Update%sRequest) returns (google.protobuf.Empty) {
    option (google.api.http) = {
      put: "/api/v1/admin/%s/{id}"
      body: "%s"
    };
  }
`, table.BusinessName, entity, entity, resourcePath, stringcase.ToSnakeCase(entity))
	case "Delete" + entity:
		return fmt.Sprintf(`  // 删除%s
  rpc Delete%s(Delete%sRequest) returns (google.protobuf.Empty) {
    option (google.api.http) = {
      delete: "/api/v1/admin/%s/{ids}"
    };
  }
`, table.BusinessName, entity, entity, resourcePath)
	default:
		return ""
	}
}

// renderProtoMessage 渲染 Proto 数据消息。
func (c *renderer) renderProtoMessage(table *Table, columns []*CodeGenColumn, form bool) string {
	entity := table.EntityName
	name := entity
	title := table.BusinessName
	if form {
		name += "Form"
		title += "表单"
	}
	var builder strings.Builder
	builder.WriteString("// " + title + "\n")
	builder.WriteString("message " + name + " {\n")
	fieldNo := int32(1)
	for _, column := range columns {
		if !form && (column.ColumnName == "deleted_at" || !generatedListIncludesColumn(column) && column.IsPrimary != 1 && column.ColumnName != "created_at" && column.ColumnName != "updated_at") {
			continue
		}
		if form && !generatedFormIncludesColumn(column) && column.IsPrimary != 1 {
			continue
		}
		builder.WriteString(c.renderProtoField(column, fieldNo, form && column.IsPrimary != 1))
		fieldNo++
	}
	if !form && table.PageType == PageTypeTree {
		builder.WriteString(fmt.Sprintf("\n  repeated %s children = 300 [(gnostic.openapi.v3.property) = {description: \"子节点树\"}]; // 子节点树\n", entity))
	}
	builder.WriteString("}\n")
	return builder.String()
}

// normalizeProtoRPCOrder 按服务分别整理已有 RPC，未识别的方法保持原有相对顺序。
func normalizeProtoRPCOrder(content string, methods []*Proto, protoPath string) string {
	serviceMethods := make(map[string][]*Proto)
	// 先按 service 拆分，避免不同实体的同名方法相互影响排序。
	for _, method := range methods {
		if method.ProtoFilePath != protoPath {
			continue
		}
		serviceName := method.TargetEntityName + "Service"
		serviceMethods[serviceName] = append(serviceMethods[serviceName], method)
	}
	serviceNames := make([]string, 0, len(serviceMethods))
	for serviceName := range serviceMethods {
		serviceNames = append(serviceNames, serviceName)
	}
	slices.Sort(serviceNames)
	for _, serviceName := range serviceNames {
		content = normalizeProtoServiceRPCOrder(content, serviceName, serviceMethods[serviceName])
	}
	return content
}

// normalizeProtoServiceRPCOrder 按固定槽位重排单个 service 的 RPC 块。
func normalizeProtoServiceRPCOrder(content string, serviceName string, methods []*Proto) string {
	serviceStart, serviceEnd := findProtoServiceBounds(content, serviceName)
	if serviceStart < 0 || serviceEnd < 0 {
		return content
	}
	bodyStart := serviceStart + 1
	body := content[bodyStart:serviceEnd]
	matches := protoRPCPattern.FindAllStringSubmatchIndex(body, -1)
	if len(matches) < 2 {
		return content
	}
	blocks := make([]CodeGenProtoRPCBlock, 0, len(matches))
	methodNames := make(map[string]struct{}, len(matches))
	firstBlockStart := -1
	lastBlockEnd := -1
	// 每个块包含紧邻方法的注释，遇到无法识别的非空间隔时保持原文不动。
	for _, match := range matches {
		if len(match) < 4 {
			return content
		}
		openOffset := strings.Index(body[match[0]:], "{")
		if openOffset < 0 {
			return content
		}
		blockEnd := findProtoBlockEnd(body, match[0]+openOffset)
		if blockEnd < 0 {
			return content
		}
		blockStart := protoDeclarationCommentStart(body, match[0])
		if lastBlockEnd >= 0 && strings.TrimSpace(body[lastBlockEnd:blockStart]) != "" {
			return content
		}
		if firstBlockStart < 0 {
			firstBlockStart = blockStart
		}
		lastBlockEnd = blockEnd + 1
		name := body[match[2]:match[3]]
		// 同名 RPC 不能在同一 service 中重复声明，保留首次出现的稳定定义。
		if _, exists := methodNames[name]; exists {
			continue
		}
		methodNames[name] = struct{}{}
		blocks = append(blocks, CodeGenProtoRPCBlock{
			Name:          name,
			Content:       strings.Trim(body[blockStart:lastBlockEnd], "\r\n"),
			OriginalIndex: len(blocks),
		})
	}
	// 已知生成方法进入固定槽位，人工扩展方法保持原有相对顺序并排在其后。
	methodOrder := make(map[string]int, len(methods))
	for i, method := range sortCodeGenProtoMethods(methods) {
		if _, ok := methodOrder[method.MethodName]; !ok {
			methodOrder[method.MethodName] = i
		}
	}
	slices.SortStableFunc(blocks, func(a CodeGenProtoRPCBlock, b CodeGenProtoRPCBlock) int {
		aOrder, aKnown := methodOrder[a.Name]
		bOrder, bKnown := methodOrder[b.Name]
		if aKnown && bKnown && aOrder != bOrder {
			return aOrder - bOrder
		}
		if aKnown != bKnown {
			if aKnown {
				return -1
			}
			return 1
		}
		return a.OriginalIndex - b.OriginalIndex
	})
	prefix := strings.TrimRight(body[:firstBlockStart], " \t\r\n")
	suffix := strings.TrimSpace(body[lastBlockEnd:])
	// 只重建 service 内连续 RPC 区域，service 外部内容和无法识别的尾部内容保持不变。
	var builder strings.Builder
	builder.WriteString(prefix)
	if prefix == "" {
		builder.WriteString("\n")
	} else {
		builder.WriteString("\n\n")
	}
	for i, block := range blocks {
		if i > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(block.Content)
	}
	if suffix != "" {
		builder.WriteString("\n\n")
		builder.WriteString(suffix)
	}
	builder.WriteString("\n")
	return content[:bodyStart] + builder.String() + content[serviceEnd:]
}

// dedupeProtoMessageBlocks 删除重复的顶层 message 声明并保留首次出现的定义。
func dedupeProtoMessageBlocks(content string) string {
	matches := protoMessagePattern.FindAllStringSubmatchIndex(content, -1)
	if len(matches) < 2 {
		return content
	}
	seen := make(map[string]struct{}, len(matches))
	var builder strings.Builder
	cursor := 0
	// 仅处理顶层 message，嵌套 message 即使同名也属于不同作用域。
	for _, match := range matches {
		if len(match) < 4 || protoBraceDepthAt(content, match[0]) != 0 {
			continue
		}
		name := content[match[2]:match[3]]
		if _, ok := seen[name]; !ok {
			seen[name] = struct{}{}
			continue
		}
		openOffset := strings.Index(content[match[0]:match[1]], "{")
		if openOffset < 0 {
			continue
		}
		blockEnd := findProtoBlockEnd(content, match[0]+openOffset)
		if blockEnd < 0 {
			continue
		}
		blockStart := protoDeclarationCommentStart(content, match[0])
		builder.WriteString(content[cursor:blockStart])
		cursor = blockEnd + 1
		for cursor < len(content) && content[cursor] == '\n' {
			cursor++
		}
	}
	if cursor == 0 {
		return content
	}
	builder.WriteString(content[cursor:])
	return builder.String()
}

// protoBraceDepthAt 返回指定位置之前的 Proto 大括号嵌套深度。
func protoBraceDepthAt(content string, position int) int {
	depth := 0
	for i := 0; i < position; i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
		}
	}
	return depth
}

// protoDeclarationCommentStart 返回声明及其紧邻注释块的起始位置。
func protoDeclarationCommentStart(content string, declarationStart int) int {
	lineStart := strings.LastIndex(content[:declarationStart], "\n") + 1
	for lineStart > 0 {
		previousLineEnd := lineStart - 1
		previousLineStart := strings.LastIndex(content[:previousLineEnd], "\n") + 1
		if !strings.HasPrefix(strings.TrimSpace(content[previousLineStart:previousLineEnd]), "//") {
			break
		}
		lineStart = previousLineStart
	}
	return lineStart
}

// findProtoDynamicGETInsertIndex 查找 service 中首个动态 GET RPC 的插入位置。
func findProtoDynamicGETInsertIndex(content string, serviceName string) int {
	serviceStart, serviceEnd := findProtoServiceBounds(content, serviceName)
	if serviceStart < 0 || serviceEnd < 0 {
		return -1
	}
	bodyStart := serviceStart + 1
	body := content[bodyStart:serviceEnd]
	dynamicGETPattern := regexp.MustCompile(`(?m)^[\t ]*get:[\t ]*"[^"]*\{[^"]*"[\t ]*$`)
	dynamicGET := dynamicGETPattern.FindStringIndex(body)
	if dynamicGET == nil {
		return -1
	}
	prefix := body[:dynamicGET[0]]
	rpcIndex := strings.LastIndex(prefix, "\n  rpc ")
	if rpcIndex < 0 {
		return -1
	}
	rpcIndex++
	commentIndex := strings.LastIndex(prefix[:rpcIndex], "\n  // ")
	if commentIndex >= 0 {
		return bodyStart + commentIndex + 1
	}
	return bodyStart + rpcIndex
}

// insertProtoImport 将 import 插入 package 声明之后。
func insertProtoImport(content string, importLine string) string {
	packageIndex := strings.Index(content, "package ")
	if packageIndex < 0 {
		return importLine + content
	}
	lineEnd := strings.Index(content[packageIndex:], "\n")
	if lineEnd < 0 {
		return content + "\n\n" + importLine
	}
	insertIndex := packageIndex + lineEnd + 1
	return content[:insertIndex] + "\n" + importLine + content[insertIndex:]
}

// ensureGoImport 确保 Go 文件包含指定 import。
func ensureGoImport(content string, importLine string) string {
	if strings.Contains(content, importLine) {
		return content
	}
	importBlock := "import (\n"
	if strings.Contains(content, importBlock) {
		return strings.Replace(content, importBlock, importBlock+"\t"+importLine+"\n", 1)
	}
	importIndex := strings.Index(content, "import ")
	if importIndex < 0 {
		return content
	}
	lineEnd := strings.Index(content[importIndex:], "\n")
	if lineEnd < 0 {
		return content
	}
	existingImport := strings.TrimSpace(content[importIndex : importIndex+lineEnd])
	return content[:importIndex] + "import (\n\t" + strings.TrimPrefix(existingImport, "import ") + "\n\t" + importLine + "\n)" + content[importIndex+lineEnd:]
}

// ensureTSNamedTypeNames 确保 TS 文件从指定模块导入类型集合。
func ensureTSNamedTypeNames(content string, importPath string, typeNames []string) string {
	missing := make([]string, 0, len(typeNames))
	seen := make(map[string]struct{}, len(typeNames))
	for _, typeName := range typeNames {
		if typeName == "" {
			continue
		}
		if _, ok := seen[typeName]; ok {
			continue
		}
		seen[typeName] = struct{}{}
		if strings.Contains(content, "type "+typeName) {
			continue
		}
		missing = append(missing, "  type "+typeName)
	}
	if len(missing) == 0 {
		return content
	}
	fromLine := "\n} from \"" + importPath + "\";"
	if !strings.Contains(content, fromLine) {
		return "import {\n" + strings.Join(missing, ",\n") + "\n} from \"" + importPath + "\";\n" + content
	}
	return strings.Replace(content, fromLine, ",\n"+strings.Join(missing, ",\n")+fromLine, 1)
}

// ensureTSCommonOptionImport 确保 TS 文件只导入方法实际使用的选项响应类型。
func ensureTSCommonOptionImport(content string, methods []*Proto) string {
	typeNames := tsCommonOptionResponseTypes(methods)
	if len(typeNames) == 0 {
		return content
	}
	if strings.Contains(content, "@/rpc/common/v1/common") {
		for _, typeName := range typeNames {
			if !strings.Contains(content, typeName) {
				fromLine := "} from \"@/rpc/common/v1/common\";"
				index := strings.Index(content, fromLine)
				if index >= 0 {
					content = content[:index] + ", " + typeName + " " + content[index:]
				}
			}
		}
		return content
	}
	importLine := "import type { " + strings.Join(typeNames, ", ") + " } from \"@/rpc/common/v1/common\";"
	insertIndex := strings.Index(content, "\n\nconst ")
	if insertIndex < 0 {
		return importLine + "\n" + content
	}
	return content[:insertIndex] + "\n" + importLine + content[insertIndex:]
}

// tsCommonOptionResponseTypes 返回选项方法实际使用的公共响应类型。
func tsCommonOptionResponseTypes(methods []*Proto) []string {
	typeNames := make([]string, 0, 2)
	seen := make(map[string]struct{}, 2)
	for _, method := range methods {
		typeName := "SelectOptionResponse"
		if method.APIKind == APIKindTree {
			typeName = "TreeOptionResponse"
		}
		if _, ok := seen[typeName]; ok {
			continue
		}
		seen[typeName] = struct{}{}
		typeNames = append(typeNames, typeName)
	}
	return typeNames
}

// protoServiceMethodExists 判断目标 Proto service 内是否已经存在指定 RPC。
func protoServiceMethodExists(content string, serviceName string, methodName string) bool {
	serviceStart, serviceEnd := findProtoServiceBounds(content, serviceName)
	if serviceStart < 0 || serviceEnd < 0 {
		return false
	}
	matches := protoRPCPattern.FindAllStringSubmatch(content[serviceStart:serviceEnd], -1)
	for _, match := range matches {
		if len(match) > 1 && match[1] == methodName {
			return true
		}
	}
	return false
}

// findProtoServiceBounds 返回指定 Proto service 的大括号起止位置。
func findProtoServiceBounds(content string, serviceName string) (int, int) {
	if serviceName == "" {
		return -1, -1
	}
	pattern := regexp.MustCompile(`(?m)^[\t ]*service[\t ]+` + regexp.QuoteMeta(serviceName) + `[\t ]*\{`)
	location := pattern.FindStringIndex(content)
	if location == nil {
		return -1, -1
	}
	openIndex := strings.LastIndex(content[location[0]:location[1]], "{") + location[0]
	closeIndex := findProtoBlockEnd(content, openIndex)
	if closeIndex < 0 {
		return -1, -1
	}
	return openIndex, closeIndex
}

// findProtoBlockEnd 返回指定左大括号对应的右大括号位置。
func findProtoBlockEnd(content string, openIndex int) int {
	depth := 0
	for i := openIndex; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// codeGenProtoMethodBusinessName 返回 Proto 方法描述使用的业务名称。
func codeGenProtoMethodBusinessName(table *Table, method *Proto) string {
	if method.TargetEntityName == "" || method.TargetEntityName == table.EntityName {
		return table.BusinessName
	}
	return DefaultString(method.TargetBusinessName, method.TargetEntityName)
}
