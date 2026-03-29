<!-- 字典数据 -->
<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestBaseDictItemTable">
      <template #tableHeader="{ selectedList }">
        <el-button v-hasPerm="['base:dict-item:create']" type="success" :icon="CirclePlus" @click="handleOpenDialog()">
          新增
        </el-button>
        <el-button
          v-hasPerm="['base:dict-item:delete']"
          type="danger"
          :icon="Delete"
          :disabled="!selectedList.length"
          @click="handleDelete(selectedList)"
        >
          删除
        </el-button>
      </template>

      <template #tagType="scope">
        <el-tag v-if="scope.row.tagType" :type="formatTagType(scope.row.tagType)" effect="plain">
          {{ scope.row.label }}
        </el-tag>
        <span v-else>无</span>
      </template>

      <template #status="scope">
        <el-switch
          v-model="scope.row.status"
          inline-prompt
          :active-value="Status.ENABLE"
          :inactive-value="Status.DISABLE"
          active-text="启用"
          inactive-text="禁用"
          :disabled="!BUTTONS['base:dict-item:status']"
          :before-change="() => handleBeforeSetStatus(scope.row)"
        />
      </template>

      <template #operation="scope">
        <el-button
          v-hasPerm="['base:dict-item:update']"
          type="primary"
          link
          :icon="EditPen"
          @click="handleOpenDialog(scope.row.id)"
        >
          编辑
        </el-button>
        <el-button v-hasPerm="['base:dict-item:delete']" type="danger" link :icon="Delete" @click="handleDelete(scope.row)">
          删除
        </el-button>
      </template>
    </ProTable>

    <el-dialog v-model="dialog.visible" :title="dialog.title" width="820px" @close="handleCloseDialog">
      <el-form ref="dataFormRef" :model="formData" :rules="rules" label-width="100px">
        <el-card shadow="never">
          <el-form-item label="字典标签" prop="label">
            <el-input v-model="formData.label" placeholder="请输入字典标签" />
          </el-form-item>
          <el-form-item label="字典值" prop="value">
            <el-input v-model="formData.value" placeholder="请输入字典值" />
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
          <el-form-item label="排序" prop="sort">
            <el-input-number v-model="formData.sort" controls-position="right" :min="1" :precision="0" :step="1" />
          </el-form-item>
          <el-form-item label="标签类型" prop="tagType">
            <el-tag v-if="formData.tagType" :type="formatTagType(formData.tagType)" class="mr-2">
              {{ formData.label || "标签预览" }}
            </el-tag>
            <el-radio-group v-model="formData.tagType">
              <el-radio value="success" border size="small">success</el-radio>
              <el-radio value="warning" border size="small">warning</el-radio>
              <el-radio value="info" border size="small">info</el-radio>
              <el-radio value="primary" border size="small">primary</el-radio>
              <el-radio value="danger" border size="small">danger</el-radio>
              <el-radio value="" border size="small">清空</el-radio>
            </el-radio-group>
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
import { computed, reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defBaseDictService } from "@/api/admin/base_dict";
import type { BaseDictItem, BaseDictItemForm, PageBaseDictItemRequest } from "@/rpc/admin/base_dict";
import { Status } from "@/rpc/common/enum";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";

defineOptions({
  name: "BaseDictItem",
  inheritAttrs: false
});

const route = useRoute();
const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const dataFormRef = ref();

const dictId = ref(Number(route.query.dictId ?? 0));

const dialog = reactive({
  title: "",
  visible: false
});

const formData = reactive<BaseDictItemForm>({
  /** 字典项ID */
  id: 0,
  /** 字典ID */
  dictId: dictId.value,
  /** 字典值 */
  value: "",
  /** 字典项标签 */
  label: "",
  /** 标签类型 */
  tagType: "",
  /** 排序 */
  sort: 1,
  /** 状态 */
  status: Status.ENABLE
});

const rules = computed(() => ({
  value: [{ required: true, message: "请输入字典值", trigger: "blur" }],
  label: [{ required: true, message: "请输入字典标签", trigger: "blur" }],
  sort: [{ required: true, message: "请输入排序", trigger: "blur" }],
  status: [{ required: true, message: "状态不能为空", trigger: "blur" }]
}));

/**
 * 规范化标签类型，兼容 Element Plus Tag 的可选值。
 */
function formatTagType(tagType: string) {
  if (["success", "info", "warning", "primary", "danger"].includes(tagType)) {
    return tagType as "success" | "info" | "warning" | "primary" | "danger";
  }
  return undefined;
}

/** 字典项表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "label", label: "字典标签", search: { el: "input" } },
  { prop: "value", label: "字典值" },
  { prop: "sort", label: "排序", align: "right" },
  { prop: "tagType", label: "标签类型", width: 120 },
  { prop: "status", label: "状态", width: 100, dictCode: "status", search: { el: "select" } },
  { prop: "createdAt", label: "创建时间", width: 180 },
  { prop: "updatedAt", label: "更新时间", width: 180 },
  { prop: "operation", label: "操作", width: 180, fixed: "right" }
];

watch(
  () => route.query.dictId,
  value => {
    dictId.value = Number(value ?? 0);
    formData.dictId = dictId.value;
    proTable.value?.search();
  }
);

/**
 * 请求字典项分页列表，并补充当前路由上的字典 ID。
 */
async function requestBaseDictItemTable(params: PageBaseDictItemRequest) {
  const data = await defBaseDictService.PageBaseDictItem(
    buildPageRequest({
      ...params,
      dictId: dictId.value
    })
  );
  return { data };
}

/**
 * 刷新字典项表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 重置字典项表单，避免弹窗之间相互污染。
 */
function resetForm() {
  dataFormRef.value?.resetFields();
  dataFormRef.value?.clearValidate();
  formData.id = 0;
  formData.dictId = dictId.value;
  formData.value = "";
  formData.label = "";
  formData.tagType = "";
  formData.sort = 1;
  formData.status = Status.ENABLE;
}

/**
 * 打开字典项编辑弹窗。
 */
function handleOpenDialog(dictItemId?: number) {
  dialog.visible = true;
  if (dictItemId) {
    dialog.title = "修改字典数据";
    defBaseDictService.GetBaseDictItem({ value: dictItemId }).then(data => {
      Object.assign(formData, data);
    });
    return;
  }

  dialog.title = "新增字典数据";
  resetForm();
}

/**
 * 关闭字典项弹窗并恢复默认值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/**
 * 提交字典项表单。
 */
function handleSubmitClick() {
  dataFormRef.value?.validate((isValid: boolean) => {
    if (!isValid) return;

    formData.dictId = dictId.value;
    const request = formData.id
      ? defBaseDictService.UpdateBaseDictItem(formData)
      : defBaseDictService.CreateBaseDictItem(formData);
    request.then(() => {
      ElMessage.success(formData.id ? "修改成功" : "新增成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 在字典项状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: BaseDictItem) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = nextStatus === Status.ENABLE ? "启用" : "禁用";
  const dictItemName = row.label || row.value || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(`是否确定${text}字典项：${dictItemName}？`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defBaseDictService.SetBaseDictItemStatus({ id: row.id, status: nextStatus });
    ElMessage.success(`${text}成功`);
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除字典项，兼容单条删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | BaseDictItem | BaseDictItem[]) {
  const dictItemList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as BaseDictItem[])
    : selected && typeof selected === "object"
      ? [selected as BaseDictItem]
      : [];
  const dictItemIds = (
    dictItemList.length
      ? dictItemList.map(item => item.id)
      : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!dictItemIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  const confirmMessage = dictItemList.length
    ? dictItemList.length === 1
      ? `是否确定删除字典项：${dictItemList[0].label || dictItemList[0].value || `ID:${dictItemList[0].id}`}？`
      : `确认删除已选中的 ${dictItemList.length} 个字典项吗？`
    : "确认删除已选中的字典项吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseDictService.DeleteBaseDictItem({ value: dictItemIds }).then(() => {
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
