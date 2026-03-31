<!-- 商品属性 -->
<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestGoodsPropTable"
      :init-param="initParam"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="dialog.title"
      width="820px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      label-width="120px"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { useRoute } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField } from "@/components/ProForm/interface";
import { defGoodsPropService } from "@/api/admin/goods_prop";
import type { GoodsProp, PageGoodsPropRequest } from "@/rpc/admin/goods_prop";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";

defineOptions({
  name: "GoodsProp",
  inheritAttrs: false
});

const route = useRoute();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();

const goodsId = computed(() => Number(route.query.goodsId ?? 0));
const initParam = computed(() => ({
  goodsId: goodsId.value
}));

const dialog = reactive({
  title: "",
  visible: false
});

const formData = reactive<GoodsProp>({
  /** 商品属性ID */
  id: 0,
  /** 商品ID */
  goodsId: goodsId.value,
  /** 商品属性值 */
  value: "",
  /** 商品属性项标签 */
  label: "",
  /** 排序 */
  sort: 1
});

const rules = reactive({
  value: [{ required: true, message: "请输入商品属性值", trigger: "blur" }],
  label: [{ required: true, message: "请输入商品属性标签", trigger: "blur" }]
});

/** 商品属性表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  {
    prop: "label",
    label: "商品属性标签",
    component: "input",
    props: { placeholder: "请输入商品属性标签" }
  },
  {
    prop: "value",
    label: "商品属性值",
    component: "textarea",
    props: { placeholder: "请输入商品属性值" }
  },
  {
    prop: "sort",
    label: "排序",
    component: "input-number",
    props: {
      min: 1,
      controlsPosition: "right",
      precision: 0,
      step: 1,
      style: { width: "100%" }
    }
  }
]);

/** 商品属性表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "label", label: "商品属性标签", search: { el: "input" } },
  { prop: "value", label: "商品属性值" },
  { prop: "sort", label: "排序", align: "right" },
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
        params: scope => ({ propId: scope.row.id }),
        onClick: (scope, params) => handleOpenDialog((params?.propId as number | undefined) ?? (scope.row as GoodsProp).id)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        onClick: scope => handleDelete(scope.row as GoodsProp)
      }
    ]
  }
];

/** 商品属性顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    onClick: () => handleOpenDialog()
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as GoodsProp[])
  }
];

/**
 * 请求商品属性列表，并带上当前商品 ID。
 */
async function requestGoodsPropTable(params: PageGoodsPropRequest) {
  const data = await defGoodsPropService.PageGoodsProp(buildPageRequest({ ...params, goodsId: goodsId.value }));
  return { data };
}

/**
 * 刷新当前商品属性表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 重置商品属性表单，确保切换商品后不会带入旧数据。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.goodsId = goodsId.value;
  formData.value = "";
  formData.label = "";
  formData.sort = 1;
}

/**
 * 打开商品属性弹窗。
 */
function handleOpenDialog(propId?: number) {
  resetForm();
  dialog.title = propId ? "修改商品属性" : "新增商品属性";
  dialog.visible = true;
  if (!propId) return;

  defGoodsPropService.GetGoodsProp({ value: propId }).then(data => {
    Object.assign(formData, data);
  });
}

/**
 * 提交商品属性表单。
 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(isValid => {
    if (!isValid) return;

    formData.goodsId = goodsId.value;
    const submitData = JSON.parse(JSON.stringify(formData)) as GoodsProp;
    const request = submitData.id
      ? defGoodsPropService.UpdateGoodsProp(submitData)
      : defGoodsPropService.CreateGoodsProp(submitData);
    request.then(() => {
      ElMessage.success(submitData.id ? "修改商品属性成功" : "新增商品属性成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 关闭商品属性弹窗并恢复默认表单值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/**
 * 删除商品属性，兼容单项删除与多选删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | GoodsProp | GoodsProp[]) {
  const propList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as GoodsProp[])
    : selected && typeof selected === "object"
      ? [selected as GoodsProp]
      : [];
  const propIds = (
    propList.length ? propList.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!propIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  const confirmMessage = propList.length
    ? propList.length === 1
      ? `是否确定删除商品属性？\n属性名称：${propList[0].label || propList[0].value || `ID:${propList[0].id}`}`
      : `确认删除已选中的 ${propList.length} 个商品属性吗？`
    : "确认删除已选中的商品属性吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defGoodsPropService.DeleteGoodsProp({ value: propIds }).then(() => {
        ElMessage.success("删除商品属性成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消删除商品属性");
    }
  );
}
</script>
