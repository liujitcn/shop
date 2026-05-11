<template>
  <section class="ai-placeholder-page">
    <div class="ai-placeholder-page__header">
      <div>
        <h1 class="ai-placeholder-page__title">AI图片</h1>
        <p class="ai-placeholder-page__desc">图片生成与图片分析能力入口预留在这里，后续按统一 AI 模块继续扩展。</p>
      </div>
    </div>
    <div class="ai-placeholder-page__body">
      <div class="ai-placeholder-card">
        <div class="ai-placeholder-card__label">当前状态</div>
        <div class="ai-placeholder-card__value">待接入</div>
        <p class="ai-placeholder-card__desc">后续接入图片生成、图片理解、素材处理等能力。</p>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
defineOptions({
  name: "AiImage"
});
</script>

<style scoped lang="scss">
.ai-placeholder-page {
  display: flex;
  gap: 20px;
  min-height: 100%;
  flex-direction: column;
}

.ai-placeholder-page__header {
  padding: 4px 2px 0;
}

.ai-placeholder-page__title {
  margin: 0;
  font-size: 24px;
  line-height: 32px;
  color: var(--admin-page-text-primary);
}

.ai-placeholder-page__desc {
  margin: 8px 0 0;
  font-size: 14px;
  line-height: 22px;
  color: var(--admin-page-text-secondary);
}

.ai-placeholder-page__body {
  display: grid;
  gap: 16px;
  grid-template-columns: minmax(0, 360px);
}

.ai-placeholder-card {
  padding: 24px;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  box-shadow: var(--admin-page-shadow);
}

.ai-placeholder-card__label {
  font-size: 13px;
  line-height: 20px;
  color: var(--admin-page-text-secondary);
}

.ai-placeholder-card__value {
  margin-top: 10px;
  font-size: 22px;
  font-weight: 700;
  line-height: 30px;
  color: var(--admin-page-text-primary);
}

.ai-placeholder-card__desc {
  margin: 10px 0 0;
  font-size: 14px;
  line-height: 22px;
  color: var(--admin-page-text-secondary);
}
</style>
