<!-- 推荐版本管理 -->
<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestRecommendModelVersionTable"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="dialog.title"
      width="720px"
      :model="dialogFormData"
      :fields="dialogFields"
      :rules="dialogRules"
      label-width="120px"
      @confirm="handleSubmitDialog"
      @close="handleCloseDialog"
    >
      <template #targetVersion="{ model }">
        <div class="summary-lines">
          <div>场景：{{ renderSceneText(currentRow?.scene) }}</div>
          <div>模型：{{ currentRow?.modelName || "--" }}</div>
          <div>类型：{{ currentRow?.modelType || "--" }}</div>
          <div>选中版本：{{ currentRow?.version || "--" }}</div>
          <div>当前生效缓存版本：{{ currentRow?.effectiveVersion || "--" }}</div>
          <div v-if="model.action === RecommendModelVersionPublishAction.RECOMMEND_MODEL_VERSION_PUBLISH_ACTION_SET_ROLLBACK">
            回滚目标：{{ model.rollbackVersion || "--" }}
          </div>
        </div>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, h, reactive, ref, resolveComponent, type VNode } from "vue";
import { ElMessage, ElMessageBox, type FormRules } from "element-plus";
import { Promotion, RefreshLeft, SwitchButton } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance, RenderScope } from "@/components/ProTable/interface";
import type { ProFormField } from "@/components/ProForm/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { useDictStore } from "@/stores/modules/dict";
import { defRecommendModelVersionService } from "@/api/admin/recommend_model_version";
import {
  RecommendModelVersionPublishAction,
  type PageRecommendModelVersionRequest,
  type RecommendModelVersion,
  type UpdateRecommendModelVersionPublishRequest
} from "@/rpc/admin/recommend_model_version";
import { RecommendScene, Status } from "@/rpc/common/enum";
import { buildPageRequest } from "@/utils/proTable";

defineOptions({
  name: "RecommendModelVersion",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const dictStore = useDictStore();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const currentRow = ref<RecommendModelVersion>();
const recommendSceneDictCode = "recommend_scene";
const recommendSceneFallbackMap: Record<number, string> = {
  [RecommendScene.HOME]: "首页",
  [RecommendScene.GOODS_DETAIL]: "商品详情",
  [RecommendScene.CART]: "购物车",
  [RecommendScene.PROFILE]: "个人中心",
  [RecommendScene.ORDER_DETAIL]: "订单详情",
  [RecommendScene.ORDER_PAID]: "支付成功"
};

const dialog = reactive({
  visible: false,
  title: ""
});

const dialogFormData = reactive<UpdateRecommendModelVersionPublishRequest>({
  /** 推荐版本ID */
  id: 0,
  /** 发布动作 */
  action: RecommendModelVersionPublishAction.RECOMMEND_MODEL_VERSION_PUBLISH_ACTION_PUBLISH,
  /** 缓存版本 */
  cacheVersion: "",
  /** 回滚版本 */
  rollbackVersion: "",
  /** 灰度比例 */
  grayRatio: undefined,
  /** 发布人 */
  publishedBy: "",
  /** 发布说明 */
  publishedReason: ""
});

/**
 * 预加载推荐场景字典，确保筛选项和弹窗文案优先复用字典配置。
 */
async function initRecommendSceneDictionary() {
  try {
    await dictStore.loadDictionaries();
  } catch (_error) {
    // 字典加载失败时保留页面兜底文案，避免影响推荐版本管理主流程。
  }
}

void initRecommendSceneDictionary();

const statusOptions = [
  { label: "启用", value: Status.ENABLE },
  { label: "禁用", value: Status.DISABLE }
];

/** 当前弹窗字段配置。 */
const dialogFields = computed<ProFormField[]>(() => {
  const fieldList: ProFormField[] = [
    {
      prop: "targetVersion",
      label: "当前对象",
      component: "slot",
      slotName: "targetVersion"
    }
  ];

  // 正式发布时，允许直接调整缓存版本和灰度比例。
  if (dialogFormData.action === RecommendModelVersionPublishAction.RECOMMEND_MODEL_VERSION_PUBLISH_ACTION_PUBLISH) {
    fieldList.push({
      prop: "cacheVersion",
      label: "缓存版本",
      component: "input",
      props: { placeholder: "默认使用当前选中版本" }
    });
    fieldList.push({
      prop: "grayRatio",
      label: "灰度比例",
      component: "input-number",
      props: {
        min: 0,
        max: 1,
        step: 0.1,
        precision: 2,
        controlsPosition: "right",
        style: { width: "100%" }
      }
    });
  }

  // 设置回滚时，允许显式指定回滚版本。
  if (dialogFormData.action === RecommendModelVersionPublishAction.RECOMMEND_MODEL_VERSION_PUBLISH_ACTION_SET_ROLLBACK) {
    fieldList.push({
      prop: "rollbackVersion",
      label: "回滚版本",
      component: "input",
      props: { placeholder: "默认使用当前选中版本" }
    });
  }

  fieldList.push({
    prop: "publishedBy",
    label: "发布人",
    component: "input",
    props: { placeholder: "请输入发布人" }
  });
  fieldList.push({
    prop: "publishedReason",
    label: "发布说明",
    component: "textarea",
    props: { placeholder: "请输入本次发布说明", rows: 3 }
  });
  return fieldList;
});

/** 当前弹窗校验规则。 */
const dialogRules = computed<FormRules>(() => ({
  publishedBy: [{ required: true, message: "请输入发布人", trigger: "blur" }],
  rollbackVersion:
    dialogFormData.action === RecommendModelVersionPublishAction.RECOMMEND_MODEL_VERSION_PUBLISH_ACTION_SET_ROLLBACK
      ? [{ required: true, message: "请输入回滚版本", trigger: "blur" }]
      : [],
  grayRatio:
    dialogFormData.action === RecommendModelVersionPublishAction.RECOMMEND_MODEL_VERSION_PUBLISH_ACTION_PUBLISH
      ? [
          {
            validator: (_rule: unknown, value: number | undefined, callback: (error?: Error) => void) => {
              if (value === undefined || value === null) {
                callback();
                return;
              }
              if (value < 0 || value > 1) {
                callback(new Error("灰度比例必须在 0 到 1 之间"));
                return;
              }
              callback();
            },
            trigger: "blur"
          }
        ]
      : []
}));

/**
 * 渲染推荐版本状态标签。
 */
function renderStatusCell(scope: RenderScope) {
  const row = scope.row as RecommendModelVersion;
  return renderTag(row.status === Status.ENABLE ? "启用" : "禁用", row.status === Status.ENABLE ? "success" : "danger");
}

/**
 * 渲染发布摘要，方便在列表里直接看到当前生效信息。
 */
function renderPublishSummary(scope: RenderScope) {
  const row = scope.row as RecommendModelVersion;
  const publish = row.publish;
  const lineList = [
    `缓存版本：${publish?.cacheVersion || row.version || "--"}`,
    `回滚版本：${publish?.rollbackVersion || "--"}`,
    `灰度比例：${publish?.grayRatio ?? 1}`,
    `发布人：${publish?.publishedBy || "--"}`,
    `发布时间：${publish?.publishedAt || "--"}`
  ];
  return renderSummaryBlock(lineList);
}

/**
 * 渲染调参摘要，便于排查训练策略是否已启用。
 */
function renderTuneSummary(scope: RenderScope) {
  const row = scope.row as RecommendModelVersion;
  const tune = row.tune;
  const lineList = [
    `自动调参：${tune?.enabled ? "启用" : "关闭"}`,
    `目标指标：${tune?.targetMetric || "--"}`,
    `尝试次数：${tune?.trialCount ?? 0}`
  ];
  return renderSummaryBlock(lineList);
}

/**
 * 渲染最近一次训练摘要。
 */
function renderLatestTrainSummary(scope: RenderScope) {
  const row = scope.row as RecommendModelVersion;
  const latest = row.tune?.latest;
  if (!latest) return renderSummaryBlock([]);
  const scoreSummary = formatScoreSummary(latest.score);
  const versionSummary = latest.version || (latest.versions?.length ? latest.versions.join(",") : "--");
  const lineList = [
    `任务：${latest.task || "--"}`,
    `模型类型：${latest.modelType || "--"}`,
    `训练后端：${latest.backend || "--"}`,
    `版本：${versionSummary}`,
    `最优值：${latest.bestValue ?? 0}`,
    `指标：${scoreSummary || "--"}`,
    `产物目录：${latest.artifactDir || "--"}`,
    `训练时间：${latest.trainedAt || "--"}`
  ];
  return renderSummaryBlock(lineList);
}

/**
 * 渲染最近一次评估摘要。
 */
function renderLatestEvalSummary(scope: RenderScope) {
  const row = scope.row as RecommendModelVersion;
  const latestEval = row.tune?.latestEval;
  if (!latestEval) return renderSummaryBlock([]);
  const lineList = [
    `报告日期：${latestEval.reportDate || "--"}`,
    `策略：${latestEval.strategyName || "--"}`,
    `样本量：${latestEval.sampleSize ?? 0}`,
    `请求数：${latestEval.requestCount ?? 0}`,
    `曝光数：${latestEval.exposureCount ?? 0}`,
    `点击数：${latestEval.clickCount ?? 0}`,
    `支付数：${latestEval.payCount ?? 0}`,
    `CTR：${formatMetric(latestEval.ctr)}`,
    `CVR：${formatMetric(latestEval.cvr)}`,
    `NDCG：${formatMetric(latestEval.ndcg)}`,
    `Precision：${formatMetric(latestEval.precision)}`,
    `Recall：${formatMetric(latestEval.recall)}`
  ];
  return renderSummaryBlock(lineList);
}

/**
 * 渲染操作列。
 */
function renderOperationCell(scope: RenderScope) {
  const row = scope.row as RecommendModelVersion;
  const actionNodes: VNode[] = [];

  if (BUTTONS.value["shop:recommend-model-version:publish"]) {
    actionNodes.push(
      h(
        resolveComponent("el-button"),
        {
          key: `publish-${row.id}`,
          type: "primary",
          link: true,
          icon: Promotion,
          onClick: () => handleOpenPublishDialog(row)
        },
        () => "正式发布"
      )
    );
  }

  if (BUTTONS.value["shop:recommend-model-version:rollback"]) {
    actionNodes.push(
      h(
        resolveComponent("el-button"),
        {
          key: `rollback-${row.id}`,
          type: "warning",
          link: true,
          icon: RefreshLeft,
          onClick: () => handleOpenRollbackDialog(row)
        },
        () => "设置回滚"
      )
    );
  }

  if (BUTTONS.value["shop:recommend-model-version:rollback-clear"]) {
    actionNodes.push(
      h(
        resolveComponent("el-button"),
        {
          key: `rollback-clear-${row.id}`,
          type: "danger",
          link: true,
          icon: SwitchButton,
          onClick: () => handleClearRollback(row)
        },
        () => "清空回滚"
      )
    );
  }

  if (!actionNodes.length) return "--";
  return h("div", { class: "summary-actions" }, actionNodes);
}

/** 推荐版本表格列配置。 */
const columns: ColumnProps[] = [
  {
    prop: "scene",
    label: "场景",
    minWidth: 120,
    search: { el: "select" },
    dictCode: recommendSceneDictCode,
    dictValueType: "number"
  },
  { prop: "modelName", label: "模型名称", minWidth: 160, search: { el: "input" } },
  { prop: "modelType", label: "模型类型", minWidth: 120, search: { el: "input" } },
  { prop: "version", label: "版本号", minWidth: 120, search: { el: "input" } },
  { prop: "effectiveVersion", label: "生效缓存版本", minWidth: 140 },
  { prop: "status", label: "状态", minWidth: 100, search: { el: "select" }, enum: statusOptions, render: renderStatusCell },
  { prop: "publish", label: "发布摘要", minWidth: 260, render: renderPublishSummary },
  { prop: "tune", label: "调参摘要", minWidth: 180, render: renderTuneSummary },
  { prop: "latest", label: "最近训练摘要", minWidth: 300, render: renderLatestTrainSummary },
  { prop: "latestEval", label: "最近评估摘要", minWidth: 300, render: renderLatestEvalSummary },
  { prop: "updatedAt", label: "更新时间", minWidth: 180 },
  {
    prop: "operation",
    label: "操作",
    width: 240,
    fixed: "right",
    render: renderOperationCell
  }
];

/** 页面顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [];

/**
 * 请求推荐版本列表，并交给 ProTable 统一处理分页与筛选。
 */
async function requestRecommendModelVersionTable(params: PageRecommendModelVersionRequest) {
  const data = await defRecommendModelVersionService.PageRecommendModelVersion(buildPageRequest(params));
  return { data };
}

/**
 * 刷新推荐版本表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 重置发布弹窗状态，避免切换动作时残留上一次输入。
 */
function resetDialogForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  dialogFormData.id = 0;
  dialogFormData.action = RecommendModelVersionPublishAction.RECOMMEND_MODEL_VERSION_PUBLISH_ACTION_PUBLISH;
  dialogFormData.cacheVersion = "";
  dialogFormData.rollbackVersion = "";
  dialogFormData.grayRatio = undefined;
  dialogFormData.publishedBy = "";
  dialogFormData.publishedReason = "";
  currentRow.value = undefined;
}

/**
 * 打开正式发布弹窗，并回填当前版本信息。
 */
function handleOpenPublishDialog(row: RecommendModelVersion) {
  resetDialogForm();
  currentRow.value = row;
  dialog.title = "正式发布推荐版本";
  dialog.visible = true;
  dialogFormData.id = row.id;
  dialogFormData.action = RecommendModelVersionPublishAction.RECOMMEND_MODEL_VERSION_PUBLISH_ACTION_PUBLISH;
  dialogFormData.cacheVersion = row.version || row.publish?.cacheVersion || "";
  dialogFormData.grayRatio = row.publish?.grayRatio ?? 1;
}

/**
 * 打开回滚设置弹窗，并回填当前回滚目标。
 */
function handleOpenRollbackDialog(row: RecommendModelVersion) {
  resetDialogForm();
  currentRow.value = row;
  dialog.title = "设置推荐版本回滚";
  dialog.visible = true;
  dialogFormData.id = row.id;
  dialogFormData.action = RecommendModelVersionPublishAction.RECOMMEND_MODEL_VERSION_PUBLISH_ACTION_SET_ROLLBACK;
  dialogFormData.rollbackVersion = row.publish?.rollbackVersion || row.version || "";
}

/**
 * 关闭弹窗并恢复默认值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetDialogForm();
}

/**
 * 提交发布或回滚动作。
 */
function handleSubmitDialog() {
  formDialogRef.value?.validate()?.then(isValid => {
    if (!isValid) return;
    const request = JSON.parse(JSON.stringify(dialogFormData)) as UpdateRecommendModelVersionPublishRequest;
    defRecommendModelVersionService.PublishRecommendModelVersion(request).then(res => {
      const actionText = renderActionText(request.action);
      ElMessage.success(`${actionText}成功`);
      handleCloseDialog();
      refreshTable();
      // 返回执行摘要时，使用弹窗展示任务回写结果，便于排查发布链路。
      if (res.summary?.length) {
        ElMessageBox.alert(res.summary.join("<br />"), `${actionText}摘要`, {
          dangerouslyUseHTMLString: true
        });
      }
    });
  });
}

/**
 * 清空当前版本的回滚配置。
 */
function handleClearRollback(row: RecommendModelVersion) {
  ElMessageBox.confirm(`确认清空场景“${renderSceneText(row.scene)}”当前版本的回滚配置吗？`, "清空回滚", {
    type: "warning"
  }).then(() => {
    defRecommendModelVersionService
      .PublishRecommendModelVersion({
        id: row.id,
        action: RecommendModelVersionPublishAction.RECOMMEND_MODEL_VERSION_PUBLISH_ACTION_CLEAR_ROLLBACK,
        cacheVersion: "",
        rollbackVersion: "",
        grayRatio: undefined,
        publishedBy: "",
        publishedReason: ""
      })
      .then(res => {
        ElMessage.success("清空回滚成功");
        refreshTable();
        if (res.summary?.length) {
          ElMessageBox.alert(res.summary.join("<br />"), "清空回滚摘要", {
            dangerouslyUseHTMLString: true
          });
        }
      });
  });
}

/**
 * 优先根据数据字典渲染推荐场景文案，缺失时退回页面兜底值。
 */
function renderSceneText(scene?: RecommendScene) {
  if (scene === undefined || scene === RecommendScene.RECOMMEND_SCENE_UNKNOWN) return "--";

  const matchedItem = dictStore.getDictionary(recommendSceneDictCode).find(dictItem => Number(dictItem.value) === Number(scene));
  if (matchedItem?.label) return matchedItem.label;

  return recommendSceneFallbackMap[Number(scene)] ?? "--";
}

/**
 * 将发布动作枚举转换为成功提示文案。
 */
function renderActionText(action: RecommendModelVersionPublishAction) {
  switch (action) {
    case RecommendModelVersionPublishAction.RECOMMEND_MODEL_VERSION_PUBLISH_ACTION_SET_ROLLBACK:
      return "设置回滚";
    case RecommendModelVersionPublishAction.RECOMMEND_MODEL_VERSION_PUBLISH_ACTION_CLEAR_ROLLBACK:
      return "清空回滚";
    default:
      return "正式发布";
  }
}

/**
 * 将训练指标映射格式化为可读文本。
 */
function formatScoreSummary(score?: Map<string, number> | Record<string, number>) {
  if (!score) return "";
  const entryList = score instanceof Map ? Array.from(score.entries()) : Object.entries((score as Record<string, number>) ?? {});
  return entryList
    .filter(([key]) => !!key)
    .map(([key, value]) => `${key}=${formatMetric(value)}`)
    .join(" / ");
}

/**
 * 将数值指标格式化为保留四位小数的字符串。
 */
function formatMetric(value?: number) {
  if (value === undefined || value === null || Number.isNaN(Number(value))) return "--";
  return Number(value).toFixed(4);
}

/**
 * 渲染统一的摘要块。
 */
function renderSummaryBlock(lineList: string[]) {
  if (!lineList.length) {
    return h("div", { class: "summary-empty" }, "--");
  }
  return h(
    "div",
    { class: "summary-lines" },
    lineList.map((line, index) =>
      h(
        "div",
        {
          key: `${line}-${index}`
        },
        line
      )
    )
  );
}

/**
 * 渲染统一标签。
 */
function renderTag(label: string, type: "primary" | "success" | "warning" | "danger") {
  return h(
    resolveComponent("el-tag"),
    {
      type
    },
    () => label
  );
}
</script>

<style scoped lang="scss">
.summary-lines {
  line-height: 1.6;
}

.summary-empty {
  color: var(--el-text-color-placeholder);
}

.summary-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
</style>
