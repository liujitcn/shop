package codegen

import (
	adminv1 "shop/api/gen/go/admin/v1"
)

// Generation 汇总一次预览或写入使用的完整生成结果。
type Generation struct {
	Table            *Table                        // 应用本次输出路径后的生成对象
	GeneratedMethods []*Proto                      // 本次实际参与生成的方法
	OutputPaths      *adminv1.CodeGenOutputPaths   // 校验后的输出路径
	Files            []*adminv1.CodeGenPreviewFile // 已完成增量合并的预览文件
}

// PrepareGeneration 准备一次预览或写入所需的全部纯生成内容。
func PrepareGeneration(
	table *Table,
	columns []*CodeGenColumn,
	methods []*Proto,
	requestedPaths *adminv1.CodeGenOutputPaths,
	tableComment string,
) (*Generation, error) {
	return prepareGenerationWithRenderer(table, columns, methods, requestedPaths, &renderer{tableComment: tableComment})
}

// prepareGenerationWithRenderer 使用指定文件视图准备单次预览或批次内合并结果。
func prepareGenerationWithRenderer(
	table *Table,
	columns []*CodeGenColumn,
	methods []*Proto,
	requestedPaths *adminv1.CodeGenOutputPaths,
	renderer *renderer,
) (*Generation, error) {
	outputPaths, err := renderer.resolveCodeGenOutputPaths(table, requestedPaths)
	if err != nil {
		return nil, err
	}
	generationTable, generationMethods := renderer.applyCodeGenOutputPaths(table, methods, outputPaths)
	generatedMethods := renderer.generatedProtoMethods(generationTable, columns, generationMethods)
	return &Generation{
		Table:            generationTable,
		GeneratedMethods: generatedMethods,
		OutputPaths:      outputPaths,
		Files:            renderer.buildPreviewFiles(generationTable, columns, generationMethods, outputPaths),
	}, nil
}

// InspectProtoMethods 检查当前配置需要的 Proto 方法及其仓库状态。
func InspectProtoMethods(table *Table, columns []*CodeGenColumn, savedMethods []*Proto) []*ProtoCheck {
	checks := (&renderer{}).buildExpectedProtoChecks(table, columns)
	for _, check := range checks {
		saved := findSavedProtoMethod(savedMethods, check)
		if saved != nil {
			applySavedProtoMethod(check, saved)
		}
		check.Exists, check.Message = ProtoMethodExists(check.ProtoFilePath, check.TargetEntityName, check.MethodName)
		if check.Exists {
			check.GenerateWhenMissing = false
		}
	}
	return checks
}

// applySavedProtoMethod 使用已保存的 Proto 配置覆盖模板推导值。
func applySavedProtoMethod(check *ProtoCheck, saved *Proto) {
	check.GenerateWhenMissing = saved.GenerateWhenMissing == 1
	check.TargetBusinessName = saved.TargetBusinessName
	if saved.ParentColumn != "" {
		check.ParentColumn = saved.ParentColumn
	}
	if saved.LabelColumn != "" {
		check.LabelColumn = saved.LabelColumn
	}
	if saved.ValueColumn != "" {
		check.ValueColumn = saved.ValueColumn
	}
	if saved.APIKind == APIKindStatus && saved.ColumnName != "" {
		check.ColumnName = saved.ColumnName
	}
}
