package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	systemadminv1 "shop/api/gen/go/system/admin/v1"
	coreBiz "shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/service/system/admin/codegen"
	"shop/service/system/admin/dto"

	"github.com/liujitcn/go-utils/stringcase"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gen/field"
)

var codeGenGenerationProcessLock sync.Mutex

// codeGenProgressReporter 将单个生成对象的执行步骤写入内存任务。
type codeGenProgressReporter struct {
	manager *codegen.Manager // 内存任务管理器
	taskID  string           // 批量生成任务ID
	tableID int64            // 当前生成对象ID
}

// codeGenCommandTarget 描述批量生成命令关联的业务表和进度上报器。
type codeGenCommandTarget struct {
	tableID   int64
	tableName string
	progress  *codeGenProgressReporter
}

// codeGenCommandResult 保存单个业务表在共享命令链中的最终结果。
type codeGenCommandResult struct {
	message string
	err     error
}

// codeGenBatchContext 保存批次预检后的生成计划和字段快照。
type codeGenBatchContext struct {
	plan           *codegen.BatchGeneration
	columnsByTable map[int64][]*codegen.CodeGenColumn
}

// CodeGenCase 管理代码预览、批量生成与任务进度。
type CodeGenCase struct {
	*coreBiz.BaseCase
	tx                data.Transaction
	codeGenTableCase  *CodeGenTableCase
	codeGenColumnCase *CodeGenColumnCase
	codeGenProtoCase  *CodeGenProtoCase
	baseMenuCase      *BaseMenuCase
	progressManager   *codegen.Manager
}

// NewCodeGenCase 创建代码生成执行业务实例。
func NewCodeGenCase(
	baseCase *coreBiz.BaseCase,
	tx data.Transaction,
	codeGenTableCase *CodeGenTableCase,
	codeGenColumnCase *CodeGenColumnCase,
	codeGenProtoCase *CodeGenProtoCase,
	baseMenuCase *BaseMenuCase,
	progressManager *codegen.Manager,
) *CodeGenCase {
	return &CodeGenCase{
		BaseCase:          baseCase,
		tx:                tx,
		codeGenTableCase:  codeGenTableCase,
		codeGenColumnCase: codeGenColumnCase,
		codeGenProtoCase:  codeGenProtoCase,
		baseMenuCase:      baseMenuCase,
		progressManager:   progressManager,
	}
}

// GetCodeGenTask 查询当前用户可访问的生成任务快照。
func (c *CodeGenCase) GetCodeGenTask(ctx context.Context, taskID string) (*systemadminv1.CodeGenTask, error) {
	if taskID == "" {
		return nil, errorsx.InvalidArgument("生成任务ID不能为空")
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	task, ok := c.progressManager.Snapshot(taskID, authInfo.UserId)
	if !ok {
		return nil, errorsx.ResourceNotFound("代码生成任务不存在或已过期")
	}
	return task, nil
}

// PreviewCodeGen 根据现有表、字段和Proto配置预览生成结果。
func (c *CodeGenCase) PreviewCodeGen(ctx context.Context, tableID int64, requestedPaths *systemadminv1.CodeGenOutputPaths) (*systemadminv1.PreviewCodeGenResponse, error) {
	table, columns, protos, err := c.loadCodeGenContext(ctx, tableID)
	if err != nil {
		return nil, err
	}
	var generation *codegen.Generation
	generation, err = codegen.PrepareGeneration(table, columns, protos, requestedPaths, table.TableComment)
	if err != nil {
		return nil, err
	}
	if err = c.validateGeneratedOptionMethods(ctx, generation.Table, columns, generation.GeneratedMethods); err != nil {
		return nil, err
	}
	if err = c.validateCodeGenParentMenu(ctx, generation.Table.ParentMenuID); err != nil {
		return nil, err
	}
	return &systemadminv1.PreviewCodeGenResponse{Files: generation.Files, OutputPaths: generation.OutputPaths}, nil
}

// StartCodeGenTask 校验生成对象并创建后台批量任务。
func (c *CodeGenCase) StartCodeGenTask(ctx context.Context, req *systemadminv1.StartCodeGenTaskRequest) (*systemadminv1.StartCodeGenTaskResponse, error) {
	if len(req.GetTableIds()) == 0 {
		return nil, errorsx.InvalidArgument("请选择生成对象")
	}
	if len(req.GetTableIds()) > 1 && req.GetOutputPaths() != nil {
		return nil, errorsx.InvalidArgument("批量生成不支持自定义输出路径")
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var batch *codeGenBatchContext
	batch, err = c.prepareCodeGenBatch(ctx, req.GetTableIds(), req.GetOutputPaths())
	if err != nil {
		return nil, err
	}
	tables := make([]*systemadminv1.CodeGenTaskTable, 0, len(req.GetTableIds()))
	for _, tableID := range req.GetTableIds() {
		generation := batch.plan.GenerationForTable(tableID)
		if generation == nil || generation.Table == nil {
			return nil, errorsx.Internal("批量生成计划缺少表配置")
		}
		tables = append(tables, &systemadminv1.CodeGenTaskTable{
			TableId:   tableID,
			TableName: generation.Table.TableName_,
			Status:    systemadminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_PENDING,
			Message:   "等待执行",
			Steps:     codegen.BuildProgressSteps(generation.Files, codegen.ShouldSyncMenus(generation.Table, generation.GeneratedMethods), req.GetRunCommands() && generation.Table.GenBackend == 1),
		})
	}
	task, created := c.progressManager.Create(authInfo.UserId, tables)
	if !created {
		return nil, errorsx.StateConflict("已有代码生成任务正在执行", "code_gen_task", "running", "completed")
	}
	go c.runCodeGenTask(context.WithoutCancel(ctx), task.GetTaskId(), req.GetTableIds(), req.GetRunCommands(), req.GetOutputPaths())
	return &systemadminv1.StartCodeGenTaskResponse{TaskId: task.GetTaskId()}, nil
}

// runCodeGenTask 串行执行批量任务并汇总最终状态。
func (c *CodeGenCase) runCodeGenTask(
	ctx context.Context,
	taskID string,
	tableIDs []int64,
	runCommands bool,
	requestedPaths *systemadminv1.CodeGenOutputPaths,
) {
	// 文件、生成产物和格式化都会改写共享工作树，整批任务必须串行执行。
	codeGenGenerationProcessLock.Lock()
	defer codeGenGenerationProcessLock.Unlock()
	c.progressManager.MarkTaskRunning(ctx, taskID)

	batch, err := c.prepareCodeGenBatch(ctx, tableIDs, requestedPaths)
	if err != nil {
		c.failCodeGenTask(ctx, taskID, tableIDs, err)
		return
	}
	reporters := make(map[int64]*codeGenProgressReporter, len(tableIDs))
	for _, tableID := range tableIDs {
		generation := batch.plan.GenerationForTable(tableID)
		if generation == nil || generation.Table == nil {
			c.failCodeGenTask(ctx, taskID, tableIDs, errorsx.Internal("批量生成计划缺少表配置"))
			return
		}
		reporter := &codeGenProgressReporter{manager: c.progressManager, taskID: taskID, tableID: tableID}
		reporters[tableID] = reporter
		c.progressManager.MarkTableRunning(ctx, taskID, tableID)
		c.progressManager.RegisterSteps(ctx, taskID, tableID, codegen.BuildProgressSteps(generation.Files, codegen.ShouldSyncMenus(generation.Table, generation.GeneratedMethods), runCommands && generation.Table.GenBackend == 1))
	}
	workflowCtx, cancelWorkflow := context.WithTimeout(context.WithoutCancel(ctx), codegen.WorkflowTimeout)
	defer cancelWorkflow()
	if err = c.writeCodeGenBatchFiles(workflowCtx, batch.plan, reporters); err != nil {
		c.failCodeGenTask(ctx, taskID, tableIDs, err)
		return
	}

	failedCount := 0
	commandTargets := make([]codeGenCommandTarget, 0, len(tableIDs))
	for _, tableID := range tableIDs {
		generation := batch.plan.GenerationForTable(tableID)
		reporter := reporters[tableID]
		if codegen.ShouldSyncMenus(generation.Table, generation.GeneratedMethods) {
			reporter.updateStep(workflowCtx, codegen.MenuStepID, systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_RUNNING, "正在同步", "")
			err = c.syncGeneratedMenus(workflowCtx, generation.Table, batch.columnsByTable[tableID], generation.GeneratedMethods, codegen.FrontendPageComponentPath(generation.OutputPaths.GetFrontendPageFilePath()))
			if err != nil {
				failedCount++
				reporter.updateStep(workflowCtx, codegen.MenuStepID, systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_FAILED, err.Error(), "")
				c.progressManager.MarkTableCompleted(ctx, taskID, tableID, systemadminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_FAILED, codegen.FailureRemark(err))
				continue
			}
			reporter.updateStep(workflowCtx, codegen.MenuStepID, systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_SUCCEEDED, "同步完成", "")
		}
		if runCommands && generation.Table.GenBackend == 1 {
			commandTargets = append(commandTargets, codeGenCommandTarget{tableID: tableID, tableName: generation.Table.TableName_, progress: reporter})
			continue
		}
		// 仅在该表全部生成步骤成功后更新持久化状态，失败任务保留原状态便于再次执行。
		err = c.markCodeGenTableGenerated(workflowCtx, tableID)
		if err != nil {
			failedCount++
			c.progressManager.MarkTableCompleted(ctx, taskID, tableID, systemadminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_FAILED, "更新生成状态失败："+codegen.FailureRemark(err))
			continue
		}
		c.progressManager.MarkTableCompleted(ctx, taskID, tableID, systemadminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_SUCCEEDED, "生成完成")
	}
	if len(commandTargets) > 0 {
		commandResults := c.runCodeGenCommands(workflowCtx, commandTargets)
		for _, target := range commandTargets {
			result := commandResults[target.tableID]
			if result.err != nil {
				failedCount++
				c.progressManager.MarkTableCompleted(ctx, taskID, target.tableID, systemadminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_FAILED, result.message)
				continue
			}
			// 共享命令链成功后再更新状态，确保列表不会提前显示为已生成。
			err = c.markCodeGenTableGenerated(workflowCtx, target.tableID)
			if err != nil {
				failedCount++
				c.progressManager.MarkTableCompleted(ctx, taskID, target.tableID, systemadminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_FAILED, "更新生成状态失败："+codegen.FailureRemark(err))
				continue
			}
			c.progressManager.MarkTableCompleted(ctx, taskID, target.tableID, systemadminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_SUCCEEDED, "生成完成")
		}
	}
	if failedCount > 0 {
		message := fmt.Sprintf("生成完成，成功 %d 个，失败 %d 个", len(tableIDs)-failedCount, failedCount)
		c.progressManager.MarkTaskCompleted(ctx, taskID, systemadminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_FAILED, message)
		return
	}
	c.progressManager.MarkTaskCompleted(ctx, taskID, systemadminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_SUCCEEDED, fmt.Sprintf("生成完成，共 %d 个", len(tableIDs)))
}

// prepareCodeGenBatch 加载并校验整批生成快照，所有文件冲突均在写入前返回。
func (c *CodeGenCase) prepareCodeGenBatch(ctx context.Context, tableIDs []int64, requestedPaths *systemadminv1.CodeGenOutputPaths) (*codeGenBatchContext, error) {
	inputs := make([]codegen.BatchGenerationInput, 0, len(tableIDs))
	columnsByTable := make(map[int64][]*codegen.CodeGenColumn, len(tableIDs))
	tableIDSet := make(map[int64]struct{}, len(tableIDs))
	for _, tableID := range tableIDs {
		if tableID <= 0 {
			return nil, errorsx.InvalidArgument("代码生成表配置ID不能为空")
		}
		if _, exists := tableIDSet[tableID]; exists {
			return nil, errorsx.InvalidArgument("代码生成表配置ID不能重复")
		}
		tableIDSet[tableID] = struct{}{}
		table, columns, protos, err := c.loadCodeGenContext(ctx, tableID)
		if err != nil {
			return nil, err
		}
		// 停用配置只允许查看，不能写入生成文件。
		if table.Status == codegen.StatusDisabled {
			return nil, errorsx.StateConflict("停用的代码生成表配置不能生成", "code_gen_table", "disabled", "draft_or_generated")
		}
		if !codeGenDatabaseTableNamePattern.MatchString(table.TableName_) {
			return nil, errorsx.InvalidArgument("业务表名只能包含小写字母、数字和下划线，且必须以字母开头")
		}
		inputs = append(inputs, codegen.BatchGenerationInput{
			Table:          table,
			Columns:        columns,
			Methods:        protos,
			RequestedPaths: requestedPaths,
			TableComment:   table.TableComment,
		})
		columnsByTable[tableID] = columns
	}
	plan, err := codegen.PrepareBatchGeneration(inputs)
	if err != nil {
		return nil, errorsx.InvalidArgument(codegen.FailureRemark(err)).WithCause(err)
	}
	for _, generation := range plan.Generations {
		if err = c.validateGeneratedOptionMethods(ctx, generation.Table, columnsByTable[generation.Table.ID], generation.GeneratedMethods); err != nil {
			return nil, err
		}
		if err = c.validateCodeGenParentMenu(ctx, generation.Table.ParentMenuID); err != nil {
			return nil, err
		}
		for _, file := range generation.Files {
			if strings.Contains(file.GetMessage(), "文件路径不允许") {
				return nil, errorsx.InvalidArgument(file.GetMessage())
			}
		}
	}
	for _, file := range plan.Files {
		if _, err = codegen.SafeRepoFilePath(file.Path); err != nil {
			return nil, err
		}
	}
	return &codeGenBatchContext{plan: plan, columnsByTable: columnsByTable}, nil
}

// writeCodeGenBatchFiles 将批次合并后的每个文件原子写入一次，并同步全部来源步骤。
func (c *CodeGenCase) writeCodeGenBatchFiles(ctx context.Context, plan *codegen.BatchGeneration, reporters map[int64]*codeGenProgressReporter) error {
	for _, file := range plan.Files {
		for _, ref := range file.Refs {
			reporter := reporters[ref.TableID]
			if reporter != nil {
				reporter.updateStep(ctx, codegen.FileStepID(ref.FileIndex), systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_RUNNING, "正在合并写入", "")
			}
		}
		fullPath, err := codegen.SafeRepoFilePath(file.Path)
		if err == nil {
			err = writeGeneratedFile(fullPath, []byte(file.Content), file.Action)
		}
		for _, ref := range file.Refs {
			reporter := reporters[ref.TableID]
			if reporter == nil {
				continue
			}
			if err != nil {
				reporter.updateStep(ctx, codegen.FileStepID(ref.FileIndex), systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_FAILED, err.Error(), "")
				continue
			}
			reporter.updateStep(ctx, codegen.FileStepID(ref.FileIndex), systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_SUCCEEDED, "已合并写入", "")
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// failCodeGenTask 将整批预检或写入失败同步到任务和每个表的进度状态。
func (c *CodeGenCase) failCodeGenTask(ctx context.Context, taskID string, tableIDs []int64, err error) {
	message := codegen.FailureRemark(err)
	for _, tableID := range tableIDs {
		c.progressManager.MarkTableCompleted(ctx, taskID, tableID, systemadminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_FAILED, message)
	}
	c.progressManager.MarkTaskCompleted(ctx, taskID, systemadminv1.CodeGenTaskStatus_CODE_GEN_TASK_STATUS_FAILED, "批量生成失败："+message)
}

// markCodeGenTableGenerated 将完整生成成功的代码生成表标记为已生成。
func (c *CodeGenCase) markCodeGenTableGenerated(ctx context.Context, tableID int64) error {
	return c.codeGenTableCase.UpdateByID(ctx, &models.CodeGenTable{ID: tableID, Status: codegen.StatusGenerated})
}

// runCodeGenCommands 对选中业务表执行单表模型生成，并整批执行一次共享生成链。
func (c *CodeGenCase) runCodeGenCommands(ctx context.Context, targets []codeGenCommandTarget) map[int64]codeGenCommandResult {
	type commandState struct {
		failureMessages []string
		err             error
	}

	backendDir := codegen.BackendDir()
	databaseSource := c.GetConfig().GetData().GetDatabase().GetSource()
	states := make(map[int64]*commandState, len(targets))
	sharedTargets := []string{"api", "openapi", "ts", "wire"}
	eligibleTargets := make([]codeGenCommandTarget, 0, len(targets))
	for _, target := range targets {
		state := new(commandState)
		states[target.tableID] = state
		stepID := codegen.CommandStepPrefix + "gorm-gen"
		target.progress.updateStep(ctx, stepID, systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_RUNNING, "正在执行", "")
		if databaseSource == "" {
			state.err = errors.New("代码生成数据库配置缺少数据源")
			failureMessage := codegen.CommandFailureMessage("gorm-gen", "", state.err)
			state.failureMessages = append(state.failureMessages, failureMessage)
			target.progress.updateStep(ctx, stepID, systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_FAILED, failureMessage, "")
			for _, skippedTarget := range sharedTargets {
				target.progress.updateStep(ctx, codegen.CommandStepPrefix+skippedTarget, systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_SKIPPED, "模型生成失败", "")
			}
			continue
		}
		output, err := codegen.RunCommand(ctx, backendDir, "gorm-gen", "GORM_TABLE="+target.tableName, "GORM_GEN_SOURCE="+databaseSource)
		if err != nil {
			failureMessage := codegen.CommandFailureMessage("gorm-gen", output, err)
			state.failureMessages = append(state.failureMessages, failureMessage)
			state.err = err
			target.progress.updateStep(ctx, stepID, systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_FAILED, failureMessage, output)
			for _, skippedTarget := range sharedTargets {
				target.progress.updateStep(ctx, codegen.CommandStepPrefix+skippedTarget, systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_SKIPPED, "模型生成失败", "")
			}
			continue
		}
		target.progress.updateStep(ctx, stepID, systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_SUCCEEDED, "执行完成", output)
		eligibleTargets = append(eligibleTargets, target)
	}

	for index, commandTarget := range sharedTargets {
		if len(eligibleTargets) == 0 {
			break
		}
		stepID := codegen.CommandStepPrefix + commandTarget
		for _, target := range eligibleTargets {
			target.progress.updateStep(ctx, stepID, systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_RUNNING, "正在执行", "")
		}
		output, err := codegen.RunCommand(ctx, backendDir, commandTarget)
		if err != nil {
			failureMessage := codegen.CommandFailureMessage(commandTarget, output, err)
			for _, target := range eligibleTargets {
				state := states[target.tableID]
				state.failureMessages = append(state.failureMessages, failureMessage)
				state.err = err
				target.progress.updateStep(ctx, stepID, systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_FAILED, failureMessage, output)
				for _, skippedTarget := range sharedTargets[index+1:] {
					target.progress.updateStep(ctx, codegen.CommandStepPrefix+skippedTarget, systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_SKIPPED, "前序命令执行失败", "")
				}
			}
			break
		}
		for _, target := range eligibleTargets {
			target.progress.updateStep(ctx, stepID, systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_SUCCEEDED, "执行完成", output)
		}
	}

	formatCtx, cancelFormat := context.WithTimeout(context.WithoutCancel(ctx), codegen.FormatTimeout)
	defer cancelFormat()
	formatStepID := codegen.CommandStepPrefix + "fmt"
	for _, target := range targets {
		target.progress.updateStep(formatCtx, formatStepID, systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_RUNNING, "正在执行", "")
	}
	fmtOutput, fmtErr := codegen.RunCommand(formatCtx, backendDir, "fmt")
	for _, target := range targets {
		state := states[target.tableID]
		if fmtErr != nil {
			failureMessage := codegen.CommandFailureMessage("fmt", fmtOutput, fmtErr)
			state.failureMessages = append(state.failureMessages, failureMessage)
			target.progress.updateStep(formatCtx, formatStepID, systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_FAILED, failureMessage, fmtOutput)
			if state.err == nil {
				state.err = fmtErr
			} else {
				state.err = fmt.Errorf("%w；make fmt: %v", state.err, fmtErr)
			}
			continue
		}
		target.progress.updateStep(formatCtx, formatStepID, systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_SUCCEEDED, "执行完成", fmtOutput)
	}

	results := make(map[int64]codeGenCommandResult, len(states))
	for tableID, state := range states {
		result := codeGenCommandResult{err: state.err}
		if state.err != nil {
			result.message = codegen.TruncateText(strings.Join(state.failureMessages, "；"), codegen.RemarkMaxRunes)
		}
		results[tableID] = result
	}
	return results
}

// loadCodeGenContext 只读加载表、字段和Proto生成配置快照。
func (c *CodeGenCase) loadCodeGenContext(ctx context.Context, tableID int64) (*codegen.Table, []*codegen.CodeGenColumn, []*codegen.Proto, error) {
	if tableID <= 0 {
		return nil, nil, nil, errorsx.InvalidArgument("代码生成表配置ID不能为空")
	}
	tableModel, err := c.codeGenTableCase.FindByID(ctx, tableID)
	if err != nil {
		return nil, nil, nil, err
	}
	var table *codegen.Table
	table, err = codeGenTableToSnapshot(tableModel)
	if err != nil {
		return nil, nil, nil, err
	}
	var columnConfigs []*systemadminv1.CodeGenColumn
	columnConfigs, err = c.codeGenColumnCase.listCodeGenColumns(ctx, tableID)
	if err != nil {
		return nil, nil, nil, err
	}
	var databaseColumns []dto.CodeGenDatabaseColumn
	databaseColumns, err = c.codeGenColumnCase.listDatabaseColumns(ctx, tableModel.Name)
	if err != nil {
		return nil, nil, nil, err
	}
	columns := codeGenColumnsToSnapshots(columnConfigs, databaseColumns)
	query := c.codeGenProtoCase.Query(ctx).CodeGenProto
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.TableID.Eq(tableID)))
	opts = append(opts, repository.Order(query.Sort.Asc()))
	var protoModels []*models.CodeGenProto
	protoModels, err = c.codeGenProtoCase.List(ctx, opts...)
	if err != nil {
		return nil, nil, nil, err
	}
	var protos []*codegen.Proto
	protos, err = codeGenProtosToSnapshots(protoModels)
	if err != nil {
		return nil, nil, nil, err
	}
	if err = c.loadProtoTargetBusinessNames(ctx, table, protos); err != nil {
		return nil, nil, nil, err
	}
	return table, columns, protos, nil
}

// loadProtoTargetBusinessNames 从现有代码生成表配置补充外部接口目标描述。
func (c *CodeGenCase) loadProtoTargetBusinessNames(ctx context.Context, table *codegen.Table, protos []*codegen.Proto) error {
	tableNameByEntity := make(map[string]string)
	tableNames := make([]string, 0)
	for _, proto := range protos {
		if proto.TargetEntityName == "" || proto.TargetEntityName == table.EntityName {
			continue
		}
		if _, exists := tableNameByEntity[proto.TargetEntityName]; exists {
			continue
		}
		tableName := stringcase.ToSnakeCase(proto.TargetEntityName)
		tableNameByEntity[proto.TargetEntityName] = tableName
		tableNames = append(tableNames, tableName)
	}
	if len(tableNames) == 0 {
		return nil
	}
	query := c.codeGenTableCase.Query(ctx).CodeGenTable
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.Name.In(tableNames...)))
	tableConfigs, err := c.codeGenTableCase.List(ctx, opts...)
	if err != nil {
		return err
	}
	commentByTableName := make(map[string]string, len(tableConfigs))
	for _, tableConfig := range tableConfigs {
		commentByTableName[tableConfig.Name] = tableConfig.Comment
	}
	for _, proto := range protos {
		proto.TargetBusinessName = commentByTableName[tableNameByEntity[proto.TargetEntityName]]
	}
	return nil
}

// validateGeneratedOptionMethods 校验选项接口字段能安全写入公共响应类型。
func (c *CodeGenCase) validateGeneratedOptionMethods(ctx context.Context, table *codegen.Table, columns []*codegen.CodeGenColumn, methods []*codegen.Proto) error {
	targetColumnCache := map[string][]*codegen.CodeGenColumn{table.EntityName: columns}
	for _, method := range methods {
		if !codegen.IsOptionProtoMethod(method) {
			continue
		}
		targetColumns, err := c.optionTargetColumns(ctx, table, method, targetColumnCache)
		if err != nil {
			return err
		}
		if err = validateOptionLabelColumn(method, targetColumns, codegen.DefaultString(method.LabelColumn, "name"), "显示字段"); err != nil {
			return err
		}
		if err = validateOptionIntegerColumn(method, targetColumns, codegen.DefaultString(method.ValueColumn, "id"), "值字段"); err != nil {
			return err
		}
		if method.APIKind == codegen.APIKindTree {
			if err = validateOptionIntegerColumn(method, targetColumns, codegen.DefaultString(method.ParentColumn, "parent_id"), "父节点字段"); err != nil {
				return err
			}
		}
	}
	return nil
}

// optionTargetColumns 查询选项接口目标实体的数据库字段。
func (c *CodeGenCase) optionTargetColumns(ctx context.Context, table *codegen.Table, method *codegen.Proto, cache map[string][]*codegen.CodeGenColumn) ([]*codegen.CodeGenColumn, error) {
	target := codegen.DefaultString(method.TargetEntityName, table.EntityName)
	if target == table.EntityName {
		return cache[table.EntityName], nil
	}
	if columns, exists := cache[target]; exists {
		return columns, nil
	}
	tableName := stringcase.ToSnakeCase(target)
	leftTreeConfig := codegen.LeftTreeConfigFromTable(table)
	if method.TriggerType == codegen.TriggerLeftTree && leftTreeConfig.SourceValue != "" {
		tableName = leftTreeConfig.SourceValue
	}
	databaseColumns, err := c.codeGenColumnCase.listDatabaseColumns(ctx, tableName)
	if err != nil {
		return nil, err
	}
	if len(databaseColumns) == 0 {
		return nil, errorsx.InvalidArgument(fmt.Sprintf("选项接口%s的目标表%s不存在，无法校验字段类型", method.MethodName, tableName))
	}
	columns := codeGenColumnsToSnapshots(nil, databaseColumns)
	cache[target] = columns
	return columns, nil
}

// validateCodeGenParentMenu 校验生成页面的父级菜单节点。
func (c *CodeGenCase) validateCodeGenParentMenu(ctx context.Context, parentMenuID int64) error {
	if parentMenuID <= 0 {
		return errorsx.InvalidArgument("代码生成必须选择固定一级菜单下的父级菜单")
	}
	menu, err := c.baseMenuCase.FindByID(ctx, parentMenuID)
	if err != nil {
		return errorsx.InvalidArgument("父级菜单不存在").WithCause(err)
	}
	return validateBaseMenuChild(menu, _const.BASE_MENU_TYPE_MENU)
}

// syncGeneratedMenus 幂等同步生成页面及按钮权限菜单。
func (c *CodeGenCase) syncGeneratedMenus(ctx context.Context, table *codegen.Table, columns []*codegen.CodeGenColumn, methods []*codegen.Proto, resourcePath string) error {
	pageSpec, buttonSpecs := codegen.MenuSpecs(table, columns, methods, resourcePath, table.TableComment)
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		pageMenu, err := c.upsertGeneratedPageMenu(ctx, pageSpec)
		if err != nil {
			return err
		}
		for _, buttonSpec := range buttonSpecs {
			buttonSpec.Menu.ParentID = pageMenu.ID
			if err = c.upsertGeneratedButtonMenu(ctx, buttonSpec); err != nil {
				return err
			}
		}
		return c.disableStaleGeneratedStatusMenus(ctx, pageMenu.ID, table, buttonSpecs)
	})
}

// disableStaleGeneratedStatusMenus 停用本轮不再需要的状态按钮权限。
func (c *CodeGenCase) disableStaleGeneratedStatusMenus(ctx context.Context, pageMenuID int64, table *codegen.Table, buttonSpecs []codegen.CodeGenMenuSpec) error {
	expectedPaths := make(map[string]struct{}, len(buttonSpecs))
	for _, buttonSpec := range buttonSpecs {
		expectedPaths[buttonSpec.Menu.Path] = struct{}{}
	}
	query := c.baseMenuCase.Query(ctx).BaseMenu
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ParentID.Eq(pageMenuID)))
	opts = append(opts, repository.Where(query.Type.Eq(_const.BASE_MENU_TYPE_BUTTON)))
	menus, err := c.baseMenuCase.List(ctx, opts...)
	if err != nil {
		return err
	}
	statusPathPrefix := codegen.PermissionPrefix(table) + ":status"
	statusAPIPrefix := codegen.GeneratedRPCServicePath(table, table.EntityName) + "/Set"
	for _, menu := range menus {
		if _, exists := expectedPaths[menu.Path]; exists {
			continue
		}
		if menu.Path != statusPathPrefix && !strings.HasPrefix(menu.Path, statusPathPrefix+":") && !strings.Contains(menu.API, statusAPIPrefix) {
			continue
		}
		if menu.Status == _const.STATUS_DISABLE && menu.API == "[]" {
			continue
		}
		if err = c.baseMenuCase.UpdateByID(ctx, &models.BaseMenu{ID: menu.ID, Status: _const.STATUS_DISABLE, API: "[]"}); err != nil {
			return err
		}
		if err = c.baseMenuCase.casbinRuleCase.RebuildCasbinRuleByMenuID(ctx, menu.ID); err != nil {
			return err
		}
	}
	return nil
}

// upsertGeneratedPageMenu 创建或更新生成页面菜单。
func (c *CodeGenCase) upsertGeneratedPageMenu(ctx context.Context, spec codegen.CodeGenMenuSpec) (*models.BaseMenu, error) {
	query := c.baseMenuCase.Query(ctx).BaseMenu
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(
		query.Type.Eq(_const.BASE_MENU_TYPE_MENU),
		field.Or(
			query.Path.Eq(spec.Menu.Path),
			query.Name.Eq(spec.Menu.Name),
			query.Component.Eq(spec.Menu.Component),
		),
	))
	menus, err := c.baseMenuCase.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	if len(menus) == 0 {
		if err = c.baseMenuCase.createBaseMenu(ctx, spec.Menu); err != nil {
			return nil, err
		}
		return spec.Menu, nil
	}
	if menus[0].ParentID != spec.Menu.ParentID {
		return nil, errorsx.StateConflict("已生成菜单不能更换父级，请先删除原菜单后重新生成", "base_menu", fmt.Sprint(menus[0].ParentID), fmt.Sprint(spec.Menu.ParentID))
	}
	spec.Menu.ID = menus[0].ID
	if err = c.baseMenuCase.UpdateByID(ctx, spec.Menu); err != nil {
		return nil, err
	}
	if err = c.baseMenuCase.casbinRuleCase.RebuildCasbinRuleByMenuID(ctx, spec.Menu.ID); err != nil {
		return nil, err
	}
	return spec.Menu, nil
}

// upsertGeneratedButtonMenu 创建或更新生成按钮权限菜单。
func (c *CodeGenCase) upsertGeneratedButtonMenu(ctx context.Context, spec codegen.CodeGenMenuSpec) error {
	query := c.baseMenuCase.Query(ctx).BaseMenu
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ParentID.Eq(spec.Menu.ParentID)))
	opts = append(opts, repository.Where(query.Type.Eq(_const.BASE_MENU_TYPE_BUTTON)))
	menus, err := c.baseMenuCase.List(ctx, opts...)
	if err != nil {
		return err
	}
	for _, menu := range menus {
		if menu.Path != spec.Menu.Path && menu.API != spec.Menu.API {
			continue
		}
		spec.Menu.ID = menu.ID
		if err = c.baseMenuCase.UpdateByID(ctx, spec.Menu); err != nil {
			return err
		}
		return c.baseMenuCase.casbinRuleCase.RebuildCasbinRuleByMenuID(ctx, spec.Menu.ID)
	}
	return c.baseMenuCase.createBaseMenu(ctx, spec.Menu)
}

// updateStep 更新当前生成对象的单个执行步骤。
func (r *codeGenProgressReporter) updateStep(ctx context.Context, stepID string, status systemadminv1.CodeGenTaskStepStatus, message string, output string) {
	if r == nil {
		return
	}
	r.manager.UpdateStep(ctx, r.taskID, r.tableID, stepID, status, message, output)
}

// codeGenTableToSnapshot 将现有表配置转换为生成器只读快照。
func codeGenTableToSnapshot(item *models.CodeGenTable) (*codegen.Table, error) {
	leftTreeConfig := ""
	if item.LeftTreeConfig != "" {
		var config systemadminv1.CodeGenLeftTreeConfig
		err := json.Unmarshal([]byte(item.LeftTreeConfig), &config)
		if err != nil {
			return nil, errorsx.Internal("左树配置格式错误").WithCause(err)
		}
		var data []byte
		data, err = json.Marshal(codegen.CodeGenLeftTreeConfig{
			Enabled:      item.PageType == codegen.PageTypeLeftTree,
			SourceType:   codegen.OptionSourceTable,
			SourceValue:  config.GetTableName(),
			FilterColumn: config.GetFilterColumn(),
			ParentColumn: config.GetParentColumn(),
			LabelColumn:  config.GetLabelColumn(),
			ValueColumn:  config.GetValueColumn(),
		})
		if err != nil {
			return nil, errorsx.Internal("转换左树配置失败").WithCause(err)
		}
		leftTreeConfig = string(data)
	}
	return &codegen.Table{
		ID:               item.ID,
		TableName_:       item.Name,
		TableComment:     item.Comment,
		BusinessName:     codegen.DefaultString(item.Comment, item.BusinessName),
		EntityName:       item.EntityName,
		ModulePath:       item.ModulePath,
		APIPath:          item.APIPath,
		PermissionPrefix: item.PermissionPrefix,
		ParentMenuID:     item.ParentMenuID,
		PageType:         item.PageType,
		ParentColumn:     item.ParentColumn,
		TreeLabelColumn:  item.TreeLabelColumn,
		LeftTreeConfig:   leftTreeConfig,
		GenBackend:       item.GenBackend,
		GenFrontend:      item.GenFrontend,
		GenSql:           item.GenSql,
		Status:           item.Status,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}, nil
}

// codeGenColumnsToSnapshots 将完整字段配置与数据库元数据合并为生成快照。
func codeGenColumnsToSnapshots(configs []*systemadminv1.CodeGenColumn, databaseColumns []dto.CodeGenDatabaseColumn) []*codegen.CodeGenColumn {
	configByName := make(map[string]*systemadminv1.CodeGenColumn, len(configs))
	for _, config := range configs {
		configByName[config.GetColumnName()] = config
	}
	columns := make([]*codegen.CodeGenColumn, 0, len(databaseColumns))
	for index, databaseColumn := range databaseColumns {
		config := configByName[databaseColumn.ColumnName]
		if config == nil {
			config = newDefaultCodeGenColumn(0, databaseColumn, int32(index+1))
		}
		queryConfig := config.GetQueryConfig()
		listConfig := config.GetListConfig()
		formConfig := config.GetFormConfig()
		queryOption := codeGenOptionToSnapshot(queryConfig.GetOption())
		listOption := codeGenOptionToSnapshot(listConfig.GetOption())
		formOption := codeGenOptionToSnapshot(formConfig.GetOption())
		isStatus := listConfig.GetEnabled() && listConfig.GetComponent() == "switch"
		defaultValue := ""
		hasDefault := databaseColumn.ColumnDefault.Valid
		if hasDefault {
			defaultValue = databaseColumn.ColumnDefault.String
		}
		column := &codegen.CodeGenColumn{
			ID:                  config.GetId(),
			TableID:             config.GetTableId(),
			ColumnName:          databaseColumn.ColumnName,
			ColumnComment:       config.GetColumnComment(),
			DbType:              databaseColumn.DataType,
			ColumnType:          databaseColumn.ColumnType,
			DbLength:            config.GetDbLength(),
			DbScale:             config.GetDbScale(),
			DefaultValue:        defaultValue,
			HasDefault:          hasDefault,
			Extra:               databaseColumn.Extra,
			IsPrimary:           codegen.BoolToInt32(config.GetIsPrimary()),
			IsAutoIncrement:     codegen.BoolToInt32(config.GetIsAutoIncrement()),
			IsNullable:          codegen.BoolToInt32(config.GetIsNullable()),
			GoType:              config.GetGoType(),
			ProtoType:           config.GetProtoType(),
			TsType:              config.GetTsType(),
			IsQuery:             codegen.BoolToInt32(queryConfig.GetEnabled()),
			QueryOperator:       codegen.NormalizeQueryOperator(queryConfig.GetOperator()),
			QueryComponent:      queryConfig.GetComponent(),
			IsList:              codegen.BoolToInt32(listConfig.GetEnabled()),
			ListComponent:       listConfig.GetComponent(),
			IsForm:              codegen.BoolToInt32(formConfig.GetEnabled()),
			FormComponent:       formConfig.GetComponent(),
			IsRequired:          codegen.BoolToInt32(formConfig.GetRequired()),
			FormMultiple:        formConfig.GetMultiple(),
			QueryOption:         queryOption,
			ListOption:          listOption,
			FormOption:          formOption,
			IsStatusField:       codegen.BoolToInt32(isStatus),
			StatusDataType:      listOption.SourceType,
			StatusDictCode:      listOption.SourceValue,
			StatusEnabledValue:  listOption.ActiveValue,
			StatusDisabledValue: listOption.InactiveValue,
			StatusDefaultValue:  defaultValue,
			StatusGenerateAPI:   codegen.BoolToInt32(isStatus),
			StatusTableColumn:   codegen.BoolToInt32(isStatus),
			StatusSearch:        codegen.BoolToInt32(isStatus && queryConfig.GetEnabled()),
			StatusSwitch:        codegen.BoolToInt32(isStatus),
			StatusForm:          codegen.BoolToInt32(isStatus && formConfig.GetEnabled()),
			Sort:                config.GetSort(),
		}
		columns = append(columns, column)
	}
	return columns
}

// codeGenOptionToSnapshot 转换单个作用域的字段选项配置。
func codeGenOptionToSnapshot(option *systemadminv1.CodeGenColumnOptionConfig) codegen.CodeGenColumnOptionConfig {
	return codegen.CodeGenColumnOptionConfig{
		Kind:          option.GetKind(),
		SourceType:    option.GetSourceType(),
		SourceValue:   option.GetSourceValue(),
		LabelField:    option.GetLabelField(),
		ValueField:    option.GetValueField(),
		ParentField:   option.GetParentField(),
		ActiveValue:   option.GetActiveValue(),
		InactiveValue: option.GetInactiveValue(),
	}
}

// codeGenProtosToSnapshots 将现有Proto配置转换为生成器只读快照。
func codeGenProtosToSnapshots(items []*models.CodeGenProto) ([]*codegen.Proto, error) {
	protos := make([]*codegen.Proto, 0, len(items))
	for _, item := range items {
		var config systemadminv1.CodeGenProtoConfig
		if item.Config != "" {
			if err := json.Unmarshal([]byte(item.Config), &config); err != nil {
				return nil, errorsx.Internal("Proto配置格式错误").WithCause(err)
			}
		}
		columnName := ""
		if item.APIKind == codegen.APIKindStatus {
			columnName = config.GetStatusColumn()
		}
		protos = append(protos, &codegen.Proto{
			ID:                  item.ID,
			TableID:             item.TableID,
			ColumnName:          columnName,
			TriggerType:         item.TriggerType,
			APIKind:             item.APIKind,
			TargetEntityName:    item.TargetEntityName,
			MethodName:          item.MethodName,
			ProtoFilePath:       item.ProtoFilePath,
			ParentColumn:        config.GetParentColumn(),
			LabelColumn:         config.GetLabelColumn(),
			ValueColumn:         config.GetValueColumn(),
			GenerateWhenMissing: item.GenerateWhenMissing,
			Sort:                item.Sort,
		})
	}
	return protos, nil
}

// validateOptionLabelColumn 校验选项响应的显示字段存在。
func validateOptionLabelColumn(method *codegen.Proto, columns []*codegen.CodeGenColumn, columnName string, fieldLabel string) error {
	if codegen.FindColumnByName(columns, columnName) == nil {
		return errorsx.InvalidArgument(fmt.Sprintf("选项接口%s的%s%s不存在", method.MethodName, fieldLabel, columnName))
	}
	return nil
}

// validateOptionIntegerColumn 校验选项响应字段可转换为int64。
func validateOptionIntegerColumn(method *codegen.Proto, columns []*codegen.CodeGenColumn, columnName string, fieldLabel string) error {
	column := codegen.FindColumnByName(columns, columnName)
	if column == nil {
		return errorsx.InvalidArgument(fmt.Sprintf("选项接口%s的%s%s不存在", method.MethodName, fieldLabel, columnName))
	}
	goType := codegen.DefaultString(column.GoType, codegen.InferGoType(column.DbType))
	if goType != "int64" && goType != "int32" {
		return errorsx.InvalidArgument(fmt.Sprintf("选项接口%s的%s%s必须是整数类型字段", method.MethodName, fieldLabel, columnName))
	}
	return nil
}

// writeGeneratedFile 通过同目录临时文件原子写入生成内容。
func writeGeneratedFile(fullPath string, content []byte, action string) error {
	err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
	if err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	var fileInfo os.FileInfo
	fileInfo, err = os.Stat(fullPath)
	// 创建动作禁止覆盖已有文件，更新动作保留原文件权限。
	switch action {
	case "create":
		if err == nil {
			return fmt.Errorf("生成文件已存在: %s", fullPath)
		}
		if !os.IsNotExist(err) {
			return err
		}
	case "update":
		if err != nil {
			return err
		}
		mode = fileInfo.Mode().Perm()
	default:
		return fmt.Errorf("不支持的生成动作: %s", action)
	}
	var tempFile *os.File
	tempFile, err = os.CreateTemp(filepath.Dir(fullPath), "."+filepath.Base(fullPath)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()
	if err = tempFile.Chmod(mode); err != nil {
		_ = tempFile.Close()
		return err
	}
	if _, err = tempFile.Write(content); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err = tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err = tempFile.Close(); err != nil {
		return err
	}
	if action == "create" {
		return os.Link(tempPath, fullPath)
	}
	return os.Rename(tempPath, fullPath)
}
