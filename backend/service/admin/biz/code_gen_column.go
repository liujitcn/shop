package biz

import (
	"context"
	"regexp"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/errorsx"
	"shop/service/admin/dto"

	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
)

var codeGenDatabaseTableNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// CodeGenColumnCase 管理代码生成字段元数据。
type CodeGenColumnCase struct {
	dbClient *databaseGorm.Client // 数据库元数据客户端
}

// NewCodeGenColumnCase 创建代码生成字段业务实例。
func NewCodeGenColumnCase(dbClient *databaseGorm.Client) *CodeGenColumnCase {
	return &CodeGenColumnCase{dbClient: dbClient}
}

// ListCodeGenDatabaseColumns 查询指定数据库表的字段元数据。
func (c *CodeGenColumnCase) ListCodeGenDatabaseColumns(ctx context.Context, tableName string) (*adminv1.ListCodeGenDatabaseColumnsResponse, error) {
	if tableName == "" {
		return nil, errorsx.InvalidArgument("数据库表名不能为空")
	}
	if !codeGenDatabaseTableNamePattern.MatchString(tableName) {
		return nil, errorsx.InvalidArgument("数据库表名格式不正确")
	}
	var databaseColumns []dto.CodeGenDatabaseColumn
	err := c.dbClient.DB.WithContext(ctx).
		Table("information_schema.columns").
		Select("column_name, column_comment, data_type, column_type, column_key, is_nullable").
		Where("table_schema = DATABASE()").
		Where("table_name = ?", tableName).
		Order("ordinal_position").
		Find(&databaseColumns).Error
	if err != nil {
		return nil, err
	}
	columns := make([]*adminv1.CodeGenDatabaseColumn, 0, len(databaseColumns))
	for _, item := range databaseColumns {
		columnComment := item.ColumnComment
		// 数据库未配置字段注释时回退显示字段名。
		if columnComment == "" {
			columnComment = item.ColumnName
		}
		columnType := item.ColumnType
		// 数据库未返回完整类型时回退到基础数据类型。
		if columnType == "" {
			columnType = item.DataType
		}
		columns = append(columns, &adminv1.CodeGenDatabaseColumn{
			ColumnName:    item.ColumnName,
			ColumnComment: columnComment,
			DbType:        item.DataType,
			ColumnType:    columnType,
			IsPrimary:     item.ColumnKey == "PRI",
			IsNullable:    item.IsNullable == "YES",
		})
	}
	return &adminv1.ListCodeGenDatabaseColumnsResponse{Columns: columns}, nil
}
