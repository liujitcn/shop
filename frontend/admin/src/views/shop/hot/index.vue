<!-- 热门推荐 -->
<template>
  <div class="app-container">
    <div class="search-bar">
      <el-form ref="queryFormRef" :model="queryParams" :inline="true">
        <el-form-item label="热门推荐标题" prop="title">
          <el-input
            v-model="queryParams.title"
            placeholder="热门推荐标题"
            clearable
            @keyup.enter="handleQuery"
          />
        </el-form-item>
        <el-form-item label="热门推荐描述" prop="desc">
          <el-input
            v-model="queryParams.desc"
            placeholder="热门推荐描述"
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
          v-hasPerm="['shop:hot:create']"
          type="success"
          icon="plus"
          @click="handleOpenDialog()"
        >
          新增
        </el-button>
        <el-button
          v-hasPerm="['shop:hot:delete']"
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
        <el-table-column label="热门推荐标题" prop="title" />
        <el-table-column label="热门推荐描述" prop="desc" />
        <el-table-column label="排序" prop="sort" align="right" />
        <el-table-column label="状态" align="center" prop="status" width="80">
          <template #default="scope">
            <el-switch
              v-model="scope.row.status"
              inline-prompt
              :active-value="Status.ENABLE"
              :inactive-value="Status.DISABLE"
              active-text="启用"
              inactive-text="禁用"
              :disabled="!useUserStore().hasPerm('shop:hot:status')"
              @change="handleSetStatus(scope.row)"
            />
          </template>
        </el-table-column>
        <el-table-column label="创建时间" prop="createdAt" align="center" />
        <el-table-column label="更新时间" prop="updatedAt" align="center" />
        <el-table-column fixed="right" label="操作" align="center" width="220">
          <template #default="scope">
            <el-button
              v-hasPerm="['shop:hot:items']"
              type="primary"
              link
              size="small"
              :icon="List"
              @click.stop="handleOpenShopHotItem(scope.row)"
            >
              推荐选项
            </el-button>

            <el-button
              v-hasPerm="['shop:hot:update']"
              type="primary"
              link
              size="small"
              icon="edit"
              @click.stop="handleOpenDialog(scope.row.id)"
            >
              编辑
            </el-button>
            <el-button
              v-hasPerm="['shop:hot:delete']"
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
      width="1000px"
      @close="handleCloseDialog"
    >
      <el-form ref="dataFormRef" :model="formData" :rules="rules" label-width="150px">
        <el-card shadow="never">
          <el-form-item label="热门推荐标题" prop="title">
            <el-input
              v-model="formData.title"
              placeholder="请输入热门推荐标题"
            />
          </el-form-item>

          <el-form-item label="热门推荐描述" prop="desc">
            <el-input
              v-model="formData.desc"
              placeholder="请输入热门推荐描述"
            />
          </el-form-item>

          <el-form-item label="轮播图" prop="banner">
            <single-image-upload v-model="formData.banner" file-type="hot" />
          </el-form-item>

          <el-form-item label="推荐图片" prop="picture">
            <multi-image-upload v-model="formData.picture" file-type="hot" :limit="2" />
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
import { useUserStore } from "@/store";
import { List } from "@element-plus/icons-vue";
defineOptions({
  name: "ShopShopHot",
  inheritAttrs: false,
});

import { defShopHotService } from "@/api/admin/shop_hot";
import type { ShopHot, ShopHotForm, PageShopHotRequest } from "@/rpc/admin/shop_hot";

import router from "@/router";
import { Status } from "@/rpc/common/enum";

const queryFormRef = ref(ElForm);
const dataFormRef = ref(ElForm);

const loading = ref(false);
const ids = ref<number[]>([]);
const total = ref(0);

const queryParams = reactive<PageShopHotRequest>({
  /** 热门推荐标题 */
  title: "",
  /** 热门推荐编号 */
  desc: "",
  /** 状态 */
  status: undefined,
  /** 当前页码 */
  pageNum: 0,
  /** 每一页的行数 */
  pageSize: 10,
});

const tableData = ref<ShopHot[]>();

const dialog = reactive({
  title: "",
  visible: false,
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
  status: Status.ENABLE,
});

const rules = computed(() => {
  const rules: Partial<Record<string, any>> = {
    title: [{ required: true, message: "请输入热门推荐标题", trigger: "blur" }],
    desc: [{ required: true, message: "请输入热门推荐描述", trigger: "blur" }],
    banner: [{ required: true, message: "请上传轮播图", trigger: "blur" }],
    picture: [{ required: true, message: "请上传推荐图片", trigger: "blur" }],
    status: [{ required: true, message: "状态不能为空", trigger: "blur" }],
  };
  return rules;
});

// 查询
function handleQuery() {
  loading.value = true;
  defShopHotService
    .PageShopHot(queryParams)
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

/**
 * 打开弹窗
 *
 * @param hotId
 */
function handleOpenDialog(hotId?: number) {
  dialog.visible = true;
  if (hotId) {
    dialog.title = "修改热门推荐";
    defShopHotService
      .GetShopHot({
        value: hotId,
      })
      .then((data) => {
        Object.assign(formData, { ...data });
      });
  } else {
    dialog.title = "新增热门推荐";
  }
}

// 关闭弹窗
function handleCloseDialog() {
  dialog.visible = false;
  dataFormRef.value.resetFields();
  dataFormRef.value.clearValidate();

  formData.id = 0;
  formData.picture = [];
}

// 提交热门推荐表单
function handleSubmitClick() {
  dataFormRef.value.validate((isValid: boolean) => {
    if (isValid) {
      const hotId = formData.id;
      loading.value = true;
      if (hotId) {
        defShopHotService
          .UpdateShopHot(formData)
          .then(() => {
            ElMessage.success("修改成功");
            handleCloseDialog();
            handleQuery();
          })
          .finally(() => (loading.value = false));
      } else {
        defShopHotService
          .CreateShopHot(formData)
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

// 设置热门推荐状态
function handleSetStatus(row: ShopHot) {
  let text = row.status === Status.ENABLE ? "启用" : "禁用";
  ElMessageBox.confirm(`是否确定${text}热门推荐为：${row.title}?`, "提示", {
    confirmButtonText: "确认",
    cancelButtonText: "取消",
    type: "warning",
  })
    .then(() => {
      defShopHotService.SetShopHotStatus({ id: row.id, status: row.status }).then(() => {
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
 * @param hotId 热门推荐ID
 */
function handleDelete(hotId?: number) {
  const hotIds = [hotId || ids.value].join(",");
  if (!hotIds) {
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
        .DeleteShopHot({
          value: hotIds,
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

// 打开热门推荐数据
function handleOpenShopHotItem(row: ShopHot) {
  router.push({
    path: "/shop/hot-item",
    query: { hotId: row.id, title: "【" + row.title + "】热门推荐选项" },
  });
}

onMounted(() => {
  handleQuery();
});
</script>
