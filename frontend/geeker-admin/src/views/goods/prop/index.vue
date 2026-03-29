<!-- 商品属性 -->
<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestGoodsPropTable" :init-param="initParam">
      <template #tableHeader="{ selectedList }">
        <el-button v-hasPerm="['goods:prop:create']" type="success" :icon="CirclePlus" @click="handleOpenDialog()">
          新增
        </el-button>
        <el-button
          v-hasPerm="['goods:prop:delete']"
          type="danger"
          :icon="Delete"
          :disabled="!selectedList.length"
          @click="handleDelete(selectedList)"
        >
          删除
        </el-button>
      </template>

      <template #operation="scope">
        <el-button v-hasPerm="['goods:prop:update']" type="primary" link :icon="EditPen" @click="handleOpenDialog(scope.row.id)">
          编辑
        </el-button>
        <el-button v-hasPerm="['goods:prop:delete']" type="danger" link :icon="Delete" @click="handleDelete(scope.row)">
          删除
        </el-button>
      </template>
    </ProTable>

    <el-dialog v-model="dialog.visible" :title="dialog.title" width="820px" @close="handleCloseDialog">
      <ProForm ref="proFormRef" :model="formData" :fields="formFields" :rules="rules" label-width="120px" />

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
import { useRoute } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import ProForm from "@/components/ProForm/index.vue";
import type { ProFormField, ProFormInstance } from "@/components/ProForm/interface";
import { defGoodsPropService } from "@/api/admin/goods_prop";
import type { GoodsProp, PageGoodsPropRequest } from "@/rpc/admin/goods_prop";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";

defineOptions({
  name: "GoodsProp",
  inheritAttrs: false
});

const route = useRoute();
const proTable = ref<ProTableInstance>();
const proFormRef = ref<ProFormInstance>();

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
  { prop: "operation", label: "操作", width: 180, fixed: "right" }
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
  proFormRef.value?.resetFields();
  proFormRef.value?.clearValidate();
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
  dialog.visible = true;
  if (propId) {
    dialog.title = "修改商品属性";
    defGoodsPropService.GetGoodsProp({ value: propId }).then(data => {
      Object.assign(formData, data);
    });
    return;
  }

  dialog.title = "新增商品属性";
  resetForm();
}

/**
 * 提交商品属性表单。
 */
function handleSubmitClick() {
  proFormRef.value?.validate()?.then(isValid => {
    if (!isValid) return;

    formData.goodsId = goodsId.value;
    const request = formData.id ? defGoodsPropService.UpdateGoodsProp(formData) : defGoodsPropService.CreateGoodsProp(formData);
    request.then(() => {
      ElMessage.success(formData.id ? "修改成功" : "新增成功");
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
      ? `是否确定删除商品属性：${propList[0].label || propList[0].value || `ID:${propList[0].id}`}？`
      : `确认删除已选中的 ${propList.length} 个商品属性吗？`
    : "确认删除已选中的商品属性吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defGoodsPropService.DeleteGoodsProp({ value: propIds }).then(() => {
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
