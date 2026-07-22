<script setup lang="ts">
import { defGoodsInfoService } from '@/api/shop/goods_info'
import { defTenantStoreService } from '@/api/shop/tenant_store'
import type { GoodsInfo } from '@/rpc/shop/app/v1/goods_info'
import type { TenantStore } from '@/rpc/shop/app/v1/tenant_store'
import { formatPrice, formatSrc } from '@/utils'
import { goodsDetailUrl, homeTabPage } from '@/utils/navigation'
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'

const query = defineProps<{
  id?: string
}>()

const routeStoreId = Number(query.id || 0)
const storeId = Number.isFinite(routeStoreId) && routeStoreId > 0 ? routeStoreId : 0
const storeInfo = ref<TenantStore>()
const goodsList = ref<GoodsInfo[]>([])
const pageNum = ref(1)
const pageSize = 10
const total = ref(0)
const finish = ref(false)
const loading = ref(false)
const loadingStore = ref(false)
const loadFailed = ref(false)
const searchKeyword = ref('')

const formatSaleText = (saleNum: number) => {
  if (saleNum >= 10000) {
    return `${(saleNum / 10000).toFixed(1).replace('.0', '')}万+`
  }
  return String(saleNum || 0)
}

/** 加载真实门店与商品数据；接口失败时展示空态。 */
const loadData = async () => {
  if (storeId <= 0) {
    loadFailed.value = true
    finish.value = true
    await uni.setNavigationBarTitle({ title: '店铺首页' })
    return
  }

  loadingStore.value = true
  try {
    const store = await defTenantStoreService.GetTenantStore({ id: storeId })
    storeInfo.value = store
    loadFailed.value = false
    await uni.setNavigationBarTitle({ title: store.name || '店铺首页' })
    goodsList.value = []
    pageNum.value = 1
    total.value = 0
    finish.value = false
    await loadGoods()
  } catch (error) {
    console.warn(error)
    await uni.setNavigationBarTitle({ title: '店铺首页' })
    storeInfo.value = undefined
    goodsList.value = []
    total.value = 0
    loadFailed.value = true
    finish.value = true
  } finally {
    loadingStore.value = false
  }
}

/** 分页加载真实门店商品；接口失败时停止继续加载。 */
const loadGoods = async () => {
  if (loading.value || finish.value || storeId <= 0) return
  loading.value = true
  try {
    const response = await defGoodsInfoService.PageGoodsInfo({
      name: searchKeyword.value,
      category_id: 0,
      tenant_store_id: storeId,
      page_num: pageNum.value,
      page_size: pageSize,
    })
    const list = response.goods_infos || []
    goodsList.value.push(...list)
    total.value = response.total || goodsList.value.length
    if (goodsList.value.length < total.value) {
      pageNum.value++
    } else {
      finish.value = true
    }
  } catch (error) {
    console.warn(error)
    finish.value = true
  } finally {
    loading.value = false
  }
}

const onScrollToLower = () => {
  void loadGoods()
}

/** 按当前关键词重新查询本店商品。 */
const onSearch = async () => {
  if (loading.value || storeId <= 0) return
  goodsList.value = []
  pageNum.value = 1
  total.value = 0
  finish.value = false
  await loadGoods()
}

const onBack = () => {
  const pages = getCurrentPages()
  if (pages.length > 1) {
    uni.navigateBack()
    return
  }
  uni.switchTab({ url: homeTabPage })
}

onLoad(() => {
  void uni.setNavigationBarTitle({ title: '店铺首页' })
  void loadData()
})
</script>

<template>
  <scroll-view enable-back-to-top scroll-y class="store-page" @scrolltolower="onScrollToLower">
    <view class="store-hero" :class="{ 'store-hero--cover': storeInfo?.cover }">
      <image
        v-if="storeInfo?.cover"
        class="store-cover"
        mode="aspectFill"
        :src="formatSrc(storeInfo.cover)"
      />
      <view v-else class="store-cover store-cover--empty"></view>
      <view v-if="storeInfo?.cover" class="store-mask"></view>
      <view class="back-button" @tap="onBack">‹</view>
      <view class="store-search">
        <uni-icons type="search" size="18" color="#999" />
        <input
          v-model="searchKeyword"
          class="store-search__input"
          confirm-type="search"
          placeholder="搜索本店商品"
          placeholder-class="store-search__placeholder"
          @confirm="onSearch"
        />
      </view>
      <view class="store-profile">
        <image
          v-if="storeInfo?.logo"
          class="store-logo"
          mode="aspectFill"
          :src="formatSrc(storeInfo.logo)"
        />
        <view v-else class="store-logo store-logo--text">店</view>
        <view class="store-text">
          <view class="store-name">{{ storeInfo?.name }}</view>
          <view v-if="storeInfo?.intro" class="store-intro">{{ storeInfo.intro }}</view>
        </view>
      </view>
    </view>

    <view v-if="storeInfo?.notice" class="notice-panel">
      <view class="notice-row">
        <text class="notice-label">公告</text>
        <text class="notice-text">{{ storeInfo.notice }}</text>
      </view>
    </view>

    <view class="goods-section">
      <view class="section-heading">
        <view class="section-title">全部商品</view>
        <view class="section-count">{{ total }} 件</view>
      </view>
      <view class="goods-grid">
        <navigator
          v-for="goods in goodsList"
          :key="goods.id"
          class="goods-card"
          hover-class="none"
          :url="goodsDetailUrl(goods.id)"
        >
          <image class="goods-image" mode="aspectFill" :src="formatSrc(goods.picture)" />
          <view class="goods-card__content">
            <view class="goods-name">{{ goods.name }}</view>
            <view v-if="goods.desc" class="goods-desc">{{ goods.desc }}</view>
            <view class="goods-meta">
              <view class="goods-price">
                <text class="price-symbol">¥</text>
                <text>{{ formatPrice(goods.price) }}</text>
              </view>
              <text class="goods-sale">销量 {{ formatSaleText(goods.sale_num) }}</text>
            </view>
          </view>
        </navigator>
      </view>
      <view v-if="loadFailed" class="goods-empty">店铺暂不可访问</view>
      <view v-else-if="!goodsList.length && !loading" class="goods-empty">暂无商品</view>
    </view>

    <view v-if="loading || loadingStore || goodsList.length" class="loading-text">
      {{ loading || loadingStore ? '正在加载...' : finish ? '没有更多商品~' : '上拉加载更多' }}
    </view>
  </scroll-view>
</template>

<style lang="scss">
page {
  height: 100%;
  background-color: #f4f4f4;
}

.store-page {
  height: 100%;
  background-color: #f4f4f4;
}

.store-hero {
  position: relative;
  height: 360rpx;
  overflow: hidden;
  background-color: #dfeee9;
}

.store-cover {
  width: 100%;
  height: 100%;
}

.store-cover--empty {
  background-color: #fff;
}

.store-mask {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: linear-gradient(180deg, rgba(0, 0, 0, 0.06), rgba(0, 0, 0, 0.52));
}

.store-profile {
  position: absolute;
  left: 24rpx;
  right: 24rpx;
  bottom: 30rpx;
  display: flex;
  align-items: center;
}

.store-logo {
  width: 112rpx;
  height: 112rpx;
  flex-shrink: 0;
  border: 4rpx solid rgba(255, 255, 255, 0.9);
  border-radius: 16rpx;
  background-color: #fff;
}

.store-logo--text {
  display: flex;
  align-items: center;
  justify-content: center;
  color: #27ba9b;
  font-size: 38rpx;
  font-weight: 700;
}

.store-text {
  min-width: 0;
  flex: 1;
  padding-left: 20rpx;
  color: #fff;
}

.store-name {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 36rpx;
  font-weight: 700;
}

.store-intro {
  margin-top: 12rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 25rpx;
  opacity: 0.92;
}

.store-stat-panel {
  text-align: center;
  margin: -26rpx 20rpx 20rpx;
  padding: 22rpx 0;
  border-radius: 14rpx;
  position: relative;
  z-index: 2;
  background-color: #fff;
  box-shadow: 0 10rpx 26rpx rgba(15, 23, 42, 0.08);
}

.store-stat__value {
  color: #1f2937;
  font-size: 32rpx;
  font-weight: 700;
}

.store-stat__label {
  margin-top: 8rpx;
  color: #8a8f99;
  font-size: 22rpx;
}

.store-main-panel,
.goods-section {
  margin: 0 20rpx 20rpx;
  border-radius: 14rpx;
  background-color: #fff;
}

.store-main-panel {
  padding: 20rpx 24rpx 22rpx;
}

.notice-row {
  display: flex;
  font-size: 25rpx;
  line-height: 1.45;
}

.notice-label {
  width: 70rpx;
  flex-shrink: 0;
  color: #0f9f86;
  font-weight: 600;
}

.notice-text {
  min-width: 0;
  flex: 1;
  color: #3f4650;
}

.store-tabs {
  display: flex;
  height: 88rpx;
  margin-bottom: 18rpx;
  border-top: 1rpx solid #eeeeee;
  border-bottom: 1rpx solid #eeeeee;
  background-color: #fff;
}

.store-tab {
  position: relative;
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #2f3338;
  font-size: 28rpx;

  &.active {
    color: #0f9f86;
    font-weight: 700;
  }

  &.active::after {
    content: '';
    position: absolute;
    left: 50%;
    bottom: 8rpx;
    width: 44rpx;
    height: 5rpx;
    border-radius: 999rpx;
    background-color: #27ba9b;
    transform: translateX(-50%);
  }
}

.goods-section {
  padding: 24rpx;
  box-sizing: border-box;
}

.section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 22rpx;
}

.section-heading--compact {
  margin-bottom: 18rpx;
}

.section-title {
  color: #20242a;
  font-size: 30rpx;
  font-weight: 700;
}

.section-link {
  color: #0f9f86;
  font-size: 24rpx;
}

.goods-row__bottom,
.goods-meta {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
}

.goods-row__action {
  color: #0f9f86;
  font-size: 24rpx;
}

.goods-row-list {
  border-top: 1rpx solid #f0f0f0;
}

.goods-empty {
  padding: 48rpx 0;
  color: #8a8f99;
  font-size: 26rpx;
  text-align: center;
}

.goods-row {
  display: flex;
  padding: 20rpx 0;
  border-bottom: 1rpx solid #f0f0f0;

  &:last-child {
    border-bottom: 0;
  }
}

.goods-row__image {
  width: 180rpx;
  height: 180rpx;
  border-radius: 10rpx;
  flex-shrink: 0;
  background-color: #f7f7f7;
}

.goods-row__body {
  min-width: 0;
  flex: 1;
  padding-left: 22rpx;
}

.goods-row__name {
  height: 76rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  color: #262626;
  font-size: 27rpx;
  line-height: 38rpx;
}

.goods-row__spec,
.goods-spec {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #8a8f99;
  font-size: 23rpx;
}

.goods-row__spec {
  margin-top: 10rpx;
}

.goods-row__bottom {
  margin-top: 24rpx;
}

.goods-grid {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
}

.goods-card {
  width: 320rpx;
  margin-bottom: 20rpx;
  padding: 18rpx;
  border: 1rpx solid #f0f0f0;
  border-radius: 12rpx;
  box-sizing: border-box;
  background-color: #fff;
}

.goods-image {
  width: 100%;
  height: 276rpx;
  border-radius: 8rpx;
  background-color: #f7f7f7;
}

.goods-name {
  height: 72rpx;
  margin-top: 14rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  color: #262626;
  font-size: 25rpx;
  line-height: 36rpx;
}

.goods-spec {
  margin-top: 8rpx;
}

.goods-meta {
  margin-top: 14rpx;
}

.goods-price {
  color: #cf4444;
  font-size: 30rpx;
  font-weight: 700;
}

.price-symbol {
  font-size: 70%;
}

.goods-sale {
  color: #999;
  font-size: 22rpx;
}

.loading-text {
  padding: 20rpx 0 46rpx;
  color: #666;
  font-size: 28rpx;
  text-align: center;
}

/* 店铺页与商城主界面保持一致，使用紧凑头部和稳定双列商品网格。 */
.store-page {
  background-color: #f7f7f7;
}

.store-hero {
  height: 220rpx;
  border-bottom: 1rpx solid #eeeeee;
  background-color: #fff;
}

.store-hero--cover {
  height: 260rpx;
  border-bottom: 0;
}

.back-button {
  position: absolute;
  top: calc(env(safe-area-inset-top) + 16rpx);
  left: 24rpx;
  z-index: 2;
  width: 64rpx;
  height: 64rpx;
  border-radius: 50%;
  border: 1rpx solid #e5e7eb;
  color: #333;
  font-size: 52rpx;
  line-height: 58rpx;
  text-align: center;
  background-color: rgba(255, 255, 255, 0.92);
}

.store-hero--cover .back-button {
  border-color: transparent;
  color: #fff;
  background-color: rgba(0, 0, 0, 0.3);
}

.store-search {
  position: absolute;
  top: calc(env(safe-area-inset-top) + 16rpx);
  right: 24rpx;
  left: 108rpx;
  z-index: 2;
  display: flex;
  align-items: center;
  height: 64rpx;
  padding: 0 22rpx;
  border: 1rpx solid #eeeeee;
  border-radius: 32rpx;
  box-sizing: border-box;
  background-color: #f7f7f7;
}

.store-search__input {
  min-width: 0;
  height: 64rpx;
  margin-left: 12rpx;
  flex: 1;
  color: #333;
  font-size: 26rpx;
  line-height: 64rpx;
}

.store-search__placeholder {
  color: #aaa;
}

.store-hero--cover .store-search {
  border-color: rgba(255, 255, 255, 0.2);
  background-color: rgba(255, 255, 255, 0.92);
}

.store-profile {
  right: 32rpx;
  bottom: 24rpx;
  left: 32rpx;
}

.store-logo {
  width: 96rpx;
  height: 96rpx;
  border-width: 2rpx;
  border-radius: 12rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.06);
}

.store-name {
  color: #222;
  font-size: 32rpx;
  line-height: 44rpx;
}

.store-intro {
  margin-top: 4rpx;
  color: #777;
  font-size: 24rpx;
  line-height: 34rpx;
  opacity: 1;
}

.store-hero--cover .store-name,
.store-hero--cover .store-intro {
  color: #fff;
}

.store-hero--cover .store-intro {
  opacity: 0.9;
}

.notice-panel {
  margin: 20rpx 20rpx 0;
  padding: 20rpx 24rpx;
  border-radius: 12rpx;
  background-color: #fff;
}

.notice-label {
  width: auto;
  margin-right: 18rpx;
  color: #27ba9b;
}

.notice-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.goods-section {
  margin: 20rpx 0 0;
  padding: 0 20rpx;
  border-radius: 0;
  background-color: transparent;
}

.section-heading {
  height: 72rpx;
  margin: 0;
}

.section-title {
  font-size: 30rpx;
}

.section-count {
  color: #999;
  font-size: 24rpx;
}

.goods-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18rpx;
}

.goods-card {
  width: auto;
  min-width: 0;
  margin: 0;
  padding: 0;
  overflow: hidden;
  border: 0;
  border-radius: 12rpx;
  background-color: #fff;
}

.goods-image {
  display: block;
  width: 100%;
  height: auto;
  aspect-ratio: 1;
  border-radius: 0;
  object-fit: cover;
}

.goods-card__content {
  padding: 16rpx;
}

.goods-name {
  height: 72rpx;
  margin: 0;
  font-size: 26rpx;
  line-height: 36rpx;
}

.goods-desc {
  margin-top: 8rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #999;
  font-size: 22rpx;
}

.goods-meta {
  margin-top: 14rpx;
  align-items: flex-end;
  gap: 8rpx;
}

.goods-price {
  min-width: 0;
  color: #cf4444;
  font-size: 30rpx;
}

.goods-sale {
  flex-shrink: 0;
  font-size: 20rpx;
}

.loading-text {
  padding: 28rpx 0 calc(36rpx + env(safe-area-inset-bottom));
  color: #999;
  font-size: 24rpx;
}

@media screen and (min-width: 750px) {
  .store-page {
    width: 750px;
    margin: 0 auto;
  }
}
</style>
