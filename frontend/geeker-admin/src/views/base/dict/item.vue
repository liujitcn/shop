<!-- 字典数据 -->
<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestBaseDictItemTable"
    >
      <template #tagType="scope">
        <el-tag v-if="scope.row.tagType" :type="formatTagType(scope.row.tagType)" effect="plain">
          {{ scope.row.label }}
        </el-tag>
        <span v-else>无</span>
      </template>
    </ProTable>

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="dialog.title"
      width="820px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      label-width="100px"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    >
      <template #tagTypeField>
        <div class="dict-item-tag-type">
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
        </div>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
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
const formDialogRef = ref<InstanceType<typeof FormDialog>>();

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
  status: [{ required: true, message: "状态不能为空", trigger: "change" }]
}));

const statusOptions: ProFormOption[] = [
  { label: "启用", value: Status.ENABLE },
  { label: "禁用", value: Status.DISABLE }
];

/** 字典项表单字段配置。 */
const formFields: ProFormField[] = [
  { prop: "label", label: "字典标签", component: "input", props: { placeholder: "请输入字典标签" } },
  { prop: "value", label: "字典值", component: "input", props: { placeholder: "请输入字典值" } },
  {
    prop: "sort",
    label: "排序",
    component: "input-number",
    props: { min: 1, precision: 0, step: 1, controlsPosition: "right", style: { width: "100%" } }
  },
  { prop: "status", label: "状态", component: "radio-group", options: statusOptions },
  { prop: "tagType", label: "标签类型", component: "slot", slotName: "tagTypeField", colSpan: 24 }
];

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
  {
    prop: "status",
    label: "状态",
    width: 100,
    search: { el: "select" },
    cellType: "status",
    statusProps: {
      activeValue: Status.ENABLE,
      inactiveValue: Status.DISABLE,
      activeText: "启用",
      inactiveText: "禁用",
      disabled: () => !BUTTONS.value["base:dict-item:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BaseDictItem)
    }
  },
  { prop: "createdAt", label: "创建时间", width: 180 },
  { prop: "updatedAt", label: "更新时间", width: 180 },
  {
    prop: "operation",
    label: "操作",
    width: 180,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "编辑",
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["base:dict-item:update"],
        params: scope => ({ dictItemId: scope.row.id }),
        onClick: (scope, params) => handleOpenDialog((params?.dictItemId as number | undefined) ?? (scope.row as BaseDictItem).id)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["base:dict-item:delete"],
        onClick: scope => handleDelete(scope.row as BaseDictItem)
      }
    ]
  }
];

/** 字典项顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:dict-item:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:dict-item:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BaseDictItem[])
  }
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
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
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
  resetForm();
  dialog.title = dictItemId ? "修改字典数据" : "新增字典数据";
  dialog.visible = true;
  if (!dictItemId) return;

  defBaseDictService.GetBaseDictItem({ value: dictItemId }).then(data => {
    Object.assign(formData, data);
  });
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
function handleSubmit() {
  formDialogRef.value?.validate()?.then(isValid => {
    if (!isValid) return;

    formData.dictId = dictId.value;
    const submitData = JSON.parse(JSON.stringify(formData)) as BaseDictItemForm;
    const request = submitData.id
      ? defBaseDictService.UpdateBaseDictItem(submitData)
      : defBaseDictService.CreateBaseDictItem(submitData);
    request.then(() => {
      ElMessage.success(submitData.id ? "修改成功" : "新增成功");
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
  const itemName = row.label || row.value || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(`是否确定${text}字典数据：${itemName}？`, "提示", {
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
 * 删除字典项，兼容单项删除与批量删除。
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
      ? `是否确定删除字典数据：${dictItemList[0].label || dictItemList[0].value || `ID:${dictItemList[0].id}`}？`
      : `确认删除已选中的 ${dictItemList.length} 项字典数据吗？`
    : "确认删除已选中的字典数据吗？";

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

<style scoped>
.dict-item-tag-type {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}
</style>
