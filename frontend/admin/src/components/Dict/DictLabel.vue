<template>
  <el-tag v-if="tagType" :type="tagType" :size="size">{{ label }}</el-tag>
  <span v-else>{{ label }}</span>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { useDictStore } from "@/stores/modules/dict";

type TagType = "success" | "warning" | "info" | "primary" | "danger";

interface DictLabelProps {
  code: string;
  modelValue?: string | number;
  size?: "default" | "large" | "small";
}

const props = withDefaults(defineProps<DictLabelProps>(), {
  size: "default"
});

const dictStore = useDictStore();
const label = ref("");
const tagType = ref<TagType | undefined>();

/**
 * 过滤后端返回的 tag 类型，确保只传递 Element Plus 支持的枚举值。
 */
function normalizeTagType(rawTagType?: string): TagType | undefined {
  const supportedTagTypes: TagType[] = ["success", "warning", "info", "primary", "danger"];
  if (!rawTagType) return undefined;
  return supportedTagTypes.find(tag => tag === rawTagType);
}

/**
 * 根据字典值刷新标签文本和标签类型。
 */
async function refreshLabelAndTag() {
  if (!props.code) {
    label.value = "";
    tagType.value = undefined;
    return;
  }

  let dictList = dictStore.getDictionary(props.code);
  if (!dictList.length) {
    // 字典缓存可能尚未初始化，组件内部兜底触发一次加载。
    await dictStore.loadDictionaries();
    dictList = dictStore.getDictionary(props.code);
  }

  const matchedItem = dictList.find(dictItem => dictItem.value == props.modelValue);
  label.value = matchedItem?.label ?? "";
  tagType.value = normalizeTagType(matchedItem?.tagType);
}

watch(
  () => [props.code, props.modelValue],
  async () => {
    await refreshLabelAndTag();
  },
  { immediate: true }
);
</script>
