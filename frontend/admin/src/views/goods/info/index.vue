<!-- 商品 -->
<template>
  <div class="app-container">
    <el-row :gutter="20">
      <!-- 分类树 -->
      <el-col :lg="4" :xs="24" class="mb-[12px]">
        <CategoryTree v-model="queryParams.categoryId" @node-click="handleQuery" />
      </el-col>

      <!-- 商品列表 -->
      <el-col :lg="20" :xs="24">
        <div class="search-bar">
          <el-form ref="queryFormRef" :model="queryParams" :inline="true">
            <el-form-item label="商品名称" prop="name">
              <el-input
                v-model="queryParams.name"
                placeholder="请输入商品名称"
                clearable
                @keyup.enter="handleQuery"
              />
            </el-form-item>
            <el-form-item label="状态" prop="status">
              <Dict v-model="queryParams.status" code="goods_status" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" icon="search" @click="handleQuery">搜索</el-button>
              <el-button icon="refresh" @click="handleResetQuery">重置</el-button>
            </el-form-item>
          </el-form>
        </div>

        <el-card shadow="never">
          <div class="mb-10px">
            <el-button
              v-hasPerm="['goods:info:create']"
              type="success"
              icon="plus"
              @click="handleOpenDialog()"
            >
              新增
            </el-button>
            <el-button
              v-hasPerm="['goods:info:create']"
              type="danger"
              :disabled="ids.length === 0"
              icon="delete"
              @click="handleDelete()"
            >
              删除
            </el-button>
          </div>

          <el-table
            ref="dataTableRef"
            v-loading="loading"
            :data="pageData"
            highlight-current-row
            @selection-change="handleSelectionChange"
          >
            <el-table-column type="selection" width="55" align="center" />
            <el-table-column label="商品主图" min-width="150" align="center">
              <template #default="scope">
                <el-popover placement="right" :width="400" trigger="hover">
                  <img :src="scope.row.picture" width="400" height="400" />
                  <template #reference>
                    <img :src="scope.row.picture" style="max-width: 60px; max-height: 60px" />
                  </template>
                </el-popover>
              </template>
            </el-table-column>
            <el-table-column label="商品名称">
              <template #default="scope">
                <el-link
                  v-if="useUserStore().hasPerm('goods:info:detail')"
                  type="primary"
                  @click.stop="handleOpenDetail(scope.row)"
                >
                  {{ scope.row.name }}
                </el-link>
                <span v-else>{{ scope.row.name }}</span>
              </template>
            </el-table-column>
            <el-table-column label="商品描述" prop="desc" />
            <el-table-column label="初始销量" align="right">
              <template #default="scope">
                {{ scope.row.initSaleNum || 0 }}
              </template>
            </el-table-column>
            <el-table-column label="真实销量" align="right">
              <template #default="scope">
                {{ scope.row.realSaleNum || 0 }}
              </template>
            </el-table-column>
            <el-table-column label="价格（元）" align="right">
              <template #default="scope">
                {{ scope.row.price / 100 || 0 }}
              </template>
            </el-table-column>
            <el-table-column label="折扣价格（元）" align="right">
              <template #default="scope">
                {{ scope.row.discountPrice / 100 || 0 }}
              </template>
            </el-table-column>
            <el-table-column label="状态" align="center" prop="status" width="80">
              <template #default="scope">
                <el-switch
                  v-model="scope.row.status"
                  inline-prompt
                  :active-value="GoodsStatus.PUT_ON"
                  :inactive-value="GoodsStatus.PULL_OFF"
                  active-text="上架"
                  inactive-text="下架"
                  :disabled="!useUserStore().hasPerm('goods:info:status')"
                  @change="handleSetStatus(scope.row)"
                />
              </template>
            </el-table-column>
            <el-table-column label="创建时间" prop="createdAt" align="center" />
            <el-table-column label="更新时间" prop="updatedAt" align="center" />
            <el-table-column fixed="right" label="操作" width="260">
              <template #default="scope">
                <el-button
                  v-hasPerm="['goods:info:sku']"
                  type="primary"
                  link
                  size="small"
                  :icon="List"
                  @click.stop="handleOpenSku(scope.row)"
                >
                  库存
                </el-button>
                <el-button
                  v-hasPerm="['goods:info:update']"
                  type="primary"
                  size="small"
                  link
                  icon="edit"
                  @click="handleOpenDialog(scope.row)"
                >
                  编辑
                </el-button>
                <el-button
                  v-hasPerm="['goods:info:delete']"
                  type="danger"
                  size="small"
                  link
                  icon="delete"
                  @click="handleDelete(scope.row.id)"
                >
                  删除
                </el-button>
                <el-button
                  v-hasPerm="['goods:info:prop']"
                  type="primary"
                  link
                  size="small"
                  :icon="List"
                  @click.stop="handleOpenProp(scope.row)"
                >
                  属性
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
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { useUserStore } from "@/store";
import { List } from "@element-plus/icons-vue";
import { defGoodsService } from "@/api/admin/goods";
import { Goods, PageGoodsRequest } from "@/rpc/admin/goods";
import CategoryTree from "./components/category-tree.vue";

import router from "@/router";
import { GoodsStatus } from "@/rpc/common/enum";

defineOptions({
  name: "Goods",
  inheritAttrs: false,
});

const queryFormRef = ref();

const loading = ref(false);
const ids = ref<number[]>([]);
const total = ref(0);

const queryParams = reactive<PageGoodsRequest>({
  /** 商品名称 */
  name: "",
  /** 分类id */
  categoryId: 0,
  /** 状态 */
  status: undefined,
  /** 当前页码 */
  pageNum: 0,
  /** 每一页的行数 */
  pageSize: 10,
});

// 商品表格数据
const pageData = ref<Goods[]>([]);

// 查询商品
function handleQuery() {
  loading.value = true;
  defGoodsService
    .PageGoods(queryParams)
    .then((data) => {
      pageData.value = data.list;
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

// 行复选框选中项变化
function handleSelectionChange(selection: any) {
  ids.value = selection.map((item: any) => item.id);
}

// 打开商品弹窗
function handleOpenDialog(row?: Goods) {
  if (row && row.id) {
    router.push({
      path: "/goods/edit",
      query: { goodsId: row.id, title: "【" + row.name + "】商品编辑" },
    });
  } else {
    router.push({
      path: "/goods/edit",
    });
  }
}

// 设置商品状态
function handleSetStatus(row: Goods) {
  let text = row.status === GoodsStatus.PUT_ON ? "上架" : "下架";
  ElMessageBox.confirm(`是否确定${text}商品为：${row.name}?`, "提示", {
    confirmButtonText: "确认",
    cancelButtonText: "取消",
    type: "warning",
  })
    .then(() => {
      defGoodsService.SetGoodsStatus({ id: row.id, status: row.status }).then(() => {
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
 * 删除商品
 *
 * @param goodsId 商品ID
 */
function handleDelete(goodsId?: number) {
  const dictIds = [goodsId || ids.value].join(",");
  if (!dictIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }
  ElMessageBox.confirm("确认删除已选中的数据项?", "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning",
  }).then(
    () => {
      defGoodsService
        .DeleteGoods({
          value: dictIds,
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

// 打开商品属性
function handleOpenProp(row: Goods) {
  router.push({
    path: "/goods/prop",
    query: { goodsId: row.id, title: "【" + row.name + "】商品属性" },
  });
}

// 打开sku
function handleOpenSku(row: Goods) {
  router.push({
    path: "/goods/sku",
    query: { goodsId: row.id, title: "【" + row.name + "】商品规格" },
  });
}

// 打开详情
function handleOpenDetail(row: Goods) {
  router.push({
    path: "/goods/detail",
    query: { goodsId: row.id, title: "【" + row.name + "】商品详情" },
  });
}

onMounted(() => {
  handleQuery();
});
</script>
