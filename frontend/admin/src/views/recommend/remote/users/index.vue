<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="__rowKey"
      :columns="columns"
      :request-api="requestUserTable"
      :pagination="false"
      :search-col="searchCol"
    >
      <template #labels="{ row }">
        <el-space v-if="row.labels.length" wrap>
          <el-tag v-for="label in row.labels" :key="String(label)" effect="plain">{{ formatRemoteCell(label) }}</el-tag>
        </el-space>
        <span v-else>--</span>
      </template>
      <template #operation="{ row }">
        <el-space>
          <el-button :icon="View" link type="primary" @click="openDetail(row)">详情</el-button>
          <el-button :icon="Pointer" link type="primary" @click="openRecommendations(row)">推荐</el-button>
          <el-button :icon="Share" link type="primary" @click="openNeighbors(row)">相似</el-button>
          <el-button :icon="Delete" link type="danger" @click="deleteUser(row)">删除</el-button>
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

    <el-drawer v-model="detailVisible" title="用户详情" size="50%">
      <el-space v-loading="detailLoading" direction="vertical" fill>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="用户编号">{{ getUserId(detailData) || "--" }}</el-descriptions-item>
          <el-descriptions-item label="描述">
            {{ formatRemoteCell(resolveRemoteValue(detailData, ["Comment", "comment", "Description", "description"])) }}
          </el-descriptions-item>
          <el-descriptions-item label="最后活跃">
            {{ getUserLastActiveTime(detailData) }}
          </el-descriptions-item>
          <el-descriptions-item label="最后更新">
            {{ getUserLastUpdateTime(detailData) }}
          </el-descriptions-item>
          <el-descriptions-item label="标签">
            <el-space v-if="getUserLabels(detailData).length" wrap>
              <el-tag v-for="label in getUserLabels(detailData)" :key="String(label)" effect="plain">{{
                formatRemoteCell(label)
              }}</el-tag>
            </el-space>
            <span v-else>--</span>
          </el-descriptions-item>
        </el-descriptions>

        <el-card shadow="never">
          <template #header><strong>用户记录</strong></template>
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
import { Delete, Pointer, Share, View } from "@element-plus/icons-vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { defRecommendRemoteService } from "@/api/admin/recommend_remote";
import {
  formatRemoteCell,
  formatRemoteDateTime,
  resolveRemoteArray,
  resolveRemoteId,
  resolveRemoteValue,
  stringifyRemoteValue,
  type RemoteRecord
} from "../utils";

defineOptions({ name: "Users" });

/** 推荐用户表格行。 */
interface UserRow extends RemoteRecord {
  /** 表格稳定行键。 */
  __rowKey: string;
  /** 用户编号。 */
  userId: string;
  /** 标签集合。 */
  labels: unknown[];
  /** 描述。 */
  comment: string;
  /** 最后活跃时间。 */
  lastActiveTime: string;
  /** 最后更新时间。 */
  lastUpdateTime: string;
}

const userIdKeys = ["UserId", "userId", "user_id", "Id", "id"];
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

/** 用户表格列配置。 */
const columns: ColumnProps[] = [
  {
    prop: "id",
    label: "用户编号",
    minWidth: 180,
    search: { el: "input", span: 2, props: { placeholder: "输入完整用户编号查询" } },
    isShow: false
  },
  { prop: "userId", label: "用户编号", minWidth: 180, align: "left" },
  { prop: "labels", label: "标签", minWidth: 260, align: "left" },
  { prop: "comment", label: "描述", minWidth: 180, align: "left" },
  { prop: "lastActiveTime", label: "最后活跃", minWidth: 180 },
  { prop: "lastUpdateTime", label: "最后更新", minWidth: 180 },
  { prop: "operation", label: "操作", width: 300, fixed: "right" }
];

/** 查询远程推荐用户列表或单个用户。 */
async function requestUserTable(params: { id?: string; cursor?: string }) {
  try {
    const id = String(params.id ?? "").trim();
    if (id) {
      const data = await defRecommendRemoteService.GetUser({ id });
      const records = normalizeRemoteRecordList((data.raw ?? data) as RemoteRecord);
      nextCursor.value = "";
      currentPageSize.value = records.length;
      return { data: records.map(normalizeUserRow) };
    }

    const data = await defRecommendRemoteService.PageUser({
      id: "",
      cursor: params.cursor ?? "",
      n: pageSize.value
    });
    nextCursor.value = data.cursor;
    currentPageSize.value = data.list.length;
    return { data: data.list.map(item => normalizeUserRow((item.raw ?? item) as RemoteRecord, data.list.indexOf(item))) };
  } catch (error) {
    ElMessage.error("加载远程推荐用户失败");
    throw error;
  }
}

/** 将远程用户记录转换为表格行。 */
function normalizeUserRow(row: RemoteRecord, index: number): UserRow {
  return {
    ...row,
    __rowKey: `${getUserId(row) || index}-${index}`,
    userId: getUserId(row),
    labels: getUserLabels(row),
    comment: formatRemoteCell(resolveRemoteValue(row, ["Comment", "comment", "Description", "description"])),
    lastActiveTime: getUserLastActiveTime(row),
    lastUpdateTime: getUserLastUpdateTime(row)
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

/** 读取远程用户编号。 */
function getUserId(row: RemoteRecord) {
  return resolveRemoteId(row, userIdKeys);
}

/** 读取远程用户标签。 */
function getUserLabels(row: RemoteRecord) {
  return resolveRemoteArray(row, ["Labels", "labels"]);
}

/** 读取远程用户最后活跃时间。 */
function getUserLastActiveTime(row: RemoteRecord) {
  return formatRemoteDateTime(
    resolveRemoteValue(row, [
      "LastActiveTime",
      "lastActiveTime",
      "last_active_time",
      "ActiveTime",
      "activeTime",
      "active_time",
      "LastFeedbackTime",
      "lastFeedbackTime",
      "last_feedback_time"
    ])
  );
}

/** 读取远程用户最后更新时间。 */
function getUserLastUpdateTime(row: RemoteRecord) {
  return formatRemoteDateTime(
    resolveRemoteValue(row, [
      "LastUpdateTime",
      "lastUpdateTime",
      "last_update_time",
      "UpdateTime",
      "updateTime",
      "update_time",
      "Timestamp",
      "timestamp",
      "CreatedAt",
      "createdAt",
      "created_at"
    ])
  );
}

/** 加载下一页远程用户。 */
function loadNextPage() {
  if (!nextCursor.value) {
    ElMessage.warning("暂无下一页数据");
    return;
  }
  cursorStack.value.push(String(proTable.value?.searchParam.cursor ?? ""));
  proTable.value!.searchParam.cursor = nextCursor.value;
  proTable.value?.getTableList();
}

/** 加载上一页远程用户。 */
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

/** 打开远程用户详情。 */
async function openDetail(row: RemoteRecord) {
  const id = getUserId(row);
  if (!id) {
    ElMessage.warning("用户编号为空，无法查看详情");
    return;
  }
  currentDetailId.value = id;
  detailVisible.value = true;
  await reloadDetail();
}

/** 打开当前用户的推荐调试页。 */
function openRecommendations(row: RemoteRecord) {
  const id = getUserId(row);
  if (!id) {
    ElMessage.warning("用户编号为空，无法查看推荐");
    return;
  }
  router.push({ path: "/recommend/remote/recommendations", query: { type: "recommend", id } });
}

/** 打开当前用户的相似用户列表。 */
function openNeighbors(row: RemoteRecord) {
  const id = getUserId(row);
  if (!id) {
    ElMessage.warning("用户编号为空，无法查看相似用户");
    return;
  }
  router.push({ path: "/recommend/remote/neighbors", query: { type: "user", id } });
}

/** 重新加载当前远程用户详情。 */
async function reloadDetail() {
  if (!currentDetailId.value) return;

  detailLoading.value = true;
  try {
    const data = await defRecommendRemoteService.GetUser({ id: currentDetailId.value });
    detailData.value = (data.raw ?? data) as RemoteRecord;
  } catch (error) {
    ElMessage.error("加载用户详情失败");
    throw error;
  } finally {
    detailLoading.value = false;
  }
}

/** 删除远程推荐用户。 */
async function deleteUser(row: RemoteRecord) {
  const id = getUserId(row);
  if (!id) {
    ElMessage.warning("用户编号为空，无法删除");
    return;
  }
  await ElMessageBox.prompt(`请输入用户编号 ${id} 以确认删除`, "删除远程推荐用户", {
    confirmButtonText: "确认删除",
    cancelButtonText: "取消",
    inputPattern: new RegExp(`^${escapeRegExp(id)}$`),
    inputErrorMessage: "用户编号不匹配",
    type: "warning"
  });
  await defRecommendRemoteService.DeleteUser({ id });
  ElMessage.success("删除远程推荐用户成功");
  proTable.value?.getTableList();
}

/** 转义确认输入使用的正则特殊字符。 */
function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
</script>
