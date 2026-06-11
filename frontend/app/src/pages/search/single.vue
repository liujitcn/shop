<script setup lang="ts">
import type {
  SkuPopupEvent,
  SkuPopupLocalData,
} from '@/components/vk-data-goods-sku-popup/vk-data-goods-sku-popup'
import { defGoodsInfoService } from '@/api/app/goods_info'
import { defUserCartService } from '@/api/app/user_cart'
import { defUserCollectService } from '@/api/app/user_collect'
import { useUserStore } from '@/stores'
import { formatPrice, formatSrc } from '@/utils'
import {
  goodsDetailUrl,
  homeTabPage,
  navigateToLogin,
  navigateToOrderCreate,
} from '@/utils/navigation'
import type { GoodsInfo, GoodsInfoResponse, PageGoodsInfoRequest } from '@/rpc/app/v1/goods_info'
import { RecommendScene } from '@/rpc/common/v1/enum'
import { onLoad } from '@dcloudio/uni-app'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

type WheelLikeEvent = {
  deltaY?: number
  preventDefault?: () => void
}

type GoodsInfoExtra = GoodsInfo & {
  banner?: string[]
  init_sale_num?: number
  real_sale_num?: number
  initSaleNum?: number
  realSaleNum?: number
}

enum SkuMode {
  Cart = 2,
  Buy = 3,
}

const query = defineProps<{
  name?: string
  category_id?: string
  categoryName?: string
}>()

const userStore = useUserStore()
const { safeAreaInsets, windowHeight } = uni.getSystemInfoSync()

const decodeQueryText = (value?: string) => {
  if (!value) return ''
  let result = value
  for (let i = 0; i < 2; i++) {
    try {
      const decoded = decodeURIComponent(result)
      if (decoded === result) break
      result = decoded
    } catch {
      break
    }
  }
  return result
}

const resolvePageHeight = () => {
  let height = windowHeight || 667
  // #ifdef H5
  if (typeof window !== 'undefined') {
    height = window.innerHeight || height
  }
  // #endif
  return Math.max(height, 600)
}

const pageHeight = resolvePageHeight()
const previewHeight = Math.min(Math.max(Math.round(pageHeight * 0.46), 330), 400)
const cardGap = 12
const estimatedCardHeight = previewHeight + 180
const adjacentMargin = Math.max(24, Math.round((pageHeight - estimatedCardHeight - cardGap) / 2))
const pagerPreviousMargin = adjacentMargin
const pagerNextMargin = adjacentMargin
const decodedName = decodeQueryText(query.name)
const decodedCategoryName = decodeQueryText(query.categoryName)

const pageParams: Required<PageGoodsInfoRequest> = {
  name: decodedName,
  category_id: query.category_id ? Number(query.category_id) : 0,
  page_num: 1,
  page_size: 8,
}

const goodsInfoList = ref<GoodsInfo[]>([])
const finish = ref(false)
const loading = ref(false)
const activeIndex = ref(0)
const cartNum = ref(0)
const buyingGoodsId = ref(0)
const isShowSku = ref(false)
const skuMode = ref<SkuMode>(SkuMode.Cart)
const localData = ref({} as SkuPopupLocalData)
const collectMap = ref<Record<number, boolean>>({})
const pictureIndexMap = ref<Record<number, number>>({})
const detailInfoMap = ref<Record<number, GoodsInfoResponse>>({})
const detailCache = new Map<number, GoodsInfoResponse>()
let wheelTimer: ReturnType<typeof setTimeout> | undefined

const activeGoods = computed(() => goodsInfoList.value[activeIndex.value])
const emptyText = computed(() => {
  if (decodedName) return `暂无“${decodedName}”相关商品`
  if (decodedCategoryName) return `暂无${decodedCategoryName}商品`
  return '暂无可购买商品'
})

const trimZeroDecimal = (value: string) => value.replace(/\.0$/, '')

const resolveSaleNum = (item: GoodsInfo) => {
  const goods = item as GoodsInfoExtra
  if (
    goods.init_sale_num !== undefined ||
    goods.real_sale_num !== undefined ||
    goods.initSaleNum !== undefined ||
    goods.realSaleNum !== undefined
  ) {
    return (
      Number(goods.init_sale_num ?? goods.initSaleNum ?? 0) +
      Number(goods.real_sale_num ?? goods.realSaleNum ?? 0)
    )
  }
  return item.sale_num || 0
}

const formatSaleText = (item: GoodsInfo) => {
  const saleNum = resolveSaleNum(item)
  if (!saleNum) return '销量 0'
  if (saleNum >= 10000) return `销量 ${trimZeroDecimal((saleNum / 10000).toFixed(1))}万+`
  return `销量 ${saleNum}`
}

const getPictureList = (item: GoodsInfo) => {
  const goods = item as GoodsInfoExtra
  const list = detailInfoMap.value[item.id]?.banner?.length
    ? detailInfoMap.value[item.id].banner
    : goods.banner?.length
      ? goods.banner
      : [item.picture]
  return list.filter(Boolean)
}

const getActivePictureIndex = (item: GoodsInfo) => {
  return Math.min(pictureIndexMap.value[item.id] || 0, Math.max(getPictureList(item).length - 1, 0))
}

const buildRecommendContext = (position = activeIndex.value) => {
  return {
    scene: RecommendScene.UNKNOWN_RS,
    request_id: 0,
    position,
  }
}

const buildSkuLocalData = (goods: GoodsInfoResponse): SkuPopupLocalData => {
  return {
    _id: goods.id,
    name: goods.name,
    goods_thumb: goods.picture,
    spec_list: goods.spec_list.map((item) => ({
      name: item.name,
      list: item.item,
    })),
    sku_list: goods.sku_list.map((item) => ({
      _id: item.sku_code,
      goods_id: goods.id,
      goods_name: goods.name,
      image: item.picture,
      price: item.price,
      stock: item.inventory,
      sku_name_arr: item.spec_item,
    })),
  }
}

const getGoodsData = async () => {
  if (loading.value || finish.value) return
  loading.value = true
  try {
    const res = await defGoodsInfoService.PageGoodsInfo(pageParams)
    goodsInfoList.value.push(...(res.goods_infos || []))
    if (goodsInfoList.value.length < res.total) {
      pageParams.page_num++
    } else {
      finish.value = true
    }
  } finally {
    loading.value = false
  }
}

const refreshUserGoodsState = async (item: GoodsInfo | undefined) => {
  if (!item || !userStore.isAuthenticated()) return
  try {
    const [cartRes, collectRes] = await Promise.all([
      defUserCartService.CountUserCart({}),
      defUserCollectService.GetIsCollect({ goods_id: item.id }),
    ])
    cartNum.value = cartRes.count
    collectMap.value = {
      ...collectMap.value,
      [item.id]: collectRes.is_collected,
    }
  } catch (error) {
    console.error(error)
  }
}

const preloadNearEnd = async () => {
  if (activeIndex.value >= goodsInfoList.value.length - 2) {
    await getGoodsData()
  }
}

const ensureGoodsDetail = async (item: GoodsInfo | undefined) => {
  if (!item) return undefined
  let goodsDetail = detailCache.get(item.id)
  if (!goodsDetail) {
    goodsDetail = await defGoodsInfoService.GetGoodsInfo({ id: item.id })
    detailCache.set(item.id, goodsDetail)
  }
  detailInfoMap.value = {
    ...detailInfoMap.value,
    [item.id]: goodsDetail,
  }
  return goodsDetail
}

const loadActiveGoodsDetail = async () => {
  try {
    await ensureGoodsDetail(activeGoods.value)
  } catch (error) {
    console.error(error)
  }
}

const onSwiperChange: UniHelper.SwiperOnChange = (event) => {
  activeIndex.value = event.detail.current
  void refreshUserGoodsState(activeGoods.value)
  void loadActiveGoodsDetail()
  void preloadNearEnd()
}

const onPictureChange = (item: GoodsInfo, event: UniHelper.SwiperOnChangeEvent) => {
  pictureIndexMap.value = {
    ...pictureIndexMap.value,
    [item.id]: event.detail.current,
  }
}

const onWheelFeed = (event: WheelLikeEvent) => {
  event.preventDefault?.()
  if (wheelTimer || goodsInfoList.value.length <= 1 || Math.abs(event.deltaY || 0) < 10) return

  const nextIndex = activeIndex.value + ((event.deltaY || 0) > 0 ? 1 : -1)
  if (nextIndex < 0 || nextIndex >= goodsInfoList.value.length) return

  activeIndex.value = nextIndex
  void refreshUserGoodsState(activeGoods.value)
  void loadActiveGoodsDetail()
  void preloadNearEnd()
  wheelTimer = setTimeout(() => {
    wheelTimer = undefined
  }, 420)
}

const onNativeWheelFeed = (event: Event) => {
  onWheelFeed(event as WheelLikeEvent)
}

const navigateToGoods = (item: GoodsInfo) => {
  uni.navigateTo({ url: goodsDetailUrl(item.id) })
}

const onNavigateBack = () => {
  const pages = getCurrentPages()
  if (pages.length > 1) {
    uni.navigateBack({
      fail() {
        void uni.switchTab({ url: homeTabPage })
      },
    })
    return
  }
  void uni.switchTab({ url: homeTabPage })
}

const previewImage = (item: GoodsInfo) => {
  const pictureList = getPictureList(item).map((url) => formatSrc(url))
  const url = pictureList[getActivePictureIndex(item)] || formatSrc(item.picture)
  uni.previewImage({
    current: url,
    urls: pictureList.length ? pictureList : [url],
  })
}

const openSkuPopup = async (item: GoodsInfo | undefined, mode: SkuMode) => {
  if (!item) return
  buyingGoodsId.value = item.id
  try {
    const goodsDetail = await ensureGoodsDetail(item)
    if (!goodsDetail) {
      uni.showToast({ icon: 'none', title: '商品规格加载失败' })
      return
    }
    if (!goodsDetail.sku_list.length) {
      uni.showToast({ icon: 'none', title: '当前商品暂无可售规格' })
      return
    }
    localData.value = buildSkuLocalData(goodsDetail)
    skuMode.value = mode
    isShowSku.value = true
  } catch (error) {
    console.error(error)
    uni.showToast({ icon: 'none', title: '商品规格加载失败' })
  } finally {
    buyingGoodsId.value = 0
  }
}

const onCollect = async (item: GoodsInfo | undefined) => {
  if (!item) return
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin()
    return
  }
  await defUserCollectService.CreateUserCollect({
    goods_id: item.id,
    recommend_context: buildRecommendContext(activeIndex.value),
  })
  collectMap.value = {
    ...collectMap.value,
    [item.id]: !collectMap.value[item.id],
  }
  await uni.showToast({ title: collectMap.value[item.id] ? '收藏成功' : '取消成功' })
}

const onAddCart = async (event: SkuPopupEvent) => {
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin()
    return
  }
  await defUserCartService.CreateUserCart({
    goods_id: event.goods_id,
    sku_code: event._id,
    num: event.buy_num,
    recommend_context: buildRecommendContext(activeIndex.value),
  })
  const res = await defUserCartService.CountUserCart({})
  cartNum.value = res.count
  await uni.showToast({ title: '添加成功' })
  isShowSku.value = false
}

const onBuyNow = (event: SkuPopupEvent) => {
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin()
    return
  }
  isShowSku.value = false
  void navigateToOrderCreate({
    goods_id: event.goods_id,
    sku_code: event._id,
    num: event.buy_num,
  })
}

onLoad(async () => {
  await getGoodsData()
  await refreshUserGoodsState(activeGoods.value)
  void loadActiveGoodsDetail()
})

onMounted(() => {
  // #ifdef H5
  if (typeof window !== 'undefined') {
    window.addEventListener('wheel', onNativeWheelFeed, { passive: false })
  }
  // #endif
})

onBeforeUnmount(() => {
  // #ifdef H5
  if (typeof window !== 'undefined') {
    window.removeEventListener('wheel', onNativeWheelFeed)
  }
  // #endif
  if (wheelTimer) {
    clearTimeout(wheelTimer)
  }
})
</script>

<template>
  <view class="single-search-page" @wheel="onWheelFeed">
    <vk-data-goods-sku-popup
      v-model="isShowSku"
      :localData="localData"
      :mode="skuMode"
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

    <view class="header">
      <view
        class="top-bar-side top-bar-side--back"
        :style="{ top: `${(safeAreaInsets?.top || 0) + 12}px` }"
        @tap="onNavigateBack"
      >
        <text class="top-bar-back">‹</text>
      </view>
    </view>

    <swiper
      v-if="goodsInfoList.length"
      class="feed-swiper"
      vertical
      :circular="goodsInfoList.length > 1"
      :current="activeIndex"
      :duration="260"
      :style="{ height: `${pageHeight}px` }"
      :previous-margin="`${pagerPreviousMargin}px`"
      :next-margin="`${pagerNextMargin}px`"
      @change="onSwiperChange"
    >
      <swiper-item v-for="item in goodsInfoList" :key="item.id" class="goods-slide">
        <view class="goods-card">
          <view class="preview">
            <swiper
              class="picture-swiper"
              :circular="getPictureList(item).length > 1"
              @change="onPictureChange(item, $event)"
            >
              <swiper-item v-for="picture in getPictureList(item)" :key="picture">
                <image
                  class="image"
                  mode="aspectFill"
                  :src="formatSrc(picture)"
                  @tap="previewImage(item)"
                />
              </swiper-item>
            </swiper>
            <view class="indicator">
              <text class="current">{{ getActivePictureIndex(item) + 1 }}</text>
              <text class="split">/</text>
              <text class="total">{{ getPictureList(item).length || 1 }}</text>
            </view>
          </view>

          <view class="meta">
            <view class="price">
              <text class="symbol">¥</text>
              <text class="number">{{ formatPrice(item.price) }}</text>
              <text class="sales">{{ formatSaleText(item) }}</text>
            </view>
            <view class="name ellipsis" @tap="navigateToGoods(item)">{{ item.name }}</view>
            <view v-if="item.desc" class="desc">{{ item.desc }}</view>
          </view>

          <view class="toolbar-slot" :style="{ paddingBottom: `${safeAreaInsets?.bottom || 0}px` }">
            <view class="toolbar">
              <view class="icons">
                <button class="icons-button" @tap="onCollect(item)">
                  <text class="icon-heart" :class="{ active: collectMap[item.id] === true }" />{{
                    collectMap[item.id] === true ? '已收藏' : '收藏'
                  }}
                </button>
                <navigator class="icons-button" url="/pages/cart/cart2" open-type="navigate">
                  <text class="icon-cart" />购物车
                  <view v-if="cartNum > 0" class="cart-badge">{{
                    cartNum > 99 ? '99+' : cartNum
                  }}</view>
                </navigator>
              </view>
              <view class="buttons">
                <view class="addcart" @tap="openSkuPopup(item, SkuMode.Cart)">加入购物车</view>
                <view
                  class="payment"
                  :class="{ 'payment--loading': buyingGoodsId === item.id }"
                  @tap="openSkuPopup(item, SkuMode.Buy)"
                >
                  {{ buyingGoodsId === item.id ? '加载中' : '立即购买' }}
                </view>
              </view>
            </view>
          </view>
        </view>
      </swiper-item>
    </swiper>

    <XtxEmptyState
      v-else-if="finish && !loading"
      image="/static/images/empty_search.png"
      :text="emptyText"
      min-height="70vh"
    />
    <view v-else class="loading-state">正在生成单品直购流...</view>
  </view>
</template>

<style lang="scss">
page {
  height: 100%;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  background-color: #f4f4f4;
}

.single-search-page {
  position: relative;
  height: 100%;
  overflow: hidden;
  background-color: #f4f4f4;
}

.header {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 20;
  pointer-events: none;
}

.top-bar-side--back {
  position: absolute;
  left: 24rpx;
  width: 72rpx;
  height: 72rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1rpx solid rgba(255, 255, 255, 0.18);
  border-radius: 999rpx;
  color: #fff;
  background-color: rgba(17, 24, 39, 0.38);
  pointer-events: auto;
}

.top-bar-back {
  margin-top: -4rpx;
  font-size: 48rpx;
  line-height: 1;
}

.feed-swiper {
  background-color: #f4f4f4;
}

.goods-slide {
  display: flex;
  align-items: flex-start;
  justify-content: center;
  box-sizing: border-box;
}

.goods-card {
  width: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-radius: 0;
  border-bottom: 20rpx solid #f4f4f4;
  background-color: #fff;
}

.preview {
  height: v-bind('`${previewHeight}px`');
  position: relative;
  flex-shrink: 0;
  background-color: #f7f7f7;

  .picture-swiper {
    width: 100%;
    height: 100%;
  }

  .image {
    width: 100%;
    height: 100%;
  }

  .indicator {
    position: absolute;
    right: 30rpx;
    bottom: 30rpx;
    height: 56rpx;
    min-width: 86rpx;
    padding: 0 20rpx;
    border-radius: 32rpx 0 0 32rpx;
    color: #fff;
    font-family: Arial, Helvetica, sans-serif;
    line-height: 56rpx;
    text-align: center;
    background-color: rgba(0, 0, 0, 0.42);
  }

  .current {
    font-size: 34rpx;
  }

  .split {
    margin: 0 1rpx 0 2rpx;
    font-size: 30rpx;
  }

  .total {
    font-size: 30rpx;
  }
}

.meta {
  position: relative;
  flex-shrink: 0;
  border-bottom: 1rpx solid #eaeaea;

  .price {
    height: 104rpx;
    padding: 0 30rpx;
    position: relative;
    display: flex;
    align-items: center;
    color: #fff;
    font-size: 30rpx;
    box-sizing: border-box;
    background-color: #35c8a9;
  }

  .symbol {
  }

  .number {
    font-size: 48rpx;
  }

  .sales {
    position: absolute;
    top: 40rpx;
    right: 30rpx;
    color: rgba(255, 255, 255, 0.9);
    font-size: 22rpx;
  }

  .name {
    max-height: 88rpx;
    margin: 20rpx 20rpx 12rpx;
    color: #333;
    font-size: 32rpx;
    line-height: 1.4;
  }

  .desc {
    padding: 0 20rpx 24rpx;
    color: #cf4444;
    font-size: 24rpx;
    line-height: 1;
  }
}

.toolbar-slot {
  border-top: 1rpx solid #eaeaea;
  box-sizing: border-box;
  background-color: #fff;
}

.toolbar {
  position: static;
  left: auto;
  right: auto;
  bottom: auto;
  z-index: auto;
  height: 100rpx;
  padding: 0 20rpx;
  display: flex;
  align-items: center;
  justify-content: space-between;
  box-sizing: border-box;
  background-color: #fff;

  .icons {
    position: relative;
    flex: 1;
    display: flex;
    align-items: center;
    padding-right: 20rpx;
  }

  .icons-button {
    position: relative;
    flex: 1;
    margin: 0;
    padding: 0;
    border-radius: 0;
    color: #333;
    font-size: 20rpx;
    line-height: 1.4;
    text-align: center;
    background-color: #fff;

    &::after {
      border: none;
    }
  }

  .icons-button text {
    display: block;
    font-size: 34rpx;
    transition: color 0.3s ease;
  }

  .icon-heart.active::before {
    color: #ff0000 !important;
  }

  .buttons {
    display: flex;
  }

  .buttons > view {
    width: 220rpx;
    border-radius: 72rpx;
    color: #fff;
    font-size: 26rpx;
    line-height: 72rpx;
    text-align: center;
  }

  .addcart {
    background-color: #ffa868;
  }

  .payment {
    margin-left: 20rpx;
    background-color: #27ba9b;
  }

  .payment--loading {
    opacity: 0.72;
  }
}

.cart-badge {
  position: absolute;
  top: -5rpx;
  right: -5rpx;
  min-width: 36rpx;
  height: 36rpx;
  padding: 0 8rpx;
  border-radius: 100rpx;
  color: #fff;
  font-size: 20rpx;
  line-height: 36rpx;
  text-align: center;
  background-color: #ff4444;
}

.loading-state {
  padding-top: 44vh;
  color: #999;
  font-size: 26rpx;
  text-align: center;
}
</style>
