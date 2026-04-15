<script setup lang="ts">
import type {
  SkuPopupEvent,
  SkuPopupInstance,
  SkuPopupLocalData,
} from '@/components/vk-data-goods-sku-popup/vk-data-goods-sku-popup'
import { defUserCartService } from '@/api/app/user_cart'
import { defUserCollectService } from '@/api/app/user_collect'
import { defGoodsInfoService } from '@/api/app/goods_info.ts'
import { defRecommendService } from '@/api/app/recommend'
import type { GoodsInfoResponse } from '@/rpc/app/goods_info'
import type { RecommendContext } from '@/rpc/app/recommend'
import { onLoad } from '@dcloudio/uni-app'
import { useGuessList } from '@/composables'
import { useRecommendStore, useUserStore } from '@/stores'
import { computed, getCurrentInstance, nextTick, onBeforeUnmount, ref } from 'vue'
import AddressPanel from './components/AddressPanel.vue'
import ServicePanel from './components/ServicePanel.vue'
import { formatSrc, formatPrice } from '@/utils'
import { defShopServiceService } from '@/api/app/shop_service.ts'
import type { ShopService } from '@/rpc/app/shop_service.ts'
import { RecommendGoodsActionType, RecommendScene } from '@/rpc/common/enum'
import {
  goodsDetailUrl,
  homeTabPage,
  navigateToLogin,
  navigateToOrderCreate,
} from '@/utils/navigation'
// 获取会员信息
const userStore = useUserStore()
const recommendStore = useRecommendStore()
const pageInstance = getCurrentInstance()
// 获取屏幕边界到安全区域距离
const { safeAreaInsets } = uni.getSystemInfoSync()
const topBarHeight = (safeAreaInsets?.top || 0) + 44
const floatingHeaderHeight = topBarHeight
const navFadeStart = 24
const navFadeDistance = 150
const sectionNavFadeStart = 88
const sectionNavFadeDistance = 110
// 分段切换以悬浮导航栏下沿作为“到顶”基准，保留极小容差避免浮点误差抖动。
const sectionActiveTolerance = 2

type SectionKey = 'goods' | 'detail' | 'recommend'

type SectionTab = {
  key: SectionKey
  label: string
  anchorId: string
}

const sectionTabs: SectionTab[] = [
  { key: 'goods', label: '商品', anchorId: 'goods-section' },
  { key: 'detail', label: '详情', anchorId: 'detail-section' },
  { key: 'recommend', label: '推荐', anchorId: 'recommend-section' },
]

// 接收页面参数
const query = defineProps<{
  id: string
  scene?: string
  requestId?: string
  index?: string
}>()
const goodsId = Number(query.id)
const routeScene = query.scene ? (Number(query.scene) as RecommendScene) : undefined
// 当前页直接复用入口路由上的原始推荐参数，不在这里做额外转换。
const recommendContext = {
  scene: routeScene,
  requestId: query.requestId,
  position: query.index,
} as unknown as RecommendContext

// 获取商品详情信息
const goodsInfo = ref<GoodsInfoResponse>()
const isCollect = ref<boolean>(false)
const cartNum = ref<number>(0)
const serviceList = ref<ShopService[]>([])
const serviceLabelList = computed(() => serviceList.value.map((item) => item.label))
const { guessRef, onScrollToLower } = useGuessList()
const scrollTop = ref(0)
const scrollIntoView = ref('')
const activeSection = ref<SectionKey>('goods')
const sectionOffsetMap = ref<Record<SectionKey, number>>({
  goods: 0,
  detail: 0,
  recommend: 0,
})
let measureTimer: ReturnType<typeof setTimeout> | undefined

const clampProgress = (value: number) => {
  return Math.min(1, Math.max(0, value))
}

const headerProgress = computed(() => {
  return clampProgress((scrollTop.value - navFadeStart) / navFadeDistance)
})

const sectionNavProgress = computed(() => {
  return clampProgress((scrollTop.value - sectionNavFadeStart) / sectionNavFadeDistance)
})

const headerStyle = computed(() => {
  const progress = headerProgress.value
  return `background-color: rgba(255, 255, 255, ${progress}); box-shadow: 0 6rpx 18rpx rgba(15, 23, 42, ${0.05 * progress});`
})

const topBarContentStyle = computed(() => {
  const progress = headerProgress.value
  return `opacity: ${progress}; transform: translateY(${(1 - progress) * -8}px);`
})

const sectionNavStyle = computed(() => {
  const progress = sectionNavProgress.value
  return `opacity: ${progress}; transform: translateY(${(1 - progress) * -10}px); pointer-events: ${progress > 0.2 ? 'auto' : 'none'};`
})

const backButtonStyle = computed(() => {
  const progress = headerProgress.value
  return `color: ${progress > 0.55 ? '#111827' : '#ffffff'}; background-color: rgba(17, 24, 39, ${0.38 * (1 - progress)}); border-color: rgba(255, 255, 255, ${0.18 * (1 - progress)});`
})

const loadData = async () => {
  const ssRes = await defShopServiceService.ListShopService({})
  serviceList.value = ssRes.list || []
  const res = await defGoodsInfoService.GetGoodsInfo({
    value: goodsId,
  })
  goodsInfo.value = res
  // SKU组件所需格式
  localData.value = {
    _id: res.id,
    name: res.name,
    goods_thumb: res.picture,
    spec_list: res.specList.map((v) => {
      return {
        name: v.name,
        list: v.item,
      }
    }),
    sku_list: res.skuList.map((v) => {
      return {
        _id: v.skuCode,
        goods_id: res.id,
        goods_name: res.name,
        image: v.picture,
        price: v.price, // 注意：需要乘以 100
        stock: v.inventory,
        sku_name_arr: v.specItem,
      }
    }),
  }
  scheduleMeasureSections()
}

// 页面加载
onLoad(() => {
  loadData()
  void (async () => {
    try {
      await recommendStore.getAnonymousId()
      await defRecommendService.RecommendGoodsActionReport({
        eventType: RecommendGoodsActionType.VIEW,
        goodsItems: [
          {
            goodsId,
            goodsNum: 1,
            recommendContext,
          },
        ],
      })
    } catch (error) {
      console.error(error)
    }
  })()
  if (userStore.userInfo) {
    defUserCartService.CountUserCart({}).then((res) => {
      cartNum.value = res.value
    })
    defUserCollectService
      .GetIsCollect({
        goodsId,
      })
      .then((res) => {
        isCollect.value = res.value
      })
  }
})

// 轮播图变化时
const currentIndex = ref(0)
const onChange: UniHelper.SwiperOnChange = (ev) => {
  currentIndex.value = ev.detail.current
}

// 点击图片时
const onTapImage = (url: string) => {
  // 大图预览
  const urls: string[] = []
  goodsInfo.value!.banner.map((item) => {
    urls.push(formatSrc(item))
  })
  uni.previewImage({
    current: formatSrc(url),
    urls: urls,
  })
}

// uni-ui 弹出层组件 ref
const popup = ref<{
  open: (type?: UniHelper.UniPopupType) => void
  close: () => void
}>()

// 弹出层条件渲染
const popupName = ref<'address' | 'service'>()
const openPopup = (name: typeof popupName.value) => {
  // 修改弹出层名称
  popupName.value = name
  popup.value?.open()
}
// 是否显示SKU组件
const isShowSku = ref(false)
// 商品信息
const localData = ref({} as SkuPopupLocalData)
// 按钮模式
enum SkuMode {
  Both = 1,
  Cart = 2,
  Buy = 3,
}
const mode = ref<SkuMode>(SkuMode.Cart)
// 打开SKU弹窗修改按钮模式
const openSkuPopup = (val: SkuMode) => {
  // 显示SKU弹窗
  isShowSku.value = true
  // 修改按钮模式
  mode.value = val
}
// SKU组件实例
const skuPopupRef = ref<SkuPopupInstance>()
// 计算被选中的值
const selectArrText = computed(() => {
  return skuPopupRef.value?.selectArr?.join(' ').trim() || '请选择商品规格'
})

// 加入购物车事件
const onAddCart = async (ev: SkuPopupEvent) => {
  if (!userStore.userInfo) {
    navigateToLogin()
    return
  }
  await defUserCartService.CreateUserCart({
    /** 商品id */
    goodsId: ev.goods_id,
    /** 规格id */
    skuCode: ev._id,
    /** 数量 */
    num: ev.buy_num,
    recommendContext,
  })
  const res = await defUserCartService.CountUserCart({})
  cartNum.value = res.value
  await uni.showToast({ title: '添加成功' })
  isShowSku.value = false
}
// 立即购买
const onBuyNow = (ev: SkuPopupEvent) => {
  if (!userStore.userInfo) {
    navigateToLogin()
    return
  }
  isShowSku.value = false
  void navigateToOrderCreate({
    goodsId: ev.goods_id,
    skuCode: ev._id,
    num: ev.buy_num,
    scene: routeScene,
    requestId: query.requestId,
    index: query.index,
  })
}
// 收藏
const onCollect = async () => {
  if (!userStore.userInfo) {
    navigateToLogin()
    return
  }
  await defUserCollectService.CreateUserCollect({
    goodsId: goodsId,
    recommendContext,
  })
  isCollect.value = !isCollect.value
  await uni.showToast({ title: isCollect.value ? '收藏成功' : '取消成功' })
}

// 定义分享配置
const shareConfig = computed(() => {
  if (!goodsInfo.value) return {}
  return {
    title: `${goodsInfo.value.name} ¥${formatPrice(goodsInfo.value.price)}`,
    path: goodsDetailUrl(goodsInfo.value.id),
    imageUrl: formatSrc(goodsInfo.value.picture),
  }
})

// 分享给朋友
const onShareAppMessage = () => {
  return shareConfig.value
}

// 分享到朋友圈
const onShareTimeline = () => {
  return shareConfig.value
}

// 延迟重新测量分段位置，避免图片尚未渲染完成时拿到错误偏移。
const scheduleMeasureSections = () => {
  if (measureTimer) {
    clearTimeout(measureTimer)
  }
  measureTimer = setTimeout(() => {
    void measureSections()
  }, 80)
}

// 读取三个分段在滚动容器中的偏移位置，供吸顶导航高亮和点击跳转使用。
const measureSections = async () => {
  await nextTick()
  if (!pageInstance) {
    return
  }

  await new Promise<void>((resolve) => {
    const query = uni.createSelectorQuery().in(pageInstance)
    query.select('.viewport').boundingClientRect()
    sectionTabs.forEach((item) => {
      query.select(`#${item.anchorId}`).boundingClientRect()
    })
    query.exec((result) => {
      const viewportRect = result?.[0] as { top: number } | undefined
      if (!viewportRect) {
        resolve()
        return
      }

      const nextOffsetMap = { ...sectionOffsetMap.value }
      sectionTabs.forEach((item, index) => {
        const rect = result?.[index + 1] as { top: number } | undefined
        if (!rect) {
          return
        }
        // 当前元素相对滚动容器顶部的位置，加上实时滚动值后得到内容区绝对偏移。
        nextOffsetMap[item.key] = Math.max(0, rect.top - viewportRect.top + scrollTop.value)
      })
      sectionOffsetMap.value = nextOffsetMap
      updateActiveSection(scrollTop.value)
      resolve()
    })
  })
}

// 根据滚动位置切换当前分段。
// 这里不做“提前高亮”，只有当对应分段到达导航栏下沿附近时才切换选中态。
const updateActiveSection = (currentScrollTop: number) => {
  const scrollThreshold = currentScrollTop + floatingHeaderHeight + sectionActiveTolerance
  const recommendMeasured = sectionOffsetMap.value.recommend > sectionOffsetMap.value.detail

  if (recommendMeasured && scrollThreshold >= sectionOffsetMap.value.recommend) {
    activeSection.value = 'recommend'
    return
  }
  // 详情分段同样只在锚点到达顶部时高亮。
  if (scrollThreshold >= sectionOffsetMap.value.detail) {
    activeSection.value = 'detail'
    return
  }
  activeSection.value = 'goods'
}

// 点击导航时直接滚动到目标锚点。
// 锚点本身通过负边距预留了头部高度，所以滚动结束后内容会自然停在导航栏下沿。
const onTapSection = async (tab: SectionTab) => {
  activeSection.value = tab.key
  await measureSections()
  scrollIntoView.value = ''
  nextTick(() => {
    scrollIntoView.value = tab.anchorId
  })
}

// 滚动时同步吸顶导航显隐和高亮状态。
const onScrollPage = (ev: { detail: { scrollTop: number } }) => {
  scrollTop.value = ev.detail.scrollTop
  updateActiveSection(scrollTop.value)
}

// 吸顶导航显示后，左侧返回按钮仍然保留，避免用户在中段内容里失去回退入口。
const onNavigateBack = () => {
  uni.navigateBack({
    fail: () => {
      uni.switchTab({ url: homeTabPage })
    },
  })
}

// 商品详情富文本图片会影响推荐分段位置，图片加载完成后重新测量。
const onDetailImageLoad = () => {
  scheduleMeasureSections()
}

onBeforeUnmount(() => {
  if (measureTimer) {
    clearTimeout(measureTimer)
  }
})
</script>

<template>
  <!-- SKU弹窗组件 -->
  <vk-data-goods-sku-popup
    ref="skuPopupRef"
    v-model="isShowSku"
    :localData="localData"
    :mode="mode"
    add-cart-background-color="#FFA868"
    buy-now-background-color="#27BA9B"
    :actived-style="{
      color: '#27BA9B',
      borderColor: '#27BA9B',
      backgroundColor: '#E9F8F5',
    }"
    @add-cart="onAddCart"
    @buy-now="onBuyNow"
  />
  <view v-if="goodsInfo" class="header" :style="headerStyle">
    <view class="top-bar" :style="{ paddingTop: `${safeAreaInsets?.top || 0}px` }">
      <view class="top-bar-side top-bar-side--back" :style="backButtonStyle" @tap="onNavigateBack">
        <text class="top-bar-back">‹</text>
      </view>
      <view class="top-bar-content" :style="topBarContentStyle">
        <view class="section-nav section-nav--inline" :style="sectionNavStyle">
          <view
            v-for="tab in sectionTabs"
            :key="tab.key"
            class="section-nav-item"
            :class="{ active: activeSection === tab.key }"
            @tap="onTapSection(tab)"
          >
            {{ tab.label }}
          </view>
        </view>
      </view>
      <view class="top-bar-side top-bar-side--placeholder"></view>
    </view>
  </view>

  <scroll-view
    v-if="goodsInfo"
    enable-back-to-top
    scroll-y
    class="viewport"
    :scroll-into-view="scrollIntoView"
    scroll-with-animation
    @scroll="onScrollPage"
    @scrolltolower="onScrollToLower"
  >
    <view id="goods-section" class="section-anchor" />
    <!-- 基本信息 -->
    <view class="goods">
      <!-- 商品主图 -->
      <view class="preview">
        <swiper circular @change="onChange">
          <swiper-item v-for="item in goodsInfo!.banner" :key="item">
            <image class="image" mode="aspectFill" :src="formatSrc(item)" @tap="onTapImage(item)" />
          </swiper-item>
        </swiper>
        <view class="indicator">
          <text class="current">{{ currentIndex + 1 }}</text>
          <text class="split">/</text>
          <text class="total">{{ goodsInfo!.banner.length }}</text>
        </view>
      </view>

      <!-- 商品简介 -->
      <view class="meta">
        <view class="price">
          <text class="symbol">¥</text>
          <text class="number">{{ formatPrice(goodsInfo!.price) }}</text>
        </view>
        <view class="name ellipsis">{{ goodsInfo!.name }}</view>
        <view class="desc"> {{ goodsInfo!.desc }} </view>
      </view>

      <!-- 操作面板 -->
      <view class="action">
        <view class="item arrow" @tap="openSkuPopup(SkuMode.Both)">
          <text class="label">选择</text>
          <text class="text ellipsis"> {{ selectArrText }} </text>
        </view>
        <view class="item arrow" @tap="openPopup('address')">
          <text class="label">送至</text>
          <text class="text ellipsis"> 请选择收获地址 </text>
        </view>
        <view class="item arrow" @tap="openPopup('service')">
          <text class="label">服务</text>
          <text class="text ellipsis"> {{ serviceLabelList.join(' ') }} </text>
        </view>
      </view>
    </view>

    <view id="detail-section" class="section-anchor" />
    <!-- 商品详情 -->
    <view class="detail panel">
      <view class="title">
        <text>详情</text>
      </view>
      <view class="content">
        <view class="properties">
          <!-- 属性详情 -->
          <view v-for="item in goodsInfo!.propList" :key="item.label" class="item">
            <text class="label">{{ item.label }}</text>
            <text class="value">{{ item.value }}</text>
          </view>
        </view>
        <!-- 图片详情 -->
        <image
          v-for="item in goodsInfo!.detail"
          :key="item"
          class="image"
          mode="widthFix"
          :src="formatSrc(item)"
          @load="onDetailImageLoad"
        />
      </view>
    </view>

    <view id="recommend-section" class="section-anchor" />
    <!-- 商品详情推荐 -->
    <XtxGuess
      ref="guessRef"
      title="看了又看"
      :scene="RecommendScene.GOODS_DETAIL"
      :goods-id="goodsId"
    />
  </scroll-view>

  <!-- 用户操作 -->
  <view v-if="goodsInfo" class="toolbar" :style="{ paddingBottom: safeAreaInsets?.bottom + 'px' }">
    <view class="icons">
      <button class="icons-button" @tap="onCollect()">
        <text class="icon-heart" :class="{ active: isCollect }" />{{
          isCollect ? '已收藏' : '收藏'
        }}
      </button>
      <!-- #ifdef MP-WEIXIN -->
      <button class="icons-button" open-type="contact"><text class="icon-handset" />客服</button>
      <!-- #endif -->
      <navigator class="icons-button" url="/pages/cart/cart2" open-type="navigate">
        <text class="icon-cart" />购物车
        <view v-if="cartNum! > 0" class="cart-badge">{{ cartNum > 99 ? '99+' : cartNum }}</view>
      </navigator>
    </view>
    <view class="buttons">
      <view class="addcart" @tap="openSkuPopup(SkuMode.Cart)"> 加入购物车 </view>
      <view class="payment" @tap="openSkuPopup(SkuMode.Buy)"> 立即购买 </view>
    </view>
  </view>

  <!-- uni-ui 弹出层 -->
  <uni-popup ref="popup" type="bottom" background-color="#fff">
    <AddressPanel v-if="popupName === 'address'" @close="popup?.close()" />
    <ServicePanel v-if="popupName === 'service'" :list="serviceList" @close="popup?.close()" />
  </uni-popup>
</template>

<style lang="scss">
page {
  height: 100%;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.viewport {
  flex: 1;
  background-color: #f4f4f4;
}

.header {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 20;
  transition:
    background-color 0.2s ease,
    box-shadow 0.2s ease;
}

.top-bar {
  height: v-bind('`${topBarHeight}px`');
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-left: 24rpx;
  padding-right: 24rpx;
  box-sizing: border-box;
}

.top-bar-side {
  width: 72rpx;
  height: 72rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.top-bar-side--back {
  border: 1rpx solid transparent;
  border-radius: 999rpx;
  transition:
    color 0.2s ease,
    background-color 0.2s ease,
    border-color 0.2s ease;
}

.top-bar-side--placeholder {
  opacity: 0;
}

.top-bar-content {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  transition:
    opacity 0.2s ease,
    transform 0.2s ease;
}

.top-bar-back {
  margin-top: -4rpx;
  font-size: 48rpx;
  line-height: 1;
}

.section-nav {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 12rpx;
  box-sizing: border-box;
  transition:
    opacity 0.2s ease,
    transform 0.2s ease;
}

.section-nav--inline {
  height: 72rpx;
}

.section-nav-item {
  position: relative;
  padding: 0 34rpx;
  height: 100%;
  display: flex;
  align-items: center;
  font-size: 28rpx;
  color: #7a7f87;
  transition: color 0.2s ease;

  &.active {
    color: #1f2937;
    font-weight: 600;
  }

  &.active::after {
    content: '';
    position: absolute;
    left: 50%;
    bottom: 6rpx;
    width: 44rpx;
    height: 5rpx;
    border-radius: 999rpx;
    background: #27ba9b;
    transform: translateX(-50%);
  }
}

.section-anchor {
  // 通过“正高度 + 负外边距”给 scroll-into-view 预留悬浮头部空间。
  height: v-bind('`${floatingHeaderHeight}px`');
  margin-top: v-bind('`-${floatingHeaderHeight}px`');
  pointer-events: none;
}

.panel {
  margin-top: 20rpx;
  background-color: #fff;
  .title {
    display: flex;
    justify-content: space-between;
    align-items: center;
    height: 90rpx;
    line-height: 1;
    padding: 30rpx 60rpx 30rpx 6rpx;
    position: relative;
    text {
      padding-left: 10rpx;
      font-size: 28rpx;
      color: #333;
      font-weight: 600;
      border-left: 4rpx solid #27ba9b;
    }
    navigator {
      font-size: 24rpx;
      color: #666;
    }
  }
}

.arrow {
  &::after {
    position: absolute;
    top: 50%;
    right: 30rpx;
    content: '\e6c2';
    color: #ccc;
    font-family: 'erabbit' !important;
    font-size: 32rpx;
    transform: translateY(-50%);
  }
}

/* 商品信息 */
.goods {
  background-color: #fff;
  .preview {
    height: 750rpx;
    position: relative;
    .image {
      width: 750rpx;
      height: 750rpx;
    }
    .indicator {
      height: 40rpx;
      padding: 0 24rpx;
      line-height: 40rpx;
      border-radius: 30rpx;
      color: #fff;
      font-family: Arial, Helvetica, sans-serif;
      background-color: rgba(0, 0, 0, 0.3);
      position: absolute;
      bottom: 30rpx;
      right: 30rpx;
      .current {
        font-size: 26rpx;
      }
      .split {
        font-size: 24rpx;
        margin: 0 1rpx 0 2rpx;
      }
      .total {
        font-size: 24rpx;
      }
    }
  }
  .meta {
    position: relative;
    border-bottom: 1rpx solid #eaeaea;
    .price {
      height: 130rpx;
      padding: 25rpx 30rpx 0;
      color: #fff;
      font-size: 34rpx;
      box-sizing: border-box;
      background-color: #35c8a9;
    }
    .number {
      font-size: 56rpx;
    }
    .brand {
      width: 160rpx;
      height: 80rpx;
      overflow: hidden;
      position: absolute;
      top: 26rpx;
      right: 30rpx;
    }
    .name {
      max-height: 88rpx;
      line-height: 1.4;
      margin: 20rpx;
      font-size: 32rpx;
      color: #333;
    }
    .desc {
      line-height: 1;
      padding: 0 20rpx 30rpx;
      font-size: 24rpx;
      color: #cf4444;
    }
  }
  .action {
    padding-left: 20rpx;
    .item {
      height: 90rpx;
      padding-right: 60rpx;
      border-bottom: 1rpx solid #eaeaea;
      font-size: 26rpx;
      color: #333;
      position: relative;
      display: flex;
      align-items: center;
      &:last-child {
        border-bottom: 0 none;
      }
    }
    .label {
      width: 60rpx;
      color: #898b94;
      margin: 0 16rpx 0 10rpx;
    }
    .text {
      flex: 1;
      -webkit-line-clamp: 1;
    }
  }
}

/* 商品详情 */
.detail {
  padding-left: 20rpx;
  .content {
    margin-left: -20rpx;
    .image {
      width: 100%;
    }
  }
  .properties {
    padding: 0 20rpx;
    margin-bottom: 30rpx;
    .item {
      display: flex;
      line-height: 2;
      padding: 10rpx;
      font-size: 26rpx;
      color: #333;
      border-bottom: 1rpx dashed #ccc;
    }
    .label {
      width: 200rpx;
    }
    .value {
      flex: 1;
    }
  }
}

/* 同类推荐 */
.similar {
  .content {
    padding: 0 20rpx 20rpx;
    background-color: #f4f4f4;
    display: flex;
    flex-wrap: wrap;
    .goods {
      width: 340rpx;
      padding: 24rpx 20rpx 20rpx;
      margin: 20rpx 7rpx;
      border-radius: 10rpx;
      background-color: #fff;
    }
    .image {
      width: 300rpx;
      height: 260rpx;
    }
    .name {
      height: 80rpx;
      margin: 10rpx 0;
      font-size: 26rpx;
      color: #262626;
    }
    .price {
      line-height: 1;
      font-size: 20rpx;
      color: #cf4444;
    }
    .number {
      font-size: 26rpx;
      margin-left: 2rpx;
    }
  }
  navigator {
    &:nth-child(even) {
      margin-right: 0;
    }
  }
}

/* 底部工具栏 */
.toolbar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: calc((var(--window-bottom)));
  z-index: 1;
  background-color: #fff;
  height: 100rpx;
  padding: 0 20rpx;
  border-top: 1rpx solid #eaeaea;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-sizing: content-box;
  .buttons {
    display: flex;
    & > view {
      width: 220rpx;
      text-align: center;
      line-height: 72rpx;
      font-size: 26rpx;
      color: #fff;
      border-radius: 72rpx;
    }
    .addcart {
      background-color: #ffa868;
    }
    .payment {
      background-color: #27ba9b;
      margin-left: 20rpx;
    }
  }
  .icons {
    padding-right: 20rpx;
    display: flex;
    align-items: center;
    flex: 1;
    // 兼容 H5 端和 App 端的导航链接样式
    .navigator-wrap,
    .icons-button {
      flex: 1;
      text-align: center;
      line-height: 1.4;
      padding: 0;
      margin: 0;
      border-radius: 0;
      font-size: 20rpx;
      color: #333;
      background-color: #fff;
      &::after {
        border: none;
      }
    }
    text {
      display: block;
      font-size: 34rpx;
      transition: color 0.3s ease;
    }
    // 收藏按钮文字颜色变化
    &.active {
      color: #ff0000;
    }
  }
}

// 新增收藏激活样式
.icon-heart {
  position: relative;
  &::before {
    transition: color 0.3s ease;
  }
  &.active::before {
    color: #ff0000 !important;
  }
}

// 购物车角标样式
.cart-badge {
  position: absolute;
  top: -5rpx;
  right: -5rpx;
  min-width: 36rpx;
  height: 36rpx;
  line-height: 36rpx;
  text-align: center;
  background-color: #ff4444;
  color: #fff;
  border-radius: 100rpx;
  font-size: 20rpx;
  padding: 0 8rpx;
  transform: scale(0.8);
  box-shadow: 0 2rpx 8rpx rgba(255, 68, 68, 0.2);
}

// 确保按钮容器有相对定位
.icons-button {
  position: relative;
}

// 新增收藏动画
@keyframes heartBeat {
  0% {
    transform: scale(1);
  }
  50% {
    transform: scale(1.2);
  }
  100% {
    transform: scale(1);
  }
}

.icon-heart.active::before {
  animation: heartBeat 0.3s ease;
}
</style>
