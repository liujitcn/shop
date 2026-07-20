<script setup lang="ts">
import { defGoodsInfoService } from '@/api/shop/app/goods_info'
import { SkuMode, useGoodsPurchase } from '@/composables'
import { goodsDetailUrl, homeTabPage } from '@/utils/navigation'
import type {
  GoodsInfo,
  GoodsInfoResponse,
  PageGoodsInfoRequest,
} from '@/rpc/shop/app/v1/goods_info'
import { RecommendScene } from '@/rpc/shop/common/v1/enum'
import { onLoad } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'

/** 商品搜索列表补充字段。 */
type GoodsInfoExtra = GoodsInfo & {
  banner?: string[]
  init_sale_num?: number
  real_sale_num?: number
}

const query = defineProps<{
  name?: string
  category_id?: string
  categoryName?: string
}>()

const { safeAreaInsets } = uni.getSystemInfoSync()

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

const decodedName = decodeQueryText(query.name)
const decodedCategoryName = decodeQueryText(query.categoryName)
const pageParams: Required<PageGoodsInfoRequest> = {
  name: decodedName,
  category_id: query.category_id ? Number(query.category_id) : 0,
  tenant_store_id: 0,
  page_num: 1,
  page_size: 10,
}

const goodsInfoList = ref<GoodsInfo[]>([])
const detailInfoMap = ref<Record<number, GoodsInfoResponse>>({})
const detailCache = new Map<number, GoodsInfoResponse>()
const finish = ref(false)
const loading = ref(false)

const emptyText = computed(() => {
  if (decodedName) return `暂无“${decodedName}”相关商品`
  if (decodedCategoryName) return `暂无${decodedCategoryName}商品`
  return '暂无可购买商品'
})

/** 计算商品展示销量，优先使用后端返回的拆分销量字段。 */
const resolveSaleNum = (item: GoodsInfo) => {
  const goods = item as GoodsInfoExtra
  if (goods.init_sale_num !== undefined || goods.real_sale_num !== undefined) {
    return Number(goods.init_sale_num ?? 0) + Number(goods.real_sale_num ?? 0)
  }
  return item.sale_num || 0
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

const getGoodsPosition = (item: GoodsInfo) => {
  const position = goodsInfoList.value.findIndex((goods) => goods.id === item.id)
  return position >= 0 ? position : 0
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

const {
  addCart: onAddCart,
  buyNow: onBuyNow,
  buyingGoodsId,
  cartNum,
  isCollected,
  isShowSku,
  localData,
  openSkuPopup,
  refreshCartNum,
  refreshCollectState,
  skuMode,
  toggleCollect,
} = useGoodsPurchase<GoodsInfo>({
  ensureGoodsDetail,
  getRecommendContext: (item) => ({
    scene: RecommendScene.UNKNOWN_RS,
    request_id: 0,
    position: getGoodsPosition(item),
  }),
})

const getGoodsData = async () => {
  if (loading.value || finish.value) return
  loading.value = true
  try {
    const res = await defGoodsInfoService.PageGoodsInfo(pageParams)
    const existingIds = new Set(goodsInfoList.value.map((item) => item.id))
    const list = (res.goods_infos || []).filter((item) => !existingIds.has(item.id))
    goodsInfoList.value.push(...list)
    if (goodsInfoList.value.length < res.total) {
      pageParams.page_num++
    } else {
      finish.value = true
    }
  } finally {
    loading.value = false
  }
}

const navigateToGoods = (item: GoodsInfo) => {
  uni.navigateTo({ url: goodsDetailUrl(item.id) })
}

const onNavigateBack = () => {
  const pages = getCurrentPages()
  if (pages.length > 1) {
    uni.navigateBack({
      fail: () => {
        uni.switchTab({ url: homeTabPage })
      },
    })
    return
  }
  uni.switchTab({ url: homeTabPage })
}

const onShowSku = async (item: GoodsInfo, mode: SkuMode) => {
  await openSkuPopup(item, mode)
  void refreshCollectState(item)
}

const onToggleCollect = async (item: GoodsInfo) => {
  await toggleCollect(item)
}

const onScrollToLower = async () => {
  await getGoodsData()
}

onLoad(async () => {
  let title = '单品直购'
  if (decodedCategoryName) title = decodedCategoryName
  if (decodedName) title = decodedName
  await uni.setNavigationBarTitle({ title })
  await getGoodsData()
  await refreshCartNum()
  await refreshCollectState(goodsInfoList.value[0])
})
</script>

<template>
  <view
    class="back-button"
    :style="{ top: `${(safeAreaInsets?.top || 0) + 12}px` }"
    @tap="onNavigateBack"
  >
    <text class="back-button__icon">‹</text>
  </view>

  <scroll-view
    enable-back-to-top
    scroll-y
    class="single-list-page"
    @scrolltolower="onScrollToLower"
  >
    <goods-sku-popup
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

    <view v-if="goodsInfoList.length" class="goods-list">
      <view v-for="item in goodsInfoList" :key="item.id" class="goods-item">
        <view @tap="navigateToGoods(item)">
          <GoodsHero
            :pictures="getPictureList(item)"
            :price="item.price"
            :sale-num="resolveSaleNum(item)"
            :name="item.name"
            :desc="item.desc"
            image-height="750rpx"
          />
        </view>

        <GoodsActionBar
          :collected="isCollected(item)"
          :cart-num="cartNum"
          :buy-loading="buyingGoodsId === item.id"
          @collect="onToggleCollect(item)"
          @add-cart="onShowSku(item, SkuMode.Cart)"
          @buy-now="onShowSku(item, SkuMode.Buy)"
        />
      </view>
    </view>

    <EmptyState
      v-else-if="finish && !loading"
      image="/static/images/empty_search.png"
      :text="emptyText"
      min-height="60vh"
    />

    <view v-if="goodsInfoList.length || loading" class="loading-text">
      {{ finish ? '没有更多数据~' : '正在加载...' }}
    </view>
  </scroll-view>
</template>

<style lang="scss">
page {
  height: 100%;
  background-color: #f4f4f4;
}

.single-list-page {
  height: 100%;
  background-color: #f4f4f4;
}

.back-button {
  position: fixed;
  left: 24rpx;
  z-index: 20;
  width: 72rpx;
  height: 72rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1rpx solid rgba(255, 255, 255, 0.18);
  border-radius: 999rpx;
  color: #fff;
  background-color: rgba(17, 24, 39, 0.38);
}

.back-button__icon {
  margin-top: -4rpx;
  font-size: 48rpx;
  line-height: 1;
}

.goods-list {
  padding-bottom: 12rpx;
}

.goods-item {
  margin-bottom: 12rpx;
  background-color: #fff;
}

.loading-text {
  padding: 20rpx 0 40rpx;
  color: #666;
  font-size: 28rpx;
  text-align: center;
}
</style>
