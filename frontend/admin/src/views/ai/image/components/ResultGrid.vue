<template>
  <el-empty v-if="!items.length" description="暂无图片" />
  <div v-else class="ai-image-grid">
    <article v-for="item in items" :key="item.key" class="ai-image-card">
      <el-image
        class="ai-image-card__media"
        :src="item.previewUrl"
        fit="cover"
        :preview-src-list="[item.previewUrl]"
        preview-teleported
      />
      <div class="ai-image-card__meta">
        <div class="ai-image-card__name">{{ item.name || "AI图片" }}</div>
        <div class="ai-image-card__tags">
          <el-tag size="small" effect="plain">{{ item.mime_type || "image/png" }}</el-tag>
          <el-tag v-if="item.saved" size="small" effect="plain" type="success">已保存</el-tag>
        </div>
      </div>
      <div v-if="item.storage_path || item.request_id" class="ai-image-card__trace">
        <span v-if="item.request_id">批次：{{ item.request_id }}</span>
        <span v-if="item.storage_path">目录：{{ item.storage_path }}</span>
      </div>
      <div class="ai-image-card__actions">
        <el-tooltip content="复制地址" placement="top">
          <el-button circle :icon="CopyDocument" @click="handleCopyUrl(item)" />
        </el-tooltip>
        <el-tooltip content="下载图片" placement="top">
          <el-button circle :icon="Download" @click="handleDownload(item)" />
        </el-tooltip>
      </div>
    </article>
  </div>
</template>

<script setup lang="ts">
import { ElMessage } from "element-plus";
import { CopyDocument, Download } from "@element-plus/icons-vue";
import type { ImageItem } from "./types";
import { resolveImageExtension } from "./utils";

defineOptions({
  name: "ResultGrid"
});

defineProps<{
  /** 图片结果列表。 */
  items: ImageItem[];
}>();

/** 复制图片地址。 */
async function handleCopyUrl(item: ImageItem) {
  await navigator.clipboard.writeText(item.previewUrl);
  ElMessage.success("图片地址已复制");
}

/** 下载图片结果。 */
function handleDownload(item: ImageItem) {
  const link = document.createElement("a");
  link.href = item.previewUrl;
  link.download = item.name || `ai-image.${resolveImageExtension(item.mime_type)}`;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
}
</script>

<style scoped lang="scss">
.ai-image-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 16px;
}

.ai-image-card {
  position: relative;
  min-width: 0;
  overflow: hidden;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
}

.ai-image-card__media {
  display: block;
  width: 100%;
  aspect-ratio: 1;
  background: var(--admin-page-bg);
}

.ai-image-card__meta {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  justify-content: space-between;
  padding: 12px;
}

.ai-image-card__name {
  min-width: 0;
  overflow: hidden;
  font-weight: 600;
  color: var(--admin-page-text-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ai-image-card__tags {
  display: flex;
  flex-shrink: 0;
  gap: 6px;
}

.ai-image-card__trace {
  display: grid;
  gap: 4px;
  padding: 0 12px 12px;
  overflow: hidden;
  font-size: 12px;
  color: var(--admin-page-text-secondary);
  overflow-wrap: anywhere;
}

.ai-image-card__actions {
  position: absolute;
  top: 10px;
  right: 10px;
  display: flex;
  gap: 8px;
  opacity: 0;
  transition: opacity 0.2s ease;
}

.ai-image-card:hover .ai-image-card__actions {
  opacity: 1;
}
</style>
