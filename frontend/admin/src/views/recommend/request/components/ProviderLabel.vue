<template>
  <el-tooltip v-if="isInteractive" :content="tooltipLabel" effect="light" placement="top">
    <span v-bind="attrs" class="recommend-provider-label recommend-provider-label--interactive">{{ displayText }}</span>
  </el-tooltip>
  <span v-else v-bind="attrs" class="recommend-provider-label">{{ displayText }}</span>
</template>

<script setup lang="ts">
import { computed, ref, useAttrs, watch } from "vue";
import type { OptionBaseDictsResponse_BaseDictItem } from "@/rpc/admin/v1/base_dict";
import { useDictStore } from "@/stores/modules/dict";
import { RecommendStrategy } from "@/rpc/common/v1/enum";

defineOptions({
  inheritAttrs: false
});

/** 推荐器字典编码。 */
const RECOMMEND_PROVIDER_DICT_CODE = "recommend_provider";

/** 推荐器标签组件入参。 */
interface RecommendProviderLabelProps {
  /** 当前推荐器所属策略。 */
  strategy?: RecommendStrategy | string;
  /** 当前推荐器英文标识。 */
  providerName?: string;
}

const props = defineProps<RecommendProviderLabelProps>();

const attrs = useAttrs();
const dictStore = useDictStore();
const tooltipLabel = ref("");

/** 默认展示推荐器英文标识，未命中时回退为占位符。 */
const displayText = computed(() => {
  const providerName = String(props.providerName ?? "").trim();
  return providerName || "--";
});

/** 当前推荐器是否需要展示悬浮交互态。 */
const isInteractive = computed(() => Boolean(tooltipLabel.value) && displayText.value !== "--");

/**
 * 统一构建推荐器字典值，避免Gorse与本地推荐器重名时出现映射冲突。
 */
function buildRecommendProviderDictValue(strategy?: RecommendStrategy | string, providerName?: string) {
  const normalizedProviderName = String(providerName ?? "").trim();
  if (!normalizedProviderName) return "";

  if (strategy === RecommendStrategy.REMOTE_STRATEGY || strategy === "remote") {
    return `gorse:${normalizedProviderName}`;
  }
  if (strategy === RecommendStrategy.LOCAL_STRATEGY || strategy === "local") {
    return `local:${normalizedProviderName}`;
  }
  return "";
}

/**
 * 从字典列表中严格匹配推荐器字典项，仅使用策略+推荐器的唯一值。
 */
function matchRecommendProviderDictItem(
  dictList: OptionBaseDictsResponse_BaseDictItem[] = [],
  strategy?: RecommendStrategy | string,
  providerName?: string
) {
  const dictValue = buildRecommendProviderDictValue(strategy, providerName);
  if (!dictValue) return undefined;

  return dictList.find(dictItem => dictItem.value === dictValue);
}

/**
 * 刷新推荐器中文悬浮文案，统一从推荐器字典中取值。
 */
async function refreshTooltipLabel() {
  const providerName = String(props.providerName ?? "").trim();
  if (!providerName) {
    tooltipLabel.value = "";
    return;
  }

  let dictList = dictStore.getDictionary(RECOMMEND_PROVIDER_DICT_CODE);
  if (!dictList.length) {
    await dictStore.loadDictionaries();
    dictList = dictStore.getDictionary(RECOMMEND_PROVIDER_DICT_CODE);
  }

  const matchedItem = matchRecommendProviderDictItem(dictList, props.strategy, providerName);
  tooltipLabel.value = matchedItem?.label ?? "";
}

watch(
  () => [props.strategy, props.providerName],
  async () => {
    await refreshTooltipLabel();
  },
  { immediate: true }
);
</script>

<style scoped lang="scss">
.recommend-provider-label {
  display: inline-flex;
  align-items: center;
  min-width: 0;
  max-width: 100%;
  padding: 4px 10px;
  border: 1px solid transparent;
  border-radius: 999px;
  background: transparent;
  color: inherit;
  font-size: inherit;
  font-weight: inherit;
  line-height: inherit;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: middle;
  white-space: nowrap;
  transition:
    color 0.2s ease,
    border-color 0.2s ease,
    background-color 0.2s ease,
    box-shadow 0.2s ease,
    transform 0.2s ease;
}

.recommend-provider-label--interactive {
  cursor: pointer;
  background: var(--admin-page-card-bg-soft);
  border-color: var(--admin-page-card-border-soft);
}

.recommend-provider-label--interactive:hover {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border-color: var(--el-color-primary-light-5);
  box-shadow: 0 8px 18px rgb(64 158 255 / 12%);
  transform: translateY(-1px);
}
</style>
