<template>
  <div class="fallback-recommenders-field">
    <el-empty v-if="sortedRecommenders.length === 0" :image-size="72" description="请先在画布中连接兜底推荐器" />
    <div v-else class="fallback-recommenders-field__list">
      <div v-for="(item, index) in sortedRecommenders" :key="`${item}-${index}`" class="fallback-recommenders-field__item">
        <span class="fallback-recommenders-field__name">{{ item }}</span>
        <div class="fallback-recommenders-field__actions">
          <el-button plain :disabled="index === 0" title="上移" @click="moveItem(index, -1)">
            <span class="fallback-recommenders-field__arrow">↑</span>
          </el-button>
          <el-button plain :disabled="index === sortedRecommenders.length - 1" title="下移" @click="moveItem(index, 1)">
            <span class="fallback-recommenders-field__arrow">↓</span>
          </el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

interface FallbackRecommendersFieldProps {
  /** 当前兜底推荐器排序列表。 */
  modelValue?: string[];
}

const props = withDefaults(defineProps<FallbackRecommendersFieldProps>(), {
  modelValue: () => []
});
const emit = defineEmits<{
  /** 更新兜底推荐器排序。 */
  "update:modelValue": [value: string[]];
}>();

/** 过滤空值后的推荐器列表，避免空行参与排序和保存。 */
const sortedRecommenders = computed(() => props.modelValue.map(item => String(item).trim()).filter(Boolean));

/** 上移或下移指定推荐器。 */
function moveItem(index: number, offset: -1 | 1) {
  const targetIndex = index + offset;
  if (targetIndex < 0 || targetIndex >= sortedRecommenders.value.length) return;

  const nextList = [...sortedRecommenders.value];
  [nextList[index], nextList[targetIndex]] = [nextList[targetIndex], nextList[index]];
  emit("update:modelValue", nextList);
}
</script>

<style scoped lang="scss">
.fallback-recommenders-field {
  width: 100%;
}

.fallback-recommenders-field__list {
  overflow: hidden;
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
}

.fallback-recommenders-field__item {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  min-height: 58px;
  padding: 10px 12px 10px 16px;
  background: var(--admin-page-card-bg);

  & + & {
    border-top: 1px solid var(--admin-page-card-border-soft);
  }
}

.fallback-recommenders-field__name {
  min-width: 0;
  color: var(--admin-page-text-primary);
  word-break: break-all;
}

.fallback-recommenders-field__actions {
  display: inline-flex;
  flex-shrink: 0;
  gap: 8px;
  align-items: center;
}

.fallback-recommenders-field__arrow {
  display: inline-block;
  min-width: 14px;
  font-size: 16px;
  line-height: 1;
}
</style>
