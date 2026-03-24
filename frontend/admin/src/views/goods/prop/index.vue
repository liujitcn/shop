<!-- 商品属性 -->
<template>
  <div class="app-container">
    <div class="search-bar mt-5">
      <el-form ref="queryFormRef" :model="queryParams" :inline="true">
        <el-form-item label="商品属性标签" prop="label">
          <el-input
            v-model="queryParams.label"
            placeholder="商品属性标签"
            clearable
            @keyup.enter="handleQuery"
          />
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
          v-hasPerm="['goods:prop:create']"
          type="success"
          icon="plus"
          @click="handleOpenDialog()"
        >
          新增
        </el-button>
        <el-button
          v-hasPerm="['goods:prop:delete']"
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
        <el-table-column label="商品属性标签" prop="label" />
        <el-table-column label="商品属性值" prop="value" />
        <el-table-column label="排序" prop="sort" align="right" />
        <el-table-column fixed="right" label="操作" align="center" width="220">
          <template #default="scope">
            <el-button
              v-hasPerm="['goods:prop:update']"
              type="primary"
              link
              size="small"
              icon="edit"
              @click.stop="handleOpenDialog(scope.row.id)"
            >
              编辑
            </el-button>
            <el-button
              v-hasPerm="['goods:prop:delete']"
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

    <!--商品属性弹窗-->
    <el-dialog
      v-model="dialog.visible"
      :title="dialog.title"
      width="820px"
      @close="handleCloseDialog"
    >
      <el-form ref="dataFormRef" :model="formData" :rules="rules" label-width="120px">
        <el-card shadow="never">
          <el-form-item label="商品属性标签" prop="label">
            <el-input v-model="formData.label" placeholder="请输入商品属性标签" />
          </el-form-item>
          <el-form-item label="商品属性值" prop="value">
            <el-input
              v-model="formData.value"
              placeholder="请输入商品属性值"
              type="textarea"
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
defineOptions({
  name: "GoodsProp",
  inheritAttrs: false,
});

import type { GoodsProp, PageGoodsPropRequest } from "@/rpc/admin/goods_prop";
import { defGoodsPropService } from "@/api/admin/goods_prop";

const route = useRoute();

const goodsId = ref(route.query.goodsId as unknown as number);

const queryFormRef = ref(ElForm);
const dataFormRef = ref(ElForm);

const loading = ref(false);
const ids = ref<number[]>([]);
const total = ref(0);

const queryParams = reactive<PageGoodsPropRequest>({
  /** 商品id */
  goodsId: goodsId.value,
  /** 商品属性名称 */
  label: "",
  /** 当前页码 */
  pageNum: 0,
  /** 每一页的行数 */
  pageSize: 10,
});

const tableData = ref<GoodsProp[]>();

const dialog = reactive({
  title: "",
  visible: false,
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
  sort: 1,
});

// 监听路由参数变化，更新商品属性
watch(
  () => [route.query.goodsId],
  ([newGoodsId]) => {
    queryParams.goodsId = newGoodsId as unknown as number;
    goodsId.value = newGoodsId as unknown as number;
    handleQuery();
  }
);
const rules = computed(() => {
  const rules: Partial<Record<string, any>> = {
    value: [{ required: true, message: "请输入商品属性值", trigger: "blur" }],
    label: [{ required: true, message: "请输入商品属性标签", trigger: "blur" }],
  };
  return rules;
});

// 查询
function handleQuery() {
  loading.value = true;
  defGoodsPropService
    .PageGoodsProp(queryParams)
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

// 打开弹窗
function handleOpenDialog(propId?: number) {
  dialog.visible = true;
  if (propId) {
    dialog.title = "修改商品属性";
    defGoodsPropService
      .GetGoodsProp({
        value: propId,
      })
      .then((data) => {
        Object.assign(formData, { ...data });
      });
  } else {
    dialog.title = "新增商品属性";
  }
}

// 提交表单
function handleSubmitClick() {
  dataFormRef.value.validate((isValid: boolean) => {
    if (isValid) {
      loading.value = true;
      const id = formData.id;
      formData.goodsId = goodsId.value;
      if (id) {
        defGoodsPropService
          .UpdateGoodsProp(formData)
          .then(() => {
            ElMessage.success("修改成功");
            handleCloseDialog();
            handleQuery();
          })
          .finally(() => (loading.value = false));
      } else {
        defGoodsPropService
          .CreateGoodsProp(formData)
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
  formData.sort = 1;
}

/**
 * 删除商品属性
 *
 * @param propId 商品属性ID
 */
function handleDelete(propId?: number) {
  const propIds = [propId || ids.value].join(",");
  if (!propIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }
  ElMessageBox.confirm("确认删除已选中的数据项?", "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning",
  }).then(
    () => {
      defGoodsPropService
        .DeleteGoodsProp({
          value: propIds,
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
