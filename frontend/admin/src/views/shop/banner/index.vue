<!-- 轮播图 -->
<template>
  <div class="app-container">
    <div class="search-bar">
      <el-form ref="queryFormRef" :model="queryParams" :inline="true">
        <el-form-item label="位置" prop="site">
          <Dict v-model="queryParams.site" code="shop_banner_site" />
        </el-form-item>
        <el-form-item label="跳转类型" prop="type">
          <Dict v-model="queryParams.type" code="shop_banner_type" />
        </el-form-item>
        <el-form-item label="状态" prop="status">
          <Dict v-model="queryParams.status" code="status" />
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
          v-hasPerm="['shop:banner:create']"
          type="success"
          icon="plus"
          @click="handleOpenDialog()"
        >
          新增
        </el-button>
        <el-button
          v-hasPerm="['shop:banner:create']"
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
        <el-table-column label="位置" prop="site">
          <template #default="scope">
            <DictLabel v-model="scope.row.site" code="shop_banner_site" />
          </template>
        </el-table-column>
        <el-table-column prop="picture" label="图片" min-width="150" align="center">
          <template #default="scope">
            <el-popover placement="right" :width="400" trigger="hover">
              <img :src="scope.row.picture" width="400" height="400" />
              <template #reference>
                <img :src="scope.row.picture" style="max-width: 60px; max-height: 60px" />
              </template>
            </el-popover>
          </template>
        </el-table-column>
        <el-table-column label="跳转类型">
          <template #default="scope">
            <DictLabel v-model="scope.row.type" code="shop_banner_type" />
          </template>
        </el-table-column>
        <el-table-column label="跳转链接" prop="href" />
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
              :disabled="!useUserStore().hasPerm('shop:banner:status')"
              @change="handleSetStatus(scope.row)"
            />
          </template>
        </el-table-column>
        <el-table-column label="创建时间" prop="createdAt" align="center" />
        <el-table-column label="更新时间" prop="updatedAt" align="center" />
        <el-table-column fixed="right" label="操作" width="150">
          <template #default="scope">
            <el-button
              v-hasPerm="['shop:banner:update']"
              type="primary"
              size="small"
              link
              icon="edit"
              @click="handleOpenDialog(scope.row.id)"
            >
              编辑
            </el-button>
            <el-button
              v-hasPerm="['shop:banner:delete']"
              type="danger"
              size="small"
              link
              icon="delete"
              @click="handleDelete(scope.row.id)"
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

    <!-- 轮播图表单弹窗 -->
    <el-dialog
      v-model="dialog.visible"
      :title="dialog.title"
      width="500px"
      @close="handleCloseDialog"
    >
      <el-form
        ref="dataFormRef"
        :model="formData"
        :rules="rules"
        label-suffix=":"
        label-width="100px"
      >
        <el-form-item label="位置" prop="site">
          <Dict v-model="formData.site" code="shop_banner_site" />
        </el-form-item>
        <el-form-item label="照片" prop="picture">
          <single-image-upload v-model="formData.picture" file-type="banner" />
        </el-form-item>
        <el-form-item label="跳转类型" prop="type">
          <Dict v-model="formData.type" code="shop_banner_type" />
        </el-form-item>
        <el-form-item
          v-if="formData.type == ShopBannerType.GOODS_DETAIL"
          label="跳转链接"
          prop="value"
        >
          <el-select v-model="formData.href" placeholder="请选择">
            <el-option
              v-for="item in goodsList"
              :key="item.id"
              :label="item.name"
              :value="item.id + ''"
            />
          </el-select>
        </el-form-item>
        <el-form-item
          v-if="formData.type == ShopBannerType.CATEGORY_DETAIL"
          label="跳转链接"
          prop="value"
        >
          <el-tree-select
            v-model="formData.href"
            placeholder="请选择商品分类"
            :data="goodsCategoryOptions"
            filterable
          />
        </el-form-item>
        <el-form-item
          v-if="formData.type == ShopBannerType.WEB_VIEW || formData.type == ShopBannerType.MINI"
          label="跳转链接"
          prop="value"
        >
          <el-input v-model="formData.href" placeholder="请输入跳转链接" />
        </el-form-item>
        <el-form-item label="排序" prop="sort">
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
import { useUserStore } from "@/store";
import { defShopBannerService } from "@/api/admin/shop_banner";
import { PageShopBannerRequest, ShopBanner, ShopBannerForm } from "@/rpc/admin/shop_banner";
import type { ListGoodsResponse_Goods } from "@/rpc/admin/goods";
import { ref } from "vue";
import { defGoodsService } from "@/api/admin/goods";
import { ShopBannerType, Status } from "@/rpc/common/enum";
import { TreeOptionResponse_Option } from "@/rpc/common/common";
import { defGoodsCategoryService } from "@/api/admin/goods_category";

defineOptions({
  name: "ShopBanner",
  inheritAttrs: false,
});

const queryFormRef = ref();
const dataFormRef = ref();

const loading = ref(false);
const ids = ref<number[]>([]);
const total = ref(0);
const goodsList = ref<ListGoodsResponse_Goods[]>([]);
const goodsCategoryOptions = ref<Category_Option[]>([]);

const queryParams = reactive<PageShopBannerRequest>({
  /** 位置：1、首页。2、分类页 */
  site: undefined,
  /** 跳转类型1、页面2、H5 3、小程序（小程序使用） */
  type: undefined,
  /** 状态 */
  status: undefined,
  /** 当前页码 */
  pageNum: 0,
  /** 每一页的行数 */
  pageSize: 10,
});

// 轮播图表格数据
const pageData = ref<ShopBanner[]>([]);

const dialog = reactive({
  title: "",
  visible: false,
});

const formData = reactive<ShopBannerForm>({
  /** 轮播图ID */
  id: 0,
  /** 位置：1、首页。2、分类页 */
  site: undefined,
  /** 图片链接 */
  picture: "",
  /** 跳转地址 */
  href: "",
  /** 跳转类型1、页面2、H5 3、小程序（小程序使用） */
  type: undefined,
  /** 排序 */
  sort: 1,
  /** 状态 */
  status: Status.ENABLE,
});

const rules = reactive({
  site: [{ required: true, message: "请选择位置", trigger: "change" }],
  picture: [{ required: true, message: "请选择上传图片", trigger: "change" }],
  type: [{ required: true, message: "请选择跳转类型", trigger: "change" }],
  href: [{ required: true, message: "跳转地址不能为空", trigger: "blur" }],
});

interface Category_Option {
  /** 选项名 */
  label: string;
  /** 选项值 */
  value: string;
  /** 是否禁用 */
  disabled: boolean;
  /** 子节点树 */
  children: Category_Option[];
}

const categoryOption = (oldCategories: TreeOptionResponse_Option[]): Category_Option[] => {
  return oldCategories.map((oldCategory) => {
    const newCategory: Category_Option = {
      value: oldCategory.value.toString(),
      label: oldCategory.label,
      disabled: oldCategory.disabled,
      children: oldCategory.children ? categoryOption(oldCategory.children) : [],
    };
    return newCategory;
  });
};

// 查询轮播图
function handleQuery() {
  loading.value = true;
  defShopBannerService
    .PageShopBanner(queryParams)
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

// 打开轮播图弹窗
async function handleOpenDialog(bannerId?: number) {
  const listGoodsResponse = await defGoodsService.ListGoods({
    name: "",
  });
  goodsList.value = listGoodsResponse.list || [];

  const optionGoodsCategoryResponse = await defGoodsCategoryService.OptionGoodsCategory({});
  goodsCategoryOptions.value = categoryOption(optionGoodsCategoryResponse.list || []);
  dialog.visible = true;
  if (bannerId) {
    dialog.title = "修改轮播图";
    defShopBannerService
      .GetShopBanner({
        value: bannerId,
      })
      .then((data) => {
        Object.assign(formData, data);
      });
  } else {
    dialog.title = "新增轮播图";
    formData.id = 0;
  }
}

// 轮播图表单提交
function handleSubmit() {
  dataFormRef.value.validate((valid: any) => {
    if (valid) {
      loading.value = true;
      const id = formData.id;
      if (id) {
        defShopBannerService
          .UpdateShopBanner(formData)
          .then(() => {
            ElMessage.success("修改成功");
            handleCloseDialog();
            handleResetQuery();
          })
          .finally(() => (loading.value = false));
      } else {
        defShopBannerService
          .CreateShopBanner(formData)
          .then(() => {
            ElMessage.success("新增成功");
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
  formData.href = "";
}

// 关闭轮播图弹窗
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

// 设置轮播图状态
function handleSetStatus(row: ShopBanner) {
  let text = row.status === Status.ENABLE ? "启用" : "禁用";
  ElMessageBox.confirm(`是否确定${text}轮播图?`, "提示", {
    confirmButtonText: "确认",
    cancelButtonText: "取消",
    type: "warning",
  })
    .then(() => {
      defShopBannerService.SetShopBannerStatus({ id: row.id, status: row.status }).then(() => {
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
 * 删除轮播图
 *
 * @param bannerId 轮播图ID
 */
function handleDelete(bannerId?: number) {
  const dictIds = [bannerId || ids.value].join(",");
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
      defShopBannerService
        .DeleteShopBanner({
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

onMounted(() => {
  handleQuery();
});
</script>
