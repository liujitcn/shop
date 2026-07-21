package codegen

import (
	"strings"
	"testing"
)

func TestMergeGeneratedGoReceiverMethodsReplacesGeneratedAndPreservesExtensions(t *testing.T) {
	t.Parallel()

	existing := `package sample

type BaseAPICase struct{}

func NewBaseAPICase() *BaseAPICase {
	return new(BaseAPICase)
}

// PageBaseAPI 保留了旧查询判断。
func (c *BaseAPICase) PageBaseAPI(req *PageBaseApiRequest) string {
	if req.McpEnabled != nil {
		return "old generated"
	}
	return "old generated"
}

func normalizeBaseAPI() string {
	return "helper"
}

// GetBaseAPIDoc 是已有扩展方法。
func (c *BaseAPICase) GetBaseAPIDoc() string {
	return "custom body"
}
`
	generated := `// PageBaseApi 是最新生成方法。
func (c *BaseAPICase) PageBaseApi(req *PageBaseApiRequest) string {
	return "new generated"
}

// GetBaseApi 是最新生成方法。
func (c *BaseAPICase) GetBaseApi() string {
	return "new get"
}`

	merged := mergeGeneratedGoReceiverMethods(existing, generated, "BaseAPICase")
	if strings.Contains(merged, "McpEnabled") || strings.Contains(merged, "old generated") {
		t.Fatalf("旧生成方法没有被替换：\n%s", merged)
	}
	for _, expected := range []string{"new generated", "new get", "custom body", "return \"helper\""} {
		if !strings.Contains(merged, expected) {
			t.Fatalf("合并结果缺少 %q：\n%s", expected, merged)
		}
	}
	pageIndex := strings.Index(merged, "func (c *BaseAPICase) PageBaseApi")
	getIndex := strings.Index(merged, "func (c *BaseAPICase) GetBaseApi")
	customIndex := strings.Index(merged, "func (c *BaseAPICase) GetBaseAPIDoc")
	helperIndex := strings.Index(merged, "func normalizeBaseAPI")
	if pageIndex < 0 || getIndex <= pageIndex || customIndex <= getIndex || helperIndex <= customIndex {
		t.Fatalf("方法顺序不符合生成方法、扩展方法、辅助函数顺序：\n%s", merged)
	}
}

func TestMergeGeneratedProtoFileReplacesGeneratedMessagesAndPreservesExtensions(t *testing.T) {
	t.Parallel()

	existing := `syntax = "proto3";

package system.admin.v1;

import "custom/options.proto";

service BaseApiService {
  option deprecated = true;

  // 旧分页接口
  rpc PageBaseApi(PageBaseApiRequest) returns (PageBaseApiResponse) {}

  // 自定义文档接口
  rpc GetBaseApiDoc(GetBaseApiDocRequest) returns (BaseApiDoc) {}
}

message PageBaseApiRequest {
  optional bool mcp_enabled = 1;
  optional bool agent_enabled = 2;
  int64 page_num = 101;
  int64 page_size = 102;
}

message PageBaseApiResponse {}

message GetBaseApiDocRequest {
  int64 id = 1;
}

message BaseApiDoc {
  string summary = 1;
}
`
	candidate := `syntax = "proto3";

package system.admin.v1;

import "google/api/annotations.proto";

service BaseApiService {
  // 最新分页接口
  rpc PageBaseApi(PageBaseApiRequest) returns (PageBaseApiResponse) {}

  // 最新详情接口
  rpc GetBaseApi(GetBaseApiRequest) returns (BaseApiForm) {}
}

message PageBaseApiRequest {
  optional string tool_name = 1;
  int64 page_num = 101;
  int64 page_size = 102;
}

message PageBaseApiResponse {}

message GetBaseApiRequest {
  int64 id = 1;
}

message BaseApiForm {
  string tool_name = 1;
}
`

	merged := mergeGeneratedProtoFile(existing, candidate)
	if strings.Contains(merged, "mcp_enabled") || strings.Contains(merged, "agent_enabled") {
		t.Fatalf("关闭的查询字段仍保留在生成请求中：\n%s", merged)
	}
	for _, expected := range []string{
		"optional string tool_name = 1;",
		"rpc GetBaseApiDoc(GetBaseApiDocRequest) returns (BaseApiDoc)",
		"message GetBaseApiDocRequest",
		"message BaseApiDoc",
		"import \"custom/options.proto\";",
		"import \"google/api/annotations.proto\";",
		"option deprecated = true;",
	} {
		if !strings.Contains(merged, expected) {
			t.Fatalf("合并结果缺少 %q：\n%s", expected, merged)
		}
	}
	pageIndex := strings.Index(merged, "rpc PageBaseApi")
	getIndex := strings.Index(merged, "rpc GetBaseApi(")
	customIndex := strings.Index(merged, "rpc GetBaseApiDoc")
	if pageIndex < 0 || getIndex <= pageIndex || customIndex <= getIndex {
		t.Fatalf("Proto 方法顺序不符合生成方法在前、扩展方法在后：\n%s", merged)
	}
}

func TestMergeGeneratedTSClassMethodsReplacesGeneratedAndPreservesExtensions(t *testing.T) {
	t.Parallel()

	existing := `export class BaseApiServiceImpl {
  /** 旧分页实现 */
  PageBaseApi(request: PageBaseApiRequest) {
    return request.mcpEnabled;
  }

  /** 自定义文档方法 */
  GetBaseApiDoc(request: GetBaseApiDocRequest) {
    return request.id;
  }
}
`
	candidate := `export class BaseApiServiceImpl {
  /** 最新分页实现 */
  PageBaseApi(request: PageBaseApiRequest) {
    return request.toolName;
  }

  /** 最新详情实现 */
  GetBaseApi(request: GetBaseApiRequest) {
    return request.id;
  }
}
`

	merged := mergeGeneratedTSClassMethods(existing, candidate, "BaseApiServiceImpl")
	if strings.Contains(merged, "mcpEnabled") || strings.Contains(merged, "旧分页实现") {
		t.Fatalf("旧生成方法没有被替换：\n%s", merged)
	}
	for _, expected := range []string{"request.toolName", "最新详情实现", "自定义文档方法", "return request.id;"} {
		if !strings.Contains(merged, expected) {
			t.Fatalf("合并结果缺少 %q：\n%s", expected, merged)
		}
	}
	pageIndex := strings.Index(merged, "PageBaseApi(")
	getIndex := strings.Index(merged, "GetBaseApi(")
	customIndex := strings.Index(merged, "GetBaseApiDoc(")
	if pageIndex < 0 || getIndex <= pageIndex || customIndex <= getIndex {
		t.Fatalf("前端方法顺序不符合生成方法在前、扩展方法在后：\n%s", merged)
	}
}

func TestAppendMainBizMethodsKeepsQueryConditionInSyncWithProto(t *testing.T) {
	t.Parallel()

	renderer := &renderer{}
	table := &Table{
		TableName_:   "code_gen_base_api",
		TableComment: "基础接口",
		BusinessName: "基础接口",
		EntityName:   "BaseApi",
	}
	protoPath := renderer.defaultProtoPath(table)
	methods := []*Proto{
		{TriggerType: TriggerCRUD, APIKind: APIKindList, TargetEntityName: table.EntityName, MethodName: "PageBaseApi", ProtoFilePath: protoPath, GenerateWhenMissing: 1},
		{TriggerType: TriggerEntityOption, APIKind: APIKindOption, TargetEntityName: table.EntityName, MethodName: "OptionBaseApi", ProtoFilePath: protoPath, GenerateWhenMissing: 1},
		{TriggerType: TriggerCRUD, APIKind: APIKindCRUD, TargetEntityName: table.EntityName, MethodName: "GetBaseApi", ProtoFilePath: protoPath, GenerateWhenMissing: 1},
		{TriggerType: TriggerCRUD, APIKind: APIKindCRUD, TargetEntityName: table.EntityName, MethodName: "CreateBaseApi", ProtoFilePath: protoPath, GenerateWhenMissing: 1},
		{TriggerType: TriggerCRUD, APIKind: APIKindCRUD, TargetEntityName: table.EntityName, MethodName: "UpdateBaseApi", ProtoFilePath: protoPath, GenerateWhenMissing: 1},
		{TriggerType: TriggerCRUD, APIKind: APIKindCRUD, TargetEntityName: table.EntityName, MethodName: "DeleteBaseApi", ProtoFilePath: protoPath, GenerateWhenMissing: 1},
	}
	idColumn := &CodeGenColumn{
		ColumnName:    "id",
		ColumnComment: "ID",
		DbType:        "bigint",
		IsPrimary:     1,
		IsList:        1,
	}
	queryColumn := &CodeGenColumn{
		ColumnName:    "mcp_enabled",
		ColumnComment: "是否暴露为MCP工具",
		DbType:        "tinyint",
		IsQuery:       1,
		QueryOperator: "eq",
	}
	existing := renderer.renderBackendBizFile(table, []*CodeGenColumn{idColumn, queryColumn}, methods)
	if !strings.Contains(existing, "req.McpEnabled != nil") {
		t.Fatalf("测试前置内容未生成旧查询判断：\n%s", existing)
	}

	closedQueryColumn := *queryColumn
	closedQueryColumn.IsQuery = 0
	columns := []*CodeGenColumn{idColumn, &closedQueryColumn}
	merged := renderer.appendMainBizMethods(existing, table, columns, methods)
	protoContent := renderer.renderProtoFile(table, columns, methods)
	if strings.Contains(merged, "req.McpEnabled") {
		t.Fatalf("Biz 仍保留已关闭字段的查询判断：\n%s", merged)
	}
	if strings.Contains(protoContent, "mcp_enabled") {
		t.Fatalf("Proto 仍生成已关闭字段的请求参数：\n%s", protoContent)
	}
	if !strings.Contains(merged, "func (c *BaseApiCase) PageBaseApi") || !strings.Contains(protoContent, "message PageBaseApiRequest") {
		t.Fatalf("关闭查询字段不应删除分页方法或请求消息")
	}
}
