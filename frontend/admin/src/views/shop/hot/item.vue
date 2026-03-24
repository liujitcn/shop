<!-- 热门推荐数据 -->
<template>
  <div class="app-container">
    <div class="search-bar mt-5">
      <el-form ref="queryFormRef" :model="queryParams" :inline="true">
        <el-form-item label="热门推荐标题" prop="title">
          <el-input
            v-model="queryParams.title"
            placeholder="热门推荐标题"
            clearable
            @keyup.enter="handleQuery"
          />
        </el-form-item>
        <el-form-item label="状态" prop="status">
          <Dict v-model="queryParams.status" code="status" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" @click="handleQuery()">搜索</el-button>
          <el-button icon="refresh" @click="handleResetQuery()">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <el-card shadow="never">
      <div class="mb-[10px]">
        <el-button
          v-hasPerm="['shop:hot-item:create']"
          type="success"
          icon="plus"
          @click="handleOpenDialog()"
        >
          新增
        </el-button>
        <el-button
          v-hasPerm="['shop:hot-item:delete']"
          type="danger"
          :disabled="ids.length === 0"
          icon="delete"
          @click="handleDelete()"
        >
          删除
        </el-button>
      </div>

      <el-table
        v-loading="loading"
        highlight-current-row
        :data="tableData"
        border
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" align="center" />
        <el-table-column title="热门推荐标题" prop="title" />
        <el-table-column label="排序" prop="sort" />
        <el-table-column label="状态" align="center" prop="status" width="80">
          <template #default="scope">
            <el-switch
              v-model="scope.row.status"
              inline-prompt
              :active-value="Status.ENABLE"
              :inactive-value="Status.DISABLE"
              active-text="启用"
              inactive-text="禁用"
              :disabled="!useUserStore().hasPerm('shop:hot-item:status')"
              @change="handleSetStatus(scope.row)"
            />
          </template>
        </el-table-column>
        <el-table-column label="创建时间" prop="createdAt" align="center" />
        <el-table-column label="更新时间" prop="updatedAt" align="center" />
        <el-table-column fixed="right" label="操作" align="center" width="200">
          <template #default="scope">
            <el-button
              v-hasPerm="['shop:hot-item:update']"
              type="primary"
              link
              size="small"
              icon="edit"
              @click.stop="handleOpenDialog(scope.row.id)"
            >
              编辑
            </el-button>
            <el-button
              v-hasPerm="['shop:hot-item:delete']"
              type="danger"
              link
              size="small"
              icon="delete"
              @click.stop="handleDelete(scope.row.id)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <pagination
        v-if="total > 0"
        v-model:total="total"
        v-model:page="queryParams.pageNum"
        v-model:limit="queryParams.pageSize"
        @pagination="handleQuery"
      />
    </el-card>

    <!--热门推荐弹窗-->
    <el-dialog
      v-model="dialog.visible"
      :title="dialog.title"
      width="1400px"
      @close="handleCloseDialog"
    >
      <el-form ref="dataFormRef" :model="formData" :rules="rules" label-width="150px">
        <el-card shadow="never">
          <el-form-item label="热门推荐标题" prop="title">
            <el-input v-model="formData.title" placeholder="热门推荐标题" />
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
          <el-form-item label="排序">
            <el-input-number
              v-model="formData.sort"
              controls-position="right"
              :min="1"
              :precision="0"
              :step="1"
            />
          </el-form-item>
          <el-form-item label="推荐商品">
            <el-transfer
              v-model="formData.goodsIds"
              :titles="['可选商品', '已选商品']"
              filterable
              :data="transferData"
            >
              <template #default="{ option }">
                <el-popover effect="light" trigger="hover" placement="top" width="auto">
                  <template #default>
                    <div>价格: {{ formatPrice(option.price) }}</div>
                  </template>
                  <template #reference>{{ option.label }}</template>
                </el-popover>
              </template>
            </el-transfer>
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
import { ref, computed, watch } from "vue";
import { useUserStore } from "@/store";

defineOptions({
  name: "ShopHotItem",
  inheritAttrs: false,
});

import { defShopHotService } from "@/api/admin/shop_hot";
import type { ShopHotItem, ShopHotItemForm, PageShopHotItemRequest } from "@/rpc/admin/shop_hot";
import type { ListGoodsResponse_Goods } from "@/rpc/admin/goods";
import { defGoodsService } from "@/api/admin/goods";
import { formatPrice } from "../../../utils/utils";
import { Status } from "@/rpc/common/enum";

const route = useRoute();

const hotId = ref(route.query.hotId as unknown as number);

const queryFormRef = ref(ElForm);
const dataFormRef = ref(ElForm);

const loading = ref(false);
const ids = ref<number[]>([]);
const total = ref(0);

const queryParams = reactive<PageShopHotItemRequest>({
  /** 热门推荐id */
  hotId: hotId.value,
  /** 热门推荐属性名称 */
  title: "",
  /** 状态 */
  status: undefined,
  /** 当前页码 */
  pageNum: 0,
  /** 每一页的行数 */
  pageSize: 10,
});

const tableData = ref<ShopHotItem[]>();

const dialog = reactive({
  title: "",
  visible: false,
});

const formData = reactive<ShopHotItemForm>({
  /** 热门推荐ID */
  id: 0,
  /** 热门推荐ID */
  hotId: hotId.value,
  /** 热门推荐标题 */
  title: "",
  /** 排序 */
  sort: 1,
  /** 商品Id */
  goodsIds: [],
  /** 状态 */
  status: Status.ENABLE,
});

// 监听路由参数变化，更新热门推荐数据
watch(
  () => [route.query.hotId],
  ([newHotId]) => {
    queryParams.hotId = newHotId as unknown as number;
    hotId.value = newHotId as unknown as number;
    handleQuery();
  }
);
const rules = computed(() => {
  const rules: Partial<Record<string, any>> = {
    title: [{ required: true, message: "请输入热门推荐选项标题", trigger: "blur" }],
    status: [{ required: true, message: "状态不能为空", trigger: "blur" }],
  };
  return rules;
});

// 查询
function handleQuery() {
  loading.value = true;
  defShopHotService
    .PageShopHotItem(queryParams)
    .then((data) => {
      tableData.value = data.list;
      total.value = data.total;
    })
    .finally(() => {
      loading.value = false;
    });
}

// 重置查询
function handleResetQuery() {
  queryFormRef.value.resetFields();
  queryParams.pageNum = 1;
  handleQuery();
}

// 行选择
function handleSelectionChange(selection: any) {
  ids.value = selection.map((item: any) => item.id);
}
const goodsList = ref<ListGoodsResponse_Goods[]>([]);
// api 穿梭框数据
const transferData = computed(() => {
  return goodsList.value.map((item) => ({
    ...item,
    key: item.id,
    label: `${item.categoryName}/${item.name}`,
  }));
});

// 打开弹窗
async function handleOpenDialog(hotItemId?: number) {
  const listGoodsResponse = await defGoodsService.ListGoods({
    name: "",
  });
  goodsList.value = listGoodsResponse.list || [];
  dialog.visible = true;
  if (hotItemId) {
    dialog.title = "修改热门推荐数据";
    defShopHotService
      .GetShopHotItem({
        value: hotItemId,
      })
      .then((data) => {
        Object.assign(formData, { ...data });
      });
  } else {
    dialog.title = "新增热门推荐数据";
  }
}

// 提交表单
function handleSubmitClick() {
  dataFormRef.value.validate((isValid: boolean) => {
    if (isValid) {
      loading.value = true;
      const id = formData.id;
      formData.hotId = hotId.value;
      if (id) {
        defShopHotService
          .UpdateShopHotItem(formData)
          .then(() => {
            ElMessage.success("修改成功");
            handleCloseDialog();
            handleQuery();
          })
          .finally(() => (loading.value = false));
      } else {
        defShopHotService
          .CreateShopHotItem(formData)
          .then(() => {
            ElMessage.success("新增成功");
            handleCloseDialog();
            handleQuery();
          })
          .finally(() => (loading.value = false));
      }
    }
  });
}

// 关闭弹窗
function handleCloseDialog() {
  dialog.visible = false;

  dataFormRef.value.resetFields();
  dataFormRef.value.clearValidate();

  formData.id = 0;
  formData.goodsIds = [];
}

// 设置热门推荐状态
function handleSetStatus(row: ShopHotItem) {
  let text = row.status === Status.ENABLE ? "启用" : "禁用";
  ElMessageBox.confirm(`是否确定${text}热门推荐属性为：${row.title}?`, "提示", {
    confirmButtonText: "确认",
    cancelButtonText: "取消",
    type: "warning",
  })
    .then(() => {
      defShopHotService.SetShopHotItemStatus({ id: row.id, status: row.status }).then(() => {
        handleQuery();
        ElMessage.success(`${text}成功`);
      });
    })
    .catch(() => {
      if (row.status == 0) {
        row.status = 1;
      } else {
        row.status = 0;
      }
    });
}

/**
 * 删除热门推荐
 *
 * @param hotItemId 热门推荐ID
 */
function handleDelete(hotItemId?: number) {
  const hotItemIds = [hotItemId || ids.value].join(",");
  if (!hotItemIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }
  ElMessageBox.confirm("确认删除已选中的数据项?", "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning",
  }).then(
    () => {
      defShopHotService
        .DeleteShopHotItem({
          value: hotItemIds,
        })
        .then(() => {
          ElMessage.success("删除成功");
          handleResetQuery();
        });
    },
    () => {
      ElMessage.info("已取消删除");
    }
  );
}

onMounted(() => {
  handleQuery();
});
</script>
<style scoped>
/* 使用深度选择器穿透 scoped 样式 */
:deep(.el-transfer-panel) {
  width: 450px; /* 面板整体宽度 */
}

/* 调整列表区域 */
:deep(.el-transfer-panel__list) {
  height: 400px; /* 列表高度 */
  width: 100%; /* 宽度继承父容器 */
}
</style>
