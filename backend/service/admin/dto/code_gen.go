package dto

// CodeGenDatabaseTable 承载数据库表元数据查询结果。
type CodeGenDatabaseTable struct {
	TableName    string `gorm:"column:table_name"`    // 数据库表名
	TableComment string `gorm:"column:table_comment"` // 业务表描述
}

// CodeGenDatabaseColumn 承载数据库字段元数据查询结果。
type CodeGenDatabaseColumn struct {
	ColumnName    string `gorm:"column:column_name"`    // 数据库字段名
	ColumnComment string `gorm:"column:column_comment"` // 数据库字段注释
	DataType      string `gorm:"column:data_type"`      // 数据库字段类型
	ColumnType    string `gorm:"column:column_type"`    // 数据库完整字段类型
	ColumnKey     string `gorm:"column:column_key"`     // 数据库字段索引类型
	IsNullable    string `gorm:"column:is_nullable"`    // 是否允许为空
}
