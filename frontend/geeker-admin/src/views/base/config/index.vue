<!-- 系统配置 -->
<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestBaseConfigTable">
      <template #tableHeader="{ selectedList }">
        <el-button v-hasPerm="['base:config:create']" type="success" :icon="CirclePlus" @click="handleOpenDialog()">
          新增
        </el-button>
        <el-button
          v-hasPerm="['base:config:delete']"
          type="danger"
          :icon="Delete"
          :disabled="!selectedList.length"
          @click="handleDelete(selectedList)"
        >
          删除
        </el-button>
        <el-button v-hasPerm="['base:config:refresh']" color="#626aef" :icon="RefreshLeft" @click="handleRefreshCache">
          刷新缓存
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
          :disabled="!BUTTONS['base:config:status']"
          :before-change="() => handleBeforeSetStatus(scope.row)"
        />
      </template>

      <template #operation="scope">
        <el-button v-hasPerm="['base:config:update']" type="primary" link :icon="EditPen" @click="handleOpenDialog(scope.row.id)">
          编辑
        </el-button>
        <el-button v-hasPerm="['base:config:delete']" type="danger" link :icon="Delete" @click="handleDelete(scope.row)">
          删除
        </el-button>
      </template>
    </ProTable>

    <el-dialog v-model="dialog.visible" :title="dialog.title" width="1200px" @close="handleCloseDialog">
      <el-form ref="dataFormRef" :model="formData" :rules="rules" label-suffix=":" label-width="100px">
        <el-form-item label="配置名称" prop="name">
          <el-input v-model="formData.name" placeholder="请输入配置名称" :disabled="formData.id > 0" />
        </el-form-item>
        <el-form-item label="配置位置" prop="site">
          <Dict v-model="formData.site" code="base_config_site" :disabled="formData.id > 0" />
        </el-form-item>
        <el-form-item label="配置键" prop="key">
          <el-input v-model="formData.key" placeholder="请输入配置键" :disabled="formData.id > 0" />
        </el-form-item>
        <el-form-item label="配置类型" prop="type">
          <Dict v-model="formData.type" code="base_config_type" :disabled="formData.id > 0" />
        </el-form-item>
        <el-form-item v-if="formData.type == BaseConfigType.TEXT" label="配置值" prop="value">
          <el-input v-model="formData.value" placeholder="请输入配置值" />
        </el-form-item>
        <el-form-item v-if="formData.type == BaseConfigType.IMAGE" label="配置值" prop="value">
          <UploadImg v-model:image-url="formData.value" />
        </el-form-item>
        <el-form-item v-if="formData.type == BaseConfigType.RICH_TEXT" label="配置值" prop="value">
          <WangEditor v-model:value="formData.value" />
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
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button type="primary" @click="handleSubmit">确定</el-button>
          <el-button @click="handleCloseDialog">取消</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from "vue";
import { useDebounceFn } from "@vueuse/core";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen, RefreshLeft } from "@element-plus/icons-vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import UploadImg from "@/components/Upload/Img.vue";
import WangEditor from "@/components/WangEditor/index.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defBaseConfigService } from "@/api/admin/base_config";
import type { BaseConfig, BaseConfigForm, PageBaseConfigRequest } from "@/rpc/admin/base_config";
import { BaseConfigType, Status } from "@/rpc/common/enum";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";

defineOptions({
  name: "BaseConfig",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const dataFormRef = ref();

const dialog = reactive({
  title: "",
  visible: false
});

const formData = reactive<BaseConfigForm>({
  /** 配置ID */
  id: 0,
  /** 位置：枚举【BaseConfigSite】 */
  site: undefined,
  /** 配置名称 */
  name: "",
  /** 配置类型：枚举【BaseConfigType】 */
  type: undefined,
  /** 配置key */
  key: "",
  /** 配置value */
  value: "",
  /** 状态 */
  status: Status.ENABLE
});

const rules = reactive({
  site: [{ required: true, message: "请选择系统配置位置", trigger: "blur" }],
  name: [{ required: true, message: "请输入系统配置名称", trigger: "blur" }],
  type: [{ required: true, message: "请选择系统配置类型", trigger: "blur" }],
  key: [{ required: true, message: "请输入系统配置编码", trigger: "blur" }],
  value: [{ required: true, message: "配置值不能为空", trigger: "blur" }],
  status: [{ required: true, message: "请选择状态", trigger: "blur" }]
});

/** 系统配置表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "site", label: "配置位置", dictCode: "base_config_site", search: { el: "select" } },
  { prop: "name", label: "配置名称", search: { el: "input" } },
  { prop: "type", label: "配置类型", dictCode: "base_config_type", search: { el: "select" } },
  { prop: "key", label: "配置键", search: { el: "input" } },
  { prop: "status", label: "状态", width: 100, dictCode: "status", search: { el: "select" } },
  { prop: "createdAt", label: "创建时间", width: 180 },
  { prop: "updatedAt", label: "更新时间", width: 180 },
  { prop: "operation", label: "操作", width: 150, fixed: "right" }
];

/**
 * 请求系统配置列表，并由 ProTable 统一维护分页与搜索参数。
 */
async function requestBaseConfigTable(params: PageBaseConfigRequest) {
  const data = await defBaseConfigService.PageBaseConfig(buildPageRequest(params));
  return { data };
}

/**
 * 刷新系统配置表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 重置系统配置表单，避免新增时保留旧值。
 */
function resetForm() {
  dataFormRef.value?.resetFields();
  dataFormRef.value?.clearValidate();
  formData.id = 0;
  formData.site = undefined;
  formData.name = "";
  formData.type = undefined;
  formData.key = "";
  formData.value = "";
  formData.status = Status.ENABLE;
}

/**
 * 打开系统配置弹窗。
 */
function handleOpenDialog(configId?: number) {
  dialog.visible = true;
  if (configId) {
    dialog.title = "修改系统配置";
    defBaseConfigService.GetBaseConfig({ value: configId }).then(data => {
      Object.assign(formData, data);
    });
    return;
  }

  dialog.title = "新增系统配置";
  resetForm();
}

/**
 * 刷新服务端配置缓存，使用防抖避免重复点击。
 */
const handleRefreshCache = useDebounceFn(() => {
  defBaseConfigService.RefreshBaseConfig({}).then(() => {
    ElMessage.success("刷新成功");
  });
}, 1000);

/**
 * 提交系统配置表单。
 */
function handleSubmit() {
  dataFormRef.value?.validate((valid: boolean) => {
    if (!valid) return;

    const request = formData.id
      ? defBaseConfigService.UpdateBaseConfig(formData)
      : defBaseConfigService.CreateBaseConfig(formData);
    request.then(() => {
      ElMessage.success(formData.id ? "修改成功" : "新增成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 关闭系统配置弹窗并恢复默认表单值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/**
 * 在系统配置状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: BaseConfig) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = nextStatus === Status.ENABLE ? "启用" : "禁用";
  const configName = row.name || row.key || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(`是否确定${text}配置：${configName}？`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defBaseConfigService.SetBaseConfigStatus({ id: row.id, status: nextStatus });
    ElMessage.success(`${text}成功`);
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除系统配置，兼容单项删除与多选删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | BaseConfig | BaseConfig[]) {
  const configList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as BaseConfig[])
    : selected && typeof selected === "object"
      ? [selected as BaseConfig]
      : [];
  const configIds = (
    configList.length
      ? configList.map(item => item.id)
      : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!configIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  const confirmMessage = configList.length
    ? configList.length === 1
      ? `是否确定删除配置：${configList[0].name || configList[0].key || `ID:${configList[0].id}`}？`
      : `确认删除已选中的 ${configList.length} 项系统配置吗？`
    : "确认删除已选中的系统配置吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseConfigService.DeleteBaseConfig({ value: configIds }).then(() => {
        ElMessage.success("删除成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消删除");
    }
  );
}
</script>
