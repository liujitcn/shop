<!-- 规格 -->
<template>
  <div class="app-container">
    <div class="search-bar">
      <el-form ref="queryFormRef" :model="queryParams" :inline="true">
        <el-form-item label="规格编号" prop="skuCode">
          <el-input
            v-model="queryParams.skuCode"
            placeholder="请输入规格编号"
            clearable
            @keyup.enter="handleQuery"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" @click="handleQuery">搜索</el-button>
          <el-button icon="refresh" @click="handleResetQuery">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <el-card shadow="never">
      <el-table ref="dataTableRef" v-loading="loading" :data="pageData" highlight-current-row>
        <el-table-column
          v-for="(item, index) in specList"
          :key="index"
          align="center"
          :prop="'specItem' + index"
          :label="item.name"
        />
        <el-table-column label="规格编号" prop="skuCode" />
        <el-table-column label="初始销量" prop="initSaleNum" align="right">
          <template #default="scope">
            {{ scope.row.initSaleNum || 0 }}
          </template>
        </el-table-column>
        <el-table-column label="真实销量" prop="realSaleNum" align="right">
          <template #default="scope">
            {{ scope.row.realSaleNum || 0 }}
          </template>
        </el-table-column>
        <el-table-column label="价格（元）" align="right">
          <template #default="scope">
            {{ formatPrice(scope.row.price) }}
          </template>
        </el-table-column>
        <el-table-column label="折扣价格（元）" align="right">
          <template #default="scope">
            {{ formatPrice(scope.row.discountPrice) }}
          </template>
        </el-table-column>

        <el-table-column label="库存" prop="inventory" align="right" />

        <el-table-column fixed="right" label="操作" width="100">
          <template #default="scope">
            <el-button
              v-hasPerm="['goods:sku:update']"
              type="primary"
              size="small"
              link
              icon="edit"
              @click="handleOpenDialog(scope.row.id)"
            >
              编辑
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

    <!-- 规格表单弹窗 -->
    <el-dialog
      v-model="dialog.visible"
      :title="dialog.title"
      width="700px"
      @close="handleCloseDialog"
    >
      <el-form
        ref="dataFormRef"
        :model="formData"
        :rules="rules"
        label-suffix=":"
        label-width="180px"
      >
        <el-form-item label="规格编号">
          {{ formData.skuCode }}
        </el-form-item>
        <el-form-item label="规格内容">
          {{ formData.specItem }}
        </el-form-item>
        <el-form-item label="初始销量">
          {{ formData.initSaleNum }}
        </el-form-item>
        <el-form-item label="真实销量">
          {{ formData.realSaleNum }}
        </el-form-item>

        <el-form-item label="库存" prop="inventory">
          <el-input-number
            v-model="formData.inventory"
            controls-position="right"
            :min="1"
            :precision="0"
            :step="1"
          />
        </el-form-item>

        <el-form-item label="价格（元）" prop="price">
          <el-input-number v-model="formData.price" :precision="2" :step="0.1" />
        </el-form-item>

        <el-form-item label="折扣价格（元）" prop="discountPrice">
          <el-input-number v-model="formData.discountPrice" :precision="2" :step="0.1" />
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
import { formatPrice } from "@/utils/utils";

defineOptions({
  name: "GoodsSku",
  inheritAttrs: false,
});

import { defGoodsSkuService } from "@/api/admin/goods_sku";
import { defGoodsSpecService } from "@/api/admin/goods_spec";
import { GoodsSku, PageGoodsSkuRequest } from "@/rpc/admin/goods_sku";
import { GoodsSpec } from "@/rpc/admin/goods_spec";
const queryFormRef = ref();
const dataFormRef = ref();

const route = useRoute();
const goodsId = ref(route.query.goodsId as unknown as number);

const loading = ref(false);
const total = ref(0);

const queryParams = reactive<PageGoodsSkuRequest>({
  /** 商品id */
  goodsId: goodsId.value,
  /** 规格编号 */
  skuCode: "",
  /** 当前页码 */
  pageNum: 0,
  /** 每一页的行数 */
  pageSize: 10,
});

// 规格表格数据
const pageData = ref<GoodsSku[]>([]);
const specList = ref<GoodsSpec[]>([]);

const dialog = reactive({
  title: "",
  visible: false,
});

const formData = reactive<GoodsSku>({
  /** 商品SKUID */
  id: 0,
  /** 商品id */
  goodsId: 0,
  /** 商品图片 */
  picture: "",
  /** SKU SKU组成，需要与 goods_spec 数组顺序对应 */
  specItem: [],
  /** SKU编码 */
  skuCode: "",
  /** 当前价格(分) */
  price: 0,
  /** 折扣价格（分） */
  discountPrice: 0,
  /** 初始销量 */
  initSaleNum: 0,
  /** 真实销售数量 */
  realSaleNum: 0,
  /** 库存数量 */
  inventory: 0,
});

const rules = reactive({});

// 查询规格
function handleQuery() {
  loading.value = true;
  defGoodsSkuService
    .PageGoodsSku(queryParams)
    .then((data) => {
      const list = data.list || [];
      list.map((item) => {
        item.specItem.forEach((spec, index) => {
          (item as any)[`specItem${index}`] = spec;
        });
      });
      pageData.value = list;
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

// 打开规格弹窗
function handleOpenDialog(skuId: number) {
  dialog.visible = true;
  if (skuId) {
    dialog.title = "修改规格";
    defGoodsSkuService
      .GetGoodsSku({
        value: skuId,
      })
      .then((data) => {
        data.price = data.price / 100;
        data.discountPrice = data.discountPrice / 100;
        Object.assign(formData, data);
      });
  }
}

// 规格表单提交
function handleSubmit() {
  dataFormRef.value.validate((valid: any) => {
    if (valid) {
      loading.value = true;
      const submitsData = JSON.parse(JSON.stringify(formData));
      submitsData.price = submitsData.price * 100;
      submitsData.discountPrice = submitsData.discountPrice * 100;
      const id = submitsData.id;
      if (id) {
        defGoodsSkuService
          .UpdateGoodsSku(submitsData)
          .then(() => {
            ElMessage.success("修改成功");
            handleCloseDialog();
            handleResetQuery();
          })
          .finally(() => (loading.value = false));
      }
    }
  });
}

// 重置表单
function resetForm() {
  dataFormRef.value.resetFields();
  dataFormRef.value.clearValidate();
  formData.id = 0;
}

// 关闭规格弹窗
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

onMounted(() => {
  defGoodsSpecService
    .ListGoodsSpec({
      goodsId: goodsId.value,
    })
    .then((res) => {
      specList.value = res.list;
    });
  handleQuery();
});
</script>
<style scoped>
.flex-items {
  display: flex;
  align-items: center;
}
.ml-2 {
  margin-left: 0.5rem;
}
</style>
