<script setup lang="ts">
import { defGoodsInfoService } from '@/api/app/goods_info'
import { defTenantStoreService } from '@/api/app/tenant_store'
import type { GoodsInfo } from '@/rpc/app/v1/goods_info'
import type { TenantStore } from '@/rpc/app/v1/tenant_store'
import { formatPrice, formatSrc } from '@/utils'
import { goodsDetailUrl } from '@/utils/navigation'
import { onLoad } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'

type StoreTabKey = 'home' | 'goods' | 'category'
type StoreSortKey = 'default' | 'sale' | 'price'
type StoreGoods = GoodsInfo & {
  specs?: string
  badge?: string
}
type StoreCategory = {
  key: string
  name: string
}

const query = defineProps<{
  id?: string
}>()

const storeTabs: { key: StoreTabKey; label: string }[] = [
  { key: 'home', label: '首页' },
  { key: 'goods', label: '全部商品' },
  { key: 'category', label: '分类' },
]
const sortOptions: { key: StoreSortKey; label: string }[] = [
  { key: 'default', label: '综合' },
  { key: 'sale', label: '销量' },
  { key: 'price', label: '价格' },
]
const allCategoryKey = 'all'
const storeServiceList = ['官方正品', '48小时发货', '七天无理由', '售后无忧']

const routeStoreId = Number(query.id || 0)
const storeId = Number.isFinite(routeStoreId) && routeStoreId > 0 ? routeStoreId : 0
const storeInfo = ref<TenantStore>()
const goodsList = ref<StoreGoods[]>([])
const pageNum = ref(1)
const pageSize = 10
const total = ref(0)
const finish = ref(false)
const loading = ref(false)
const loadingStore = ref(false)
const loadFailed = ref(false)
const activeTab = ref<StoreTabKey>('home')
const activeSort = ref<StoreSortKey>('default')
const activeCategory = ref(allCategoryKey)

const displayStore = computed(() => storeInfo.value)
const displayGoodsList = computed<StoreGoods[]>(() => {
  return goodsList.value
})
// 商品接口只返回分类 ID，门店页按当前商品聚合出可筛选分类。
const storeCategories = computed<StoreCategory[]>(() => {
  const categoryIDs = new Set<number>()
  displayGoodsList.value.forEach((item) => {
    item.category_id?.forEach((categoryID) => {
      if (categoryID > 0) {
        categoryIDs.add(categoryID)
      }
    })
  })
  return [
    { key: allCategoryKey, name: '全部' },
    ...Array.from(categoryIDs)
      .sort((left, right) => left - right)
      .map((categoryID) => ({
        key: String(categoryID),
        name: `分类 ${categoryID}`,
      })),
  ]
})
const storeStats = computed(() => [
  { label: '综合评分', value: '4.9' },
  { label: '在售商品', value: `${Math.max(total.value, displayGoodsList.value.length)}` },
  { label: '发货时效', value: '48h' },
])
const hasGoodsCategory = (goods: StoreGoods, categoryKey: string) => {
  return goods.category_id?.some((categoryID) => String(categoryID) === categoryKey) || false
}
const filteredGoodsList = computed(() => {
  let list = displayGoodsList.value
  if (activeTab.value === 'category' && activeCategory.value !== allCategoryKey) {
    list = list.filter((item) => hasGoodsCategory(item, activeCategory.value))
  }
  if (activeSort.value === 'sale') {
    return [...list].sort((left, right) => (right.sale_num || 0) - (left.sale_num || 0))
  }
  if (activeSort.value === 'price') {
    return [...list].sort((left, right) => left.price - right.price)
  }
  return list
})
const homeGoodsList = computed(() => displayGoodsList.value.slice(0, 4))
const featuredGoods = computed(() => displayGoodsList.value[0])
const goodsCountText = computed(
  () => `本店商品 ${Math.max(total.value, displayGoodsList.value.length)}`,
)
const activeCategoryName = computed(
  () => storeCategories.value.find((item) => item.key === activeCategory.value)?.name || '全部',
)

const getCategoryCount = (category: StoreCategory) => {
  if (category.key === allCategoryKey) {
    return displayGoodsList.value.length
  }
  return displayGoodsList.value.filter((item) => hasGoodsCategory(item, category.key)).length
}

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

const onSelectTab = (key: StoreTabKey) => {
  activeTab.value = key
}

const onSelectSort = (key: StoreSortKey) => {
  activeSort.value = key
}

const onSelectCategory = (key: string) => {
  activeCategory.value = key
}

const onTapSearch = () => {
  uni.showToast({ icon: 'none', title: '店内搜索暂未开放' })
}

const onContactStore = () => {
  uni.showToast({ icon: 'none', title: '已为你唤起联系入口' })
}

onLoad(() => {
  void uni.setNavigationBarTitle({ title: '店铺首页' })
  void loadData()
})
</script>

<template>
  <scroll-view enable-back-to-top scroll-y class="store-page" @scrolltolower="onScrollToLower">
    <view class="store-hero">
      <image
        v-if="displayStore?.cover"
        class="store-cover"
        mode="aspectFill"
        :src="formatSrc(displayStore.cover)"
      />
      <view v-else class="store-cover store-cover--empty"></view>
      <view class="store-mask"></view>
      <view class="store-profile">
        <image
          v-if="displayStore?.logo"
          class="store-logo"
          mode="aspectFill"
          :src="formatSrc(displayStore.logo)"
        />
        <view v-else class="store-logo store-logo--text">店</view>
        <view class="store-text">
          <view class="store-title-row">
            <view class="store-name">{{ displayStore?.name || '店铺首页' }}</view>
            <view class="store-contact" @tap.stop="onContactStore">联系店铺</view>
          </view>
          <view class="store-intro">{{ displayStore?.intro || '欢迎光临本店' }}</view>
          <view class="store-tags">
            <text v-for="item in storeServiceList.slice(0, 2)" :key="item" class="store-tag">
              {{ item }}
            </text>
          </view>
        </view>
      </view>
    </view>

    <view class="store-stat-panel">
      <view v-for="item in storeStats" :key="item.label" class="store-stat">
        <view class="store-stat__value">{{ item.value }}</view>
        <view class="store-stat__label">{{ item.label }}</view>
      </view>
    </view>

    <view class="store-main-panel">
      <view class="store-search" @tap="onTapSearch">
        <text class="store-search__icon">⌕</text>
        <text class="store-search__text">搜索本店商品</text>
      </view>
      <view class="notice-row">
        <text class="notice-label">公告</text>
        <text class="notice-text">{{ displayStore?.notice || '欢迎光临本店' }}</text>
      </view>
      <view class="service-strip">
        <view v-for="item in storeServiceList" :key="item" class="service-item">
          <text class="service-dot">✓</text>
          <text>{{ item }}</text>
        </view>
      </view>
    </view>

    <view class="store-tabs">
      <view
        v-for="tab in storeTabs"
        :key="tab.key"
        class="store-tab"
        :class="{ active: activeTab === tab.key }"
        @tap="onSelectTab(tab.key)"
      >
        {{ tab.label }}
      </view>
    </view>

    <template v-if="activeTab === 'home'">
      <view class="feature-panel">
        <view class="section-heading">
          <view>
            <view class="section-title">店铺精选</view>
            <view class="section-subtitle">高评分好物，按销量与服务表现精选</view>
          </view>
          <view class="section-link" @tap="onSelectTab('goods')">查看全部</view>
        </view>
        <navigator
          v-if="featuredGoods"
          class="feature-card"
          hover-class="none"
          :url="goodsDetailUrl(featuredGoods.id)"
        >
          <image class="feature-image" mode="aspectFill" :src="formatSrc(featuredGoods.picture)" />
          <view class="feature-info">
            <view class="feature-badge">{{ featuredGoods.badge || '本店推荐' }}</view>
            <view class="feature-name">{{ featuredGoods.name }}</view>
            <view class="feature-desc">{{ featuredGoods.desc }}</view>
            <view class="feature-bottom">
              <view class="goods-price">
                <text class="price-symbol">¥</text>
                <text>{{ formatPrice(featuredGoods.price) }}</text>
              </view>
              <text class="feature-action">查看 ›</text>
            </view>
          </view>
        </navigator>
      </view>

      <view class="goods-section">
        <view class="section-heading section-heading--compact">
          <view class="section-title">{{ goodsCountText }}</view>
          <view class="section-link" @tap="onSelectTab('category')">按分类看</view>
        </view>
        <view class="goods-row-list">
          <navigator
            v-for="goods in homeGoodsList"
            :key="goods.id"
            class="goods-row"
            hover-class="none"
            :url="goodsDetailUrl(goods.id)"
          >
            <image class="goods-row__image" mode="aspectFill" :src="formatSrc(goods.picture)" />
            <view class="goods-row__body">
              <view class="goods-row__name">{{ goods.name }}</view>
              <view class="goods-row__spec">{{ goods.specs || goods.desc }}</view>
              <view class="goods-row__bottom">
                <view class="goods-price">
                  <text class="price-symbol">¥</text>
                  <text>{{ formatPrice(goods.price) }}</text>
                </view>
                <text class="goods-row__action">查看 ›</text>
              </view>
            </view>
          </navigator>
        </view>
        <view v-if="!homeGoodsList.length" class="goods-empty">暂无商品</view>
        <view v-if="loadFailed" class="goods-empty">店铺暂不可访问</view>
      </view>
    </template>

    <template v-else-if="activeTab === 'goods'">
      <view class="goods-section">
        <view class="sort-bar">
          <view
            v-for="item in sortOptions"
            :key="item.key"
            class="sort-item"
            :class="{ active: activeSort === item.key }"
            @tap="onSelectSort(item.key)"
          >
            {{ item.label }}
          </view>
        </view>
        <view class="goods-grid">
          <navigator
            v-for="goods in filteredGoodsList"
            :key="goods.id"
            class="goods-card"
            hover-class="none"
            :url="goodsDetailUrl(goods.id)"
          >
            <image class="goods-image" mode="aspectFill" :src="formatSrc(goods.picture)" />
            <view class="goods-name">{{ goods.name }}</view>
            <view class="goods-spec">{{ goods.specs || goods.desc }}</view>
            <view class="goods-meta">
              <view class="goods-price">
                <text class="price-symbol">¥</text>
                <text>{{ formatPrice(goods.price) }}</text>
              </view>
              <text class="goods-sale">销量 {{ formatSaleText(goods.sale_num) }}</text>
            </view>
          </navigator>
        </view>
        <view v-if="!filteredGoodsList.length" class="goods-empty">暂无商品</view>
      </view>
    </template>

    <template v-else>
      <view class="category-panel">
        <view class="category-sidebar">
          <view
            v-for="category in storeCategories"
            :key="category.key"
            class="category-item"
            :class="{ active: activeCategory === category.key }"
            @tap="onSelectCategory(category.key)"
          >
            <text>{{ category.name }}</text>
            <text class="category-count">{{ getCategoryCount(category) }}</text>
          </view>
        </view>
        <view class="category-content">
          <view class="category-title">
            {{ activeCategoryName }}
          </view>
          <view class="goods-grid goods-grid--category">
            <navigator
              v-for="goods in filteredGoodsList"
              :key="goods.id"
              class="goods-card goods-card--compact"
              hover-class="none"
              :url="goodsDetailUrl(goods.id)"
            >
              <image class="goods-image" mode="aspectFill" :src="formatSrc(goods.picture)" />
              <view class="goods-name">{{ goods.name }}</view>
              <view class="goods-price">
                <text class="price-symbol">¥</text>
                <text>{{ formatPrice(goods.price) }}</text>
              </view>
            </navigator>
          </view>
          <view v-if="!filteredGoodsList.length" class="category-empty">该分类暂无更多商品</view>
        </view>
      </view>
    </template>

    <view class="loading-text">
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
  background: linear-gradient(135deg, #27ba9b 0%, #6dc7b8 100%);
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

.store-title-row {
  display: flex;
  align-items: center;
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

.store-contact {
  height: 52rpx;
  padding: 0 24rpx;
  border: 1rpx solid rgba(255, 255, 255, 0.88);
  border-radius: 999rpx;
  flex-shrink: 0;
  font-size: 24rpx;
  line-height: 52rpx;
}

.store-intro {
  margin-top: 12rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 25rpx;
  opacity: 0.92;
}

.store-tags {
  display: flex;
  gap: 10rpx;
  margin-top: 14rpx;
}

.store-tag {
  height: 36rpx;
  padding: 0 12rpx;
  border-radius: 6rpx;
  color: #0f8f78;
  font-size: 22rpx;
  line-height: 36rpx;
  background-color: rgba(255, 255, 255, 0.92);
}

.store-stat-panel {
  display: flex;
  margin: -26rpx 20rpx 20rpx;
  padding: 22rpx 0;
  border-radius: 14rpx;
  position: relative;
  z-index: 2;
  background-color: #fff;
  box-shadow: 0 10rpx 26rpx rgba(15, 23, 42, 0.08);
}

.store-stat {
  flex: 1;
  text-align: center;
  border-right: 1rpx solid #f0f0f0;

  &:last-child {
    border-right: 0;
  }
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
.feature-panel,
.goods-section,
.category-panel {
  margin: 0 20rpx 20rpx;
  border-radius: 14rpx;
  background-color: #fff;
}

.store-main-panel {
  padding: 20rpx 24rpx 22rpx;
}

.store-search {
  height: 72rpx;
  display: flex;
  align-items: center;
  padding: 0 22rpx;
  border: 1rpx solid #eeeeee;
  border-radius: 12rpx;
  box-sizing: border-box;
  color: #9aa1aa;
  background-color: #f7f8fa;
}

.store-search__icon {
  margin-right: 12rpx;
  font-size: 34rpx;
}

.store-search__text {
  font-size: 26rpx;
}

.notice-row {
  display: flex;
  padding-top: 20rpx;
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

.service-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 14rpx;
  margin-top: 22rpx;
}

.service-item {
  display: flex;
  align-items: center;
  height: 42rpx;
  padding: 0 14rpx;
  border-radius: 8rpx;
  color: #59616d;
  font-size: 23rpx;
  background-color: #f4f8f7;
}

.service-dot {
  margin-right: 6rpx;
  color: #27ba9b;
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

.feature-panel,
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

.section-subtitle {
  margin-top: 8rpx;
  color: #8a8f99;
  font-size: 23rpx;
}

.section-link {
  color: #0f9f86;
  font-size: 24rpx;
}

.feature-card {
  display: flex;
  min-height: 250rpx;
  border: 1rpx solid #f0f0f0;
  border-radius: 12rpx;
  overflow: hidden;
}

.feature-image {
  width: 250rpx;
  height: 250rpx;
  flex-shrink: 0;
  background-color: #f7f7f7;
}

.feature-info {
  min-width: 0;
  flex: 1;
  padding: 22rpx 22rpx 18rpx;
}

.feature-badge {
  display: inline-flex;
  height: 34rpx;
  padding: 0 10rpx;
  border-radius: 6rpx;
  color: #0f9f86;
  font-size: 21rpx;
  line-height: 34rpx;
  background-color: #e9f8f5;
}

.feature-name {
  height: 76rpx;
  margin-top: 12rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  color: #1f2937;
  font-size: 27rpx;
  line-height: 38rpx;
  font-weight: 600;
}

.feature-desc {
  margin-top: 8rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #8a8f99;
  font-size: 23rpx;
}

.feature-bottom,
.goods-row__bottom,
.goods-meta {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
}

.feature-bottom {
  margin-top: 18rpx;
}

.feature-action,
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

.sort-bar {
  display: flex;
  height: 72rpx;
  margin: -4rpx -24rpx 20rpx;
  border-bottom: 1rpx solid #f0f0f0;
}

.sort-item {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #5f6670;
  font-size: 26rpx;

  &.active {
    color: #0f9f86;
    font-weight: 700;
  }
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

.goods-card--compact {
  width: 238rpx;
  padding: 14rpx;
}

.goods-image {
  width: 100%;
  height: 276rpx;
  border-radius: 8rpx;
  background-color: #f7f7f7;
}

.goods-card--compact .goods-image {
  height: 210rpx;
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

.category-panel {
  display: flex;
  min-height: 680rpx;
  overflow: hidden;
}

.category-sidebar {
  width: 170rpx;
  flex-shrink: 0;
  background-color: #f7f8fa;
}

.category-item {
  position: relative;
  height: 96rpx;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #3f4650;
  font-size: 27rpx;

  &.active {
    color: #0f9f86;
    font-weight: 700;
    background-color: #fff;
  }

  &.active::before {
    content: '';
    position: absolute;
    left: 0;
    top: 24rpx;
    width: 6rpx;
    height: 48rpx;
    border-radius: 999rpx;
    background-color: #27ba9b;
  }
}

.category-count {
  margin-top: 6rpx;
  color: #9aa1aa;
  font-size: 20rpx;
  font-weight: 400;
}

.category-content {
  min-width: 0;
  flex: 1;
  padding: 24rpx 20rpx;
  box-sizing: border-box;
}

.category-title {
  margin-bottom: 18rpx;
  color: #20242a;
  font-size: 30rpx;
  font-weight: 700;
}

.goods-grid--category {
  gap: 18rpx 0;
}

.category-empty {
  margin-top: 80rpx;
  color: #999;
  font-size: 26rpx;
  text-align: center;
}

.loading-text {
  padding: 20rpx 0 46rpx;
  color: #666;
  font-size: 28rpx;
  text-align: center;
}
</style>
