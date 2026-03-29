<!-- 字典 -->
<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestBaseDictTable">
      <template #tableHeader="{ selectedList }">
        <el-button v-hasPerm="['base:dict:create']" type="success" :icon="CirclePlus" @click="handleOpenDialog()">新增</el-button>
        <el-button
          v-hasPerm="['base:dict:delete']"
          type="danger"
          :icon="Delete"
          :disabled="!selectedList.length"
          @click="handleDelete(selectedList)"
        >
          删除
        </el-button>
      </template>

      <template #status="scope">
        <el-switch
          v-model="scope.row.status"
          inline-prompt
          :active-value="Status.ENABLE"
          :inactive-value="Status.DISABLE"
          active-text="启用"
          inactive-text="禁用"
          :disabled="!BUTTONS['base:dict:status']"
          :before-change="() => handleBeforeSetStatus(scope.row)"
        />
      </template>

      <template #operation="scope">
        <el-button v-hasPerm="['base:dict:items']" type="primary" link :icon="List" @click="handleOpenBaseDictItem(scope.row)">
          字典数据
        </el-button>
        <el-button v-hasPerm="['base:dict:update']" type="primary" link :icon="EditPen" @click="handleOpenDialog(scope.row.id)">
          编辑
        </el-button>
        <el-button v-hasPerm="['base:dict:delete']" type="danger" link :icon="Delete" @click="handleDelete(scope.row)">
          删除
        </el-button>
      </template>
    </ProTable>

    <el-dialog v-model="dialog.visible" :title="dialog.title" width="500px" @close="handleCloseDialog">
      <el-form ref="dataFormRef" :model="formData" :rules="rules" label-width="100px">
        <el-card shadow="never">
          <el-form-item label="字典名称" prop="name">
            <el-input v-model="formData.name" placeholder="请输入字典名称" />
          </el-form-item>

          <el-form-item label="字典编码" prop="code">
            <el-input v-model="formData.code" placeholder="请输入字典编码" />
          </el-form-item>

          <el-form-item label="状态" prop="status">
            <el-switch
              v-model="formData.status"
              inline-prompt
              active-text="启用"
              inactive-text="禁用"
              :active-value="Status.ENABLE"
              :inactive-value="Status.DISABLE"
            />
          </el-form-item>
        </el-card>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button type="primary" @click="handleSubmitClick">确 定</el-button>
          <el-button @click="handleCloseDialog">取 消</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen, List } from "@element-plus/icons-vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defBaseDictService } from "@/api/admin/base_dict";
import type { BaseDict, BaseDictForm, PageBaseDictRequest } from "@/rpc/admin/base_dict";
import router from "@/routers";
import { Status } from "@/rpc/common/enum";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";

defineOptions({
  name: "BaseDict",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const dataFormRef = ref();

const dialog = reactive({
  title: "",
  visible: false
});

const formData = reactive<BaseDictForm>({
  /** 字典ID */
  id: 0,
  /** 字典编号 */
  code: "",
  /** 字典名称 */
  name: "",
  /** 状态 */
  status: Status.ENABLE
});

const rules = computed(() => ({
  name: [{ required: true, message: "请输入字典名称", trigger: "blur" }],
  code: [{ required: true, message: "请输入字典编码", trigger: "blur" }],
  status: [{ required: true, message: "状态不能为空", trigger: "blur" }]
}));

/** 字典表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "name", label: "字典名称", search: { el: "input" } },
  { prop: "code", label: "字典编码", search: { el: "input" } },
  { prop: "status", label: "状态", width: 100, dictCode: "status", search: { el: "select" } },
  { prop: "createdAt", label: "创建时间", width: 180 },
  { prop: "updatedAt", label: "更新时间", width: 180 },
  { prop: "operation", label: "操作", width: 240, fixed: "right" }
];

/**
 * 请求字典列表，并由 ProTable 统一管理分页搜索。
 */
async function requestBaseDictTable(params: PageBaseDictRequest) {
  const data = await defBaseDictService.PageBaseDict(buildPageRequest(params));
  return { data };
}

/**
 * 刷新字典表格数据。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 重置字典表单，避免新增时带入上次编辑结果。
 */
function resetForm() {
  dataFormRef.value?.resetFields();
  dataFormRef.value?.clearValidate();
  formData.id = 0;
  formData.code = "";
  formData.name = "";
  formData.status = Status.ENABLE;
}

/**
 * 打开字典编辑弹窗。
 */
function handleOpenDialog(dictId?: number) {
  dialog.visible = true;
  if (dictId) {
    dialog.title = "修改字典";
    defBaseDictService.GetBaseDict({ value: dictId }).then(data => {
      Object.assign(formData, data);
    });
    return;
  }

  dialog.title = "新增字典";
  resetForm();
}

/**
 * 关闭字典弹窗并恢复表单初始值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/**
 * 提交字典表单。
 */
function handleSubmitClick() {
  dataFormRef.value?.validate((isValid: boolean) => {
    if (!isValid) return;

    const request = formData.id ? defBaseDictService.UpdateBaseDict(formData) : defBaseDictService.CreateBaseDict(formData);
    request.then(() => {
      ElMessage.success(formData.id ? "修改成功" : "新增成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 在字典状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: BaseDict) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = nextStatus === Status.ENABLE ? "启用" : "禁用";
  const dictName = row.name || row.code || String(row.id);
  try {
    await ElMessageBox.confirm(`是否确定${text}字典：${dictName}？`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defBaseDictService.SetBaseDictStatus({ id: row.id, status: nextStatus });
    ElMessage.success(`${text}成功`);
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除字典，兼容单项删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | BaseDict | BaseDict[]) {
  const dictList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as BaseDict[])
    : selected && typeof selected === "object"
      ? [selected as BaseDict]
      : [];
  const dictIds = (
    dictList.length ? dictList.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!dictIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  const confirmMessage = dictList.length
    ? dictList.length === 1
      ? `是否确定删除字典：${dictList[0].name || dictList[0].code || `ID:${dictList[0].id}`}？`
      : `确认删除已选中的 ${dictList.length} 个字典吗？`
    : "确认删除已选中的字典吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseDictService.DeleteBaseDict({ value: dictIds }).then(() => {
        ElMessage.success("删除成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消删除");
    }
  );
}

/**
 * 打开字典数据页面。
 */
function handleOpenBaseDictItem(row: BaseDict) {
  router.push({
    path: "/base/dict-item",
    query: { dictId: row.id, title: `【${row.name}】字典数据` }
  });
}
</script>
