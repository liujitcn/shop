package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	_const "shop/pkg/const"
	"shop/pkg/errorsx"
	"shop/pkg/gen/models"
	"shop/service/system/admin/codegen"

	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gen/field"
)

const codeGenRestoreRoot = "backend/data/codegen/restore"

// codeGenRestoreManifest 保存一次生成前的文件快照。
type codeGenRestoreManifest struct {
	Version       int                  `json:"version"`
	TaskID        string               `json:"task_id"`
	TableID       int64                `json:"table_id"`
	BatchTableIDs []int64              `json:"batch_table_ids"`
	Files         []codeGenRestoreFile `json:"files"`
}

// codeGenRestoreFile 保存单个文件生成前后的内容。
type codeGenRestoreFile struct {
	Path             string  `json:"path"`
	OriginalExists   bool    `json:"original_exists"`
	OriginalContent  string  `json:"original_content"`
	OriginalMode     uint32  `json:"original_mode"`
	GeneratedExists  bool    `json:"generated_exists"`
	GeneratedContent string  `json:"generated_content"`
	OwnerTableIDs    []int64 `json:"owner_table_ids"`
}

// codeGenRestoreWorkspaceFile 表示工作区文件的当前快照。
type codeGenRestoreWorkspaceFile struct {
	Path    string
	Exists  bool
	Content []byte
	Mode    os.FileMode
}

// codeGenRestoreTransaction 管理还原快照文件的失败回滚。
type codeGenRestoreTransaction struct {
	snapshots    map[string]codeGenFileSnapshot
	writtenPaths []string
	committed    bool
}

// RestoreCodeGen 还原单个或批量代码生成结果。
func (c *CodeGenCase) RestoreCodeGen(ctx context.Context, tableIDs []int64) error {
	ids, err := normalizeCodeGenTableIDs(tableIDs)
	if err != nil {
		return err
	}
	manifests := make(map[int64]*codeGenRestoreManifest, len(ids))
	for _, tableID := range ids {
		var manifest *codeGenRestoreManifest
		manifest, err = loadCodeGenRestoreManifest(tableID)
		if err != nil {
			return err
		}
		manifests[tableID] = manifest
	}
	if err = validateCodeGenRestoreBatch(ids, manifests); err != nil {
		return err
	}
	for _, manifest := range manifests {
		if err = validateCodeGenRestoreFiles(manifest.Files); err != nil {
			return err
		}
	}
	paths := make([]string, 0)
	for _, manifest := range manifests {
		for _, file := range manifest.Files {
			paths = append(paths, file.Path)
		}
	}
	for _, tableID := range ids {
		paths = append(paths, codeGenRestoreManifestPath(tableID))
	}
	fileTransaction, err := newCodeGenRestoreTransaction(paths)
	if err != nil {
		return err
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		for _, tableID := range ids {
			err = c.restoreCodeGenTable(txCtx, tableID, manifests[tableID], fileTransaction)
			if err != nil {
				return err
			}
		}
		for _, tableID := range ids {
			fileTransaction.record(codeGenRestoreManifestPath(tableID))
			manifestPath, pathErr := codegen.SafeRepoFilePath(codeGenRestoreManifestPath(tableID))
			if pathErr != nil {
				return pathErr
			}
			removeErr := os.Remove(manifestPath)
			if removeErr != nil && !os.IsNotExist(removeErr) {
				return removeErr
			}
		}
		return nil
	})
	if err != nil {
		rollbackErr := fileTransaction.rollback()
		if rollbackErr != nil {
			return errorsx.Internal("还原代码生成结果失败").WithCause(fmt.Errorf("%w；回滚快照失败：%v", err, rollbackErr))
		}
		return err
	}
	fileTransaction.commit()
	return nil
}

// RestoreAvailable 判断代码生成表是否存在可用还原快照。
func RestoreAvailable(tableID int64) bool {
	if tableID <= 0 {
		return false
	}
	path, err := codegen.SafeRepoFilePath(codeGenRestoreManifestPath(tableID))
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// captureCodeGenWorkspaceSnapshot 读取生成命令可能改写的工作区文件。
func captureCodeGenWorkspaceSnapshot(batchFiles []*codegen.BatchFile) (map[string]codeGenRestoreWorkspaceFile, error) {
	paths, err := codeGenWorkspacePaths()
	if err != nil {
		return nil, err
	}
	for _, file := range batchFiles {
		if _, exists := paths[file.Path]; !exists {
			paths[file.Path] = struct{}{}
		}
	}
	snapshots := make(map[string]codeGenRestoreWorkspaceFile, len(paths))
	for path := range paths {
		snapshot, readErr := readCodeGenWorkspaceFile(path)
		if readErr != nil {
			return nil, readErr
		}
		snapshots[path] = snapshot
	}
	return snapshots, nil
}

// buildCodeGenRestoreManifests 构建本次生成实际改写文件的还原快照。
func buildCodeGenRestoreManifests(taskID string, tableIDs []int64, batch *codegen.BatchGeneration, before map[string]codeGenRestoreWorkspaceFile) (map[int64]*codeGenRestoreManifest, error) {
	after, err := captureCodeGenWorkspaceSnapshot(batch.Files)
	if err != nil {
		return nil, err
	}
	ownersByPath := make(map[string][]int64)
	for _, file := range batch.Files {
		owners := make([]int64, 0, len(file.Refs))
		for _, ref := range file.Refs {
			owners = appendUniqueInt64(owners, ref.TableID)
		}
		ownersByPath[file.Path] = owners
	}
	manifests := make(map[int64]*codeGenRestoreManifest, len(tableIDs))
	for _, tableID := range tableIDs {
		manifests[tableID] = &codeGenRestoreManifest{Version: 1, TaskID: taskID, TableID: tableID, BatchTableIDs: append([]int64(nil), tableIDs...)}
	}
	allPaths := make(map[string]struct{}, len(before)+len(after))
	for path := range before {
		allPaths[path] = struct{}{}
	}
	for path := range after {
		allPaths[path] = struct{}{}
	}
	for path := range allPaths {
		afterFile, afterExists := after[path]
		if !afterExists {
			afterFile = codeGenRestoreWorkspaceFile{Path: path}
		}
		beforeFile, beforeExists := before[path]
		if !beforeExists {
			beforeFile = codeGenRestoreWorkspaceFile{Path: path}
		}
		if workspaceFilesEqual(beforeFile, afterFile) {
			continue
		}
		owners := append([]int64(nil), ownersByPath[path]...)
		if len(owners) == 0 {
			owners = append(owners, tableIDs...)
		}
		for _, tableID := range owners {
			manifest := manifests[tableID]
			if manifest == nil {
				continue
			}
			manifest.Files = append(manifest.Files, codeGenRestoreFile{
				Path:             path,
				OriginalExists:   beforeFile.Exists,
				OriginalContent:  string(beforeFile.Content),
				OriginalMode:     uint32(beforeFile.Mode.Perm()),
				GeneratedExists:  afterFile.Exists,
				GeneratedContent: string(afterFile.Content),
				OwnerTableIDs:    append([]int64(nil), owners...),
			})
		}
	}
	for _, manifest := range manifests {
		sort.Slice(manifest.Files, func(i, j int) bool { return manifest.Files[i].Path < manifest.Files[j].Path })
	}
	return manifests, nil
}

// SaveCodeGenRestoreManifests 持久化本次生成的还原快照。
func SaveCodeGenRestoreManifests(manifests map[int64]*codeGenRestoreManifest) error {
	paths := make([]string, 0, len(manifests))
	for tableID := range manifests {
		paths = append(paths, codeGenRestoreManifestPath(tableID))
	}
	transaction, err := newCodeGenRestoreTransaction(paths)
	if err != nil {
		return err
	}
	for tableID, manifest := range manifests {
		content, marshalErr := json.Marshal(manifest)
		if marshalErr != nil {
			err = marshalErr
			break
		}
		manifestPath, pathErr := codegen.SafeRepoFilePath(codeGenRestoreManifestPath(tableID))
		if pathErr != nil {
			err = pathErr
			break
		}
		if writeErr := writeCodeGenRestoreFile(manifestPath, content); writeErr != nil {
			err = writeErr
			break
		}
		transaction.record(codeGenRestoreManifestPath(tableID))
	}
	if err != nil {
		rollbackErr := transaction.rollback()
		if rollbackErr != nil {
			return errorsx.Internal("保存代码生成还原快照失败").WithCause(fmt.Errorf("%w；回滚快照失败：%v", err, rollbackErr))
		}
		return err
	}
	transaction.commit()
	return nil
}

// listGeneratedMenus 查询当前代码生成对象关联的页面和按钮菜单。
func (c *CodeGenCase) listGeneratedMenus(ctx context.Context, table *codegen.Table, columns []*codegen.CodeGenColumn, methods []*codegen.Proto, resourcePath string) ([]*models.BaseMenu, error) {
	pageSpec, buttonSpecs := codegen.MenuSpecs(table, columns, methods, resourcePath, table.TableComment)
	query := c.baseMenuCase.Query(ctx).BaseMenu
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.Type.Eq(_const.BASE_MENU_TYPE_MENU)))
	opts = append(opts, repository.Where(field.Or(
		query.Path.Eq(pageSpec.Menu.Path),
		query.Name.Eq(pageSpec.Menu.Name),
		query.Component.Eq(pageSpec.Menu.Component),
	)))
	pages, err := c.baseMenuCase.List(ctx, opts...)
	if err != nil || len(pages) == 0 {
		return pages, err
	}
	page := pages[0]
	menus := []*models.BaseMenu{page}
	childOpts := make([]repository.QueryOption, 0, 2)
	childOpts = append(childOpts, repository.Where(query.ParentID.Eq(page.ID)))
	childOpts = append(childOpts, repository.Where(query.Type.Eq(_const.BASE_MENU_TYPE_BUTTON)))
	children, err := c.baseMenuCase.List(ctx, childOpts...)
	if err != nil {
		return nil, err
	}
	expectedPaths := make(map[string]struct{}, len(buttonSpecs))
	expectedAPIs := make(map[string]struct{}, len(buttonSpecs))
	for _, spec := range buttonSpecs {
		expectedPaths[spec.Menu.Path] = struct{}{}
		expectedAPIs[spec.Menu.API] = struct{}{}
	}
	statusPathPrefix := codegen.PermissionPrefix(table) + ":status"
	statusAPIPrefix := codegen.GeneratedRPCServicePath(table, table.EntityName) + "/Set"
	for _, child := range children {
		_, expectedPath := expectedPaths[child.Path]
		_, expectedAPI := expectedAPIs[child.API]
		if expectedPath || expectedAPI || strings.HasPrefix(child.Path, statusPathPrefix) || strings.Contains(child.API, statusAPIPrefix) {
			menus = append(menus, child)
		}
	}
	return menus, nil
}

// restoreCodeGenTable 还原单个代码生成对象的文件、菜单和状态。
func (c *CodeGenCase) restoreCodeGenTable(ctx context.Context, tableID int64, manifest *codeGenRestoreManifest, transaction *codeGenRestoreTransaction) error {
	table, columns, protos, err := c.loadCodeGenContext(ctx, tableID)
	if err != nil {
		return err
	}
	generation, err := codegen.PrepareGeneration(table, columns, protos, nil, table.TableComment)
	if err != nil {
		return err
	}
	if codegen.ShouldSyncMenus(generation.Table, generation.GeneratedMethods) {
		err = c.removeGeneratedMenus(ctx, generation.Table, columns, generation.GeneratedMethods, codegen.FrontendPageComponentPath(generation.OutputPaths.GetFrontendPageFilePath()))
		if err != nil {
			return err
		}
	}
	for _, file := range manifest.Files {
		fullPath, pathErr := codegen.SafeRepoFilePath(file.Path)
		if pathErr != nil {
			return pathErr
		}
		if file.OriginalExists {
			mode := os.FileMode(file.OriginalMode)
			if mode == 0 {
				mode = 0o644
			}
			err = writeGeneratedFileAtomically(fullPath, []byte(file.OriginalContent), mode, false)
		} else {
			err = os.Remove(fullPath)
			if os.IsNotExist(err) {
				err = nil
			}
		}
		if err != nil {
			return err
		}
		transaction.record(file.Path)
	}
	query := c.codeGenTableCase.Query(ctx).CodeGenTable
	return c.codeGenTableCase.Update(
		ctx,
		&models.CodeGenTable{ID: tableID, Status: codegen.StatusDraft},
		repository.Where(query.ID.Eq(tableID)),
		repository.Select(query.Status),
	)
}

// removeGeneratedMenus 删除代码生成对象产生的页面和按钮权限。
func (c *CodeGenCase) removeGeneratedMenus(ctx context.Context, table *codegen.Table, columns []*codegen.CodeGenColumn, methods []*codegen.Proto, resourcePath string) error {
	menus, err := c.listGeneratedMenus(ctx, table, columns, methods, resourcePath)
	if err != nil || len(menus) == 0 {
		return err
	}
	ids := make([]int64, 0, len(menus)-1)
	for index := len(menus) - 1; index >= 1; index-- {
		ids = append(ids, menus[index].ID)
	}
	if len(ids) > 0 {
		err = c.baseMenuCase.DeleteByIDs(ctx, ids)
		if err != nil {
			return err
		}
		err = c.baseMenuCase.casbinRuleCase.DeleteCasbinRuleByMenuIDs(ctx, ids)
		if err != nil {
			return err
		}
	}
	query := c.baseMenuCase.Query(ctx).BaseMenu
	childCount, err := c.baseMenuCase.Count(ctx, repository.Where(query.ParentID.Eq(menus[0].ID)))
	if err != nil {
		return err
	}
	if childCount > 0 {
		err = c.baseMenuCase.Update(
			ctx,
			&models.BaseMenu{ID: menus[0].ID, Status: _const.STATUS_DISABLE, API: "[]"},
			repository.Where(query.ID.Eq(menus[0].ID)),
			repository.Select(query.Status, query.API),
		)
		if err != nil {
			return err
		}
		return c.baseMenuCase.casbinRuleCase.RebuildCasbinRuleByMenuID(ctx, menus[0].ID)
	}
	err = c.baseMenuCase.DeleteByIDs(ctx, []int64{menus[0].ID})
	if err != nil {
		return err
	}
	return c.baseMenuCase.casbinRuleCase.DeleteCasbinRuleByMenuIDs(ctx, []int64{menus[0].ID})
}

// validateCodeGenRestoreFiles 确认工作区仍是生成后的快照，避免覆盖人工修改。
func validateCodeGenRestoreFiles(files []codeGenRestoreFile) error {
	for _, file := range files {
		fullPath, err := codegen.SafeRepoFilePath(file.Path)
		if err != nil {
			return err
		}
		current, readErr := os.ReadFile(fullPath)
		currentExists := readErr == nil
		if readErr != nil && !os.IsNotExist(readErr) {
			return readErr
		}
		if currentExists != file.GeneratedExists || currentExists && string(current) != file.GeneratedContent {
			return errorsx.StateConflict("生成文件已被修改，无法安全还原", "code_gen_file", file.Path, "generated")
		}
	}
	return nil
}

// validateCodeGenRestoreBatch 校验批量生成共享文件必须整批还原。
func validateCodeGenRestoreBatch(ids []int64, manifests map[int64]*codeGenRestoreManifest) error {
	selected := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		selected[id] = struct{}{}
	}
	for _, manifest := range manifests {
		for _, batchID := range manifest.BatchTableIDs {
			if _, ok := selected[batchID]; !ok {
				return errorsx.StateConflict("批量生成的共享文件必须整批还原", "code_gen_task", manifest.TaskID, "restore")
			}
		}
	}
	return nil
}

// normalizeCodeGenTableIDs 去重并校验代码生成表配置ID。
func normalizeCodeGenTableIDs(tableIDs []int64) ([]int64, error) {
	ids := make([]int64, 0, len(tableIDs))
	seen := make(map[int64]struct{}, len(tableIDs))
	for _, tableID := range tableIDs {
		if tableID <= 0 {
			return nil, errorsx.InvalidArgument("代码生成表配置ID必须大于0")
		}
		if _, ok := seen[tableID]; ok {
			continue
		}
		seen[tableID] = struct{}{}
		ids = append(ids, tableID)
	}
	if len(ids) == 0 {
		return nil, errorsx.InvalidArgument("请选择还原的代码生成表")
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

// codeGenRestoreManifestPath 返回代码生成还原快照路径。
func codeGenRestoreManifestPath(tableID int64) string {
	return filepath.ToSlash(filepath.Join(codeGenRestoreRoot, fmt.Sprintf("%d.json", tableID)))
}

// loadCodeGenRestoreManifest 加载单个代码生成还原快照。
func loadCodeGenRestoreManifest(tableID int64) (*codeGenRestoreManifest, error) {
	path, err := codegen.SafeRepoFilePath(codeGenRestoreManifestPath(tableID))
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, errorsx.StateConflict("代码生成还原快照不存在，请重新生成", "code_gen_table", fmt.Sprint(tableID), "restore")
	}
	if err != nil {
		return nil, err
	}
	manifest := new(codeGenRestoreManifest)
	if err = json.Unmarshal(content, manifest); err != nil {
		return nil, errorsx.Internal("代码生成还原快照格式错误").WithCause(err)
	}
	if manifest.TableID != tableID || manifest.Version != 1 {
		return nil, errorsx.StateConflict("代码生成还原快照已失效，请重新生成", "code_gen_table", fmt.Sprint(tableID), "restore")
	}
	return manifest, nil
}

// newCodeGenRestoreTransaction 创建还原快照文件事务。
func newCodeGenRestoreTransaction(paths []string) (*codeGenRestoreTransaction, error) {
	transaction := &codeGenRestoreTransaction{snapshots: make(map[string]codeGenFileSnapshot, len(paths))}
	for _, path := range paths {
		fullPath, err := codegen.SafeRepoFilePath(path)
		if err != nil {
			return nil, err
		}
		if _, exists := transaction.snapshots[fullPath]; exists {
			continue
		}
		fileInfo, statErr := os.Stat(fullPath)
		if os.IsNotExist(statErr) {
			transaction.snapshots[fullPath] = codeGenFileSnapshot{fullPath: fullPath}
			continue
		}
		if statErr != nil {
			return nil, statErr
		}
		content, readErr := os.ReadFile(fullPath)
		if readErr != nil {
			return nil, readErr
		}
		transaction.snapshots[fullPath] = codeGenFileSnapshot{fullPath: fullPath, exists: true, content: content, mode: fileInfo.Mode().Perm()}
	}
	return transaction, nil
}

// record 记录还原快照文件写入。
func (t *codeGenRestoreTransaction) record(path string) {
	if t == nil || t.committed {
		return
	}
	fullPath, err := codegen.SafeRepoFilePath(path)
	if err == nil {
		t.writtenPaths = append(t.writtenPaths, fullPath)
	}
}

// rollback 回滚还原快照文件变更。
func (t *codeGenRestoreTransaction) rollback() error {
	if t == nil || t.committed {
		return nil
	}
	var rollbackErr error
	for index := len(t.writtenPaths) - 1; index >= 0; index-- {
		snapshot := t.snapshots[t.writtenPaths[index]]
		var err error
		if snapshot.exists {
			err = writeGeneratedFileAtomically(snapshot.fullPath, snapshot.content, snapshot.mode, false)
		} else {
			err = os.Remove(snapshot.fullPath)
			if os.IsNotExist(err) {
				err = nil
			}
		}
		if err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
		}
	}
	return rollbackErr
}

// commit 提交还原快照文件事务。
func (t *codeGenRestoreTransaction) commit() {
	if t != nil {
		t.committed = true
		t.writtenPaths = nil
	}
}

// writeCodeGenRestoreFile 原子写入还原快照文件。
func writeCodeGenRestoreFile(path string, content []byte) error {
	return writeGeneratedFileAtomically(path, content, 0o644, false)
}

// codeGenWorkspacePaths 收集生成命令可能改写的源代码路径。
func codeGenWorkspacePaths() (map[string]struct{}, error) {
	paths := make(map[string]struct{})
	backendPath, err := codegen.SafeRepoFilePath("backend")
	if err != nil {
		return nil, err
	}
	rootPath := filepath.Dir(backendPath)
	roots := []string{
		filepath.Join(rootPath, "backend"),
		filepath.Join(rootPath, "frontend/admin/src/api"),
		filepath.Join(rootPath, "frontend/admin/src/rpc"),
		filepath.Join(rootPath, "frontend/admin/src/views"),
	}
	for _, root := range roots {
		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if path == filepath.Join(rootPath, "backend/data") || path == filepath.Join(rootPath, "backend/logs") {
					return filepath.SkipDir
				}
				return nil
			}
			if !isCodeGenWorkspaceFile(path) {
				return nil
			}
			relative, relErr := filepath.Rel(rootPath, path)
			if relErr != nil {
				return relErr
			}
			paths[filepath.ToSlash(relative)] = struct{}{}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return paths, nil
}

// isCodeGenWorkspaceFile 判断文件是否属于生成命令可能改写的源代码产物。
func isCodeGenWorkspaceFile(path string) bool {
	if filepath.Base(path) == "openapi.yaml" && strings.Contains(filepath.ToSlash(path), "/backend/internal/cmd/server/assets/") {
		return true
	}
	switch filepath.Ext(path) {
	case ".go", ".proto", ".ts", ".vue":
		return true
	default:
		return false
	}
}

// readCodeGenWorkspaceFile 读取仓库相对路径文件状态。
func readCodeGenWorkspaceFile(path string) (codeGenRestoreWorkspaceFile, error) {
	fullPath, err := codegen.SafeRepoFilePath(path)
	if err != nil {
		return codeGenRestoreWorkspaceFile{}, err
	}
	fileInfo, statErr := os.Stat(fullPath)
	if os.IsNotExist(statErr) {
		return codeGenRestoreWorkspaceFile{Path: path}, nil
	}
	if statErr != nil {
		return codeGenRestoreWorkspaceFile{}, statErr
	}
	content, readErr := os.ReadFile(fullPath)
	if readErr != nil {
		return codeGenRestoreWorkspaceFile{}, readErr
	}
	return codeGenRestoreWorkspaceFile{Path: path, Exists: true, Content: content, Mode: fileInfo.Mode().Perm()}, nil
}

// workspaceFilesEqual 判断两个工作区文件快照是否一致。
func workspaceFilesEqual(left codeGenRestoreWorkspaceFile, right codeGenRestoreWorkspaceFile) bool {
	return left.Exists == right.Exists && (!left.Exists || reflect.DeepEqual(left.Content, right.Content))
}

// appendUniqueInt64 向ID切片追加未出现的值。
func appendUniqueInt64(values []int64, value int64) []int64 {
	for _, item := range values {
		if item == value {
			return values
		}
	}
	return append(values, value)
}
