package biz

import (
	"testing"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/service/admin/dto"
)

// TestNormalizeCodeGenColumnConfigColumnComment 验证字段描述完整继承数据库注释并保留用户修改。
func TestNormalizeCodeGenColumnConfigColumnComment(t *testing.T) {
	tests := []struct {
		name            string
		columnComment   string
		databaseComment string
		want            string
	}{
		{
			name:            "完整使用数据库注释",
			databaseComment: "用户性别：枚举【BaseUserGender】",
			want:            "用户性别：枚举【BaseUserGender】",
		},
		{
			name:            "保留用户修改",
			columnComment:   "用户性别",
			databaseComment: "用户性别：枚举【BaseUserGender】",
			want:            "用户性别",
		},
		{
			name: "无注释时回退字段名",
			want: "gender",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			column := &adminv1.CodeGenColumn{
				ColumnName:    "gender",
				ColumnComment: test.columnComment,
			}
			databaseColumn := dto.CodeGenDatabaseColumn{
				ColumnName:    "gender",
				ColumnComment: test.databaseComment,
			}
			normalizeCodeGenColumnConfig(column, databaseColumn)
			// 字段描述必须保持完整且只在用户未修改时回退。
			if column.GetColumnComment() != test.want {
				t.Fatalf("字段描述 = %q，期望 %q", column.GetColumnComment(), test.want)
			}
		})
	}
}
