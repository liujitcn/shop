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
import type { GoodsInfo, GoodsInfoResponse } from '@/rpc/app/goods_info'
import type { RecommendContext } from '@/rpc/app/recommend'
import { onLoad } from '@dcloudio/uni-app'
import { useRecommendStore, useUserStore } from '@/stores'
import { computed, ref } from 'vue'
import AddressPanel from './components/AddressPanel.vue'
import ServicePanel from './components/ServicePanel.vue'
import { formatSrc, formatPrice } from '@/utils'
import { navigateToLogin } from '@/utils/login'
import { defShopServiceService } from '@/api/app/shop_service.ts'
import type { ShopService } from '@/rpc/app/shop_service.ts'
import { RecommendGoodsActionType } from '@/rpc/common/enum'
// 获取会员信息
const userStore = useUserStore()
const recommendStore = useRecommendStore()
// 获取屏幕边界到安全区域距离
const { safeAreaInsets } = uni.getSystemInfoSync()

// 接收页面参数
const query = defineProps<{
  id: string
  scene?: string
  requestId?: string
  index?: string
}>()
const goodsId = Number(query.id)
// 当前页直接复用入口路由上的原始推荐参数，不在这里做额外转换。
const recommendContext = {
  scene: query.scene,
  requestId: query.requestId,
  position: query.index,
} as unknown as RecommendContext

// 获取商品详情信息
const goodsInfo = ref<GoodsInfoResponse>()
const goodsInfoList = ref<GoodsInfo[]>([])
const isCollect = ref<boolean>(false)
const cartNum = ref<number>(0)
const serviceList = ref<ShopService[]>([])
const serviceLabelList = computed(() => serviceList.value.map((item) => item.label))

const loadData = async () => {
  const ssRes = await defShopServiceService.ListShopService({})
  serviceList.value = ssRes.list || []
  const res = await defGoodsInfoService.GetGoodsInfo({
    value: goodsId,
  })
  goodsInfo.value = res
  const pageRes = await defGoodsInfoService.PageGoodsInfo({
    name: '',
    categoryId: res.categoryId,
    pageNum: 1,
    pageSize: 30,
  })
  goodsInfoList.value = pageRes.list || []
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
}

// 页面加载
onLoad(() => {
  loadData()
  void (async () => {
    try {
      await recommendStore.getAnonymousId()
      await defRecommendService.RecommendGoodsActionReport({
        eventType: RecommendGoodsActionType.RECOMMEND_GOODS_ACTION_VIEW,
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
  try {
    await recommendStore.getAnonymousId()
    await defRecommendService.RecommendGoodsActionReport({
      eventType: RecommendGoodsActionType.RECOMMEND_GOODS_ACTION_CART,
      goodsItems: [
        {
          goodsId,
          goodsNum: ev.buy_num,
          recommendContext,
        },
      ],
    })
  } catch (error) {
    console.error(error)
  }
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
  const params = [
    `goodsId=${encodeURIComponent(String(ev.goods_id))}`,
    `skuCode=${encodeURIComponent(String(ev._id))}`,
    `num=${encodeURIComponent(String(ev.buy_num))}`,
  ]
  if (query.scene) {
    params.push(`scene=${encodeURIComponent(query.scene)}`)
  }
  if (query.requestId) {
    params.push(`requestId=${encodeURIComponent(query.requestId)}`)
  }
  if (query.index !== undefined && query.index !== '') {
    params.push(`index=${encodeURIComponent(query.index)}`)
  }
  uni.navigateTo({
    url: `/pagesOrder/create/create?${params.join('&')}`,
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
  if (isCollect.value) {
    try {
      await recommendStore.getAnonymousId()
      await defRecommendService.RecommendGoodsActionReport({
        eventType: RecommendGoodsActionType.RECOMMEND_GOODS_ACTION_COLLECT,
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
  }
  await uni.showToast({ title: isCollect.value ? '收藏成功' : '取消成功' })
}

// 定义分享配置
const shareConfig = computed(() => {
  if (!goodsInfo.value) return {}
  return {
    title: `${goodsInfo.value.name} ¥${formatPrice(goodsInfo.value.price)}`,
    path: `/pages/goods/goods?id=${goodsInfo.value.id}`,
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
  <scroll-view v-if="goodsInfo" enable-back-to-top scroll-y class="viewport">
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
        />
      </view>
    </view>

    <!-- 同类推荐 -->
    <view class="similar panel">
      <view class="title">
        <text>同类推荐</text>
      </view>
      <view class="content">
        <navigator
          v-for="item in goodsInfoList"
          :key="item.id"
          class="goods"
          hover-class="none"
          :url="`/pages/goods/goods?id=${item.id}`"
        >
          <image class="image" mode="aspectFill" :src="formatSrc(item.picture)" />
          <view class="name ellipsis">{{ item.name }}</view>
          <view class="price">
            <text class="symbol">¥</text>
            <text class="number">{{ formatPrice(item.price) }}</text>
          </view>
        </navigator>
      </view>
    </view>
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
  background-color: #f4f4f4;
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
