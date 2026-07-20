<template>
  <div class="goods-h5-preview">
    <el-tooltip :disabled="!iconOnly" :content="tooltip || buttonText" placement="bottom">
      <el-button :type="buttonType" :plain="plain" :size="size" :circle="circle || iconOnly" @click="handleOpenPreview">
        <el-icon><View /></el-icon>
        <span v-if="!iconOnly">{{ buttonText }}</span>
      </el-button>
    </el-tooltip>

    <el-drawer
      v-model="drawerVisible"
      title="H5预览"
      :size="previewDrawerSize"
      append-to-body
      destroy-on-close
      @opened="handlePreviewDrawerOpened"
      @closed="handlePreviewDrawerClosed"
    >
      <div ref="previewAreaRef" class="goods-h5-preview__area">
        <div v-if="previewUrl" class="goods-h5-preview__scale-box" :style="previewScaleBoxStyle">
          <div ref="previewShellRef" class="goods-h5-preview__phone-shell" :style="previewShellStyle">
            <div class="goods-h5-preview__phone-head">
              <span class="goods-h5-preview__camera"></span>
            </div>
            <iframe
              ref="previewFrameRef"
              class="goods-h5-preview__frame"
              :src="previewUrl"
              title="商品H5预览"
              loading="lazy"
              @load="handlePreviewFrameLoad"
            />
          </div>
        </div>
        <el-empty v-else class="goods-h5-preview__empty" description="暂无可预览商品" :image-size="80" />
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref } from "vue";
import type { CSSProperties } from "vue";
import { ElMessage } from "element-plus";
import { buildGoodsH5PreviewUrl } from "@/utils/utils";

defineOptions({
  name: "GoodsH5PreviewDrawer",
  inheritAttrs: false
});

/** 预览按钮类型，统一收敛为 Element Plus 支持的按钮枚举值。 */
type PreviewButtonType = "" | "default" | "primary" | "success" | "warning" | "info" | "danger" | "text";

const props = withDefaults(
  defineProps<{
    goodsId?: string | number;
    buttonText?: string;
    buttonType?: PreviewButtonType;
    plain?: boolean;
    size?: "large" | "default" | "small";
    circle?: boolean;
    iconOnly?: boolean;
    tooltip?: string;
  }>(),
  {
    goodsId: "",
    buttonText: "H5预览",
    buttonType: "primary",
    plain: false,
    size: "default",
    circle: false,
    iconOnly: false,
    tooltip: ""
  }
);

const drawerVisible = ref(false);
const previewAreaRef = ref<HTMLDivElement>();
const previewShellRef = ref<HTMLDivElement>();
const previewFrameRef = ref<HTMLIFrameElement>();
const previewScale = ref(1);
const previewShellSize = ref({
  width: 405,
  height: 774
});
let previewResizeObserver: ResizeObserver | undefined;
let previewResizeFrame = 0;
/** 抽屉宽度随管理端窗口收缩，避免小屏下抽屉自身横向溢出。 */
const previewDrawerSize = "min(440px, calc(100vw - 24px))";

/** 根据商品ID实时生成 H5 商品详情页地址。 */
const previewUrl = computed(() => buildGoodsH5PreviewUrl(props.goodsId));
const previewScaleBoxStyle = computed<CSSProperties>(() => ({
  width: `${previewShellSize.value.width * previewScale.value}px`,
  height: `${previewShellSize.value.height * previewScale.value}px`
}));
const previewShellStyle = computed<CSSProperties>(() => ({
  "--goods-h5-preview-scale": String(previewScale.value)
}) as CSSProperties);

/** 打开预览抽屉，未保存商品时给出明确提示。 */
function handleOpenPreview() {
  if (!previewUrl.value) {
    ElMessage.warning("请先保存商品后再预览 H5");
    return;
  }
  drawerVisible.value = true;
}

/** 抽屉打开后根据可用宽度计算预览壳缩放比例。 */
function handlePreviewDrawerOpened() {
  nextTick(() => {
    bindPreviewResizeObserver();
    updatePreviewScale();
  });
}

/** 抽屉关闭时清理尺寸观察器，避免重复绑定。 */
function handlePreviewDrawerClosed() {
  unbindPreviewResizeObserver();
  previewScale.value = 1;
}

/** 绑定预览区域尺寸监听，适配浏览器窗口和抽屉宽度变化。 */
function bindPreviewResizeObserver() {
  unbindPreviewResizeObserver();
  if (!previewAreaRef.value || typeof ResizeObserver === "undefined") {
    window.addEventListener("resize", scheduleUpdatePreviewScale);
    scheduleUpdatePreviewScale();
    return;
  }
  previewResizeObserver = new ResizeObserver(scheduleUpdatePreviewScale);
  previewResizeObserver.observe(previewAreaRef.value);
  if (previewShellRef.value) previewResizeObserver.observe(previewShellRef.value);
}

/** 解绑预览区域尺寸监听。 */
function unbindPreviewResizeObserver() {
  previewResizeObserver?.disconnect();
  previewResizeObserver = undefined;
  window.removeEventListener("resize", scheduleUpdatePreviewScale);
  if (previewResizeFrame) {
    window.cancelAnimationFrame(previewResizeFrame);
    previewResizeFrame = 0;
  }
}

/** 合并同一帧内的尺寸变化，减少拖拽窗口时的重复计算。 */
function scheduleUpdatePreviewScale() {
  if (previewResizeFrame) window.cancelAnimationFrame(previewResizeFrame);
  previewResizeFrame = window.requestAnimationFrame(() => {
    previewResizeFrame = 0;
    updatePreviewScale();
  });
}

/** 按管理端当前可用宽度缩放预览外壳，保持 iframe 内 H5 视口宽度稳定。 */
function updatePreviewScale() {
  const areaWidth = previewAreaRef.value?.clientWidth || 0;
  const shellWidth = previewShellRef.value?.offsetWidth || previewShellSize.value.width;
  const shellHeight = previewShellRef.value?.offsetHeight || previewShellSize.value.height;
  if (!areaWidth || !shellWidth) return;

  previewShellSize.value = {
    width: shellWidth,
    height: shellHeight
  };
  previewScale.value = Math.min(1, Number((areaWidth / shellWidth).toFixed(3)));
}

/** 隐藏预览 iframe 内的桌面滚动条占位，保持 H5 按真实移动端宽度排版。 */
function handlePreviewFrameLoad() {
  try {
    const frameDocument = previewFrameRef.value?.contentDocument;
    if (!frameDocument?.head) return;

    const styleId = "shop-admin-h5-preview-style";
    let styleElement = frameDocument.getElementById(styleId) as HTMLStyleElement | null;
    if (!styleElement) {
      styleElement = frameDocument.createElement("style");
      styleElement.id = styleId;
      frameDocument.head.appendChild(styleElement);
    }
    styleElement.textContent = `
      *,
      html,
      body {
        -ms-overflow-style: none !important;
        scrollbar-width: none !important;
      }

      *::-webkit-scrollbar {
        width: 0 !important;
        height: 0 !important;
      }
    `;
  } catch {
    return;
  }
}

onBeforeUnmount(() => {
  unbindPreviewResizeObserver();
});
</script>

<style scoped lang="scss">
.goods-h5-preview__area {
  display: flex;
  justify-content: center;
  width: 100%;
  overflow: hidden;
}
.goods-h5-preview__scale-box {
  position: relative;
  flex: 0 0 auto;
}
.goods-h5-preview__phone-shell {
  position: absolute;
  top: 0;
  left: 0;
  box-sizing: border-box;
  width: 405px;
  padding: 18px 14px 14px;
  overflow: hidden;
  background: linear-gradient(180deg, var(--admin-page-card-bg) 0%, var(--admin-page-card-bg-soft) 100%);
  border: 1px solid var(--admin-page-card-border);
  border-radius: 28px;
  box-shadow: var(--admin-page-shadow);
  transform: scale(var(--goods-h5-preview-scale));
  transform-origin: left top;
}
.goods-h5-preview__phone-head {
  display: flex;
  justify-content: center;
  margin-bottom: 12px;
}
.goods-h5-preview__camera {
  width: 78px;
  height: 8px;
  background: var(--admin-page-card-border-muted);
  border-radius: 999px;
}
.goods-h5-preview__frame {
  display: block;
  width: 375px;
  height: clamp(560px, calc(100vh - 160px), 720px);
  overflow: hidden;
  background: #ffffff;
  border: none;
  border-radius: 18px;
}
.goods-h5-preview__empty {
  width: 100%;
}
</style>
