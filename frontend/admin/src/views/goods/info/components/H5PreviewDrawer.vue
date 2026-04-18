<template>
  <div class="goods-h5-preview">
    <el-tooltip :disabled="!iconOnly" :content="tooltip || buttonText" placement="bottom">
      <el-button :type="buttonType" :plain="plain" :size="size" :circle="circle || iconOnly" @click="handleOpenPreview">
        <el-icon><View /></el-icon>
        <span v-if="!iconOnly">{{ buttonText }}</span>
      </el-button>
    </el-tooltip>

    <el-drawer v-model="drawerVisible" title="H5预览" size="440px" append-to-body destroy-on-close>
      <div class="goods-h5-preview__toolbar">
        <el-input :model-value="previewUrl" readonly />
        <div class="goods-h5-preview__actions">
          <el-button type="primary" @click="handleOpenInNewWindow">新窗口打开</el-button>
        </div>
      </div>

      <div class="goods-h5-preview__phone-shell">
        <div class="goods-h5-preview__phone-head">
          <span class="goods-h5-preview__camera"></span>
        </div>
        <iframe v-if="previewUrl" class="goods-h5-preview__frame" :src="previewUrl" title="商品H5预览" loading="lazy" />
        <el-empty v-else description="暂无可预览商品" :image-size="80" />
      </div>

      <p class="goods-h5-preview__tip">当前预览展示的是已保存到商城端的商品详情效果。</p>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { ElMessage } from "element-plus";
import { buildGoodsH5PreviewUrl } from "@/utils/utils";

defineOptions({
  name: "GoodsH5PreviewDrawer",
  inheritAttrs: false
});

const props = withDefaults(
  defineProps<{
    goodsId?: string | number;
    buttonText?: string;
    buttonType?: string;
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

/** 根据商品ID实时生成 H5 商品详情页地址。 */
const previewUrl = computed(() => buildGoodsH5PreviewUrl(props.goodsId));

/** 打开预览抽屉，未保存商品时给出明确提示。 */
function handleOpenPreview() {
  if (!previewUrl.value) {
    ElMessage.warning("请先保存商品后再预览 H5");
    return;
  }
  drawerVisible.value = true;
}

/** 在新窗口打开商城 H5 页面，便于对照真实环境查看。 */
function handleOpenInNewWindow() {
  if (!previewUrl.value) {
    ElMessage.warning("请先保存商品后再预览 H5");
    return;
  }
  window.open(previewUrl.value, "_blank", "noopener,noreferrer");
}
</script>

<style scoped lang="scss">
.goods-h5-preview__toolbar {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 16px;
}

.goods-h5-preview__actions {
  display: flex;
  justify-content: flex-end;
}

.goods-h5-preview__phone-shell {
  position: relative;
  padding: 18px 14px 14px;
  overflow: hidden;
  border: 1px solid var(--admin-page-card-border);
  border-radius: 28px;
  background: linear-gradient(180deg, var(--admin-page-card-bg) 0%, var(--admin-page-card-bg-soft) 100%);
  box-shadow: var(--admin-page-shadow);
}

.goods-h5-preview__phone-head {
  display: flex;
  justify-content: center;
  margin-bottom: 12px;
}

.goods-h5-preview__camera {
  width: 78px;
  height: 8px;
  border-radius: 999px;
  background: var(--admin-page-card-border-muted);
}

.goods-h5-preview__frame {
  width: 100%;
  height: 720px;
  overflow: hidden;
  background: #fff;
  border: none;
  border-radius: 18px;
}

.goods-h5-preview__tip {
  margin: 12px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--admin-page-text-secondary);
}

@media (width <= 768px) {
  .goods-h5-preview__frame {
    height: 560px;
  }
}
</style>
