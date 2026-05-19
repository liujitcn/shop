<template>
  <div class="ai-image-loading" aria-live="polite">
    <div class="ai-image-loading__preview">
      <span class="ai-image-loading__grain" />
      <span class="ai-image-loading__horizon" />
      <span class="ai-image-loading__subject ai-image-loading__subject--main" />
      <span class="ai-image-loading__subject ai-image-loading__subject--side" />
      <el-icon class="ai-image-loading__icon"><Picture /></el-icon>
      <span class="ai-image-loading__spark ai-image-loading__spark--one" />
      <span class="ai-image-loading__spark ai-image-loading__spark--two" />
      <span class="ai-image-loading__spark ai-image-loading__spark--three" />
      <span class="ai-image-loading__scan" />
    </div>
    <div class="ai-image-loading__stages" aria-hidden="true">
      <span v-for="stage in generatingStages" :key="stage">{{ stage }}</span>
    </div>
    <div class="ai-image-loading__dots" aria-hidden="true">
      <span />
      <span />
      <span />
    </div>
    <div class="ai-image-loading__text">
      <strong>正在生成图片</strong>
      <small>模型正在构图、补光和渲染细节</small>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Picture } from "@element-plus/icons-vue";

defineOptions({
  name: "GeneratingPreview"
});

/** 图片生成过程阶段文案。 */
const generatingStages = ["构图", "补光", "细节", "出图"];
</script>

<style scoped lang="scss">
.ai-image-loading {
  display: grid;
  min-height: 360px;
  padding: 28px;
  overflow: hidden;
  place-items: center;
  background:
    radial-gradient(circle at 18% 18%, rgb(64 158 255 / 16%), transparent 34%),
    radial-gradient(circle at 82% 22%, rgb(103 194 58 / 14%), transparent 30%), var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
}

.ai-image-loading__preview {
  position: relative;
  width: min(280px, 80vw);
  aspect-ratio: 1;
  overflow: hidden;
  background: linear-gradient(145deg, #f8fbff 0%, #e8f4ef 48%, #dfeffc 100%);
  border: 1px solid rgb(64 158 255 / 20%);
  border-radius: 8px;
  box-shadow: 0 18px 50px rgb(24 38 66 / 16%);
}

.ai-image-loading__grain,
.ai-image-loading__horizon,
.ai-image-loading__subject,
.ai-image-loading__spark,
.ai-image-loading__scan {
  position: absolute;
  display: block;
}

.ai-image-loading__grain {
  inset: 0;
  background-image:
    linear-gradient(90deg, rgb(255 255 255 / 18%) 1px, transparent 1px),
    linear-gradient(0deg, rgb(255 255 255 / 18%) 1px, transparent 1px);
  background-size: 18px 18px;
  opacity: 0.45;
}

.ai-image-loading__horizon {
  right: 0;
  bottom: 0;
  left: 0;
  height: 42%;
  background: linear-gradient(180deg, rgb(103 194 58 / 0%), rgb(103 194 58 / 26%));
}

.ai-image-loading__subject {
  background: linear-gradient(135deg, #67c23a, #f3d35f);
  border-radius: 8px;
  animation: floatSubject 2.4s ease-in-out infinite;
}

.ai-image-loading__subject--main {
  right: 86px;
  bottom: 64px;
  width: 92px;
  height: 112px;
}

.ai-image-loading__subject--side {
  right: 58px;
  bottom: 72px;
  width: 48px;
  height: 72px;
  background: linear-gradient(135deg, #409eff, #67c23a);
  animation-delay: 0.2s;
}

.ai-image-loading__icon {
  position: absolute;
  top: 50%;
  left: 50%;
  z-index: 2;
  font-size: 46px;
  color: rgb(255 255 255 / 86%);
  transform: translate(-50%, -50%);
}

.ai-image-loading__spark {
  width: 10px;
  height: 10px;
  background: #fff;
  border-radius: 50%;
  box-shadow: 0 0 18px rgb(255 255 255 / 90%);
  animation: blinkSpark 1.8s ease-in-out infinite;
}

.ai-image-loading__spark--one {
  top: 58px;
  left: 58px;
}

.ai-image-loading__spark--two {
  top: 88px;
  right: 54px;
  animation-delay: 0.3s;
}

.ai-image-loading__spark--three {
  right: 94px;
  bottom: 46px;
  animation-delay: 0.6s;
}

.ai-image-loading__scan {
  top: -30%;
  left: -20%;
  width: 140%;
  height: 34%;
  background: linear-gradient(180deg, transparent, rgb(255 255 255 / 40%), transparent);
  transform: rotate(-10deg);
  animation: scanImage 2.2s ease-in-out infinite;
}

.ai-image-loading__stages {
  display: flex;
  gap: 8px;
  justify-content: center;
  margin-top: 20px;
}

.ai-image-loading__stages span {
  padding: 3px 9px;
  font-size: 12px;
  color: var(--admin-page-text-secondary);
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border);
  border-radius: 999px;
}

.ai-image-loading__dots {
  display: flex;
  gap: 6px;
  justify-content: center;
  margin-top: 14px;
}

.ai-image-loading__dots span {
  width: 7px;
  height: 7px;
  background: var(--el-color-primary);
  border-radius: 50%;
  animation: pulseDot 1s ease-in-out infinite;
}

.ai-image-loading__dots span:nth-child(2) {
  animation-delay: 0.15s;
}

.ai-image-loading__dots span:nth-child(3) {
  animation-delay: 0.3s;
}

.ai-image-loading__text {
  display: grid;
  gap: 4px;
  margin-top: 12px;
  text-align: center;
}

.ai-image-loading__text strong {
  color: var(--admin-page-text-primary);
}

.ai-image-loading__text small {
  color: var(--admin-page-text-secondary);
}

@keyframes scanImage {
  0% {
    transform: translateY(0) rotate(-10deg);
  }

  100% {
    transform: translateY(420%) rotate(-10deg);
  }
}

@keyframes blinkSpark {
  0%,
  100% {
    opacity: 0.28;
    transform: scale(0.8);
  }

  50% {
    opacity: 1;
    transform: scale(1.2);
  }
}

@keyframes pulseDot {
  0%,
  100% {
    opacity: 0.35;
    transform: translateY(0);
  }

  50% {
    opacity: 1;
    transform: translateY(-4px);
  }
}

@keyframes floatSubject {
  0%,
  100% {
    transform: translateY(0);
  }

  50% {
    transform: translateY(-8px);
  }
}
</style>
