<!-- 商品 -->
<template>
  <div class="main-box">
    <TreeFilter
      label="name"
      title="分类列表"
      :request-api="requestCategoryTreeFilter"
      :show-all="false"
      :default-value="categoryFilterValue"
      @change="changeTreeFilter"
    />

    <div class="table-box">
      <ProTable
        ref="proTable"
        row-key="id"
        :columns="columns"
        :header-actions="headerActions"
        :request-api="requestGoodsTable"
        :init-param="initParam"
      >
        <template #name="scope">
          <el-link v-if="BUTTONS['goods:info:detail']" type="primary" @click.stop="handleOpenDetail(scope.row)">
            {{ scope.row.name }}
          </el-link>
          <span v-else>{{ scope.row.name }}</span>
        </template>
      </ProTable>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen, List, Tickets } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import TreeFilter from "@/components/TreeFilter/index.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defGoodsService } from "@/api/admin/goods";
import { defGoodsCategoryService } from "@/api/admin/goods_category";
import type { Goods, PageGoodsRequest } from "@/rpc/admin/goods";
import type { TreeOptionResponse_Option } from "@/rpc/common/common";
import { GoodsStatus } from "@/rpc/common/enum";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";

defineOptions({
  name: "Goods",
  inheritAttrs: false
});

type CategoryFilterNode = {
  id: string;
  name: string;
  children?: CategoryFilterNode[];
};

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const router = useRouter();

const initParam = reactive({
  categoryId: undefined as number | undefined
});
const categoryFilterValue = ref("");

/** 商品表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  {
    prop: "picture",
    label: "商品主图",
    minWidth: 150,
    cellType: "image",
    imageProps: {
      previewWidth: 400,
      previewHeight: 400,
      width: 60,
      height: 60
    }
  },
  { prop: "name", label: "商品名称", minWidth: 200, search: { el: "input" } },
  { prop: "categoryName", label: "分类", width: 140 },
  { prop: "desc", label: "商品描述", minWidth: 200 },
  { prop: "initSaleNum", label: "初始销量", align: "right" },
  { prop: "realSaleNum", label: "真实销量", align: "right" },
  { prop: "price", label: "价格（元）", align: "right", cellType: "money" },
  { prop: "discountPrice", label: "折扣价格（元）", align: "right", cellType: "money" },
  {
    prop: "status",
    label: "状态",
    width: 100,
    search: { el: "select" },
    cellType: "status",
    statusProps: {
      activeValue: GoodsStatus.PUT_ON,
      inactiveValue: GoodsStatus.PULL_OFF,
      activeText: "上架",
      inactiveText: "下架",
      disabled: () => !BUTTONS.value["goods:info:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as Goods)
    }
  },
  { prop: "createdAt", label: "创建时间", width: 180 },
  { prop: "updatedAt", label: "更新时间", width: 180 },
  {
    prop: "operation",
    label: "操作",
    width: 300,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "库存",
        type: "primary",
        link: true,
        icon: List,
        hidden: () => !BUTTONS.value["goods:info:sku"],
        onClick: scope => handleOpenSku(scope.row as Goods)
      },
      {
        label: "属性",
        type: "primary",
        link: true,
        icon: Tickets,
        hidden: () => !BUTTONS.value["goods:info:prop"],
        onClick: scope => handleOpenProp(scope.row as Goods)
      },
      {
        label: "编辑",
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["goods:info:update"],
        onClick: scope => handleOpenDialog(scope.row as Goods)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["goods:info:delete"],
        onClick: scope => handleDelete(scope.row as Goods)
      }
    ]
  }
];

/** 商品顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["goods:info:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["goods:info:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as Goods[])
  }
];

/**
 * 递归转换分类树筛选数据，适配 TreeFilter 组件字段。
 */
function transformCategoryFilterNodes(options: TreeOptionResponse_Option[] = []): CategoryFilterNode[] {
  return options.map(option => ({
    id: String(option.value),
    name: option.label,
    children: transformCategoryFilterNodes(option.children ?? [])
  }));
}

/**
 * 请求分类树筛选数据。
 */
async function requestCategoryTreeFilter() {
  const response = await defGoodsCategoryService.OptionGoodsCategory({});
  return {
    data: transformCategoryFilterNodes(response.list ?? [])
  };
}

/**
 * 切换分类树筛选时同步更新表格查询参数。
 */
function changeTreeFilter(value: string) {
  categoryFilterValue.value = value ?? "";
  initParam.categoryId = value ? Number(value) : undefined;
  if (proTable.value) {
    proTable.value.pageable.pageNum = 1;
    proTable.value.search();
  }
}

/**
 * 请求商品分页列表，并统一处理分页参数。
 */
async function requestGoodsTable(params: PageGoodsRequest) {
  const data = await defGoodsService.PageGoods(
    buildPageRequest({
      ...params,
      categoryId: initParam.categoryId
    })
  );
  return { data };
}

/**
 * 统一处理页面跳转；若路由实例未生效，则降级为浏览器地址跳转。
 */
function navigateTo(path: string, query?: Record<string, string | number>) {
  const target = { path, query };
  router.push(target).catch(() => {
    const resolved = router.resolve(target);
    window.location.href = resolved.href;
  });
}

/**
 * 打开商品编辑页。
 */
function handleOpenDialog(row?: Goods) {
  if (row?.id) {
    navigateTo("/goods/edit", { goodsId: row.id, title: `【${row.name}】商品编辑` });
    return;
  }

  navigateTo("/goods/edit");
}

/**
 * 在商品状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: Goods) {
  const nextStatus = row.status === GoodsStatus.PUT_ON ? GoodsStatus.PULL_OFF : GoodsStatus.PUT_ON;
  const text = nextStatus === GoodsStatus.PUT_ON ? "上架" : "下架";
  const goodsName = row.name || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(`是否确定${text}商品？\n商品名称：${goodsName}`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defGoodsService.SetGoodsStatus({ id: row.id, status: nextStatus });
    ElMessage.success(`${text}成功`);
    proTable.value?.getTableList();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除商品，兼容单条删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | Goods | Goods[] | { [key: string]: any }[]) {
  const goodsList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as Goods[])
    : selected && typeof selected === "object"
      ? [selected as Goods]
      : [];
  const goodsIds = (
    goodsList.length ? goodsList.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!goodsIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  const confirmMessage = goodsList.length
    ? goodsList.length === 1
      ? `是否确定删除商品？\n商品名称：${goodsList[0].name || `ID:${goodsList[0].id}`}`
      : `确认删除已选中的 ${goodsList.length} 个商品吗？`
    : "确认删除已选中的商品吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defGoodsService.DeleteGoods({ value: goodsIds }).then(() => {
        ElMessage.success("删除商品成功");
        proTable.value?.search();
      });
    },
    () => {
      ElMessage.info("已取消删除商品");
    }
  );
}

/**
 * 打开商品属性页。
 */
function handleOpenProp(row: Goods) {
  navigateTo("/goods/prop", { goodsId: row.id, title: `【${row.name}】商品属性` });
}

/**
 * 打开商品规格页。
 */
function handleOpenSku(row: Goods) {
  navigateTo("/goods/sku", { goodsId: row.id, title: `【${row.name}】商品规格` });
}

/**
 * 打开商品详情页。
 */
function handleOpenDetail(row: Goods) {
  navigateTo("/goods/detail", { goodsId: row.id, title: `【${row.name}】商品详情` });
}
</script>
