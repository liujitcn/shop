<!-- 热门推荐 -->
<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestShopHotTable">
      <template #tableHeader="{ selectedList }">
        <el-button v-hasPerm="['shop:hot:create']" type="success" :icon="CirclePlus" @click="handleOpenDialog()">新增</el-button>
        <el-button
          v-hasPerm="['shop:hot:delete']"
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
          :disabled="!BUTTONS['shop:hot:status']"
          :before-change="() => handleBeforeSetStatus(scope.row)"
        />
      </template>

      <template #operation="scope">
        <el-button v-hasPerm="['shop:hot:items']" type="primary" link :icon="List" @click="handleOpenShopHotItem(scope.row)">
          推荐选项
        </el-button>
        <el-button v-hasPerm="['shop:hot:update']" type="primary" link :icon="EditPen" @click="handleOpenDialog(scope.row.id)">
          编辑
        </el-button>
        <el-button v-hasPerm="['shop:hot:delete']" type="danger" link :icon="Delete" @click="handleDelete(scope.row)">
          删除
        </el-button>
      </template>
    </ProTable>

    <el-dialog v-model="dialog.visible" :title="dialog.title" width="1000px" @close="handleCloseDialog">
      <ProForm ref="proFormRef" :model="formData" :fields="formFields" :rules="rules" label-width="150px">
        <template #pictureUpload>
          <UploadImgs v-model:file-list="pictureFileList" :limit="2" />
        </template>
      </ProForm>

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
import type { UploadUserFile } from "element-plus";
import { CirclePlus, Delete, EditPen, List } from "@element-plus/icons-vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import ProForm from "@/components/ProForm/index.vue";
import type { ProFormField, ProFormInstance, ProFormOption } from "@/components/ProForm/interface";
import UploadImgs from "@/components/Upload/Imgs.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defShopHotService } from "@/api/admin/shop_hot";
import type { PageShopHotRequest, ShopHot, ShopHotForm } from "@/rpc/admin/shop_hot";
import router from "@/routers";
import { Status } from "@/rpc/common/enum";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";

defineOptions({
  name: "ShopShopHot",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const proFormRef = ref<ProFormInstance>();

const dialog = reactive({
  title: "",
  visible: false
});

const formData = reactive<ShopHotForm>({
  /** 热门推荐ID */
  id: 0,
  /** 商城热门推荐标题 */
  title: "",
  /** 商城热门推荐名称 */
  desc: "",
  /** 轮播图 */
  banner: "",
  /** 图片 */
  picture: [],
  /** 排序 */
  sort: 1,
  /** 状态 */
  status: Status.ENABLE
});

/** 将热门推荐图片值适配为上传组件需要的文件列表结构。 */
const pictureFileList = computed<UploadUserFile[]>({
  get: () => (formData.picture ?? []).map(url => ({ name: url.split("/").pop() ?? "image", url })),
  set: value => {
    formData.picture = value.map(item => item.url ?? "").filter(Boolean);
  }
});

const rules = computed(() => ({
  title: [{ required: true, message: "请输入热门推荐标题", trigger: "blur" }],
  desc: [{ required: true, message: "请输入热门推荐描述", trigger: "blur" }],
  banner: [{ required: true, message: "请上传轮播图", trigger: "blur" }],
  picture: [{ required: true, message: "请上传推荐图片", trigger: "blur" }],
  status: [{ required: true, message: "状态不能为空", trigger: "blur" }]
}));

const statusOptions: ProFormOption[] = [
  { label: "启用", value: Status.ENABLE },
  { label: "禁用", value: Status.DISABLE }
];

/** 热门推荐表单字段配置。 */
const formFields: ProFormField[] = [
  { prop: "title", label: "热门推荐标题", component: "input", props: { placeholder: "请输入热门推荐标题" } },
  { prop: "desc", label: "热门推荐描述", component: "input", props: { placeholder: "请输入热门推荐描述" } },
  { prop: "banner", label: "轮播图", component: "image-upload" },
  { prop: "picture", label: "推荐图片", component: "slot", slotName: "pictureUpload" },
  {
    prop: "sort",
    label: "排序",
    component: "input-number",
    props: { min: 1, precision: 0, step: 1, controlsPosition: "right", style: { width: "100%" } }
  },
  { prop: "status", label: "状态", component: "radio-group", options: statusOptions }
];

/** 热门推荐表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "title", label: "热门推荐标题", search: { el: "input" } },
  { prop: "desc", label: "热门推荐描述", search: { el: "input" } },
  { prop: "sort", label: "排序", align: "right" },
  { prop: "status", label: "状态", width: 100, dictCode: "status", search: { el: "select" } },
  { prop: "createdAt", label: "创建时间", width: 180 },
  { prop: "updatedAt", label: "更新时间", width: 180 },
  { prop: "operation", label: "操作", width: 240, fixed: "right" }
];

/**
 * 请求热门推荐列表，交给 ProTable 统一处理分页和筛选。
 */
async function requestShopHotTable(params: PageShopHotRequest) {
  const data = await defShopHotService.PageShopHot(buildPageRequest(params));
  return { data };
}

/**
 * 刷新热门推荐表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 重置热门推荐表单，清理上次编辑残留。
 */
function resetForm() {
  proFormRef.value?.resetFields();
  proFormRef.value?.clearValidate();
  formData.id = 0;
  formData.title = "";
  formData.desc = "";
  formData.banner = "";
  formData.picture = [];
  formData.sort = 1;
  formData.status = Status.ENABLE;
}

/**
 * 打开热门推荐弹窗。
 */
function handleOpenDialog(hotId?: number) {
  dialog.visible = true;
  if (hotId) {
    dialog.title = "修改热门推荐";
    defShopHotService.GetShopHot({ value: hotId }).then(data => {
      Object.assign(formData, data);
    });
    return;
  }

  dialog.title = "新增热门推荐";
  resetForm();
}

/**
 * 关闭热门推荐弹窗并恢复默认值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/**
 * 提交热门推荐表单。
 */
function handleSubmitClick() {
  proFormRef.value?.validate()?.then(isValid => {
    if (!isValid) return;

    const request = formData.id ? defShopHotService.UpdateShopHot(formData) : defShopHotService.CreateShopHot(formData);
    request.then(() => {
      ElMessage.success(formData.id ? "修改成功" : "新增成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 在热门推荐状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: ShopHot) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = nextStatus === Status.ENABLE ? "启用" : "禁用";
  const hotName = row.title || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(`是否确定${text}热门推荐：${hotName}？`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defShopHotService.SetShopHotStatus({ id: row.id, status: nextStatus });
    ElMessage.success(`${text}成功`);
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除热门推荐，兼容单行删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | ShopHot | ShopHot[]) {
  const hotList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as ShopHot[])
    : selected && typeof selected === "object"
      ? [selected as ShopHot]
      : [];
  const hotIds = (
    hotList.length ? hotList.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!hotIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  const confirmMessage = hotList.length
    ? hotList.length === 1
      ? `是否确定删除热门推荐：${hotList[0].title || `ID:${hotList[0].id}`}？`
      : `确认删除已选中的 ${hotList.length} 个热门推荐吗？`
    : "确认删除已选中的热门推荐吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defShopHotService.DeleteShopHot({ value: hotIds }).then(() => {
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
 * 打开热门推荐选项页面。
 */
function handleOpenShopHotItem(row: ShopHot) {
  router.push({
    path: "/shop/hot-item",
    query: { hotId: row.id, title: `【${row.title}】热门推荐选项` }
  });
}
</script>
