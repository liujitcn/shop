<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="__rowKey"
      :columns="columns"
      :request-api="requestItemTable"
      :pagination="false"
      :search-col="searchCol"
    >
      <template #categories="{ row }">
        <el-space v-if="row.categories.length" wrap>
          <el-tag v-for="category in row.categories" :key="String(category)" effect="plain" type="info">
            {{ formatRemoteCell(category) }}
          </el-tag>
        </el-space>
        <span v-else>--</span>
      </template>
      <template #labels="{ row }">
        <el-space v-if="row.labels.length" wrap>
          <el-tag v-for="label in row.labels" :key="String(label)" effect="plain">{{ formatRemoteCell(label) }}</el-tag>
        </el-space>
        <span v-else>--</span>
      </template>
      <template #isHidden="{ row }">
        <el-tag :type="row.isHidden ? 'danger' : 'success'" effect="light">{{ row.isHidden ? "已隐藏" : "展示中" }}</el-tag>
      </template>
      <template #operation="{ row }">
        <el-space>
          <el-button :icon="View" link type="primary" @click="openDetail(row)">详情</el-button>
          <el-button :icon="Pointer" link type="primary" @click="openNeighbors(row)">相似</el-button>
          <el-button :icon="Delete" link type="danger" @click="deleteItem(row)">删除</el-button>
        </el-space>
      </template>
      <template #pagination>
        <div class="el-pagination">
          <el-space>
            <span>游标分页</span>
            <el-tag effect="plain" type="info">当前页 {{ currentPageSize }} 条</el-tag>
            <span>每页条数</span>
            <el-input-number
              v-model="pageSize"
              :min="1"
              :max="200"
              :step="10"
              controls-position="right"
              @change="resetCursorPage"
            />
            <el-button :disabled="!hasPreviousPage" @click="loadPreviousPage">上一页</el-button>
            <el-button :disabled="!nextCursor" type="primary" @click="loadNextPage">下一页</el-button>
          </el-space>
        </div>
      </template>
    </ProTable>

    <el-drawer v-model="detailVisible" title="商品详情" size="50%">
      <el-space v-loading="detailLoading" direction="vertical" fill>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="商品编号">{{ getItemId(detailData) || "--" }}</el-descriptions-item>
          <el-descriptions-item label="是否隐藏">
            <el-tag :type="isItemHidden(detailData) ? 'danger' : 'success'" effect="light">
              {{ isItemHidden(detailData) ? "已隐藏" : "展示中" }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="最后更新">
            {{
              formatRemoteDateTime(resolveRemoteValue(detailData, ["Timestamp", "timestamp", "LastUpdateTime", "lastUpdateTime"]))
            }}
          </el-descriptions-item>
          <el-descriptions-item label="分类">
            <el-space v-if="getItemCategories(detailData).length" wrap>
              <el-tag v-for="category in getItemCategories(detailData)" :key="String(category)" effect="plain" type="info">
                {{ formatRemoteCell(category) }}
              </el-tag>
            </el-space>
            <span v-else>--</span>
          </el-descriptions-item>
          <el-descriptions-item label="标签">
            <el-space v-if="getItemLabels(detailData).length" wrap>
              <el-tag v-for="label in getItemLabels(detailData)" :key="String(label)" effect="plain">{{
                formatRemoteCell(label)
              }}</el-tag>
            </el-space>
            <span v-else>--</span>
          </el-descriptions-item>
          <el-descriptions-item label="描述">
            {{ formatRemoteCell(resolveRemoteValue(detailData, ["Comment", "comment", "Description", "description"])) }}
          </el-descriptions-item>
        </el-descriptions>

        <el-card shadow="never">
          <template #header><strong>商品记录</strong></template>
          <el-input :model-value="stringifyRemoteValue(detailData)" type="textarea" :rows="12" readonly />
        </el-card>
      </el-space>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import { Delete, Pointer, View } from "@element-plus/icons-vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { defRecommendRemoteService } from "@/api/admin/recommend_remote";
import {
  formatRemoteCell,
  formatRemoteDateTime,
  resolveRemoteArray,
  resolveRemoteBoolean,
  resolveRemoteId,
  resolveRemoteValue,
  stringifyRemoteValue,
  type RemoteRecord
} from "../utils";

defineOptions({ name: "Items" });

/** 推荐商品表格行。 */
interface ItemRow extends RemoteRecord {
  /** 表格稳定行键。 */
  __rowKey: string;
  /** 商品编号。 */
  itemId: string;
  /** 分类集合。 */
  categories: unknown[];
  /** 标签集合。 */
  labels: unknown[];
  /** 是否隐藏。 */
  isHidden: boolean;
  /** 描述。 */
  comment: string;
  /** 更新时间。 */
  updateTime: string;
}

const itemIdKeys = ["ItemId", "itemId", "item_id", "Id", "id"];
const router = useRouter();
const proTable = ref<ProTableInstance>();
const detailLoading = ref(false);
const detailVisible = ref(false);
const currentDetailId = ref("");
const detailData = ref<RemoteRecord>({});
const nextCursor = ref("");
const cursorStack = ref<string[]>([]);
const pageSize = ref(20);
const currentPageSize = ref(0);
const searchCol = { xs: 1, sm: 2, md: 3, lg: 6, xl: 6 };

/** 是否存在上一页游标。 */
const hasPreviousPage = computed(() => cursorStack.value.length > 0);

/** 商品表格列配置。 */
const columns: ColumnProps[] = [
  {
    prop: "id",
    label: "商品编号",
    minWidth: 180,
    search: { el: "input", span: 2, props: { placeholder: "输入完整商品编号查询" } },
    isShow: false
  },
  { prop: "itemId", label: "商品编号", minWidth: 180, align: "left" },
  { prop: "categories", label: "分类", minWidth: 220, align: "left" },
  { prop: "labels", label: "标签", minWidth: 260, align: "left" },
  { prop: "isHidden", label: "状态", minWidth: 120 },
  { prop: "comment", label: "描述", minWidth: 220, align: "left" },
  { prop: "updateTime", label: "更新时间", minWidth: 180 },
  { prop: "operation", label: "操作", width: 240, fixed: "right" }
];

/** 查询远程推荐商品列表或单个商品。 */
async function requestItemTable(params: { id?: string; cursor?: string }) {
  try {
    const id = String(params.id ?? "").trim();
    if (id) {
      const data = await defRecommendRemoteService.GetItem({ id });
      const records = normalizeRemoteRecordList((data.raw ?? data) as RemoteRecord);
      nextCursor.value = "";
      currentPageSize.value = records.length;
      return { data: records.map(normalizeItemRow) };
    }

    const data = await defRecommendRemoteService.PageItem({
      id: "",
      cursor: params.cursor ?? "",
      n: pageSize.value
    });
    nextCursor.value = data.cursor;
    currentPageSize.value = data.list.length;
    return { data: data.list.map(item => normalizeItemRow((item.raw ?? item) as RemoteRecord, data.list.indexOf(item))) };
  } catch (error) {
    ElMessage.error("加载远程推荐商品失败");
    throw error;
  }
}

/** 将远程商品记录转换为表格行。 */
function normalizeItemRow(row: RemoteRecord, index: number): ItemRow {
  return {
    ...row,
    __rowKey: `${getItemId(row) || index}-${index}`,
    itemId: getItemId(row),
    categories: getItemCategories(row),
    labels: getItemLabels(row),
    isHidden: isItemHidden(row),
    comment: formatRemoteCell(resolveRemoteValue(row, ["Comment", "comment", "Description", "description"])),
    updateTime: formatRemoteDateTime(resolveRemoteValue(row, ["Timestamp", "timestamp", "LastUpdateTime", "lastUpdateTime"]))
  };
}

/** 将单条或列表响应转换为远程记录列表。 */
function normalizeRemoteRecordList(value: unknown) {
  if (Array.isArray(value)) return value.filter(isRecord);
  if (isRecord(value)) return [value];
  return [];
}

/** 判断值是否为远程推荐记录。 */
function isRecord(value: unknown): value is RemoteRecord {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

/** 读取远程商品编号。 */
function getItemId(row: RemoteRecord) {
  return resolveRemoteId(row, itemIdKeys);
}

/** 读取远程商品分类。 */
function getItemCategories(row: RemoteRecord) {
  return resolveRemoteArray(row, ["Categories", "categories"]);
}

/** 读取远程商品标签。 */
function getItemLabels(row: RemoteRecord) {
  return resolveRemoteArray(row, ["Labels", "labels"]);
}

/** 判断远程商品是否隐藏。 */
function isItemHidden(row: RemoteRecord) {
  return resolveRemoteBoolean(row, ["IsHidden", "isHidden", "is_hidden"]);
}

/** 加载下一页远程商品。 */
function loadNextPage() {
  if (!nextCursor.value) {
    ElMessage.warning("暂无下一页数据");
    return;
  }
  cursorStack.value.push(String(proTable.value?.searchParam.cursor ?? ""));
  proTable.value!.searchParam.cursor = nextCursor.value;
  proTable.value?.getTableList();
}

/** 加载上一页远程商品。 */
function loadPreviousPage() {
  const previousCursor = cursorStack.value.pop();
  if (previousCursor === undefined) {
    ElMessage.warning("暂无上一页数据");
    return;
  }
  proTable.value!.searchParam.cursor = previousCursor;
  proTable.value?.getTableList();
}

/** 重置游标分页并刷新第一页。 */
function resetCursorPage() {
  cursorStack.value = [];
  nextCursor.value = "";
  if (proTable.value) {
    proTable.value.searchParam.cursor = "";
  }
  proTable.value?.getTableList();
}

/** 打开远程商品详情。 */
async function openDetail(row: RemoteRecord) {
  const id = getItemId(row);
  if (!id) {
    ElMessage.warning("商品编号为空，无法查看详情");
    return;
  }
  currentDetailId.value = id;
  detailVisible.value = true;
  await reloadDetail();
}

/** 打开当前商品的相似内容页。 */
function openNeighbors(row: RemoteRecord) {
  const id = getItemId(row);
  if (!id) {
    ElMessage.warning("商品编号为空，无法查看相似内容");
    return;
  }
  router.push({ path: "/recommend/remote/neighbors", query: { type: "item", id } });
}

/** 重新加载当前远程商品详情。 */
async function reloadDetail() {
  if (!currentDetailId.value) return;

  detailLoading.value = true;
  try {
    const data = await defRecommendRemoteService.GetItem({ id: currentDetailId.value });
    detailData.value = (data.raw ?? data) as RemoteRecord;
  } catch (error) {
    ElMessage.error("加载商品详情失败");
    throw error;
  } finally {
    detailLoading.value = false;
  }
}

/** 删除远程推荐商品。 */
async function deleteItem(row: RemoteRecord) {
  const id = getItemId(row);
  if (!id) {
    ElMessage.warning("商品编号为空，无法删除");
    return;
  }
  await ElMessageBox.prompt(`请输入商品编号 ${id} 以确认删除`, "删除远程推荐商品", {
    confirmButtonText: "确认删除",
    cancelButtonText: "取消",
    inputPattern: new RegExp(`^${escapeRegExp(id)}$`),
    inputErrorMessage: "商品编号不匹配",
    type: "warning"
  });
  await defRecommendRemoteService.DeleteItem({ id });
  ElMessage.success("删除远程推荐商品成功");
  proTable.value?.getTableList();
}

/** 转义确认输入使用的正则特殊字符。 */
function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
</script>
