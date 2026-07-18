package codegen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/errorsx"

	"github.com/liujitcn/go-utils/stringcase"
)

// --- 管理端 API 与页面模板渲染 ---

// FrontendPageComponentPath 根据页面文件路径推导动态路由组件路径。
func FrontendPageComponentPath(path string) string {
	relativePath := strings.TrimPrefix(path, "frontend/admin/src/views/")
	return strings.TrimSuffix(relativePath, "/index.vue")
}

// SafeRepoFilePath 返回仓库内安全文件路径。
func SafeRepoFilePath(path string) (string, error) {
	if path == "" {
		return "", errorsx.InvalidArgument("文件路径不允许为空")
	}
	if filepath.IsAbs(path) {
		return "", errorsx.InvalidArgument("文件路径不允许使用绝对路径")
	}
	cleanPath := filepath.Clean(path)
	if cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return "", errorsx.InvalidArgument("文件路径不允许跳出仓库目录")
	}

	root, err := filepath.Abs(repoRoot())
	if err != nil {
		return "", err
	}
	fullPath := filepath.Join(root, cleanPath)
	var relPath string
	relPath, err = filepath.Rel(root, fullPath)
	if err != nil {
		return "", err
	}
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", errorsx.InvalidArgument("文件路径不允许跳出仓库目录")
	}
	return fullPath, nil
}

// BackendDir 返回当前仓库的后端目录。
func BackendDir() string {
	return filepath.Join(repoRoot(), "backend")
}

// BoolToInt32 转换布尔数据库值。
func BoolToInt32(value bool) int32 {
	if value {
		return 1
	}
	return 0
}

// DefaultString 返回默认字符串。
func DefaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// PermissionPrefix 返回页面权限前缀。
func PermissionPrefix(table *Table) string {
	if table.PermissionPrefix != "" {
		return table.PermissionPrefix
	}
	return strings.ReplaceAll(strings.Trim(table.ModulePath, "/"), "/", ":") + ":" + stringcase.ToSnakeCase(table.EntityName)
}

// IsOptionProtoMethod 判断方法是否是选择项接口。
func IsOptionProtoMethod(method *Proto) bool {
	return method.APIKind == APIKindOption || method.APIKind == APIKindTree && (method.TriggerType == TriggerEntityOption || method.TriggerType == TriggerFieldOption || method.TriggerType == TriggerLeftTree)
}

// InferTSType 按数据库类型推断 TS 类型。
func InferTSType(dbType string) string {
	protoType := InferProtoType(dbType)
	if protoType == "bool" {
		return "boolean"
	}
	if protoType == "int64" || protoType == "int32" || protoType == "double" {
		return "number"
	}
	return "string"
}

// InferGoType 按数据库类型推断 Go 类型。
func InferGoType(dbType string) string {
	if isBoolDBType(dbType) {
		return "bool"
	}
	lowerType := strings.ToLower(dbType)
	if strings.Contains(lowerType, "bigint") {
		return "int64"
	}
	if isNumericDBType(lowerType) {
		if strings.Contains(lowerType, "decimal") || strings.Contains(lowerType, "float") || strings.Contains(lowerType, "double") {
			return "float64"
		}
		return "int32"
	}
	if isDateTimeDBType(lowerType) {
		return "time.Time"
	}
	return "string"
}

// InferProtoType 按数据库类型推断 Proto 类型。
func InferProtoType(dbType string) string {
	lowerType := strings.ToLower(dbType)
	if isBoolDBType(lowerType) {
		return "bool"
	}
	if strings.Contains(lowerType, "bigint") {
		return "int64"
	}
	if strings.Contains(lowerType, "int") {
		return "int32"
	}
	if strings.Contains(lowerType, "decimal") || strings.Contains(lowerType, "float") || strings.Contains(lowerType, "double") {
		return "double"
	}
	return "string"
}

// renderFrontendAPIFile 渲染前端 API 文件占位内容。
func (c *renderer) renderFrontendAPIFile(table *Table, columns []*CodeGenColumn, methods []*Proto) string {
	entity := table.EntityName
	snakeEntity := stringcase.ToSnakeCase(entity)
	urlConst := stringcase.UpperSnakeCase(entity) + "_URL"
	statusMethods := methodsByKinds(methods, APIKindStatus)
	hasTree := firstMethodByKind(methods, APIKindTree, TriggerPageTree) != nil
	optionMethod := currentEntityOptionMethod(table, methods)
	methodNames := protoMethodNameSet(methods)
	_, hasPage := methodNames["Page"+entity]
	_, hasGet := methodNames["Get"+entity]
	_, hasCreate := methodNames["Create"+entity]
	_, hasUpdate := methodNames["Update"+entity]
	_, hasDelete := methodNames["Delete"+entity]
	var typeImports []string
	if hasCreate {
		typeImports = append(typeImports, "  type Create"+entity+"Request")
	}
	if hasDelete {
		typeImports = append(typeImports, "  type Delete"+entity+"Request")
	}
	if hasGet {
		typeImports = append(typeImports, "  type Get"+entity+"Request")
	}
	if hasTree {
		typeImports = append(typeImports, "  type Tree"+entity+"Request", "  type Tree"+entity+"Response")
	} else if hasPage {
		typeImports = append(typeImports, "  type Page"+entity+"Request", "  type Page"+entity+"Response")
	}
	if optionMethod != nil {
		typeImports = append(typeImports, "  type "+optionMethod.MethodName+"Request")
	}
	for _, statusMethod := range statusMethods {
		typeImports = append(typeImports, "  type "+statusMethod.MethodName+"Request")
	}
	if hasGet || hasCreate || hasUpdate {
		typeImports = append(typeImports, "  type "+entity+"Form")
	}
	typeImports = append(typeImports, "  type "+entity+"Service")
	if hasUpdate {
		typeImports = append(typeImports, "  type Update"+entity+"Request")
	}

	emptyImport := ""
	if hasCreate || hasUpdate || hasDelete || len(statusMethods) > 0 {
		emptyImport = "import type { Empty } from \"@/rpc/google/protobuf/empty\";\n"
	}
	optionImport := ""
	if optionMethod != nil {
		optionImport = "import type { " + strings.Join(tsCommonOptionResponseTypes([]*Proto{optionMethod}), ", ") + " } from \"@/rpc/common/v1/common\";\n"
	}
	var methodsBuilder strings.Builder
	if hasTree {
		methodsBuilder.WriteString(fmt.Sprintf(`
  /** 查询树形列表 */
  Tree%s(request: Tree%sRequest): Promise<Tree%sResponse> {
    return service<Tree%sRequest, Tree%sResponse>({
      url: %s + "/tree",
      method: "get",
      params: request
    });
  }
`, entity, entity, entity, entity, entity, urlConst))
	}
	if optionMethod != nil {
		returnType := "SelectOptionResponse"
		comment := "查询下拉选择"
		if optionMethod.APIKind == APIKindTree {
			returnType = "TreeOptionResponse"
			comment = "查询树形选择"
		}
		methodsBuilder.WriteString(fmt.Sprintf(`
  /** %s */
  %s(request: %sRequest): Promise<%s> {
    return service<%sRequest, %s>({
      url: %s + "/option",
      method: "get",
      params: request
    });
  }
`, comment, optionMethod.MethodName, optionMethod.MethodName, returnType, optionMethod.MethodName, returnType, urlConst))
	}
	coreMethods := fmt.Sprintf(`
  /** 查询分页列表 */
  Page%s(request: Page%sRequest): Promise<Page%sResponse> {
    return service<Page%sRequest, Page%sResponse>({
      url: %s,
      method: "get",
      params: request
    });
  }

  /** 查询详情 */
  Get%s(request: Get%sRequest): Promise<%sForm> {
    return service<Get%sRequest, %sForm>({
      url: %s + "/" + request.id,
      method: "get"
    });
  }

  /** 创建 */
  Create%s(request: Create%sRequest): Promise<Empty> {
    return service<%sForm | undefined, Empty>({
      url: %s,
      method: "post",
      data: request.%s
    });
  }

  /** 更新 */
  Update%s(request: Update%sRequest): Promise<Empty> {
    return service<%sForm | undefined, Empty>({
      url: %s + "/" + request.id,
      method: "put",
      data: request.%s
    });
  }

  /** 删除 */
  Delete%s(request: Delete%sRequest): Promise<Empty> {
    return service<Delete%sRequest, Empty>({
      url: %s + "/" + request.ids,
      method: "delete"
    });
  }
`, entity, entity, entity, entity, entity, urlConst, entity, entity, entity, entity, entity, urlConst, entity, entity, entity, urlConst, snakeEntity, entity, entity, entity, urlConst, snakeEntity, entity, entity, entity, urlConst)
	if hasTree {
		detailMethodIndex := strings.Index(coreMethods, "  /** 查询详情 */")
		if detailMethodIndex >= 0 {
			coreMethods = coreMethods[detailMethodIndex:]
		}
	}
	methodsBuilder.WriteString(coreMethods)
	for _, statusMethod := range statusMethods {
		methodsBuilder.WriteString(fmt.Sprintf(`
  /** 设置状态 */
  %s(request: %sRequest): Promise<Empty> {
    return service<%sRequest, Empty>({
      url: %s + "/" + request.id + "/%s",
      method: "put",
      data: request
    });
  }
`, statusMethod.MethodName, statusMethod.MethodName, statusMethod.MethodName, urlConst, statusResourcePath(table, statusMethod)))
	}
	content := renderTemplate("frontend_api.tmpl", frontendAPITemplateData{
		Entity:       entity,
		BusinessName: table.BusinessName,
		TypeImports:  strings.Join(typeImports, ",\n"),
		RPCImport:    frontendRPCImportPath(c.defaultProtoPath(table)),
		EmptyImport:  emptyImport,
		OptionImport: optionImport,
		URLConst:     urlConst,
		ResourcePath: c.frontendResourcePath(table),
		Methods:      methodsBuilder.String(),
	})
	content = removeTSClassMethods(content, missingCoreMethodNames(table, methods))
	return reorderTSClassMethods(content, entity+"ServiceImpl")
}

// renderExternalTargetFrontendAPIFile 渲染外部目标实体的最小前端 API 文件。
func (c *renderer) renderExternalTargetFrontendAPIFile(table *Table, methods []*Proto) string {
	entity := table.EntityName
	urlConst := stringcase.UpperSnakeCase(entity) + "_URL"
	var typeImports []string
	for _, method := range methods {
		typeImports = append(typeImports, "  type "+method.MethodName+"Request")
	}
	typeImports = append(typeImports, "  type "+entity+"Service")
	return renderTemplate("frontend_api.tmpl", frontendAPITemplateData{
		Entity:       entity,
		BusinessName: table.BusinessName,
		TypeImports:  strings.Join(typeImports, ",\n"),
		RPCImport:    frontendRPCImportPath(c.defaultProtoPath(table)),
		OptionImport: "import type { " + strings.Join(tsCommonOptionResponseTypes(methods), ", ") + " } from \"@/rpc/common/v1/common\";\n",
		URLConst:     urlConst,
		ResourcePath: c.frontendResourcePath(table),
		Methods:      c.renderExternalTargetFrontendAPIMethods(table, methods),
	})
}

// appendExternalTargetFrontendAPIMethods 向已有前端 API 文件追加外部目标选项方法。
func (c *renderer) appendExternalTargetFrontendAPIMethods(content string, table *Table, methods []*Proto) string {
	className := table.EntityName + "ServiceImpl"
	if findTSClassEndIndex(content, className) < 0 {
		return content
	}
	missingMethods := make([]*Proto, 0, len(methods))
	for _, method := range methods {
		if tsClassMethodExists(content, className, method.MethodName) {
			continue
		}
		missingMethods = append(missingMethods, method)
	}
	if len(missingMethods) == 0 {
		return content
	}
	typeNames := make([]string, 0, len(missingMethods))
	for _, method := range missingMethods {
		typeNames = append(typeNames, method.MethodName+"Request")
	}
	content = ensureTSNamedTypeNames(content, frontendRPCImportPath(c.defaultProtoPath(table)), typeNames)
	content = ensureTSCommonOptionImport(content, missingMethods)
	index := findTSClassEndIndex(content, className)
	if index < 0 {
		return content
	}
	content = content[:index] + "\n" + c.renderExternalTargetFrontendAPIMethods(table, missingMethods) + content[index:]
	return reorderTSClassMethods(content, className)
}

// renderExternalTargetFrontendAPIMethods 渲染外部目标实体选项 API 方法。
func (c *renderer) renderExternalTargetFrontendAPIMethods(table *Table, methods []*Proto) string {
	urlConst := stringcase.UpperSnakeCase(table.EntityName) + "_URL"
	var builder strings.Builder
	for _, method := range methods {
		returnType := "SelectOptionResponse"
		comment := "查询下拉选择"
		if method.APIKind == APIKindTree {
			returnType = "TreeOptionResponse"
			comment = "查询树形选择"
		}
		builder.WriteString(fmt.Sprintf(`
  /** %s */
  %s(request: %sRequest): Promise<%s> {
    return service<%sRequest, %s>({
      url: %s + "/option",
      method: "get",
      params: request
    });
  }
`, comment, method.MethodName, method.MethodName, returnType, method.MethodName, returnType, urlConst))
	}
	return builder.String()
}

// renderFrontendPageFile 渲染前端页面内容。
func (c *renderer) renderFrontendPageFile(table *Table, columns []*CodeGenColumn, methods []*Proto, paths *adminv1.CodeGenOutputPaths) string {
	// 页面中的查询、列表与表单共用字段排序，Proto 仍使用调用方保留的数据库字段顺序。
	columns = slices.Clone(columns)
	slices.SortStableFunc(columns, func(left *CodeGenColumn, right *CodeGenColumn) int {
		if left.Sort < right.Sort {
			return -1
		}
		if left.Sort > right.Sort {
			return 1
		}
		return 0
	})
	entity := table.EntityName
	pluralEntity := pluralize(entity)
	snakeEntity := stringcase.ToSnakeCase(entity)
	frontendAPIPath := strings.TrimPrefix(paths.GetFrontendApiFilePath(), "frontend/admin/src/")
	frontendAPIImport := "@/" + strings.TrimSuffix(frontendAPIPath, filepath.Ext(frontendAPIPath))
	frontendRPCImport := frontendRPCImportPath(paths.GetProtoFilePath())
	listField := stringcase.ToSnakeCase(pluralEntity)
	statusTypeImport := renderFrontendStatusTypeImports(columns, methods)
	proFormTypeImport := "ProFormField"
	for _, column := range columns {
		hasTypedOptions := false
		for _, scope := range frontendOptionScopes(column) {
			if scope.option.SourceType == OptionSourceStatic || scope.option.SourceType == OptionSourceTable {
				hasTypedOptions = true
				break
			}
		}
		if statusNeedsFrontendOptions(column) || hasTypedOptions {
			proFormTypeImport += ", ProFormOption"
			break
		}
	}
	script := fmt.Sprintf(`<script setup lang="ts">
import { computed, reactive, ref } from "vue";
%s
import type { FormRules } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { %s } from "@/components/ProForm/interface";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { def%sService } from "%s";
%s
import type { Page%sRequest, %s, %sForm%s } from "%s";
%s
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";

defineOptions({
  name: "%s",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();

const dialog = reactive({
  title: "",
  visible: false
});

const formData = reactive<%sForm>({
%s
});

const rules = reactive<FormRules>({
%s
});
%s
/** %s表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
%s
]);

/** %s表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
%s
  {
    prop: "operation",
    label: "操作",
    width: 150,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "编辑",
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["%s:update"],
        onClick: scope => handleOpenDialog((scope.row as %s).id)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["%s:delete"],
        onClick: scope => handleDelete(scope.row as %s)
      }
    ]
  }
];

/** %s顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["%s:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["%s:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as %s[])
  }
];

%s
/**
 * 请求%s列表，并适配 ProTable 固定列表字段。
 */
async function request%sTable(params: Page%sRequest) {
  const requestParams = buildPageRequest(params);
  const data = await def%sService.Page%s(requestParams);
  const compatData = data as typeof data & { list?: typeof data.%s };
  const list = compatData.%s ?? compatData.list ?? [];
  return { data: { ...data, list } };
}

/**
 * 刷新%s表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 重置%s表单。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
%s
}

/**
 * 打开%s弹窗。
 */
async function handleOpenDialog(id?: number) {
  resetForm();
%s  dialog.title = id ? "修改%s" : "新增%s";
  dialog.visible = true;
  if (!id) return;

  const data = await def%sService.Get%s({ id });
  Object.assign(formData, data);
}

/**
 * 提交%s表单。
 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;

    const request = formData.id
      ? def%sService.Update%s({ id: formData.id, %s: formData })
      : def%sService.Create%s({ %s: formData });
    request.then(() => {
      ElMessage.success(formData.id ? "修改%s成功" : "新增%s成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}
%s
/**
 * 删除%s，兼容单项删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | %s | %s[]) {
  const rowList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as %s[])
    : selected && typeof selected === "object"
      ? [selected as %s]
      : [];
  const ids = (
    rowList.length ? rowList.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!ids) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  const confirmMessage = rowList.length === 1 ? "是否确定删除%s？" : "确认删除已选中的%s吗？";
  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      def%sService.Delete%s({ ids }).then(() => {
        ElMessage.success("删除%s成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消删除%s");
    }
  );
}

/**
 * 关闭%s弹窗。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}
</script>
	`, renderFrontendDateImport(columns), proFormTypeImport, entity, frontendAPIImport, c.renderFrontendOptionImports(table, columns, methods), entity, entity, entity, statusTypeImport, frontendRPCImport, c.renderFrontendEnumImports(columns), entity, entity, c.renderFrontendFormDefaults(columns), c.renderFrontendRules(columns), c.renderFrontendStatusOptions(columns)+c.renderFrontendOptionState(columns, methods), table.BusinessName, c.renderFrontendFormFields(columns), table.BusinessName, c.renderFrontendColumns(table, columns, methods), PermissionPrefix(table), entity, PermissionPrefix(table), entity, table.BusinessName, PermissionPrefix(table), PermissionPrefix(table), entity, "", table.BusinessName, entity, entity, entity, entity, listField, listField, table.BusinessName, table.BusinessName, c.renderFrontendResetForm(columns), table.BusinessName, c.renderFrontendLoadOptionsCall(columns, methods), table.BusinessName, table.BusinessName, entity, entity, table.BusinessName, entity, entity, snakeEntity, entity, entity, snakeEntity, table.BusinessName, table.BusinessName, c.renderFrontendStatusHandlers(table, columns, methods), table.BusinessName, entity, entity, entity, entity, table.BusinessName, table.BusinessName, entity, entity, table.BusinessName, table.BusinessName, table.BusinessName)
	content := renderTemplate("frontend_page.tmpl", frontendPageTemplateData{Entity: entity, BusinessName: table.BusinessName, Script: script})
	return c.applyFrontendPageType(content, table, methods, frontendAPIImport)
}

// applyFrontendPageType 根据页面类型补充树表格或左树右表结构。
func (c *renderer) applyFrontendPageType(content string, table *Table, methods []*Proto, frontendAPIImport string) string {
	switch table.PageType {
	case PageTypeTree:
		return c.applyFrontendTreePage(content, table, methods)
	case PageTypeLeftTree:
		return c.applyFrontendLeftTreePage(content, table, methods, frontendAPIImport)
	default:
		return content
	}
}

// applyFrontendTreePage 将普通分页模板调整为树形表格模板。
func (c *renderer) applyFrontendTreePage(content string, table *Table, methods []*Proto) string {
	treeMethod := firstMethodByKind(methods, APIKindTree, TriggerPageTree)
	if treeMethod == nil {
		return content
	}
	pluralEntity := pluralize(table.EntityName)
	content = strings.ReplaceAll(content, "Page"+table.EntityName+"Request", "Tree"+table.EntityName+"Request")
	content = strings.Replace(content, `import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";`, `import { normalizeSelectedIds } from "@/utils/proTable";`, 1)
	requestMarker := fmt.Sprintf(`:request-api="request%sTable"`, table.EntityName)
	treeProps := requestMarker + `
      :pagination="false"
      :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"`
	content = strings.Replace(content, requestMarker, treeProps, 1)

	functionName := "request" + table.EntityName + "Table"
	start := strings.Index(content, "async function "+functionName+"(")
	end := findTSFunctionEndIndex(content, start)
	if start < 0 || end < 0 {
		return content
	}
	requestFunction := fmt.Sprintf(`async function %s(params: Tree%sRequest) {
  const data = await def%sService.%s(params);
  return { data: data.%s ?? [] };
}`, functionName, table.EntityName, table.EntityName, treeMethod.MethodName, stringcase.ToSnakeCase(pluralEntity))
	return content[:start] + requestFunction + content[end+1:]
}

// applyFrontendLeftTreePage 将普通分页模板调整为左树右表模板。
func (c *renderer) applyFrontendLeftTreePage(content string, table *Table, methods []*Proto, frontendAPIImport string) string {
	config := LeftTreeConfigFromTable(table)
	filterColumn := DefaultString(config.FilterColumn, "parent_id")
	requestMarker := fmt.Sprintf(`:request-api="request%sTable"`, table.EntityName)
	content = strings.Replace(content, requestMarker, requestMarker+` :init-param="initParam"`, 1)
	content = strings.Replace(content, `  <div class="table-box">`, fmt.Sprintf(`  <div class="main-box">
    <TreeFilter
      label="name"
      title="筛选"
      :request-api="request%sTreeFilter"
      :default-value="treeFilterValue"
      @change="changeTreeFilter"
    />

    <div class="table-box">`, table.EntityName), 1)
	closingIndex := strings.LastIndex(content, "  </div>\n</template>")
	if closingIndex >= 0 {
		content = content[:closingIndex] + "    </div>\n  </div>\n</template>" + content[closingIndex+len("  </div>\n</template>"):]
	}
	content = strings.Replace(content, `import FormDialog from "@/components/Dialog/FormDialog.vue";`, `import FormDialog from "@/components/Dialog/FormDialog.vue";
import TreeFilter from "@/components/TreeFilter/index.vue";`, 1)

	treeMethod := firstMethodByKind(methods, APIKindTree, TriggerLeftTree)
	serviceName := ""
	requestCall := "return { data: [] };"
	if treeMethod != nil {
		serviceName = "def" + treeMethod.TargetEntityName + "Service"
		requestCall = fmt.Sprintf("const data = await %s.%s({});\n  return { data: transform%sTreeNodes(data.list ?? []) };", serviceName, treeMethod.MethodName, table.EntityName)
		if treeMethod.TargetEntityName != table.EntityName {
			importLine := fmt.Sprintf(`import { %s } from "@/api/admin/%s";`, serviceName, stringcase.ToSnakeCase(treeMethod.TargetEntityName))
			// 左树与表单字段可能依赖同一个外部服务，避免重复导入同名服务实例。
			if !strings.Contains(content, importLine) {
				apiImport := fmt.Sprintf(`import { def%sService } from "%s";`, table.EntityName, frontendAPIImport)
				content = strings.Replace(content, apiImport, apiImport+"\n"+importLine, 1)
			}
		}
	}

	refMarker := fmt.Sprintf("const formDialogRef = ref<InstanceType<typeof FormDialog>>();")
	state := fmt.Sprintf(`%s

type %sTreeOption = {
  value: number;
  label: string;
  children?: %sTreeOption[];
};

type %sFilterNode = {
  id: string;
  name: string;
  children?: %sFilterNode[];
};

const initParam = reactive({
  %s: undefined as number | undefined
});
const treeFilterValue = ref("");`, refMarker, table.EntityName, table.EntityName, table.EntityName, table.EntityName, filterColumn)
	content = strings.Replace(content, refMarker, state, 1)

	requestComment := fmt.Sprintf("/**\n * 请求%s列表", table.BusinessName)
	helper := fmt.Sprintf(`/** 转换左侧树筛选节点。 */
function transform%sTreeNodes(options: %sTreeOption[] = []): %sFilterNode[] {
  return options.map(option => ({
    id: String(option.value),
    name: option.label,
    children: transform%sTreeNodes(option.children ?? [])
  }));
}

/** 请求左侧树筛选数据。 */
async function request%sTreeFilter() {
  %s
}

/** 切换左侧树筛选条件。 */
function changeTreeFilter(value: string) {
  treeFilterValue.value = value ?? "";
  initParam.%s = value ? Number(value) : undefined;
}

`, table.EntityName, table.EntityName, table.EntityName, table.EntityName, table.EntityName, requestCall, filterColumn)
	content = strings.Replace(content, requestComment, helper+requestComment, 1)
	openDialogMarker := `  resetForm();
  dialog.title = id ?`
	openDialogReplacement := fmt.Sprintf(`  resetForm();
  // 新增时继承当前左树节点，保证记录归入正在查看的分组。
  if (!id && initParam.%s !== undefined) {
    formData.%s = initParam.%s;
  }
  dialog.title = id ?`, filterColumn, filterColumn, filterColumn)
	return strings.Replace(content, openDialogMarker, openDialogReplacement, 1)
}

// defaultProtoPath 获取默认 Proto 文件路径。
func (c *renderer) defaultProtoPath(table *Table) string {
	if table.APIPath != "" {
		return table.APIPath
	}
	return "backend/api/protos/admin/v1/" + stringcase.ToSnakeCase(table.EntityName) + ".proto"
}

// frontendResourcePath 获取前端接口资源路径。
func (c *renderer) frontendResourcePath(table *Table) string {
	resourcePath := strings.Trim(table.ModulePath, "/")
	snakeEntity := stringcase.ToSnakeCase(table.EntityName)
	if resourcePath == "" {
		return snakeEntity
	}
	entityResourcePath := resourcePathByEntity(table.EntityName)
	if resourcePath == entityResourcePath {
		return resourcePath
	}
	modulePrefix := strings.ReplaceAll(resourcePath, "/", "_") + "_"
	if strings.HasPrefix(snakeEntity, modulePrefix) {
		snakeEntity = strings.TrimPrefix(snakeEntity, modulePrefix)
	}
	return resourcePath + "/" + snakeEntity
}

// renderFrontendColumns 渲染完整表格列配置。
func (c *renderer) renderFrontendColumns(table *Table, columns []*CodeGenColumn, methods []*Proto) string {
	list := make([]string, 0, len(columns))
	statusColumnCount := len(statusColumns(columns))
	for _, column := range columns {
		if (!generatedListIncludesColumn(column) && !generatedQueryIncludesColumn(column)) || column.IsPrimary == 1 || column.ColumnName == "deleted_at" {
			continue
		}
		list = append(list, renderFrontendColumn(table.EntityName, column, findStatusMethodForColumn(column, methods), statusColumnCount))
	}
	return strings.Join(list, ",\n") + ","
}

// renderFrontendFormDefaults 渲染表单默认值。
func (c *renderer) renderFrontendFormDefaults(columns []*CodeGenColumn) string {
	lines := make([]string, 0, len(columns))
	for _, column := range columns {
		if !generatedFormIncludesColumn(column) && column.IsPrimary != 1 {
			continue
		}
		lines = append(lines, fmt.Sprintf("  %s: %s", column.ColumnName, frontendDefaultValue(column)))
	}
	return strings.Join(lines, ",\n")
}

// renderFrontendResetForm 渲染表单重置语句。
func (c *renderer) renderFrontendResetForm(columns []*CodeGenColumn) string {
	lines := make([]string, 0, len(columns))
	for _, column := range columns {
		if !generatedFormIncludesColumn(column) && column.IsPrimary != 1 {
			continue
		}
		lines = append(lines, fmt.Sprintf("  formData.%s = %s;", column.ColumnName, frontendDefaultValue(column)))
	}
	return strings.Join(lines, "\n")
}

// renderFrontendRules 渲染表单校验规则。
func (c *renderer) renderFrontendRules(columns []*CodeGenColumn) string {
	lines := make([]string, 0)
	for _, column := range columns {
		if !generatedFormIncludesColumn(column) || column.IsRequired != 1 {
			continue
		}
		trigger := "blur"
		if column.IsStatusField == 1 && column.StatusForm == 1 || column.FormComponent == "switch" || isSelectComponent(column.FormComponent) {
			trigger = "change"
		}
		lines = append(lines, fmt.Sprintf("  %s: [{ required: true, message: \"%s不能为空\", trigger: %q }]", column.ColumnName, DefaultString(column.ColumnComment, column.ColumnName), trigger))
	}
	return strings.Join(lines, ",\n")
}

// renderFrontendStatusOptions 渲染状态选项。
func (c *renderer) renderFrontendStatusOptions(columns []*CodeGenColumn) string {
	statusColumnList := statusColumns(columns)
	var builder strings.Builder
	for _, column := range statusColumnList {
		if !statusNeedsFrontendOptions(column) {
			continue
		}
		builder.WriteString(fmt.Sprintf(`const %s: ProFormOption[] = [
  { label: "启用", value: %s },
  { label: "禁用", value: %s }
];
`, statusOptionsVariable(column, len(statusColumnList)), statusValueExpression(column, column.StatusEnabledValue, "1"), statusValueExpression(column, column.StatusDisabledValue, "2")))
	}
	return builder.String()
}

// renderFrontendFormFields 渲染 ProForm 字段配置。
func (c *renderer) renderFrontendFormFields(columns []*CodeGenColumn) string {
	fields := make([]string, 0, len(columns))
	for _, column := range columns {
		if !generatedFormIncludesColumn(column) || column.IsPrimary == 1 {
			continue
		}
		fields = append(fields, c.renderFrontendFormField(column))
	}
	return strings.Join(fields, ",\n")
}

// renderFrontendFormField 渲染单个 ProForm 字段。
func (c *renderer) renderFrontendFormField(column *CodeGenColumn) string {
	component := DefaultString(column.FormComponent, "input")
	label := DefaultString(column.ColumnComment, column.ColumnName)
	if component == "switch" {
		// 开关提交值由表单范围的独立配置决定。
		return fmt.Sprintf(`  { prop: "%s", label: "%s", component: "switch", props: { activeValue: %s, inactiveValue: %s } }`, column.ColumnName, label, statusValueExpression(column, column.FormOption.ActiveValue, "1"), statusValueExpression(column, column.FormOption.InactiveValue, "2"))
	}
	option := column.FormOption
	if option.Kind != "" && isSelectComponent(component) {
		props := `props: { placeholder: "请选择", filterable: true, style: { width: "100%" } }`
		// 多选树形选择使用复选框，并允许选择任意层级节点。
		if isFormTreeMultiple(column) {
			props = `props: { multiple: true, showCheckbox: true, checkStrictly: true, nodeKey: "value", placeholder: "请选择", filterable: true, style: { width: "100%" } }`
		}
		switch option.SourceType {
		case OptionSourceDict:
			if option.SourceValue != "" {
				if component == "radio-group" {
					return fmt.Sprintf(`  { prop: "%s", label: "%s", component: "dict", props: { code: %q, codeType: %q, type: "radio" } }`, column.ColumnName, label, option.SourceValue, frontendDictValueType(column))
				}
				if component == "checkbox-group" {
					return fmt.Sprintf(`  { prop: "%s", label: "%s", component: "dict", props: { code: %q, codeType: %q, type: "checkbox" } }`, column.ColumnName, label, option.SourceValue, frontendDictValueType(column))
				}
				return fmt.Sprintf(`  { prop: "%s", label: "%s", component: "dict", props: { code: %q, codeType: %q } }`, column.ColumnName, label, option.SourceValue, frontendDictValueType(column))
			}
		case OptionSourceStatic:
			return fmt.Sprintf(`  { prop: "%s", label: "%s", component: "%s", options: %sOptions, %s }`, column.ColumnName, label, component, frontendOptionVar(column, "form"), props)
		case OptionSourceTable:
			return fmt.Sprintf(`  { prop: "%s", label: "%s", component: "%s", options: %sOptions.value, %s }`, column.ColumnName, label, component, frontendOptionVar(column, "form"), props)
		}
	}
	if component == "input-number" {
		return fmt.Sprintf(`  { prop: "%s", label: "%s", component: "input-number", props: { min: 0, precision: %d, controlsPosition: "right", style: { width: "100%%" } } }`, column.ColumnName, label, column.DbScale)
	}
	if component == "date-picker" {
		dbType := strings.ToLower(DefaultString(column.ColumnType, column.DbType))
		if strings.Contains(dbType, "datetime") || strings.Contains(dbType, "timestamp") {
			return fmt.Sprintf(`  { prop: "%s", label: "%s", component: "date-picker", props: { type: "datetime", valueFormat: "YYYY-MM-DD HH:mm:ss", placeholder: "请选择%s", style: { width: "100%%" } } }`, column.ColumnName, label, label)
		}
		return fmt.Sprintf(`  { prop: "%s", label: "%s", component: "date-picker", props: { type: "date", valueFormat: "YYYY-MM-DD", placeholder: "请选择%s", style: { width: "100%%" } } }`, column.ColumnName, label, label)
	}
	return fmt.Sprintf(`  { prop: "%s", label: "%s", component: "%s", props: { placeholder: "请输入%s" } }`, column.ColumnName, label, component, label)
}

// renderFrontendOptionState 渲染静态选项、数据表选项状态与加载方法。
func (c *renderer) renderFrontendOptionState(columns []*CodeGenColumn, methods []*Proto) string {
	var builder strings.Builder
	hasTableSource := false
	for _, column := range columns {
		for _, scope := range frontendOptionScopes(column) {
			if scope.option.SourceType != OptionSourceStatic && scope.option.SourceType != OptionSourceTable {
				continue
			}
			switch scope.option.SourceType {
			case OptionSourceStatic:
				builder.WriteString(fmt.Sprintf("const %sOptions: ProFormOption[] = %s;\n", frontendOptionVar(column, scope.name), renderFrontendStaticOptions(scope.option)))
			case OptionSourceTable:
				hasTableSource = true
				builder.WriteString(fmt.Sprintf("const %sOptions = ref<ProFormOption[]>([]);\n", frontendOptionVar(column, scope.name)))
			}
		}
	}
	if builder.Len() == 0 {
		return ""
	}
	if !hasTableSource {
		return builder.String()
	}
	builder.WriteString("\n/** 加载表单选择项。 */\nasync function loadFormOptions() {\n")
	for _, column := range columns {
		for _, scope := range frontendOptionScopes(column) {
			if scope.option.SourceType != OptionSourceTable {
				continue
			}
			method := findOptionMethodForConfig(scope.option, methods)
			if method == nil {
				continue
			}
			variable := frontendOptionVar(column, scope.name)
			serviceName := "def" + method.TargetEntityName + "Service"
			builder.WriteString(fmt.Sprintf("  const %sResponse = await %s.%s({});\n", variable, serviceName, method.MethodName))
			builder.WriteString(fmt.Sprintf("  %sOptions.value = (%sResponse.list ?? []) as ProFormOption[];\n", variable, variable))
		}
	}
	builder.WriteString("}\n\nvoid loadFormOptions();\n")
	return builder.String()
}

// renderFrontendOptionImports 渲染字段选项依赖的外部 API import。
func (c *renderer) renderFrontendOptionImports(table *Table, columns []*CodeGenColumn, methods []*Proto) string {
	imports := make([]string, 0)
	seen := make(map[string]struct{})
	for _, column := range columns {
		for _, scope := range frontendOptionScopes(column) {
			if scope.option.SourceType != OptionSourceTable {
				continue
			}
			method := findOptionMethodForConfig(scope.option, methods)
			if method == nil || method.TargetEntityName == table.EntityName {
				continue
			}
			if _, ok := seen[method.TargetEntityName]; ok {
				continue
			}
			seen[method.TargetEntityName] = struct{}{}
			imports = append(imports, fmt.Sprintf("import { def%sService } from \"@/api/admin/%s\";", method.TargetEntityName, stringcase.ToSnakeCase(method.TargetEntityName)))
		}
	}
	return strings.Join(imports, "\n")
}

// renderFrontendEnumImports 渲染状态枚举导入。
func (c *renderer) renderFrontendEnumImports(columns []*CodeGenColumn) string {
	enumNames := make([]string, 0)
	seen := make(map[string]struct{})
	for _, column := range statusColumns(columns) {
		if !statusUsesEnumImport(column) {
			continue
		}
		if _, ok := seen[column.StatusEnumName]; ok {
			continue
		}
		seen[column.StatusEnumName] = struct{}{}
		enumNames = append(enumNames, column.StatusEnumName)
	}
	if len(enumNames) == 0 {
		return ""
	}
	return fmt.Sprintf("import { %s } from \"@/rpc/common/v1/enum\";", strings.Join(enumNames, ", "))
}

// renderFrontendLoadOptionsCall 渲染打开弹窗时加载选项调用。
func (c *renderer) renderFrontendLoadOptionsCall(columns []*CodeGenColumn, _ []*Proto) string {
	for _, column := range columns {
		for _, scope := range frontendOptionScopes(column) {
			if scope.option.SourceType == OptionSourceTable {
				return "  await loadFormOptions();\n"
			}
		}
	}
	return ""
}

// renderFrontendStatusHandlers 渲染全部状态切换处理方法。
func (c *renderer) renderFrontendStatusHandlers(table *Table, columns []*CodeGenColumn, methods []*Proto) string {
	statusColumnList := statusColumns(columns)
	var builder strings.Builder
	for _, column := range statusColumnList {
		method := findStatusMethodForColumn(column, methods)
		if column.StatusSwitch != 1 || method == nil {
			continue
		}
		builder.WriteString(c.renderFrontendStatusHandler(table, column, method, statusHandlerName(column, len(statusColumnList))))
	}
	return builder.String()
}

// renderFrontendStatusHandler 渲染单个状态切换处理方法。
func (c *renderer) renderFrontendStatusHandler(table *Table, column *CodeGenColumn, statusMethod *Proto, handlerName string) string {
	if column == nil || column.StatusSwitch != 1 || statusMethod == nil {
		return ""
	}
	enabled := statusValueExpression(column, column.StatusEnabledValue, "1")
	disabled := statusValueExpression(column, column.StatusDisabledValue, "2")
	return fmt.Sprintf(`
/**
 * 切换%s状态前先确认并调用后端状态接口。
 */
async function %s(row: %s) {
  const currentStatus = (row as unknown as Record<string, unknown>)["%s"];
  const nextStatus = currentStatus === %s ? %s : %s;
  const text = nextStatus === %s ? "启用" : "禁用";
  try {
    await ElMessageBox.confirm("是否确定" + text + "%s？", "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await def%sService.%s({ id: row.id, status: nextStatus as %sRequest["status"] });
    ElMessage.success(text + "成功");
    refreshTable();
    return true;
  } catch {
    return false;
  }
}
`, DefaultString(column.ColumnComment, column.ColumnName), handlerName, table.EntityName, column.ColumnName, enabled, disabled, enabled, enabled, table.BusinessName, table.EntityName, statusMethod.MethodName, statusMethod.MethodName)
}

// findTSFunctionEndIndex 查找 TypeScript 函数体结束位置。
func findTSFunctionEndIndex(content string, functionStart int) int {
	if functionStart < 0 {
		return -1
	}
	openIndex := strings.Index(content[functionStart:], "{")
	if openIndex < 0 {
		return -1
	}
	openIndex += functionStart
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

// validateCodeGenOutputPathLayout 校验输出路径仍位于生成代码可编译和可加载的目录中。
func validateCodeGenOutputPathLayout(paths *adminv1.CodeGenOutputPaths) error {
	if !strings.HasPrefix(paths.GetProtoFilePath(), "backend/api/protos/") || filepath.Ext(paths.GetProtoFilePath()) != ".proto" {
		return errorsx.InvalidArgument("Proto文件必须位于backend/api/protos目录且使用.proto扩展名")
	}
	if filepath.Dir(paths.GetBackendBizFilePath()) != "backend/service/admin/biz" || filepath.Ext(paths.GetBackendBizFilePath()) != ".go" {
		return errorsx.InvalidArgument("后端Biz文件必须位于backend/service/admin/biz目录且使用.go扩展名")
	}
	if filepath.Dir(paths.GetBackendServiceFilePath()) != "backend/service/admin" || filepath.Ext(paths.GetBackendServiceFilePath()) != ".go" {
		return errorsx.InvalidArgument("后端Service文件必须位于backend/service/admin目录且使用.go扩展名")
	}
	if !strings.HasPrefix(paths.GetFrontendApiFilePath(), "frontend/admin/src/") || filepath.Ext(paths.GetFrontendApiFilePath()) != ".ts" {
		return errorsx.InvalidArgument("前端API文件必须位于frontend/admin/src目录且使用.ts扩展名")
	}
	if !strings.HasPrefix(paths.GetFrontendPageFilePath(), "frontend/admin/src/views/") || !strings.HasSuffix(paths.GetFrontendPageFilePath(), "/index.vue") {
		return errorsx.InvalidArgument("前端页面文件必须位于frontend/admin/src/views目录且文件名为index.vue")
	}
	if filepath.Ext(paths.GetSqlFilePath()) != ".sql" {
		return errorsx.InvalidArgument("SQL文件必须使用.sql扩展名")
	}
	return nil
}

// frontendRPCImportPath 根据 Proto 文件路径推导前端生成类型导入路径。
func frontendRPCImportPath(protoPath string) string {
	relativePath := strings.TrimPrefix(protoPath, "backend/api/protos/")
	return "@/rpc/" + strings.TrimSuffix(relativePath, filepath.Ext(relativePath))
}

// repoRoot 返回仓库根目录。
func repoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	// 后端命令通常在 backend 目录执行，这里回到仓库根目录。
	if filepath.Base(wd) == "backend" {
		return filepath.Dir(wd)
	}
	return wd
}

// isManagedAuditColumn 判断字段是否由基础设施自动维护、默认无需人工配置。
func isManagedAuditColumn(columnName string) bool {
	return columnName == "created_by" || columnName == "updated_by" || columnName == "created_at" || columnName == "updated_at" || columnName == "deleted_at"
}

// pluralize 生成项目常用复数方法名。
func pluralize(value string) string {
	if strings.HasSuffix(value, "s") {
		return value + "es"
	}
	if strings.HasSuffix(value, "y") {
		return strings.TrimSuffix(value, "y") + "ies"
	}
	return value + "s"
}

// statusAPIColumns 返回启用状态接口的字段。
func statusAPIColumns(columns []*CodeGenColumn) []*CodeGenColumn {
	list := make([]*CodeGenColumn, 0)
	for _, column := range columns {
		if column.IsStatusField == 1 && column.StatusGenerateAPI == 1 {
			list = append(list, column)
		}
	}
	return list
}

// findStatusColumn 根据字段名查找状态字段。
func findStatusColumn(columns []*CodeGenColumn, columnName string) *CodeGenColumn {
	if columnName != "" {
		for _, column := range columns {
			if column.ColumnName == columnName && column.IsStatusField == 1 {
				return column
			}
		}
	}
	statusColumnList := statusColumns(columns)
	if len(statusColumnList) == 0 {
		return nil
	}
	return statusColumnList[0]
}

// statusProtoType 返回状态字段在 Proto 中的类型。
func statusProtoType(column *CodeGenColumn) string {
	if column == nil {
		return "int32"
	}
	return DefaultString(column.ProtoType, InferProtoType(column.DbType))
}

// statusMethodNameForColumn 根据状态字段数量生成兼容且唯一的方法名。
func statusMethodNameForColumn(table *Table, column *CodeGenColumn, statusColumnCount int) string {
	if statusColumnCount <= 1 {
		return "Set" + table.EntityName + "Status"
	}
	fieldName := stringcase.ToPascalCase(column.ColumnName)
	if strings.HasSuffix(fieldName, "Status") {
		return "Set" + table.EntityName + fieldName
	}
	return "Set" + table.EntityName + fieldName + "Status"
}

// statusResourcePath 返回状态接口对应的 HTTP 子资源路径。
func statusResourcePath(table *Table, method *Proto) string {
	if method.MethodName == "Set"+table.EntityName+"Status" || method.ColumnName == "" {
		return "status"
	}
	return "status/" + strings.ReplaceAll(method.ColumnName, "_", "-")
}

// renderFrontendDateImport 仅在列表包含日期展示列时引入前端格式化依赖。
func renderFrontendDateImport(columns []*CodeGenColumn) string {
	for _, column := range columns {
		if generatedListIncludesColumn(column) && column.IsPrimary != 1 && column.ColumnName != "deleted_at" && column.ListComponent == "date" &&
			(column.IsStatusField != 1 || column.StatusTableColumn != 1 && column.StatusSearch != 1) {
			return `import dayjs from "dayjs";`
		}
	}
	return ""
}

// renderFrontendColumn 渲染前端表格列配置。
func renderFrontendColumn(entityName string, column *CodeGenColumn, statusMethod *Proto, statusColumnCount int) string {
	if column.IsStatusField == 1 && (column.StatusTableColumn == 1 || column.StatusSearch == 1) {
		search := ""
		if column.StatusSearch == 1 {
			search = ", search: { el: \"select\" }"
		}
		visibility := ""
		// 仅用于查询的状态字段不展示为表格列，也不进入列设置。
		if column.StatusTableColumn != 1 {
			visibility = ", isShow: false, isSetting: false"
		}
		dictConfig := ""
		if statusUsesDictionary(column) {
			dictConfig = fmt.Sprintf(", dictCode: %q, dictValueType: %q", column.StatusDictCode, frontendDictValueType(column))
		}
		if column.StatusSwitch == 1 && statusMethod != nil {
			return fmt.Sprintf(`  {
    prop: "%s",
    label: "%s",
    width: 100%s%s%s,
    cellType: "status",
    statusProps: {
      activeValue: %s,
      inactiveValue: %s,
      activeText: "启用",
      inactiveText: "禁用"%s,
      beforeChange: scope => %s(scope.row as %s)
    }
  }`, column.ColumnName, column.ColumnComment, dictConfig, search, visibility, statusValueExpression(column, column.StatusEnabledValue, "1"), statusValueExpression(column, column.StatusDisabledValue, "2"), "", statusHandlerName(column, statusColumnCount), entityName)
		}
		if statusUsesDictionary(column) {
			return fmt.Sprintf(`  { prop: "%s", label: "%s"%s%s%s }`, column.ColumnName, column.ColumnComment, dictConfig, search, visibility)
		}
		return fmt.Sprintf(`  { prop: "%s", label: "%s", enum: %s%s%s }`, column.ColumnName, column.ColumnComment, statusOptionsVariable(column, statusColumnCount), search, visibility)
	}
	optionConfig := frontendColumnOptionConfig(column)
	searchConfig := frontendSearchConfig(column)
	// 仅作为普通查询条件的字段保留搜索配置，但不展示为列表列。
	if !generatedListIncludesColumn(column) && generatedQueryIncludesColumn(column) {
		return fmt.Sprintf(`  { prop: "%s", label: "%s"%s%s, isShow: false, isSetting: false }`, column.ColumnName, column.ColumnComment, optionConfig, searchConfig)
	}
	if column.ListComponent == "image" {
		return fmt.Sprintf(`  { prop: "%s", label: "%s", cellType: "image"%s%s }`, column.ColumnName, column.ColumnComment, optionConfig, searchConfig)
	}
	if column.ListComponent == "date" {
		dbType := strings.ToLower(DefaultString(column.ColumnType, column.DbType))
		format := "YYYY-MM-DD"
		if strings.Contains(dbType, "datetime") || strings.Contains(dbType, "timestamp") {
			format = "YYYY-MM-DD HH:mm:ss"
		} else if strings.Contains(dbType, "time") {
			format = "HH:mm:ss"
		}
		return fmt.Sprintf(`  { prop: "%s", label: "%s", render: scope => scope.row.%s ? dayjs(scope.row.%s).format(%q) : "--"%s%s }`, column.ColumnName, column.ColumnComment, column.ColumnName, column.ColumnName, format, optionConfig, searchConfig)
	}
	return fmt.Sprintf(`  { prop: "%s", label: "%s"%s%s }`, column.ColumnName, column.ColumnComment, optionConfig, searchConfig)
}

// frontendColumnOptionConfig 渲染列表或查询选择组件的数据源配置。
func frontendColumnOptionConfig(column *CodeGenColumn) string {
	if !generatedListIncludesColumn(column) || column.IsStatusField == 1 || !isSelectComponent(column.ListComponent) {
		return ""
	}
	option := column.ListOption
	switch option.SourceType {
	case OptionSourceDict:
		if option.SourceValue != "" {
			return fmt.Sprintf(", dictCode: %q, dictValueType: %q", option.SourceValue, frontendDictValueType(column))
		}
	case OptionSourceStatic, OptionSourceTable:
		if option.Kind != "" {
			return ", enum: " + frontendOptionVar(column, "list") + "Options"
		}
	}
	return ""
}

// frontendSearchConfig 渲染普通查询字段的搜索配置。
func frontendSearchConfig(column *CodeGenColumn) string {
	if !generatedQueryIncludesColumn(column) {
		return ""
	}
	component := DefaultString(column.QueryComponent, "input")
	if component == "date-picker" && effectiveQueryOperator(column) == "between" {
		return `, search: { el: "date-picker", props: { type: "daterange", editable: false, valueFormat: "YYYY-MM-DD" } }`
	}
	if component == "input-number" {
		return `, search: { el: "input-number" }`
	}
	if component == "select" || component == "tree-select" {
		option := column.QueryOption
		switch option.SourceType {
		case OptionSourceDict:
			return fmt.Sprintf(`, search: { el: "%s", dictCode: %q, dictValueType: %q }`, component, option.SourceValue, frontendDictValueType(column))
		case OptionSourceStatic, OptionSourceTable:
			if option.Kind != "" {
				return fmt.Sprintf(`, search: { el: "%s", enum: %sOptions }`, component, frontendOptionVar(column, "query"))
			}
		}
		return fmt.Sprintf(`, search: { el: "%s" }`, component)
	}
	if component == "date-picker" {
		return `, search: { el: "date-picker" }`
	}
	return `, search: { el: "input" }`
}

// statusOptionsVariable 返回状态字段对应的前端选项变量名。
func statusOptionsVariable(column *CodeGenColumn, statusColumnCount int) string {
	if statusColumnCount <= 1 {
		return "statusOptions"
	}
	return stringcase.ToCamelCase(column.ColumnName) + "StatusOptions"
}

// renderFrontendStaticOptions 将静态选项 JSON 渲染为安全的 TypeScript 字面量。
func renderFrontendStaticOptions(option CodeGenColumnOptionConfig) string {
	var options []CodeGenStaticOption
	err := json.Unmarshal([]byte(option.SourceValue), &options)
	if err != nil {
		return "[]"
	}
	var content []byte
	content, err = json.Marshal(options)
	if err != nil {
		return "[]"
	}
	return string(content)
}

// renderFrontendStatusTypeImports 渲染状态切换处理函数依赖的请求类型。
func renderFrontendStatusTypeImports(columns []*CodeGenColumn, methods []*Proto) string {
	typeNames := make([]string, 0)
	for _, column := range statusColumns(columns) {
		if column.StatusSwitch != 1 {
			continue
		}
		method := findStatusMethodForColumn(column, methods)
		if method != nil {
			typeNames = append(typeNames, method.MethodName+"Request")
		}
	}
	if len(typeNames) == 0 {
		return ""
	}
	return ", " + strings.Join(typeNames, ", ")
}

// statusColumns 返回全部状态字段。
func statusColumns(columns []*CodeGenColumn) []*CodeGenColumn {
	list := make([]*CodeGenColumn, 0)
	for _, column := range columns {
		if column.IsStatusField == 1 {
			list = append(list, column)
		}
	}
	return list
}

// findStatusMethodForColumn 查找状态字段对应的接口方法。
func findStatusMethodForColumn(column *CodeGenColumn, methods []*Proto) *Proto {
	var onlyStatusMethod *Proto
	statusMethodCount := 0
	for _, method := range methods {
		if method.APIKind != APIKindStatus {
			continue
		}
		statusMethodCount++
		onlyStatusMethod = method
		if method.ColumnName == column.ColumnName {
			return method
		}
	}
	if statusMethodCount == 1 {
		return onlyStatusMethod
	}
	return nil
}

// statusHandlerName 返回状态字段对应的前端切换处理函数名。
func statusHandlerName(column *CodeGenColumn, statusColumnCount int) string {
	if statusColumnCount <= 1 {
		return "handleBeforeSetStatus"
	}
	return "handleBeforeSet" + stringcase.ToPascalCase(column.ColumnName) + "Status"
}

// statusNeedsFrontendOptions 判断状态字段是否需要生成前端静态选项。
func statusNeedsFrontendOptions(column *CodeGenColumn) bool {
	return column != nil && column.IsStatusField == 1 && !statusUsesDictionary(column) && (column.StatusTableColumn == 1 || column.StatusSearch == 1 || column.StatusForm == 1)
}

// statusUsesDictionary 判断状态选项是否来自字典。
func statusUsesDictionary(column *CodeGenColumn) bool {
	return column != nil && column.IsStatusField == 1 && column.StatusDataType == "dict"
}

// frontendOptionScope 表示字段在单个前端作用域中使用的选项配置。
type frontendOptionScope struct {
	name   string
	option CodeGenColumnOptionConfig
}

// frontendOptionScopes 返回前端实际消费的查询、列表和表单选项配置。
func frontendOptionScopes(column *CodeGenColumn) []frontendOptionScope {
	scopes := make([]frontendOptionScope, 0, 3)
	if frontendQueryUsesOptions(column) && column.QueryOption.Kind != "" {
		scopes = append(scopes, frontendOptionScope{name: "query", option: column.QueryOption})
	}
	if generatedListIncludesColumn(column) && column.IsStatusField != 1 && isSelectComponent(column.ListComponent) && column.ListOption.Kind != "" {
		scopes = append(scopes, frontendOptionScope{name: "list", option: column.ListOption})
	}
	if generatedFormIncludesColumn(column) && isSelectComponent(column.FormComponent) && column.FormOption.Kind != "" {
		scopes = append(scopes, frontendOptionScope{name: "form", option: column.FormOption})
	}
	return scopes
}

// frontendQueryUsesOptions 判断普通查询字段是否使用选择型组件。
func frontendQueryUsesOptions(column *CodeGenColumn) bool {
	if column.IsStatusField == 1 || !generatedQueryIncludesColumn(column) {
		return false
	}
	return column.QueryComponent == "select" || column.QueryComponent == "tree-select"
}

// generatedRequestIncludesColumn 判断请求是否需要包含显式查询字段或左树关联字段。
func generatedRequestIncludesColumn(table *Table, column *CodeGenColumn) bool {
	if generatedQueryIncludesColumn(column) {
		return true
	}
	leftTreeConfig := LeftTreeConfigFromTable(table)
	return leftTreeConfig.Enabled && leftTreeConfig.FilterColumn != "" && column.ColumnName == leftTreeConfig.FilterColumn
}

// codeGenRequestColumns 为后端查询模板补入左树隐式关联字段。
func codeGenRequestColumns(table *Table, columns []*CodeGenColumn) []*CodeGenColumn {
	leftTreeConfig := LeftTreeConfigFromTable(table)
	if !leftTreeConfig.Enabled || leftTreeConfig.FilterColumn == "" {
		return columns
	}
	list := make([]*CodeGenColumn, 0, len(columns))
	for _, column := range columns {
		if column.ColumnName != leftTreeConfig.FilterColumn || generatedQueryIncludesColumn(column) {
			list = append(list, column)
			continue
		}
		requestColumn := *column
		requestColumn.IsQuery = 1
		requestColumn.QueryOperator = "eq"
		list = append(list, &requestColumn)
	}
	return list
}

// generatedQueryIncludesColumn 判断查询请求是否包含字段。
func generatedQueryIncludesColumn(column *CodeGenColumn) bool {
	return column.IsQuery == 1 || column.IsStatusField == 1 && column.StatusSearch == 1
}

// generatedListIncludesColumn 判断列表响应是否包含字段。
func generatedListIncludesColumn(column *CodeGenColumn) bool {
	return column.IsList == 1 || column.IsStatusField == 1 && column.StatusTableColumn == 1
}

// generatedFormIncludesColumn 判断新增编辑表单是否包含字段。
func generatedFormIncludesColumn(column *CodeGenColumn) bool {
	return column.IsForm == 1 || column.IsStatusField == 1 && column.StatusForm == 1
}

// frontendDictValueType 返回前端字典组件使用的值类型。
func frontendDictValueType(column *CodeGenColumn) string {
	if DefaultString(column.TsType, InferTSType(column.DbType)) == "string" {
		return "string"
	}
	return "number"
}

// frontendDefaultValue 返回前端表单默认值表达式。
func frontendDefaultValue(column *CodeGenColumn) string {
	if column.IsPrimary == 1 {
		return "0"
	}
	if column.IsStatusField == 1 && column.StatusDefaultValue != "" {
		return statusValueExpression(column, column.StatusDefaultValue, "1")
	}
	if isFormTreeMultiple(column) {
		return "[]"
	}
	// 关联、字典等选择型字段未选择时不能传递数值零值。
	if isSelectComponent(DefaultString(column.FormComponent, "input")) {
		return "undefined"
	}
	if column.DefaultValue != "" {
		if DefaultString(column.TsType, "string") == "string" {
			return fmt.Sprintf("%q", column.DefaultValue)
		}
		return column.DefaultValue
	}
	switch DefaultString(column.TsType, InferTSType(column.DbType)) {
	case "number":
		return "0"
	case "boolean":
		return "false"
	default:
		return "\"\""
	}
}

// isFormTreeMultiple 判断字段是否使用 JSON 存储的多选树形选择。
func isFormTreeMultiple(column *CodeGenColumn) bool {
	return column != nil && column.FormMultiple && column.FormComponent == "tree-select" && strings.EqualFold(column.DbType, "json")
}

// statusValueExpression 渲染状态值表达式。
func statusValueExpression(column *CodeGenColumn, value string, fallback string) string {
	if value == "" {
		return fallback
	}
	if statusUsesEnumImport(column) && strings.Contains(value, ".") {
		return value
	}
	if statusUsesEnumImport(column) {
		return column.StatusEnumName + "." + value
	}
	if column != nil && DefaultString(column.TsType, "number") == "string" {
		return fmt.Sprintf("%q", value)
	}
	return value
}

// statusUsesEnumImport 判断状态值是否需要从公共枚举导入。
func statusUsesEnumImport(column *CodeGenColumn) bool {
	return column != nil && column.StatusDataType == "enum" && column.StatusEnumName != ""
}

// frontendOptionVar 返回字段指定作用域的选择项状态变量前缀。
func frontendOptionVar(column *CodeGenColumn, scope string) string {
	return stringcase.ToCamelCase(column.ColumnName) + stringcase.ToPascalCase(scope)
}

// isSelectComponent 判断组件是否属于选择型控件。
func isSelectComponent(component string) bool {
	return component == "select" || component == "tree-select" || component == "radio-group" || component == "checkbox-group" || component == "dict"
}

// findOptionMethodForConfig 查找选项配置对应的接口。
func findOptionMethodForConfig(option CodeGenColumnOptionConfig, methods []*Proto) *Proto {
	targetEntity := stringcase.ToPascalCase(option.SourceValue)
	apiKind := APIKindOption
	if option.Kind == APIKindTree {
		apiKind = APIKindTree
	}
	for _, method := range methods {
		if method.TargetEntityName == targetEntity && method.APIKind == apiKind {
			return method
		}
	}
	return nil
}

// isBoolDBType 判断数据库类型是否表示布尔值。
func isBoolDBType(dbType string) bool {
	lowerType := strings.ToLower(dbType)
	return strings.Contains(lowerType, "bool") || strings.Contains(lowerType, "tinyint(1)")
}

// isNumericDBType 判断数据库类型是否表示数值。
func isNumericDBType(dbType string) bool {
	lowerType := strings.ToLower(dbType)
	return strings.Contains(lowerType, "int") ||
		strings.Contains(lowerType, "decimal") ||
		strings.Contains(lowerType, "float") ||
		strings.Contains(lowerType, "double")
}

// isDateTimeDBType 判断数据库类型是否表示日期时间。
func isDateTimeDBType(dbType string) bool {
	lowerType := strings.ToLower(dbType)
	return strings.Contains(lowerType, "date") || strings.Contains(lowerType, "time")
}

// dedupeProtoChecks 去重 Proto 检查项。
func dedupeProtoChecks(checks []*ProtoCheck) []*ProtoCheck {
	list := make([]*ProtoCheck, 0, len(checks))
	seen := make(map[string]struct{}, len(checks))
	for _, check := range checks {
		key := check.ProtoFilePath + ":" + check.TargetEntityName + ":" + check.MethodName
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		list = append(list, check)
	}
	slices.SortFunc(list, func(a *ProtoCheck, b *ProtoCheck) int {
		if a.ProtoFilePath != b.ProtoFilePath {
			return strings.Compare(a.ProtoFilePath, b.ProtoFilePath)
		}
		if a.TargetEntityName != b.TargetEntityName {
			return strings.Compare(a.TargetEntityName, b.TargetEntityName)
		}
		aPosition := codeGenProtoMethodPosition(a.APIKind, a.TriggerType, a.MethodName)
		bPosition := codeGenProtoMethodPosition(b.APIKind, b.TriggerType, b.MethodName)
		if aPosition != bPosition {
			return aPosition - bPosition
		}
		return strings.Compare(a.MethodName, b.MethodName)
	})
	return list
}

// findSavedProtoMethod 查找已保存的生成选择。
func findSavedProtoMethod(methods []*Proto, check *ProtoCheck) *Proto {
	legacyMethodName := legacyPluralProtoMethodName(check.TargetEntityName, check.MethodName)
	for _, method := range methods {
		if method.ProtoFilePath != check.ProtoFilePath || method.TargetEntityName != check.TargetEntityName {
			continue
		}
		if method.MethodName == check.MethodName || legacyMethodName != "" && method.MethodName == legacyMethodName {
			return method
		}
	}
	return nil
}

// legacyPluralProtoMethodName 返回旧版 Page、Tree、Option 复数契约名。
func legacyPluralProtoMethodName(entity string, methodName string) string {
	for _, prefix := range []string{"Page", "Tree", "Option"} {
		if methodName == prefix+entity {
			return prefix + pluralize(entity)
		}
	}
	return ""
}
