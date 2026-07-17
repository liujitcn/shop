package biz

import (
	"context"
	"math"
	"regexp"
	"strings"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/service/admin/dto"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
)

var codeGenDatabaseTableNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// CodeGenColumnCase 管理代码生成字段元数据与生成配置。
type CodeGenColumnCase struct {
	*data.CodeGenColumnRepository
	dbClient         *databaseGorm.Client
	tx               data.Transaction
	codeGenTableRepo *data.CodeGenTableRepository
	mapper           *mapper.CopierMapper[adminv1.CodeGenColumn, models.CodeGenColumn]
}

// NewCodeGenColumnCase 创建代码生成字段业务实例。
func NewCodeGenColumnCase(
	codeGenColumnRepo *data.CodeGenColumnRepository,
	dbClient *databaseGorm.Client,
	tx data.Transaction,
	codeGenTableRepo *data.CodeGenTableRepository,
) *CodeGenColumnCase {
	columnMapper := mapper.NewCopierMapper[adminv1.CodeGenColumn, models.CodeGenColumn]()
	columnMapper.AppendConverters(mapper.NewJSONTypeConverter[*adminv1.CodeGenColumnQueryConfig]().NewConverterPair())
	columnMapper.AppendConverters(mapper.NewJSONTypeConverter[*adminv1.CodeGenColumnListConfig]().NewConverterPair())
	columnMapper.AppendConverters(mapper.NewJSONTypeConverter[*adminv1.CodeGenColumnFormConfig]().NewConverterPair())
	return &CodeGenColumnCase{
		CodeGenColumnRepository: codeGenColumnRepo,
		dbClient:                dbClient,
		tx:                      tx,
		codeGenTableRepo:        codeGenTableRepo,
		mapper:                  columnMapper,
	}
}

// ListCodeGenDatabaseColumns 查询指定数据库表的字段元数据。
func (c *CodeGenColumnCase) ListCodeGenDatabaseColumns(ctx context.Context, tableName string) (*adminv1.ListCodeGenDatabaseColumnsResponse, error) {
	databaseColumns, err := c.listDatabaseColumns(ctx, tableName)
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

// ListCodeGenColumns 查询允许用户维护的字段配置。
func (c *CodeGenColumnCase) ListCodeGenColumns(ctx context.Context, tableID int64) (*adminv1.ListCodeGenColumnsResponse, error) {
	columns, err := c.listCodeGenColumns(ctx, tableID)
	if err != nil {
		return nil, err
	}
	return &adminv1.ListCodeGenColumnsResponse{
		CodeGenColumns: filterConfigurableCodeGenColumns(columns),
	}, nil
}

// listCodeGenColumns 查询数据库字段与已保存生成配置的完整合并结果。
func (c *CodeGenColumnCase) listCodeGenColumns(ctx context.Context, tableID int64) ([]*adminv1.CodeGenColumn, error) {
	if tableID <= 0 {
		return nil, errorsx.InvalidArgument("代码生成表配置ID不能为空")
	}
	table, err := c.codeGenTableRepo.FindByID(ctx, tableID)
	if err != nil {
		return nil, err
	}
	var databaseColumns []dto.CodeGenDatabaseColumn
	databaseColumns, err = c.listDatabaseColumns(ctx, table.Name)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).CodeGenColumn
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.TableID.Eq(tableID)))
	opts = append(opts, repository.Order(query.ID.Asc()))
	var savedColumns []*models.CodeGenColumn
	savedColumns, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return c.mergeCodeGenColumns(tableID, databaseColumns, savedColumns), nil
}

// SaveCodeGenColumns 保存代码生成字段配置快照。
func (c *CodeGenColumnCase) SaveCodeGenColumns(ctx context.Context, req *adminv1.SaveCodeGenColumnsRequest) error {
	if req.GetTableId() <= 0 {
		return errorsx.InvalidArgument("代码生成表配置ID不能为空")
	}
	table, err := c.codeGenTableRepo.FindByID(ctx, req.GetTableId())
	if err != nil {
		return err
	}
	var databaseColumns []dto.CodeGenDatabaseColumn
	databaseColumns, err = c.listDatabaseColumns(ctx, table.Name)
	if err != nil {
		return err
	}
	// 保存字段配置前必须确认目标表仍有可用字段。
	if len(databaseColumns) == 0 {
		return errorsx.ResourceNotFound("数据库表字段不存在")
	}
	requestColumns := make(map[string]*adminv1.CodeGenColumn, len(req.GetCodeGenColumns()))
	for _, column := range req.GetCodeGenColumns() {
		if column == nil || column.GetColumnName() == "" {
			return errorsx.InvalidArgument("字段名不能为空")
		}
		// 同一个字段只能保存一份配置，避免覆盖顺序影响最终结果。
		if _, exists := requestColumns[column.GetColumnName()]; exists {
			return errorsx.InvalidArgument("字段" + column.GetColumnName() + "配置重复")
		}
		requestColumns[column.GetColumnName()] = column
	}
	columns := make([]*models.CodeGenColumn, 0, len(databaseColumns))
	for index, databaseColumn := range databaseColumns {
		column := requestColumns[databaseColumn.ColumnName]
		// 前端未提交的系统字段按元数据默认值保存，保证数据库字段快照完整。
		if column == nil {
			column = newDefaultCodeGenColumn(req.GetTableId(), databaseColumn, int32(index+1))
		}
		normalizeCodeGenColumnConfig(column, databaseColumn)
		if err = validateCodeGenColumnConfig(column); err != nil {
			return err
		}
		item := c.mapper.ToEntity(column)
		item.ID = 0
		item.TableID = req.GetTableId()
		item.ColumnName = databaseColumn.ColumnName
		columns = append(columns, item)
		delete(requestColumns, databaseColumn.ColumnName)
	}
	// 数据库元数据中不存在的请求字段不能写入配置快照。
	if len(requestColumns) > 0 {
		for columnName := range requestColumns {
			return errorsx.InvalidArgument("字段" + columnName + "不存在")
		}
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		// 使用完整快照替换旧字段配置，确保排序和默认值一致。
		query := c.Query(ctx).CodeGenColumn
		if err = c.Delete(ctx, repository.Where(query.TableID.Eq(req.GetTableId()))); err != nil {
			return err
		}
		return c.BatchCreate(ctx, columns)
	})
}

// DeleteByTableIDs 删除多个代码生成表配置关联的字段配置。
func (c *CodeGenColumnCase) DeleteByTableIDs(ctx context.Context, tableIDs []int64) error {
	if len(tableIDs) == 0 {
		return nil
	}
	query := c.Query(ctx).CodeGenColumn
	return c.Delete(ctx, repository.Where(query.TableID.In(tableIDs...)))
}

// listDatabaseColumns 查询数据库字段生成所需的完整元数据。
func (c *CodeGenColumnCase) listDatabaseColumns(ctx context.Context, tableName string) ([]dto.CodeGenDatabaseColumn, error) {
	if tableName == "" {
		return nil, errorsx.InvalidArgument("数据库表名不能为空")
	}
	if !codeGenDatabaseTableNamePattern.MatchString(tableName) {
		return nil, errorsx.InvalidArgument("数据库表名格式不正确")
	}
	var columns []dto.CodeGenDatabaseColumn
	// information_schema 没有业务生成模型，表名经白名单校验后使用参数化查询读取字段元数据。
	err := c.dbClient.DB.WithContext(ctx).
		Table("information_schema.columns").
		Select("column_name, column_comment, data_type, column_type, column_key, is_nullable, extra, ordinal_position, character_maximum_length, numeric_precision, numeric_scale").
		Where("table_schema = DATABASE()").
		Where("table_name = ?", tableName).
		Order("ordinal_position").
		Find(&columns).Error
	return columns, err
}

// mergeCodeGenColumns 合并数据库字段元数据与已保存配置。
func (c *CodeGenColumnCase) mergeCodeGenColumns(tableID int64, databaseColumns []dto.CodeGenDatabaseColumn, savedColumns []*models.CodeGenColumn) []*adminv1.CodeGenColumn {
	savedByName := make(map[string]*models.CodeGenColumn, len(savedColumns))
	for _, column := range savedColumns {
		savedByName[column.ColumnName] = column
	}
	columns := make([]*adminv1.CodeGenColumn, 0, len(databaseColumns))
	for index, databaseColumn := range databaseColumns {
		column := newDefaultCodeGenColumn(tableID, databaseColumn, int32(index+1))
		// 已保存配置只覆盖用户可编辑部分，数据库字段属性始终以实时元数据为准。
		if saved := savedByName[databaseColumn.ColumnName]; saved != nil {
			savedColumn := c.mapper.ToDTO(saved)
			column.Id = savedColumn.GetId()
			column.ColumnComment = savedColumn.GetColumnComment()
			column.QueryConfig = savedColumn.GetQueryConfig()
			column.ListConfig = savedColumn.GetListConfig()
			column.FormConfig = savedColumn.GetFormConfig()
		}
		normalizeCodeGenColumnConfig(column, databaseColumn)
		columns = append(columns, column)
	}
	return columns
}

// filterConfigurableCodeGenColumns 过滤字段配置接口不允许维护的数据库字段。
func filterConfigurableCodeGenColumns(columns []*adminv1.CodeGenColumn) []*adminv1.CodeGenColumn {
	configurableColumns := make([]*adminv1.CodeGenColumn, 0, len(columns))
	for _, column := range columns {
		// 主键和软删除字段由基础设施维护，不通过字段配置接口返回。
		if column.GetIsPrimary() || column.GetColumnName() == "deleted_at" {
			continue
		}
		configurableColumns = append(configurableColumns, column)
	}
	return configurableColumns
}

// newDefaultCodeGenColumn 根据数据库字段元数据创建默认生成配置。
func newDefaultCodeGenColumn(tableID int64, item dto.CodeGenDatabaseColumn, sort int32) *adminv1.CodeGenColumn {
	columnComment := item.ColumnComment
	// 数据库未配置字段注释时使用字段名作为展示名称。
	if columnComment == "" {
		columnComment = item.ColumnName
	}
	lengthValue := item.CharacterMaximumLength
	// 数值字段没有字符长度时使用数值精度。
	if !lengthValue.Valid {
		lengthValue = item.NumericPrecision
	}
	var columnLength int32
	// 超出 Proto int32 范围的数据库长度不写入生成配置。
	if lengthValue.Valid && lengthValue.Int64 <= math.MaxInt32 {
		columnLength = int32(lengthValue.Int64)
	}
	var columnScale int32
	// 超出 Proto int32 范围的小数位不写入生成配置。
	if item.NumericScale.Valid && item.NumericScale.Int64 <= math.MaxInt32 {
		columnScale = int32(item.NumericScale.Int64)
	}
	isPrimary := item.ColumnKey == "PRI"
	isAutoIncrement := strings.Contains(strings.ToLower(item.Extra), "auto_increment")
	protoType := inferCodeGenProtoType(item.ColumnType)
	tsType := "string"
	// TypeScript 数值与布尔类型跟随 Proto 标量类型映射。
	if protoType == "bool" {
		tsType = "boolean"
	} else if protoType == "int64" || protoType == "int32" || protoType == "double" {
		tsType = "number"
	}
	column := &adminv1.CodeGenColumn{
		TableId:         tableID,
		ColumnName:      item.ColumnName,
		ColumnComment:   columnComment,
		DbType:          item.ColumnType,
		DbLength:        columnLength,
		DbScale:         columnScale,
		IsPrimary:       isPrimary,
		IsAutoIncrement: isAutoIncrement,
		IsNullable:      item.IsNullable == "YES",
		GoType:          inferCodeGenGoType(item.ColumnType),
		ProtoType:       protoType,
		TsType:          tsType,
		Sort:            sort,
	}
	normalizeCodeGenColumnConfig(column, item)
	return column
}

// normalizeCodeGenColumnConfig 补齐缺失的结构化字段配置。
func normalizeCodeGenColumnConfig(column *adminv1.CodeGenColumn, item dto.CodeGenDatabaseColumn) {
	// 旧配置没有保存字段描述时，完整沿用数据库原始注释。
	if column.ColumnComment == "" {
		column.ColumnComment = item.ColumnComment
		if column.ColumnComment == "" {
			column.ColumnComment = item.ColumnName
		}
	}
	// 缺失的查询配置按字段名称和数据库类型推导。
	if column.QueryConfig == nil {
		column.QueryConfig = defaultCodeGenQueryConfig(item.ColumnName, item.ColumnType)
	}
	// 查询选项独立保存，不能与列表或表单配置共用。
	if column.QueryConfig.Option == nil {
		column.QueryConfig.Option = &adminv1.CodeGenColumnOptionConfig{}
	}
	// 缺失的列表配置按字段类型选择基础展示组件。
	if column.ListConfig == nil {
		component := "text"
		// 状态字段默认生成可操作开关，日期字段使用日期展示。
		if isCodeGenStatusColumn(item.ColumnName) {
			component = "switch"
		} else if isCodeGenDateTimeType(item.ColumnType) {
			component = "date"
		}
		column.ListConfig = &adminv1.CodeGenColumnListConfig{
			Enabled:   !isManagedCodeGenColumn(item.ColumnName),
			Component: component,
			Option:    &adminv1.CodeGenColumnOptionConfig{},
		}
	}
	// 列表选项独立保存，不能与查询或表单配置共用。
	if column.ListConfig.Option == nil {
		column.ListConfig.Option = &adminv1.CodeGenColumnOptionConfig{}
	}
	// 缺失的表单配置按数据库约束推导。
	if column.FormConfig == nil {
		column.FormConfig = defaultCodeGenFormConfig(item)
	}
	// 表单选项独立保存，不能与查询或列表配置共用。
	if column.FormConfig.Option == nil {
		column.FormConfig.Option = &adminv1.CodeGenColumnOptionConfig{}
	}
}

// validateCodeGenColumnConfig 校验结构化字段配置的业务完整性。
func validateCodeGenColumnConfig(column *adminv1.CodeGenColumn) error {
	err := validateCodeGenOptionConfig(column.GetColumnName(), "查询", column.GetQueryConfig().GetOption())
	if err != nil {
		return err
	}
	err = validateCodeGenListOptionConfig(column.GetColumnName(), column.GetListConfig())
	if err != nil {
		return err
	}
	err = validateCodeGenOptionConfig(column.GetColumnName(), "表单", column.GetFormConfig().GetOption())
	if err != nil {
		return err
	}
	return nil
}

// validateCodeGenListOptionConfig 校验列表组件与选项配置是否匹配。
func validateCodeGenListOptionConfig(columnName string, config *adminv1.CodeGenColumnListConfig) error {
	option := config.GetOption()
	// 列表只允许使用已经确认的七种展示组件。
	switch config.GetComponent() {
	case "text", "image", "money", "date":
		if option.GetKind() != "" || option.GetSourceType() != "" || option.GetSourceValue() != "" || option.GetLabelField() != "" || option.GetValueField() != "" || option.GetParentField() != "" || option.GetActiveValue() != "" || option.GetInactiveValue() != "" {
			return errorsx.InvalidArgument("字段" + columnName + "的列表组件不需要选项配置")
		}
		return nil
	case "switch":
		if !config.GetEnabled() {
			return validateCodeGenOptionConfig(columnName, "列表", option)
		}
		if option.GetKind() != "switch" || option.GetSourceType() != "dict" || option.GetSourceValue() == "" || option.GetActiveValue() == "" || option.GetInactiveValue() == "" {
			return errorsx.InvalidArgument("字段" + columnName + "的列表开关配置不完整")
		}
		if option.GetActiveValue() == option.GetInactiveValue() {
			return errorsx.InvalidArgument("字段" + columnName + "的列表开关开启值和关闭值不能相同")
		}
		return nil
	case "select":
		if !config.GetEnabled() {
			return validateCodeGenOptionConfig(columnName, "列表", option)
		}
		if option.GetKind() != "option" {
			return errorsx.InvalidArgument("字段" + columnName + "的列表下拉选项配置不完整")
		}
	case "tree-select":
		if !config.GetEnabled() {
			return validateCodeGenOptionConfig(columnName, "列表", option)
		}
		if option.GetKind() != "tree" {
			return errorsx.InvalidArgument("字段" + columnName + "的列表树形选项配置不完整")
		}
	default:
		return errorsx.InvalidArgument("字段" + columnName + "的列表组件不支持")
	}
	return validateCodeGenOptionConfig(columnName, "列表", option)
}

// validateCodeGenOptionConfig 校验查询、列表或表单自己的选项配置。
func validateCodeGenOptionConfig(columnName string, scope string, option *adminv1.CodeGenColumnOptionConfig) error {
	// 未启用选项能力时不允许只残留部分来源字段。
	if option.GetKind() == "" {
		if option.GetSourceType() != "" || option.GetSourceValue() != "" || option.GetLabelField() != "" || option.GetValueField() != "" || option.GetParentField() != "" || option.GetActiveValue() != "" || option.GetInactiveValue() != "" {
			return errorsx.InvalidArgument("字段" + columnName + "的" + scope + "选项形态不能为空")
		}
		return nil
	}
	// 开关值只允许由列表开关配置维护。
	if option.GetKind() != "switch" && (option.GetActiveValue() != "" || option.GetInactiveValue() != "") {
		return errorsx.InvalidArgument("字段" + columnName + "的" + scope + "选项不能配置开关值")
	}
	// 树形选项只能使用数据库表构建真实父子关系。
	if option.GetKind() == "tree" && option.GetSourceType() != "table" {
		return errorsx.InvalidArgument("字段" + columnName + "的" + scope + "树形选项只能使用数据表来源")
	}
	// 启用选项能力后，来源只能是静态数据、字典或数据表。
	switch option.GetSourceType() {
	case "static", "dict", "table":
	default:
		return errorsx.InvalidArgument("字段" + columnName + "的" + scope + "选项来源配置不完整")
	}
	// 所有选项来源都必须配置具体来源值。
	if option.GetSourceValue() == "" {
		return errorsx.InvalidArgument("字段" + columnName + "的" + scope + "选项来源值不能为空")
	}
	// 数据表选项必须明确展示字段、值字段以及树形父级字段。
	if option.GetSourceType() == "table" && (option.GetLabelField() == "" || option.GetValueField() == "" || option.GetKind() == "tree" && option.GetParentField() == "") {
		return errorsx.InvalidArgument("字段" + columnName + "的" + scope + "数据表选项字段配置不完整")
	}
	return nil
}

// defaultCodeGenQueryConfig 创建默认查询配置。
func defaultCodeGenQueryConfig(columnName string, dbType string) *adminv1.CodeGenColumnQueryConfig {
	enabled := columnName == "name" || columnName == "title" || columnName == "code" || isCodeGenStatusColumn(columnName)
	operator := "like"
	component := "input"
	// 日期时间字段默认使用区间查询。
	if isCodeGenDateTimeType(dbType) {
		operator = "between"
		component = "date-picker"
	} else if isCodeGenBoolType(dbType) || isCodeGenNumericType(dbType) || isCodeGenStatusColumn(columnName) {
		operator = "eq"
		component = "input-number"
	}
	// 布尔值和状态字段使用选项组件表达有限值集合。
	if isCodeGenBoolType(dbType) || isCodeGenStatusColumn(columnName) {
		component = "select"
	}
	return &adminv1.CodeGenColumnQueryConfig{
		Enabled:   enabled,
		Operator:  operator,
		Component: component,
		Option:    &adminv1.CodeGenColumnOptionConfig{},
	}
}

// defaultCodeGenFormConfig 创建默认表单录入配置。
func defaultCodeGenFormConfig(item dto.CodeGenDatabaseColumn) *adminv1.CodeGenColumnFormConfig {
	isPrimary := item.ColumnKey == "PRI"
	isAutoIncrement := strings.Contains(strings.ToLower(item.Extra), "auto_increment")
	enabled := !isPrimary && !isAutoIncrement && !isManagedCodeGenColumn(item.ColumnName)
	component := "input"
	// 表单组件根据数据库字段类型选择，优先匹配更具体的布尔和数值类型。
	if isCodeGenStatusColumn(item.ColumnName) || isCodeGenBoolType(item.ColumnType) {
		component = "switch"
	} else if isCodeGenNumericType(item.ColumnType) {
		component = "input-number"
	} else if isCodeGenDateTimeType(item.ColumnType) {
		component = "date-picker"
	} else if strings.Contains(strings.ToLower(item.ColumnType), "text") {
		component = "textarea"
	}
	return &adminv1.CodeGenColumnFormConfig{
		Enabled:   enabled,
		Component: component,
		Required:  enabled && item.IsNullable != "YES",
		Option:    &adminv1.CodeGenColumnOptionConfig{},
	}
}

// inferCodeGenGoType 推断 Go 字段类型。
func inferCodeGenGoType(dbType string) string {
	// tinyint(1) 等布尔字段优先映射为 bool。
	if isCodeGenBoolType(dbType) {
		return "bool"
	}
	lowerType := strings.ToLower(dbType)
	// bigint 需要保留 64 位整数范围。
	if strings.Contains(lowerType, "bigint") {
		return "int64"
	}
	// 其余数值字段按是否包含小数映射。
	if isCodeGenNumericType(lowerType) {
		if strings.Contains(lowerType, "decimal") || strings.Contains(lowerType, "float") || strings.Contains(lowerType, "double") {
			return "float64"
		}
		return "int32"
	}
	// Go 层保留数据库日期时间语义。
	if isCodeGenDateTimeType(lowerType) {
		return "time.Time"
	}
	return "string"
}

// inferCodeGenProtoType 推断 Proto 字段类型。
func inferCodeGenProtoType(dbType string) string {
	lowerType := strings.ToLower(dbType)
	// Proto 标量类型按数据库类型范围依次匹配。
	if isCodeGenBoolType(lowerType) {
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

// isManagedCodeGenColumn 判断字段是否由基础设施维护。
func isManagedCodeGenColumn(columnName string) bool {
	return columnName == "created_by" || columnName == "updated_by" || columnName == "created_at" || columnName == "updated_at" || columnName == "deleted_at"
}

// isCodeGenStatusColumn 判断字段是否表示状态。
func isCodeGenStatusColumn(columnName string) bool {
	return columnName == "status" || columnName == "state" || strings.HasSuffix(columnName, "_status") || strings.HasSuffix(columnName, "_state")
}

// isCodeGenBoolType 判断数据库字段是否表示布尔值。
func isCodeGenBoolType(dbType string) bool {
	lowerType := strings.ToLower(dbType)
	return lowerType == "bool" || lowerType == "boolean" || strings.Contains(lowerType, "tinyint(1)")
}

// isCodeGenNumericType 判断数据库字段是否表示数值。
func isCodeGenNumericType(dbType string) bool {
	lowerType := strings.ToLower(dbType)
	return strings.Contains(lowerType, "int") || strings.Contains(lowerType, "decimal") || strings.Contains(lowerType, "float") || strings.Contains(lowerType, "double")
}

// isCodeGenDateTimeType 判断数据库字段是否表示日期时间。
func isCodeGenDateTimeType(dbType string) bool {
	lowerType := strings.ToLower(dbType)
	return strings.Contains(lowerType, "date") || strings.Contains(lowerType, "time")
}
