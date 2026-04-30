<!-- 商品详情 -->
<template>
  <div v-loading="loading" class="app-container">
    <el-card class="goods-hero-card" shadow="never">
      <div class="goods-hero">
        <div class="goods-cover-panel">
          <el-image
            class="goods-cover-image"
            :src="coverImageSrc"
            :preview-src-list="coverPreviewList"
            fit="cover"
            preview-teleported
          >
            <template #error>
              <div class="image-placeholder">暂无主图</div>
            </template>
          </el-image>
        </div>

        <div class="goods-summary-panel">
          <div class="goods-summary-toolbar">
            <GoodsH5PreviewDrawer
              :goods-id="goodsId"
              class="goods-summary-toolbar__preview"
              button-text="预览"
              size="default"
              plain
              tooltip="预览"
            />
          </div>

          <div class="goods-summary-grid">
            <div class="goods-summary-card">
              <span class="goods-summary-card__label">总库存</span>
              <strong class="goods-summary-card__value">{{ totalInventory }}</strong>
            </div>
            <div class="goods-summary-card">
              <span class="goods-summary-card__label">销量</span>
              <strong class="goods-summary-card__value">{{ totalSaleNum }}</strong>
            </div>
            <div class="goods-summary-card">
              <span class="goods-summary-card__label">原价区间</span>
              <strong class="goods-summary-card__value">{{ originPriceRangeText }}</strong>
            </div>
            <div class="goods-summary-card">
              <span class="goods-summary-card__label">折后价区间</span>
              <strong class="goods-summary-card__value">{{ discountPriceRangeText }}</strong>
            </div>
            <button type="button" class="goods-summary-card goods-summary-card--action" @click="handleNavigateSection('prop')">
              <span class="goods-summary-card__label">商品属性</span>
              <strong class="goods-summary-card__value">{{ propCount }}</strong>
            </button>
            <button type="button" class="goods-summary-card goods-summary-card--action" @click="handleNavigateSection('spec')">
              <span class="goods-summary-card__label">规格项</span>
              <strong class="goods-summary-card__value">{{ specCount }}</strong>
            </button>
            <button type="button" class="goods-summary-card goods-summary-card--action" @click="handleNavigateSection('sku')">
              <span class="goods-summary-card__label">商品规格</span>
              <strong class="goods-summary-card__value">{{ skuCount }}</strong>
            </button>
            <button type="button" class="goods-summary-card goods-summary-card--action" @click="handleNavigateSection('comment')">
              <span class="goods-summary-card__label">商品评论</span>
              <strong class="goods-summary-card__value">{{ commentCount }}</strong>
            </button>
          </div>
        </div>
      </div>
    </el-card>

    <el-card class="goods-detail-panel" shadow="never">
      <el-tabs v-model="activeTabName" class="goods-detail-tabs">
        <el-tab-pane label="商品信息" name="basic">
          <div class="detail-tab-panel detail-tab-panel--basic">
            <div class="detail-info-panel">
              <el-descriptions :column="2" border class="goods-descriptions">
                <el-descriptions-item label="分类">{{ formData.category_name || "-" }}</el-descriptions-item>
                <el-descriptions-item label="上架状态">
                  <DictLabel :model-value="formData.status" code="goods_status" size="default" />
                </el-descriptions-item>
                <el-descriptions-item label="标题" :span="2">{{ formData.name || "-" }}</el-descriptions-item>
                <el-descriptions-item label="描述" :span="2">{{ formData.desc || "-" }}</el-descriptions-item>
                <el-descriptions-item label="轮播图" :span="2">
                  <div class="detail-media-list">
                    <el-image
                      v-for="(img, index) in bannerImageList"
                      :key="`banner-${index}`"
                      class="detail-media-item"
                      :src="img"
                      :preview-src-list="bannerImageList"
                      :initial-index="index"
                      fit="cover"
                      preview-teleported
                    >
                      <template #error>
                        <div class="detail-media-item__placeholder">图片加载失败</div>
                      </template>
                    </el-image>
                    <div v-if="!bannerImageList.length" class="detail-media-empty">暂无轮播图</div>
                  </div>
                </el-descriptions-item>
                <el-descriptions-item label="详情" :span="2">
                  <div class="detail-media-list">
                    <el-image
                      v-for="(img, index) in detailImageList"
                      :key="`detail-${index}`"
                      class="detail-media-item"
                      :src="img"
                      :preview-src-list="detailImageList"
                      :initial-index="index"
                      fit="cover"
                      preview-teleported
                    >
                      <template #error>
                        <div class="detail-media-item__placeholder">图片加载失败</div>
                      </template>
                    </el-image>
                    <div v-if="!detailImageList.length" class="detail-media-empty">暂无详情</div>
                  </div>
                </el-descriptions-item>
              </el-descriptions>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="商品属性" name="prop">
          <ProTable
            class="detail-table-content"
            row-key="label"
            :data="formData.prop_list"
            :columns="propColumns"
            :pagination="false"
            :tool-button="false"
          />
        </el-tab-pane>

        <el-tab-pane label="规格项" name="spec">
          <ProTable
            class="detail-table-content"
            row-key="name"
            :data="formData.spec_list"
            :columns="specColumns"
            :pagination="false"
            :tool-button="false"
          />
        </el-tab-pane>

        <el-tab-pane label="商品规格" name="sku">
          <ProTable
            class="detail-table-content"
            row-key="sku_code"
            :data="formData.sku_list"
            :columns="skuColumns"
            :pagination="false"
            :tool-button="false"
          >
            <template #picture="scope">
              <div class="sku-image-cell">
                <el-image
                  v-if="getSkuPictureSrc(scope.row.picture)"
                  class="sku-image-cell__thumb"
                  :src="getSkuPictureSrc(scope.row.picture)"
                  :preview-src-list="getSkuPicturePreviewList(scope.row.picture)"
                  fit="cover"
                  preview-teleported
                >
                  <template #error>
                    <div class="sku-image-cell__empty">暂无规格图</div>
                  </template>
                </el-image>
                <div v-else class="sku-image-cell__empty">暂无规格图</div>
              </div>
            </template>
          </ProTable>
        </el-tab-pane>

        <el-tab-pane label="商品评论" name="comment">
          <ProTable
            :key="`goods-comment-${goodsId}`"
            ref="commentTableRef"
            class="detail-table-content"
            row-key="id"
            :columns="commentColumns"
            :request-api="requestGoodsCommentTable"
          >
            <template #goods_name_snapshot="scope">
              <span>{{ scope.row.goods_name_snapshot || "未命名商品" }}</span>
            </template>
            <template #discussion_count="scope">
              <el-space>
                <span>{{ scope.row.discussion_count || 0 }}</span>
                <el-link
                  v-if="BUTTONS['comment:detail'] && scope.row.pending_discussion_count > 0"
                  type="danger"
                  @click.stop="handleOpenPendingDiscussion(scope.row)"
                >
                  待审 {{ scope.row.pending_discussion_count }}
                </el-link>
                <el-tag v-else-if="scope.row.pending_discussion_count > 0" type="danger">
                  待审 {{ scope.row.pending_discussion_count }}
                </el-tag>
              </el-space>
            </template>
          </ProTable>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <el-dialog v-model="approveDialog.visible" title="评论审核" width="560px" destroy-on-close @closed="handleResetApproveDialog">
      <div v-if="approveDialog.row" class="approve-preview">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="商品名称">{{ approveDialog.row.goods_name_snapshot }}</el-descriptions-item>
          <el-descriptions-item label="用户昵称">{{ approveDialog.row.user_name_snapshot }}</el-descriptions-item>
          <el-descriptions-item label="评论内容">
            <div class="approve-content">{{ approveDialog.row.content || "暂无评论内容" }}</div>
          </el-descriptions-item>
        </el-descriptions>

        <section class="comment-section">
          <h3>评论图片</h3>
          <div v-if="approveImageList.length" class="image-list">
            <el-image
              v-for="item in approveImageList"
              :key="item"
              :src="item"
              :preview-src-list="approveImageList"
              fit="cover"
              class="comment-image"
            />
          </div>
          <el-empty v-else description="暂无图片" :image-size="80" />
        </section>
      </div>

      <template #footer>
        <el-button @click="handleCancelApprove">取消</el-button>
        <el-button type="success" :loading="approveDialog.loading" @click="handleConfirmApprove">通过</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, reactive, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import DictLabel from "@/components/Dict/DictLabel.vue";
import ProTable from "@/components/ProTable/index.vue";
import GoodsH5PreviewDrawer from "../components/H5PreviewDrawer.vue";
import { type GoodsInfoForm } from "@/rpc/admin/v1/goods_info";
import { type GoodsSku } from "@/rpc/admin/v1/goods_sku";
import { defGoodsInfoService } from "@/api/admin/goods_info";
import { defCommentInfoService } from "@/api/admin/comment_info";
import type { CommentInfo, PageCommentInfosRequest } from "@/rpc/admin/v1/comment_info";
import { CommentStatus, GoodsStatus } from "@/rpc/common/v1/enum";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { useTabsStore } from "@/stores/modules/tabs";
import { formatPrice } from "@/utils/utils";
import { buildPageRequest } from "@/utils/proTable";
import { navigateTo } from "@/utils/router";

defineOptions({
  name: "GoodsDetail",
  inheritAttrs: false
});

/** 扩展 SKU 行数据，兼容详情页动态规格列渲染。 */
type GoodsSkuTableRow = GoodsSku & Record<string, string | number | string[] | undefined>;

/** 商品详情页标签页名称。 */
type GoodsDetailTabName = "basic" | "sku" | "spec" | "prop" | "comment";

/** 顶部快捷入口支持跳转的分区。 */
type GoodsDetailSectionKey = "sku" | "spec" | "prop" | "comment";

/** 评论审核确认弹窗状态。 */
type ApproveDialogState = {
  /** 弹窗是否显示 */
  visible: boolean;
  /** 审核请求提交状态 */
  loading: boolean;
  /** 当前待审核评论 */
  row?: CommentInfo;
};

const route = useRoute();
const router = useRouter();
const tabsStore = useTabsStore();
const { BUTTONS } = useAuthButtons();
const loading = ref(false);
const goodsId = ref(0);
const activeTabName = ref<GoodsDetailTabName>("basic");
const commentTableRef = ref<ProTableInstance>();
const goodsDetailRequestId = ref(0);
const approveDialog = reactive<ApproveDialogState>({
  visible: false,
  loading: false,
  row: undefined
});

/** 商品详情工作区标题固定为“商品详情”，避免跨页面跳转时沿用旧标题。 */
const workspaceTitle = "商品详情";

/** 判断当前是否仍停留在商品详情页，避免离开后继续改写其他页面标题。 */
function isCurrentGoodsDetailRoute() {
  return route.name === "GoodsDetail" || route.path.includes("/goods/detail/");
}

/** 创建详情页商品默认值，避免切换记录时出现残留数据。 */
function createDefaultGoodsDetailForm(): GoodsInfoForm {
  return {
    /** 商品ID */
    id: 0,
    /** 分类ID列表 */
    category_id: [],
    /** 名称 */
    name: "",
    /** 描述 */
    desc: "",
    /** 商品图片 */
    picture: "",
    /** 轮播图 */
    banner: [],
    /** 商品详情 */
    detail: [],
    /** 状态 */
    status: GoodsStatus.PUT_ON,
    /** 分类名称 */
    category_name: "",
    /** 商品属性 */
    prop_list: [],
    /** 商品SKU */
    sku_list: [],
    /** 商品规格 */
    spec_list: []
  };
}

const formData = reactive<GoodsInfoForm>(createDefaultGoodsDetailForm());

/** 将任意图片项提取为可渲染的地址字符串。 */
function extractImageValue(image: unknown) {
  if (typeof image === "string") {
    return image.trim();
  }
  if (!image || typeof image !== "object") {
    return "";
  }

  const imageRecord = image as Record<string, unknown>;
  const candidateList = [imageRecord.url, imageRecord.imageUrl, imageRecord.src, imageRecord.value];
  const matchedValue = candidateList.find(item => typeof item === "string" && item.trim());
  return typeof matchedValue === "string" ? matchedValue.trim() : "";
}

/** 将图片字段统一转成数组，兼容 JSON 字符串、逗号分隔值与对象数组。 */
function parseImageList(images: unknown): string[] {
  if (Array.isArray(images)) {
    return images.map(item => extractImageValue(item)).filter(Boolean);
  }

  if (images && typeof images === "object") {
    const imageValue = extractImageValue(images);
    return imageValue ? [imageValue] : [];
  }

  if (typeof images !== "string") {
    return [];
  }

  const value = images.trim();
  if (!value) {
    return [];
  }

  if ((value.startsWith("[") && value.endsWith("]")) || (value.startsWith("{") && value.endsWith("}"))) {
    try {
      return parseImageList(JSON.parse(value));
    } catch {
      return [];
    }
  }

  if (value.includes(",")) {
    return value
      .split(",")
      .map(item => item.trim())
      .filter(Boolean);
  }

  return [value];
}

/** 解析当前运行环境下的商品静态资源根地址，优先兼容本地 `/shop` 挂载。 */
function resolveGoodsStaticBase() {
  const configuredBase = String(import.meta.env.VITE_APP_STATIC_URL ?? "").trim();
  if (configuredBase) {
    if (/^https?:\/\//.test(configuredBase)) {
      return configuredBase.replace(/\/$/, "");
    }
    return new URL(`${configuredBase.replace(/\/$/, "")}/`, `${window.location.origin}/`).toString().replace(/\/$/, "");
  }
  return `${window.location.origin}/shop`;
}

/** 统一补齐图片地址，兼容后端返回相对路径的场景。 */
function normalizeImageSrc(image?: string) {
  const value = String(image ?? "").trim();
  if (!value) return "";
  if (/^(https?:)?\/\//.test(value) || value.startsWith("data:") || value.startsWith("blob:")) {
    return value;
  }

  const staticBase = resolveGoodsStaticBase();
  if (value.startsWith("/shop/")) {
    return new URL(value, `${window.location.origin}/`).toString();
  }
  if (value.startsWith("/")) {
    return new URL(value.replace(/^\/+/, ""), `${staticBase}/`).toString();
  }

  const normalizedPath = value.replace(/^\.\/+/, "").replace(/^shop\/+/, "");
  return new URL(normalizedPath, `${staticBase}/`).toString();
}

/** 统一规范化图片列表，避免缩略图和预览图地址不一致。 */
function normalizeImageList(images: string[] = []) {
  return images.map(item => normalizeImageSrc(item)).filter(Boolean);
}

/** 规范化详情页返回数据，兼容图片字段返回字符串场景。 */
function normalizeGoodsDetailForm(data?: Partial<GoodsInfoForm>): GoodsInfoForm {
  return {
    ...createDefaultGoodsDetailForm(),
    ...data,
    category_id: Array.isArray(data?.category_id) ? data.category_id : [],
    picture: extractImageValue(data?.picture),
    banner: parseImageList(data?.banner),
    detail: parseImageList(data?.detail),
    prop_list: Array.isArray(data?.prop_list) ? data.prop_list : [],
    sku_list: Array.isArray(data?.sku_list) ? data.sku_list : [],
    spec_list: Array.isArray(data?.spec_list) ? data.spec_list : []
  };
}

/** 统一计算主图地址。 */
const coverImageSrc = computed(() => normalizeImageSrc(formData.picture));

/** 统一计算主图预览列表，避免空图片时打开预览报错。 */
const coverPreviewList = computed(() => (coverImageSrc.value ? [coverImageSrc.value] : []));

/** 统一计算轮播图缩略图与预览列表。 */
const bannerImageList = computed(() => normalizeImageList(formData.banner));

/** 统一计算详情图缩略图与预览列表。 */
const detailImageList = computed(() => normalizeImageList(formData.detail));

/** 当前审核弹窗图片列表。 */
const approveImageList = computed<string[]>(() => {
  const imgList = approveDialog.row?.img;
  // 兼容列表行图片字段为空或被省略的场景。
  if (!Array.isArray(imgList)) return [];
  return normalizeImageList(imgList);
});

/** 统计属性项数量。 */
const propCount = computed(() => formData.prop_list.length);

/** 统计规格项数量。 */
const specCount = computed(() => formData.spec_list.length);

/** 统计商品规格数量。 */
const skuCount = computed(() => formData.sku_list.length);

/** 统计当前商品评论数量，统一从 ProTable 暴露的总数方法读取。 */
const commentCount = computed(() => commentTableRef.value?.getTotal?.() ?? 0);

/** 详情页按 SKU 汇总库存，避免额外维护聚合字段。 */
const totalInventory = computed(() => formData.sku_list.reduce((total, item) => total + Number(item.inventory ?? 0), 0));

/** 详情页按 SKU 汇总真实销量，便于头部直接查看销售表现。 */
const totalSaleNum = computed(() => formData.sku_list.reduce((total, item) => total + Number(item.real_sale_num ?? 0), 0));

/** 统一格式化规格价格区间，避免头部重复散落区间拼装逻辑。 */
function buildPriceRangeText(priceList: number[]) {
  const validPriceList = priceList.filter(price => price > 0);
  if (!validPriceList.length) {
    return "-";
  }

  const minPrice = Math.min(...validPriceList);
  const maxPrice = Math.max(...validPriceList);
  if (minPrice === maxPrice) {
    return `￥${formatPrice(minPrice)}`;
  }
  return `￥${formatPrice(minPrice)} - ￥${formatPrice(maxPrice)}`;
}

/** 汇总规格原价区间，便于在头部快速查看价格跨度。 */
const originPriceRangeText = computed(() => buildPriceRangeText(formData.sku_list.map(item => Number(item.price ?? 0))));

/** 汇总规格折后价区间，便于在头部直接对比优惠前后价格。 */
const discountPriceRangeText = computed(() =>
  buildPriceRangeText(formData.sku_list.map(item => Number(item.discount_price ?? 0)))
);

/** 页面分区和标签页映射，统一控制概览区跳转行为。 */
const detailSectionMap: Record<GoodsDetailSectionKey, GoodsDetailTabName> = {
  sku: "sku",
  spec: "spec",
  prop: "prop",
  comment: "comment"
};

/** 重置详情页表单，避免切换商品时残留上一条记录。 */
function resetGoodsDetailForm() {
  Object.assign(formData, createDefaultGoodsDetailForm());
}

/** 从路由中同步当前商品ID，统一兼容 query 字符串场景。 */
function syncGoodsIdFromRoute() {
  // 优先使用路径参数，兼容少量历史 query 链接仍携带 goodsId 的场景。
  goodsId.value = Number(route.params.goodsId ?? route.query.goodsId ?? 0);
  return goodsId.value;
}

/** 商品属性明细表格列配置。 */
const propColumns: ColumnProps[] = [
  { prop: "label", label: "名称", minWidth: 180 },
  { prop: "value", label: "内容", minWidth: 220 },
  { prop: "sort", label: "排序", align: "right", minWidth: 100 }
];

/** 规格项明细表格列配置。 */
const specColumns: ColumnProps[] = [
  { prop: "name", label: "名称", minWidth: 160 },
  {
    prop: "item",
    label: "规格值",
    minWidth: 220,
    render: scope => (Array.isArray(scope.row.item) ? scope.row.item.join(" / ") : "--")
  },
  { prop: "sort", label: "排序", align: "right", minWidth: 100 }
];

/** 商品规格明细表格列配置。 */
const skuColumns = computed<ColumnProps[]>(() => {
  // 根据规格定义动态拼接表头，保证不同商品规格也能完整展示。
  const dynamicSpecColumns = formData.spec_list.map((item, index) => ({
    prop: `spec_item${index}`,
    label: item.name,
    align: "center",
    minWidth: 120
  }));

  return [
    ...dynamicSpecColumns,
    {
      prop: "spec_item",
      label: "规格组合",
      minWidth: 180,
      render: scope => ((scope.row.spec_item as string[] | undefined) ?? []).join(" / ")
    },
    { prop: "picture", label: "规格图", minWidth: 110, align: "center" },
    { prop: "sku_code", label: "SKU", minWidth: 160 },
    { prop: "init_sale_num", label: "初始销量", align: "right", minWidth: 100 },
    { prop: "real_sale_num", label: "销量", align: "right", minWidth: 100 },
    { prop: "price", label: "售价", align: "right", minWidth: 110, cellType: "money" },
    { prop: "discount_price", label: "折后价", align: "right", minWidth: 130, cellType: "money" },
    { prop: "inventory", label: "库存", align: "right", minWidth: 100 }
  ];
});

/** 商品详情页评论表格列配置，商品名称只展示文本，不再跳回商品详情。 */
const commentColumns: ColumnProps[] = [
  {
    prop: "goods_picture_snapshot",
    label: "商品图",
    width: 90,
    cellType: "image",
    imageProps: {
      width: 48,
      height: 48,
      src: scope => normalizeImageSrc(scope.row.goods_picture_snapshot),
      previewSrc: scope => normalizeImageSrc(scope.row.goods_picture_snapshot)
    }
  },
  { prop: "goods_name_snapshot", label: "商品名称", minWidth: 220 },
  { prop: "user_name_snapshot", label: "用户昵称", minWidth: 120, search: { el: "input", key: "user_name" } },
  { prop: "goods_score", label: "评分", width: 90, search: { el: "input-number", props: { min: 1, max: 5, precision: 0 } } },
  { prop: "content", label: "评论内容", minWidth: 260, showOverflowTooltip: true },
  { prop: "discussion_count", label: "讨论数", width: 130 },
  { prop: "status", label: "审核状态", width: 120, dictCode: "comment_status", search: { el: "select" } },
  { prop: "created_at", label: "评价时间", minWidth: 170 },
  {
    prop: "operation",
    label: "操作",
    width: 150,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "详情",
        type: "primary",
        link: true,
        icon: View,
        hidden: () => !BUTTONS.value["comment:detail"],
        onClick: scope => handleOpenCommentDetail((scope.row as CommentInfo).id)
      },
      {
        label: "通过",
        type: "success",
        link: true,
        icon: CircleCheck,
        hidden: scope => !BUTTONS.value["comment:status"] || (scope.row as CommentInfo).status === CommentStatus.APPROVED_CS,
        onClick: scope => handleApproveComment(scope.row as CommentInfo)
      }
    ]
  }
];

/** 请求当前商品评论列表，并固定附加商品ID，避免混入其他商品评论。 */
async function requestGoodsCommentTable(params: PageCommentInfosRequest) {
  if (!goodsId.value) {
    return { data: { list: [], total: 0 } };
  }
  const data = await defCommentInfoService.PageCommentInfos(
    buildPageRequest({ ...params, goods_id: goodsId.value }) as PageCommentInfosRequest
  );
  return { data: { ...data, list: data.comment_infos ?? [] } };
}

/** 刷新当前商品评论列表。 */
function refreshCommentTable() {
  commentTableRef.value?.getTableList();
}

/** 打开评论详情页面。 */
function handleOpenCommentDetail(commentId: number) {
  if (!commentId) {
    ElMessage.warning("评论记录不存在");
    return;
  }
  void navigateTo(router, `/admin/comment/detail/${commentId}`);
}

/** 打开评论详情页的待审核讨论标签。 */
function handleOpenPendingDiscussion(row: CommentInfo) {
  if (!row.id) {
    ElMessage.warning("评论记录不存在");
    return;
  }
  void navigateTo(router, `/admin/comment/detail/${row.id}`, {
    tab: "discussion",
    discussionStatus: CommentStatus.PENDING_REVIEW_CS
  });
}

/** 审核通过单条评论。 */
function handleApproveComment(row: CommentInfo) {
  approveDialog.row = row;
  approveDialog.visible = true;
}

/** 取消当前评论审核确认。 */
function handleCancelApprove() {
  approveDialog.visible = false;
}

/** 弹窗关闭后重置当前审核记录。 */
function handleResetApproveDialog() {
  // 请求提交中不清理记录，避免确认请求读取不到当前评论。
  if (approveDialog.loading) return;
  approveDialog.row = undefined;
}

/** 确认通过当前评论审核。 */
async function handleConfirmApprove() {
  const row = approveDialog.row;
  // 弹窗没有选中评论时，不发起审核请求。
  if (!row) return;

  approveDialog.loading = true;
  try {
    await defCommentInfoService.SetCommentInfoStatus({ id: row.id, status: CommentStatus.APPROVED_CS, reason: "" });
    ElMessage.success("评论审核通过");
    approveDialog.visible = false;
    refreshCommentTable();
  } finally {
    approveDialog.loading = false;
  }
}

/** 同步当前页签和浏览器标题，确保从其他业务页跳转进来也显示为“商品详情”。 */
function syncWorkspaceTitle() {
  tabsStore.setTabsTitle(workspaceTitle);
  document.title = `${workspaceTitle} - ${import.meta.env.VITE_GLOB_APP_TITLE}`;
}

// 监听路由参数变化，更新商品详情数据。
watch(
  () => [route.params.goodsId, route.query.goodsId],
  () => {
    if (!isCurrentGoodsDetailRoute()) return;
    const currentGoodsId = syncGoodsIdFromRoute();
    syncWorkspaceTitle();
    if (!currentGoodsId) {
      resetGoodsDetailForm();
      return;
    }
    handleQuery(currentGoodsId);
  },
  { immediate: true }
);

/**
 * 查询商品详情，并将 SKU 规格项展开到平铺字段，便于表格动态列直接渲染。
 */
function handleQuery(targetGoodsId: number = goodsId.value) {
  if (!targetGoodsId) return;
  const requestId = ++goodsDetailRequestId.value;
  loading.value = true;
  defGoodsInfoService
    .GetGoodsInfo({
      id: targetGoodsId
    })
    .then(data => {
      if (requestId !== goodsDetailRequestId.value) return;
      const normalizedData = normalizeGoodsDetailForm(data);
      normalizedData.sku_list.forEach(item => {
        // 将规格数组转成扁平字段，避免在表格单元格里重复解析。
        item.spec_item.forEach((spec, index) => {
          (item as GoodsSkuTableRow)[`spec_item${index}`] = spec;
        });
      });
      resetGoodsDetailForm();
      Object.assign(formData, normalizedData);
    })
    .finally(() => {
      if (requestId !== goodsDetailRequestId.value) return;
      loading.value = false;
    });
}

/** 统一补齐 SKU 规格图地址，兼容后端返回相对路径的场景。 */
function getSkuPictureSrc(picture?: string) {
  return normalizeImageSrc(picture);
}

/** 统一生成规格图预览列表，避免图片为空时传入无效预览数据。 */
function getSkuPicturePreviewList(picture?: string) {
  const pictureSrc = getSkuPictureSrc(picture);
  return pictureSrc ? [pictureSrc] : [];
}

/**
 * 从顶部快捷入口切换到对应标签页。
 */
function handleNavigateSection(sectionKey: GoodsDetailSectionKey) {
  activeTabName.value = detailSectionMap[sectionKey];
}

onActivated(() => {
  if (!isCurrentGoodsDetailRoute()) return;
  syncWorkspaceTitle();
  const currentGoodsId = syncGoodsIdFromRoute();
  if (!currentGoodsId || loading.value) return;
  handleQuery(currentGoodsId);
});
</script>

<style scoped lang="scss">
.goods-hero-card {
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
}

.goods-hero-card {
  margin-bottom: 18px;
}

:deep(.goods-hero-card .el-card__body) {
  padding: 16px;
}

.goods-hero {
  --goods-summary-panel-gap: 12px;
  --goods-summary-card-column-gap: 16px;
  --goods-summary-card-row-gap: 16px;
  --goods-summary-card-height: 66px;
  --goods-summary-toolbar-height: 32px;
  display: grid;
  grid-template-columns: 200px minmax(0, 1fr);
  gap: 16px;
  align-items: start;
}

.goods-cover-panel {
  display: flex;
  box-sizing: border-box;
  height: calc(
    var(--goods-summary-toolbar-height) + var(--goods-summary-panel-gap) + var(--goods-summary-card-height) * 2 +
      var(--goods-summary-card-row-gap)
  );
  min-width: 0;
  padding: 8px;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: calc(var(--admin-page-radius) + 2px);
  background: var(--admin-page-card-bg-soft);
}

.goods-cover-image {
  width: 100%;
  height: 100%;
  overflow: hidden;
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg-muted);
}

.goods-summary-panel {
  display: flex;
  flex: 1;
  min-width: 0;
  flex-direction: column;
  gap: var(--goods-summary-panel-gap);
  align-self: start;
}

.goods-summary-toolbar {
  display: flex;
  justify-content: flex-end;
}

.goods-summary-toolbar__preview {
  flex: 0 0 auto;
}

.goods-summary-toolbar__preview :deep(.el-button) {
  min-width: 96px;
}

.goods-summary-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  grid-auto-rows: var(--goods-summary-card-height);
  // 显式统一概览卡片横向与纵向间距，避免视觉上出现“左右大、上下挤”的差异。
  column-gap: var(--goods-summary-card-column-gap);
  row-gap: var(--goods-summary-card-row-gap);
}

.goods-summary-card {
  display: flex;
  box-sizing: border-box;
  flex-direction: column;
  justify-content: center;
  height: 100%;
  min-width: 0;
  min-height: 0;
  gap: 1px;
  padding: 8px 12px;
  overflow: hidden;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg-soft);
}

.goods-summary-card__label {
  font-size: 12px;
  line-height: 1.4;
  color: var(--admin-page-text-secondary);
}

.goods-summary-card__value {
  min-width: 0;
  overflow: hidden;
  font-size: 18px;
  font-weight: 700;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--admin-page-text-primary);
}

.goods-summary-card--action {
  width: auto;
  appearance: none;
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.2s ease;
}

.goods-summary-card--action:hover {
  border-color: var(--admin-page-card-border-muted);
}

.goods-detail-panel {
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg);
  box-shadow: var(--admin-page-shadow);
}

:deep(.goods-detail-panel .el-card__body) {
  padding: 0;
}

.goods-detail-tabs :deep(.el-tabs__header) {
  margin: 0;
  padding: 0 16px;
}

.goods-detail-tabs :deep(.el-tabs__nav-wrap::after) {
  height: 1px;
  background-color: var(--admin-page-divider-strong);
}

.goods-detail-tabs :deep(.el-tabs__item) {
  height: 36px;
  padding: 0 8px;
  font-size: 14px;
  font-weight: 600;
}

.goods-detail-tabs :deep(.el-tabs__content) {
  padding: 0;
}

.detail-tab-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.detail-tab-panel--basic {
  padding: 16px;
}

.detail-info-panel {
  display: block;
}

.detail-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  font-size: 16px;
  font-weight: 600;
  color: var(--admin-page-text-primary);
}

.detail-media-list {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.detail-media-item,
.detail-media-empty {
  width: 112px;
  height: 112px;
  overflow: hidden;
  border: 1px solid var(--admin-page-card-border-muted);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg-muted);
}

.detail-media-item {
  display: block;
}

.detail-media-item__placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  padding: 8px;
  font-size: 12px;
  line-height: 1.5;
  text-align: center;
  color: var(--admin-page-text-placeholder);
  background: var(--admin-page-card-bg-muted);
}

.detail-media-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 8px;
  font-size: 12px;
  line-height: 1.5;
  text-align: center;
  color: var(--admin-page-text-placeholder);
}

.detail-table-content {
  display: block;
}

.goods-descriptions :deep(.el-descriptions__label) {
  width: 110px;
  font-weight: 600;
}

.goods-descriptions :deep(.el-descriptions__cell) {
  padding: 10px 14px;
}

.sku-image-cell {
  display: flex;
  justify-content: center;
}

.sku-image-cell__thumb {
  width: 60px;
  height: 60px;
  overflow: hidden;
  border: 1px solid var(--admin-page-card-border-muted);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg-muted);
}

.sku-image-cell__empty {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 60px;
  height: 60px;
  padding: 6px;
  font-size: 12px;
  line-height: 1.4;
  text-align: center;
  color: var(--admin-page-text-placeholder);
  border: 1px solid var(--admin-page-card-border-muted);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg-muted);
}

.image-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  min-height: 184px;
  font-size: 14px;
  color: var(--admin-page-text-placeholder);
  background: var(--admin-page-card-bg-muted);
}

.comment-section {
  margin-top: 18px;
}

.image-list {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.comment-image {
  width: 88px;
  height: 88px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
}

.approve-content {
  line-height: 1.6;
  color: var(--el-text-color-primary);
  white-space: pre-wrap;
}

@media (width <= 1200px) {
  .goods-hero {
    grid-template-columns: 180px minmax(0, 1fr);
  }
}

@media (width <= 992px) {
  .goods-hero {
    grid-template-columns: 1fr;
  }

  .goods-cover-panel {
    height: auto;
  }

  .goods-cover-image {
    height: 220px;
  }

  .goods-summary-toolbar {
    width: 100%;
  }

  .goods-summary-toolbar__preview :deep(.el-button) {
    width: 100%;
  }
}

@media (width <= 768px) {
  .goods-summary-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 520px) {
  .goods-summary-grid {
    grid-template-columns: 1fr;
  }
}
</style>
