<template>
  <ProDialog
    v-model="dialog.visible"
    :title="dialog.title"
    width="1100px"
    destroy-on-close
    @close="handleCloseDialog"
    @closed="handleClosedDialog"
  >
    <div v-loading="loading" class="recommend-request-event-dialog">
      <div class="recommend-request-event-dialog__summary">
        <el-descriptions :column="2" border class="recommend-request-event-dialog__descriptions">
          <el-descriptions-item label="商品名称">{{ currentItem?.goods_name || "--" }}</el-descriptions-item>
          <el-descriptions-item label="结果位置">{{ currentItem?.position ?? "--" }}</el-descriptions-item>
          <el-descriptions-item label="商品状态">
            <DictLabel :model-value="currentItem?.goods_status" code="goods_status" />
          </el-descriptions-item>
          <el-descriptions-item label="事件条数">{{ eventData.total }}</el-descriptions-item>
        </el-descriptions>
      </div>

      <div class="recommend-request-event-dialog__table">
        <div class="recommend-request-event-dialog__header">
          <span>关联事件明细</span>
          <span class="recommend-request-event-dialog__header-extra">共 {{ eventData.total }} 条</span>
        </div>

        <ProTable
          row-key="id"
          :data="eventData.recommend_events"
          :columns="eventColumns"
          :pagination="false"
          :tool-button="false"
          :border="true"
        />
      </div>
    </div>

    <template #footer>
      <el-button @click="handleCloseDialog">关闭</el-button>
    </template>
  </ProDialog>
</template>

<script setup lang="ts">
import { reactive, ref } from "vue";
import type { ColumnProps } from "@/components/ProTable/interface";
import ProDialog from "@/components/Dialog/ProDialog.vue";
import DictLabel from "@/components/Dict/DictLabel.vue";
import ProTable from "@/components/ProTable/index.vue";
import { defRecommendRequestService } from "@/api/admin/recommend_request";
import type { ListRecommendRequestEventsResponse, RecommendRequestItem } from "@/rpc/admin/v1/recommend_request";

defineOptions({
  name: "RecommendRequestEventDialog",
  inheritAttrs: false
});

/** 打开关联事件弹窗时需要的入参。 */
type RecommendRequestEventDialogOpenOptions = {
  /** 推荐请求记录ID。 */
  requestRecordId: number;
  /** 当前选中的推荐商品。 */
  item: RecommendRequestItem;
};

const dialog = reactive({
  title: "关联事件",
  visible: false
});
const loading = ref(false);
const requestId = ref(0);
const currentRequestRecordId = ref(0);
const currentItem = ref<RecommendRequestItem>();

/** 推荐事件表格列配置。 */
const eventColumns: ColumnProps[] = [
  { prop: "actor_type", label: "主体类型", minWidth: 120, dictCode: "recommend_actor_type" },
  { prop: "scene", label: "推荐场景", minWidth: 120, dictCode: "recommend_scene" },
  { prop: "event_type", label: "事件类型", minWidth: 120, dictCode: "recommend_event_type" },
  { prop: "goods_num", label: "商品数量", minWidth: 100, align: "right" },
  { prop: "position", label: "结果位置", minWidth: 100, align: "right" },
  { prop: "event_at", label: "事件时间", minWidth: 180 }
];

/** 创建默认推荐事件响应，避免弹窗切换商品时残留上一条数据。 */
function createDefaultEventData(): ListRecommendRequestEventsResponse {
  return {
    recommend_events: [],
    total: 0
  };
}

const eventData = reactive<ListRecommendRequestEventsResponse>(createDefaultEventData());

/** 重置弹窗内的事件数据。 */
function resetEventData() {
  Object.assign(eventData, createDefaultEventData());
}

/** 重置弹窗状态，避免关闭后残留上一条商品数据。 */
function resetDialogState() {
  requestId.value += 1;
  loading.value = false;
  currentRequestRecordId.value = 0;
  currentItem.value = undefined;
  resetEventData();
}

/**
 * 打开关联事件弹窗，并默认查询当前推荐商品的事件数据。
 */
function openDialog(options: RecommendRequestEventDialogOpenOptions) {
  if (!options.requestRecordId || !options.item?.goods_id) {
    ElMessage.warning("当前商品缺少关联事件查询参数");
    return;
  }

  currentRequestRecordId.value = options.requestRecordId;
  currentItem.value = { ...options.item };
  dialog.title = "关联事件";
  dialog.visible = true;
  void queryEventData();
}

/** 关闭关联事件弹窗。 */
function handleCloseDialog() {
  requestId.value += 1;
  loading.value = false;
  dialog.visible = false;
}

/** 弹窗完全关闭后重置内部状态。 */
function handleClosedDialog() {
  resetDialogState();
}

/** 查询当前商品的关联事件数据。 */
async function queryEventData() {
  const item = currentItem.value;
  if (!currentRequestRecordId.value || !item?.goods_id) {
    resetEventData();
    return;
  }

  const currentQueryId = ++requestId.value;
  loading.value = true;
  resetEventData();
  try {
    const data = await defRecommendRequestService.ListRecommendRequestEvents({
      request_record_id: currentRequestRecordId.value,
      goods_id: item.goods_id,
      position: item.position
    });
    if (currentQueryId !== requestId.value) return;
    Object.assign(eventData, data);
  } catch {
    if (currentQueryId !== requestId.value) return;
    resetEventData();
  } finally {
    if (currentQueryId !== requestId.value) return;
    loading.value = false;
  }
}

defineExpose({
  openDialog
});
</script>

<style scoped lang="scss">
.recommend-request-event-dialog {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.recommend-request-event-dialog__summary,
.recommend-request-event-dialog__table {
  padding: 16px;
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg-soft);
}

.recommend-request-event-dialog__table {
  background: var(--admin-page-card-bg);
}

.recommend-request-event-dialog__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
  color: var(--admin-page-text-primary);
}

.recommend-request-event-dialog__header span:first-child {
  font-size: 16px;
  font-weight: 600;
}

.recommend-request-event-dialog__header-extra {
  font-size: 12px;
  color: var(--admin-page-text-secondary);
}

.recommend-request-event-dialog__descriptions {
  :deep(.el-descriptions__label) {
    width: 140px;
  }
}
</style>
