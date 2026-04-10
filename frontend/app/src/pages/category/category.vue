<script setup lang="ts">
import { defGoodsCategoryService } from '@/api/app/goods_category'
import { defShopBannerService } from '@/api/app/shop_banner'
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import PageSkeleton from './components/PageSkeleton.vue'
import type { ShopBanner } from '@/rpc/app/shop_banner'
import type { GoodsCategory } from '@/rpc/app/goods_category'
import { formatSrc, formatPrice } from '@/utils'
import { ShopBannerSite } from '@/rpc/common/enum.ts'
import { goodsDetailUrl } from '@/utils/navigation'

// 获取轮播图数据
const bannerList = ref<ShopBanner[]>([])
const getBannerData = async () => {
  const res = await defShopBannerService.ListShopBanner({
    site: ShopBannerSite.CATEGORY,
  })
  bannerList.value = res.list || []
}

// 获取分类列表数据
const categoryList = ref<GoodsCategory[]>([])
const subCategoryList = ref<GoodsCategory[]>([])
const activeMap = ref<Map<number, GoodsCategory[]>>()
const activeIndex = ref(0)
const getCategoryTopData = async () => {
  const topRes = await defGoodsCategoryService.ListGoodsCategory({
    parentId: 0,
  })
  categoryList.value = topRes.list || []
  // 查询二级分类
  if (categoryList.value.length) {
    const subRes = await defGoodsCategoryService.ListGoodsCategory({
      parentId: categoryList.value[0].id,
    })
    subCategoryList.value = subRes.list
    if (activeMap.value === undefined) {
      activeMap.value = new Map<number, GoodsCategory[]>()
    }
    activeMap!.value.set(0, subCategoryList.value)
  }
}

const getSubCategoryData = async (index: number) => {
  activeIndex.value = index
  if (activeMap!.value?.has(index)) {
    subCategoryList.value = activeMap!.value.get(index) || []
  } else {
    const res = await defGoodsCategoryService.ListGoodsCategory({
      parentId: categoryList.value[index].id,
    })
    subCategoryList.value = res.list || []
    if (activeMap.value === undefined) {
      activeMap.value = new Map<number, GoodsCategory[]>()
    }
    activeMap!.value.set(index, subCategoryList.value)
  }
}

const MAX_NAME_LENGTH = 5

const shouldShowFullName = (name: string) => {
  return name.length > MAX_NAME_LENGTH
}

const showCategoryName = (name: string) => {
  uni.showToast({
    title: name,
    icon: 'none',
  })
}

// 是否数据加载完毕
const isFinish = ref(false)
// 页面加载
onLoad(async () => {
  await Promise.all([getBannerData(), getCategoryTopData()])
  isFinish.value = true
})
</script>

<template>
  <view class="viewport" v-if="isFinish">
    <!-- 分类 -->
    <view class="categories">
      <!-- 左侧：一级分类 -->
      <scroll-view class="primary" scroll-y>
        <view
          v-for="(item, index) in categoryList"
          :key="item.id"
          class="item"
          :class="{ active: index === activeIndex }"
          @tap="getSubCategoryData(index)"
          @longpress="shouldShowFullName(item.name) && showCategoryName(item.name)"
        >
          <text class="name" :title="shouldShowFullName(item.name) ? item.name : ''">
            {{ item.name }}
          </text>
        </view>
      </scroll-view>
      <!-- 右侧：二级分类 -->
      <scroll-view enable-back-to-top class="secondary" scroll-y>
        <!-- 焦点图 -->
        <XtxSwiper class="banner" :list="bannerList" />
        <!-- 内容区域 -->
        <view class="panel" v-for="item in subCategoryList" :key="item.id">
          <view class="title">
            <text class="name">{{ item.name }}</text>
            <navigator
              class="more"
              hover-class="none"
              :url="`/pages/search/index?categoryId=${encodeURIComponent(String(item.id))}&categoryName=${encodeURIComponent(item.name)}`"
              >全部</navigator
            >
          </view>
          <view class="section">
            <navigator
              v-for="goodsItem in item.goods"
              :key="goodsItem.id"
              class="goods"
              hover-class="none"
              :url="goodsDetailUrl(goodsItem.id)"
            >
              <image class="image" :src="formatSrc(goodsItem.picture)"></image>
              <view class="name ellipsis">{{ goodsItem.name }}</view>
              <view class="price">
                <text class="symbol">¥</text>
                <text class="number">{{ formatPrice(goodsItem.price) }}</text>
              </view>
            </navigator>
          </view>
        </view>
      </scroll-view>
    </view>
  </view>
  <PageSkeleton v-else />
</template>

<style lang="scss">
@use './styles/category.scss';
</style>
