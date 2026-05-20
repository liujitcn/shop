<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestAiImageTable"
      :init-param="initParam"
    />

    <CreateDialog v-model="createVisible" @created="handleCreatedImage" />
    <DetailDialog v-model="detailVisible" :image-id="activeImageId" @refreshed="refreshTable" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { CirclePlus, Refresh, View } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import { defAiImageService } from "@/api/base/ai_image";
import type { AiImage, PageAiImagesRequest } from "@/rpc/base/v1/ai_image";
import { Terminal } from "@/rpc/common/v1/enum";
import { buildPageRequest } from "@/utils/proTable";
import { formatSrc } from "@/utils/utils";
import CreateDialog from "./components/CreateDialog.vue";
import DetailDialog from "./components/DetailDialog.vue";
import { aiImageStatusOptions } from "./components/types";
import { formatTimestamp } from "./components/utils";

defineOptions({
  name: "AiImage"
});

const proTable = ref<ProTableInstance>();
const createVisible = ref(false);
const detailVisible = ref(false);
const activeImageId = ref("");

const initParam = computed<PageAiImagesRequest>(() => ({
  status: undefined,
  keyword: "",
  terminal: Terminal.TERMINAL_ADMIN,
  page_num: 1,
  page_size: 10
}));

/** AI 图片表格列配置。 */
const columns: ColumnProps[] = [
  {
    prop: "images",
    label: "图片",
    width: 96,
    cellType: "image",
    imageProps: {
      width: 56,
      height: 56,
      src: scope => firstImageSrc(scope.row as AiImage),
      previewSrc: scope => firstImageSrc(scope.row as AiImage)
    }
  },
  {
    prop: "keyword",
    label: "提示词",
    minWidth: 260,
    search: { el: "input", key: "keyword", props: { placeholder: "请输入提示词" } },
    showOverflowTooltip: true,
    render: scope => {
      const row = scope.row as AiImage;
      return row.prompt || row.original_prompt || "--";
    }
  },
  {
    prop: "status",
    label: "状态",
    width: 100,
    enum: aiImageStatusOptions,
    tag: true,
    search: { el: "select" }
  },
  { prop: "model", label: "模型", minWidth: 130 },
  { prop: "size", label: "尺寸", width: 120 },
  { prop: "quality", label: "质量", width: 90 },
  { prop: "n", label: "数量", width: 80, align: "right" },
  {
    prop: "created_at",
    label: "创建时间",
    minWidth: 180,
    render: scope => formatTimestamp((scope.row as AiImage).created_at) || "--"
  },
  {
    prop: "operation",
    label: "操作",
    width: 110,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "详情",
        type: "primary",
        link: true,
        icon: View,
        onClick: scope => handleOpenDetail((scope.row as AiImage).id)
      }
    ]
  }
];

/** AI 图片顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增AI图片",
    type: "success",
    icon: CirclePlus,
    onClick: () => {
      createVisible.value = true;
    }
  },
  {
    label: "刷新",
    type: "primary",
    icon: Refresh,
    onClick: () => refreshTable()
  }
];

/** 请求 AI 图片列表，并由 ProTable 统一管理分页和筛选。 */
async function requestAiImageTable(params: PageAiImagesRequest) {
  const data = await defAiImageService.PageAiImages(buildPageRequest(params));
  return { data: { ...data, list: data.images ?? [] } };
}

/** 刷新 AI 图片表格。 */
function refreshTable() {
  proTable.value?.getTableList();
}

/** 创建图片后刷新列表，避免自动跳到详情打断当前操作。 */
function handleCreatedImage() {
  refreshTable();
}

/** 打开 AI 图片详情。 */
function handleOpenDetail(imageId: string) {
  activeImageId.value = imageId;
  detailVisible.value = true;
}

/** 解析首图地址，供表格图片列展示与预览复用。 */
function firstImageSrc(row: AiImage) {
  const image = (row.images ?? [])[0];
  return image?.url ? formatSrc(image.url) : "";
}
</script>
