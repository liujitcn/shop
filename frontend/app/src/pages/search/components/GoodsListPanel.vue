<script setup lang="ts">
import type { GoodsInfo } from '@/rpc/app/v1/goods_info'
import { ref } from 'vue'
import { formatPrice, formatSrc } from '@/utils'
import { goodsDetailUrl } from '@/utils/navigation'

type PromotionItem = {
  title: string
  desc?: string
}

type GoodsListItem = GoodsInfo & {
  /** 多图列表，后端列表只有主图时会自动退回 picture。 */
  banner?: string[]
  pictureList?: string[]
  /** 划线价，单位保持与 price 一致。 */
  marketPrice?: number
  /** 标题前置标签，例如“自营”。传空字符串可隐藏。 */
  titleBadge?: string
  /** 服务标签，例如国家补贴、支持送礼。 */
  tagList?: string[]
  /** 店铺信息，当前列表接口没有返回时使用默认文案。 */
  shopName?: string
  shopLogo?: string
  shopDesc?: string
  /** 顶部促销条，留给国补/换新等活动数据接入。 */
  promoList?: PromotionItem[]
}

const props = defineProps<{
  list: GoodsListItem[]
}>()

const defaultTagList = ['官方正品', '支持送礼', '7天价保']

const getItemKey = (item: GoodsListItem, index: number) => `${item.id || 'goods'}-${index}`
const activeIndexMap = ref<Record<string, number>>({})

const getPictureList = (item: GoodsListItem) => {
  const list = item.banner?.length ? item.banner : item.pictureList
  if (list?.length) {
    return list.filter(Boolean)
  }
  return item.picture ? [item.picture] : []
}

const getActiveIndex = (item: GoodsListItem, index: number) => {
  return activeIndexMap.value[getItemKey(item, index)] || 0
}

const onSwiperChange = (item: GoodsListItem, index: number, ev: UniHelper.SwiperOnChangeEvent) => {
  activeIndexMap.value = {
    ...activeIndexMap.value,
    [getItemKey(item, index)]: ev.detail.current,
  }
}

const priceInteger = (price: number) => formatPrice(price).split('.')[0]
const priceDecimal = (price: number) => formatPrice(price).split('.')[1] || '00'

const trimZeroDecimal = (value: string) => value.replace(/\.0$/, '')

const formatSaleText = (sale_num: number) => {
  if (!sale_num) {
    return '销量 0'
  }
  if (sale_num >= 10000) {
    return `销量 ${trimZeroDecimal((sale_num / 10000).toFixed(1))}万+`
  }
  return `销量 ${sale_num}`
}

const resolveTitleBadge = (item: GoodsListItem) => {
  if (item.titleBadge !== undefined) {
    return item.titleBadge
  }
  return '自营'
}

const resolveTagList = (item: GoodsListItem) => {
  const customTags = item.tagList?.filter(Boolean)
  if (customTags?.length) {
    return customTags.slice(0, 5)
  }
  if (item.desc) {
    return [item.desc, ...defaultTagList.slice(1)].slice(0, 5)
  }
  return defaultTagList
}

const resolveShopName = (item: GoodsListItem) => item.shopName || '官方旗舰店'
const resolveShopDesc = (item: GoodsListItem) => item.shopDesc || '品质保障'

const navigateToGoods = (item: GoodsListItem) => {
  uni.navigateTo({ url: goodsDetailUrl(item.id) })
}

const onTapAction = (item: GoodsListItem) => {
  // 列表接口没有默认 SKU，加入购物车/立即购买统一进入详情页完成规格选择。
  navigateToGoods(item)
}
</script>

<template>
  <view class="goods-list-panel">
    <view v-for="(item, index) in props.list" :key="getItemKey(item, index)" class="goods-card">
      <view class="goods-media" @tap="navigateToGoods(item)">
        <swiper
          v-if="getPictureList(item).length > 1"
          class="goods-swiper"
          circular
          @change="onSwiperChange(item, index, $event)"
        >
          <swiper-item v-for="picture in getPictureList(item)" :key="picture">
            <image class="goods-image" mode="aspectFit" :src="formatSrc(picture)" />
          </swiper-item>
        </swiper>
        <image
          v-else-if="getPictureList(item).length === 1"
          class="goods-image"
          mode="aspectFit"
          :src="formatSrc(getPictureList(item)[0])"
        />
        <view v-else class="image-placeholder">暂无图片</view>
        <view class="image-count">
          {{ getActiveIndex(item, index) + 1 }}/{{ getPictureList(item).length || 1 }}
        </view>
      </view>

      <view v-if="item.promoList?.length" class="promo-strip">
        <view v-for="promo in item.promoList" :key="promo.title" class="promo-item">
          <text class="promo-title">{{ promo.title }}</text>
          <text v-if="promo.desc" class="promo-desc">{{ promo.desc }}</text>
        </view>
      </view>

      <view class="goods-info">
        <view class="price-row">
          <view class="price-main">
            <text class="currency">¥</text>
            <text class="price-int">{{ priceInteger(item.price) }}</text>
            <text class="price-decimal">.{{ priceDecimal(item.price) }}</text>
            <text class="price-suffix">到手价</text>
            <text v-if="item.marketPrice && item.marketPrice > item.price" class="market-price">
              ¥{{ formatPrice(item.marketPrice) }}
            </text>
          </view>
          <text class="sales">{{ formatSaleText(item.sale_num) }}</text>
        </view>

        <view class="title-row" @tap="navigateToGoods(item)">
          <text v-if="resolveTitleBadge(item)" class="title-badge">{{
            resolveTitleBadge(item)
          }}</text>
          <text class="goods-name">{{ item.name }}</text>
          <text class="title-arrow">›</text>
        </view>

        <view class="tag-row">
          <text
            v-for="(tag, tagIndex) in resolveTagList(item)"
            :key="`${tag}-${tagIndex}`"
            class="tag"
            :class="{ 'tag--primary': tagIndex === 0 }"
          >
            {{ tag }}
          </text>
        </view>

        <view class="shop-action-row">
          <view class="shop-info" @tap="navigateToGoods(item)">
            <image
              v-if="item.shopLogo"
              class="shop-logo"
              mode="aspectFill"
              :src="formatSrc(item.shopLogo)"
            />
            <view v-else class="shop-logo shop-logo--text">铺</view>
            <view class="shop-text">
              <view class="shop-name">{{ resolveShopName(item) }}</view>
              <view class="shop-desc">{{ resolveShopDesc(item) }} ›</view>
            </view>
          </view>
          <view class="action-buttons">
            <view class="action-button action-button--cart" @tap.stop="onTapAction(item)"
              >加入购物车</view
            >
            <view class="action-button action-button--buy" @tap.stop="onTapAction(item)"
              >立即购买</view
            >
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<style lang="scss">
.goods-list-panel {
  padding: 20rpx 20rpx 0;
}

.goods-card {
  margin-bottom: 24rpx;
  overflow: hidden;
  border-radius: 24rpx;
  background-color: #fff;
}

.goods-media {
  position: relative;
  height: 710rpx;
  background-color: #fff;
}

.goods-swiper,
.goods-image,
.image-placeholder {
  width: 100%;
  height: 100%;
}

.goods-image {
  display: block;
}

.image-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  color: #999;
  font-size: 28rpx;
  background-color: #f7f7f7;
}

.image-count {
  position: absolute;
  right: 24rpx;
  bottom: 24rpx;
  min-width: 84rpx;
  height: 48rpx;
  padding: 0 18rpx;
  border-radius: 999rpx;
  box-sizing: border-box;
  color: #fff;
  font-size: 28rpx;
  font-weight: 600;
  line-height: 48rpx;
  text-align: center;
  background-color: rgba(0, 0, 0, 0.56);
}

.promo-strip {
  display: flex;
  min-height: 104rpx;
  color: #fff8c7;
  background: linear-gradient(90deg, #249b52 0%, #21a35a 68%, #fff3a9 100%);
}

.promo-item {
  position: relative;
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 0 20rpx;
  box-sizing: border-box;

  &::after {
    position: absolute;
    top: 24rpx;
    right: 0;
    width: 1rpx;
    height: 56rpx;
    content: '';
    background-color: rgba(255, 248, 199, 0.5);
  }

  &:last-child::after {
    display: none;
  }
}

.promo-title {
  font-size: 34rpx;
  font-weight: 700;
  line-height: 1.1;
}

.promo-desc {
  margin-top: 8rpx;
  font-size: 22rpx;
  line-height: 1.2;
}

.goods-info {
  padding: 24rpx 24rpx 20rpx;
}

.price-row {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
}

.price-main {
  display: flex;
  align-items: baseline;
  min-width: 0;
  color: #ff3131;
}

.currency {
  font-size: 30rpx;
  font-weight: 700;
}

.price-int {
  font-size: 58rpx;
  font-weight: 800;
  line-height: 1;
  letter-spacing: -2rpx;
}

.price-decimal {
  font-size: 30rpx;
  font-weight: 800;
}

.price-suffix {
  margin-left: 10rpx;
  font-size: 26rpx;
  font-weight: 500;
}

.market-price {
  margin-left: 18rpx;
  color: #8f8f99;
  font-size: 28rpx;
  text-decoration: line-through;
}

.sales {
  flex-shrink: 0;
  margin-left: 20rpx;
  color: #8b8f9b;
  font-size: 28rpx;
}

.title-row {
  display: flex;
  align-items: center;
  margin-top: 14rpx;
}

.title-badge {
  flex-shrink: 0;
  height: 34rpx;
  margin-right: 10rpx;
  padding: 0 8rpx;
  border-radius: 6rpx;
  color: #fff;
  font-size: 24rpx;
  font-weight: 700;
  line-height: 34rpx;
  background-color: #ff3131;
}

.goods-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  color: #20212a;
  font-size: 34rpx;
  font-weight: 700;
  line-height: 1.28;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.title-arrow {
  flex-shrink: 0;
  margin-left: 12rpx;
  color: #20212a;
  font-size: 54rpx;
  line-height: 34rpx;
}

.tag-row {
  display: flex;
  flex-wrap: wrap;
  margin-top: 18rpx;
  overflow: hidden;
}

.tag {
  max-width: 258rpx;
  height: 40rpx;
  margin-right: 10rpx;
  margin-bottom: 10rpx;
  padding: 0 10rpx;
  overflow: hidden;
  border: 1rpx solid #f0caa8;
  border-radius: 6rpx;
  box-sizing: border-box;
  color: #b8742b;
  font-size: 24rpx;
  line-height: 38rpx;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tag--primary {
  border-color: #8be0b2;
  color: #16a35a;
}

.shop-action-row {
  display: flex;
  align-items: center;
  margin-top: 14rpx;
}

.shop-info {
  flex: 1;
  display: flex;
  align-items: center;
  min-width: 0;
}

.shop-logo {
  flex-shrink: 0;
  width: 72rpx;
  height: 72rpx;
  margin-right: 14rpx;
  border-radius: 50%;
  background-color: #f4f4f4;
}

.shop-logo--text {
  color: #ff3b30;
  font-size: 28rpx;
  font-weight: 700;
  line-height: 72rpx;
  text-align: center;
  background-color: #fff0ec;
}

.shop-text {
  min-width: 0;
}

.shop-name,
.shop-desc {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.shop-name {
  color: #262936;
  font-size: 28rpx;
  line-height: 1.25;
}

.shop-desc {
  margin-top: 6rpx;
  color: #9296a3;
  font-size: 24rpx;
}

.action-buttons {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  margin-left: 14rpx;
}

.action-button {
  width: 188rpx;
  height: 76rpx;
  border-radius: 12rpx;
  font-size: 30rpx;
  font-weight: 700;
  line-height: 76rpx;
  text-align: center;
}

.action-button--cart {
  margin-right: 14rpx;
  color: #b66b20;
  background-color: #ffe5c4;
}

.action-button--buy {
  color: #fff;
  background-color: #ff333d;
}
</style>
