package biz

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	systemadminv1 "shop/api/gen/go/system/admin/v1"
	_const "shop/pkg/const"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/service/system/admin/codegen"
	"shop/service/system/admin/dto"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/go-utils/stringcase"
	"github.com/liujitcn/gorm-kit/repository"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
)

const (
	// codeGenPageTypeNormal 表示普通表格页面。
	codeGenPageTypeNormal = "normal"
)

// CodeGenTableCase 管理代码生成表配置。
type CodeGenTableCase struct {
	*data.CodeGenTableRepository
	dbClient          *databaseGorm.Client // 数据库元数据客户端
	tx                data.Transaction
	baseMenuCase      *BaseMenuCase
	codeGenColumnCase *CodeGenColumnCase
	codeGenProtoCase  *CodeGenProtoCase
	formMapper        *mapper.CopierMapper[systemadminv1.CodeGenTableForm, models.CodeGenTable]
	mapper            *mapper.CopierMapper[systemadminv1.CodeGenTable, models.CodeGenTable]
}

// NewCodeGenTableCase 创建代码生成表配置业务实例。
func NewCodeGenTableCase(
	codeGenTableRepo *data.CodeGenTableRepository,
	dbClient *databaseGorm.Client,
	tx data.Transaction,
	baseMenuCase *BaseMenuCase,
	codeGenColumnCase *CodeGenColumnCase,
	codeGenProtoCase *CodeGenProtoCase,
) *CodeGenTableCase {
	formMapper := mapper.NewCopierMapper[systemadminv1.CodeGenTableForm, models.CodeGenTable]()
	formMapper.AppendConverters(mapper.NewJSONTypeConverter[*systemadminv1.CodeGenLeftTreeConfig]().NewConverterPair())
	formMapper.AppendConverters(mapper.NewGenericTypeConverterPair(
		false,
		int32(0),
		func(value bool) int32 {
			if value {
				return 1
			}
			return 0
		},
		func(value int32) bool {
			return value == 1
		},
	))
	return &CodeGenTableCase{
		CodeGenTableRepository: codeGenTableRepo,
		dbClient:               dbClient,
		tx:                     tx,
		baseMenuCase:           baseMenuCase,
		codeGenColumnCase:      codeGenColumnCase,
		codeGenProtoCase:       codeGenProtoCase,
		formMapper:             formMapper,
		mapper:                 mapper.NewCopierMapper[systemadminv1.CodeGenTable, models.CodeGenTable](),
	}
}

// ListCodeGenDatabaseTable 查询当前数据库表元数据。
func (c *CodeGenTableCase) ListCodeGenDatabaseTable(ctx context.Context) (*systemadminv1.ListCodeGenDatabaseTableResponse, error) {
	query := c.Query(ctx).CodeGenTable
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Order(query.Name.Asc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	usedTableNames := make(map[string]bool, len(list))
	for _, item := range list {
		usedTableNames[item.Name] = true
	}
	var tableInfos []dto.CodeGenDatabaseTable
	tableInfos, err = c.listDatabaseTables(ctx, nil)
	if err != nil {
		return nil, err
	}
	tables := make([]*systemadminv1.CodeGenDatabaseTable, 0, len(tableInfos))
	for _, tableInfo := range tableInfos {
		businessName := tableInfo.TableName
		pathSegments := strings.Split(businessName, "_")
		modulePath := businessName
		// 多段表名默认将最后一段作为资源名，其余部分作为模块路径。
		if len(pathSegments) > 1 {
			modulePath = strings.Join(pathSegments[:len(pathSegments)-1], "/")
		}
		tables = append(tables, &systemadminv1.CodeGenDatabaseTable{
			Name:             tableInfo.TableName,
			Comment:          tableInfo.TableComment,
			Disabled:         usedTableNames[tableInfo.TableName],
			BusinessName:     businessName,
			EntityName:       stringcase.ToPascalCase(businessName),
			ModulePath:       modulePath,
			ApiPath:          codegen.DefaultProtoDirectory,
			PermissionPrefix: strings.Join(pathSegments, ":"),
		})
	}
	return &systemadminv1.ListCodeGenDatabaseTableResponse{Tables: tables}, nil
}

// ListCodeGenProtoDirectory 查询可用于代码生成的 Proto 目录。
func (c *CodeGenTableCase) ListCodeGenProtoDirectory(_ context.Context) (*systemadminv1.ListCodeGenProtoDirectoryResponse, error) {
	directories, err := c.listCodeGenProtoDirectories()
	if err != nil {
		return nil, err
	}
	items := make([]*systemadminv1.CodeGenProtoDirectory, 0, len(directories))
	for _, directory := range directories {
		items = append(items, &systemadminv1.CodeGenProtoDirectory{Path: directory})
	}
	return &systemadminv1.ListCodeGenProtoDirectoryResponse{Directories: items}, nil
}

// PageCodeGenTable 查询代码生成表配置分页数据。
func (c *CodeGenTableCase) PageCodeGenTable(ctx context.Context, req *systemadminv1.PageCodeGenTableRequest) (*systemadminv1.PageCodeGenTableResponse, error) {
	query := c.Query(ctx).CodeGenTable
	opts := make([]repository.QueryOption, 0, 6)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.Name != nil {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	if req.BusinessName != nil {
		opts = append(opts, repository.Where(query.BusinessName.Like("%"+req.GetBusinessName()+"%")))
	}
	if req.ModulePath != nil {
		opts = append(opts, repository.Where(query.ModulePath.Like("%"+req.GetModulePath()+"%")))
	}
	if req.PageType != nil {
		opts = append(opts, repository.Where(query.PageType.Eq(req.GetPageType())))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(req.GetStatus())))
	}
	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	codeGenTables := make([]*systemadminv1.CodeGenTable, 0, len(list))
	for _, item := range list {
		codeGenTables = append(codeGenTables, c.mapper.ToDTO(item))
	}
	return &systemadminv1.PageCodeGenTableResponse{CodeGenTables: codeGenTables, Total: int32(total)}, nil
}

// GetCodeGenTable 查询代码生成表配置。
func (c *CodeGenTableCase) GetCodeGenTable(ctx context.Context, id int64) (*systemadminv1.CodeGenTableForm, error) {
	if id <= 0 {
		return nil, errorsx.InvalidArgument("代码生成表配置ID不能为空")
	}
	item, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	form := c.formMapper.ToDTO(item)
	if form.LeftTreeConfig == nil {
		form.LeftTreeConfig = &systemadminv1.CodeGenLeftTreeConfig{}
	}
	return form, nil
}

// CreateCodeGenTable 创建代码生成表配置。
func (c *CodeGenTableCase) CreateCodeGenTable(ctx context.Context, req *systemadminv1.CodeGenTableForm) error {
	item, err := c.codeGenTableFormToModel(ctx, 0, req)
	if err != nil {
		return err
	}
	item.ID = 0
	return c.Create(ctx, item)
}

// UpdateCodeGenTable 更新代码生成表配置。
func (c *CodeGenTableCase) UpdateCodeGenTable(ctx context.Context, id int64, req *systemadminv1.CodeGenTableForm) error {
	if id <= 0 {
		return errorsx.InvalidArgument("代码生成表配置ID不能为空")
	}
	item, err := c.codeGenTableFormToModel(ctx, id, req)
	if err != nil {
		return err
	}
	item.ID = id
	query := c.Query(ctx).CodeGenTable
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(id)))
	opts = append(opts, repository.Select(
		query.Name,
		query.Comment,
		query.BusinessName,
		query.EntityName,
		query.ModulePath,
		query.APIPath,
		query.PermissionPrefix,
		query.ParentMenuID,
		query.PageType,
		query.ParentColumn,
		query.TreeLabelColumn,
		query.LeftTreeConfig,
		query.GenBackend,
		query.GenFrontend,
		query.GenSql,
		query.Status,
		query.Remark,
	))
	return c.Update(ctx, item, opts...)
}

// DeleteCodeGenTable 删除代码生成表配置。
func (c *CodeGenTableCase) DeleteCodeGenTable(ctx context.Context, ids string) error {
	idList := _string.ConvertStringToInt64Array(ids)
	if len(idList) == 0 {
		return nil
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err := c.codeGenColumnCase.DeleteByTableIDs(ctx, idList)
		if err != nil {
			return err
		}
		err = c.codeGenProtoCase.DeleteByTableIDs(ctx, idList)
		if err != nil {
			return err
		}
		return c.DeleteByIDs(ctx, idList)
	})
}

// listDatabaseTables 查询当前数据库的表名与表描述，可按表名缩小范围。
func (c *CodeGenTableCase) listDatabaseTables(ctx context.Context, tableNames []string) ([]dto.CodeGenDatabaseTable, error) {
	query := c.dbClient.DB.WithContext(ctx).
		Table("information_schema.tables").
		Select("table_name, table_comment").
		Where("table_schema = DATABASE()").
		Where("table_type = ?", "BASE TABLE")
	if len(tableNames) > 0 {
		query = query.Where("table_name IN ?", tableNames)
	}
	var tableInfos []dto.CodeGenDatabaseTable
	err := query.Order("table_name").Find(&tableInfos).Error
	return tableInfos, err
}

// codeGenTableFormToModel 转换代码生成表配置保存模型，并校验生成所需的关联配置。
func (c *CodeGenTableCase) codeGenTableFormToModel(ctx context.Context, currentID int64, req *systemadminv1.CodeGenTableForm) (*models.CodeGenTable, error) {
	apiPath := req.GetApiPath()
	if apiPath == "" {
		apiPath = codegen.DefaultProtoDirectory
	}
	directories, err := c.listCodeGenProtoDirectories()
	if err != nil {
		return nil, err
	}
	directoryIndex := sort.SearchStrings(directories, apiPath)
	if directoryIndex == len(directories) || directories[directoryIndex] != apiPath {
		return nil, errorsx.InvalidArgument("请选择有效的Proto目录")
	}
	parentMenuID := req.GetParentMenuId()
	if parentMenuID <= 0 {
		return nil, errorsx.InvalidArgument("请选择父级菜单")
	}
	var menu *models.BaseMenu
	menu, err = c.baseMenuCase.FindByID(ctx, parentMenuID)
	if err != nil {
		return nil, errorsx.InvalidArgument("父级菜单不存在").WithCause(err)
	}
	// 生成页面只能挂载到目录或普通菜单节点。
	if menu.Type != _const.BASE_MENU_TYPE_FOLDER && menu.Type != _const.BASE_MENU_TYPE_MENU {
		return nil, errorsx.InvalidArgument("父级菜单只能选择目录或菜单")
	}
	query := c.Query(ctx).CodeGenTable
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.Name.Eq(req.GetName())))
	if currentID > 0 {
		opts = append(opts, repository.Where(query.ID.Neq(currentID)))
	}
	var count int64
	count, err = c.Count(ctx, opts...)
	if err != nil {
		return nil, err
	}
	// 同一数据库表只能对应一条配置，避免后续生成路径相互覆盖。
	if count > 0 {
		return nil, errorsx.UniqueConflict("业务表已被代码生成表配置选择", "code_gen_table", "name", "")
	}
	item := c.formMapper.ToEntity(req)
	item.APIPath = apiPath
	// 未指定页面类型时使用最通用的普通表格。
	if item.PageType == "" {
		item.PageType = codeGenPageTypeNormal
	}
	if req.GetLeftTreeConfig() == nil {
		item.LeftTreeConfig = ""
	}
	return item, nil
}

// listCodeGenProtoDirectories 查询 Proto 根目录下实际包含 Proto 文件的目录。
func (c *CodeGenTableCase) listCodeGenProtoDirectories() ([]string, error) {
	protoRoot := filepath.Join(codegen.BackendDir(), "api", "proto")
	directorySet := make(map[string]struct{})
	err := filepath.WalkDir(protoRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".proto" {
			return nil
		}
		directory, err := filepath.Rel(protoRoot, filepath.Dir(path))
		if err != nil {
			return err
		}
		directory = filepath.ToSlash(directory)
		if _, ok := codegen.ProtoTargetByDirectory(directory); ok {
			directorySet[directory] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	directories := make([]string, 0, len(directorySet))
	for directory := range directorySet {
		directories = append(directories, directory)
	}
	sort.Strings(directories)
	return directories, nil
}
