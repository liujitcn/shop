<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import type { PageUserCollectRequest, UserCollect } from '@/rpc/shop/app/v1/user_collect'
import { defUserCollectService } from '@/api/shop/app/user_collect'
import { formatSrc, formatPrice } from '@/utils'
import {
  goodsDetailUrl,
  navigateToLogin,
  switchTabToHome,
  tenantStoreUrl,
} from '@/utils/navigation'
import { useUserStore } from '@/stores'

const userStore = useUserStore()
// 分页参数
const pageParams: PageUserCollectRequest = {
  page_num: 1,
  page_size: 10,
}
// 猜你喜欢的列表
const collectList = ref<UserCollect[]>([])
// 优化空列表状态，默认展示列表
const showCollectList = ref(false)
// 已结束标记
const finish = ref(false)
// 获取数据
const getCollectData = async () => {
  // 退出分页判断
  if (finish.value === true) {
    return uni.showToast({ icon: 'none', title: '没有更多数据~' })
  }
  const res = await defUserCollectService.PageUserCollect(pageParams)
  // 数组追加
  const list = res.user_collects || []
  collectList.value.push(...list)
  // 分页条件
  if (collectList.value.length < res.total) {
    // 页码累加
    pageParams.page_num++
  } else {
    finish.value = true
  }

  showCollectList.value = collectList.value.length > 0
}

// 组件挂载完毕
onLoad(async () => {
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin()
    return
  }
  await getCollectData()
})

// 滚动触底
const onScrollToLower = async () => {
  await getCollectData()
}

// 点击删除按钮
const onDeleteCollect = (id: number) => {
  // 弹窗二次确认
  uni.showModal({
    content: '是否删除',
    confirmColor: '#27BA9B',
    success: async (res) => {
      if (res.confirm) {
        // 后端删除单品
        await defUserCollectService.DeleteUserCollect({ ids: id + '' })
        // 删除成功，界面中删除订单
        const index = collectList.value.findIndex((v) => v.id === id)
        collectList.value.splice(index, 1)
        showCollectList.value = collectList.value.length > 0
      }
    },
  })
}

// 切换首页
const goIndex = () => {
  void switchTabToHome()
}

const goStore = (storeID: number) => {
  void uni.navigateTo({ url: tenantStoreUrl(storeID) })
}
</script>

<template>
  <scroll-view enable-back-to-top scroll-y class="scroll-view" @scrolltolower="onScrollToLower">
    <!-- 购物车列表 -->
    <view class="collect-list" v-if="showCollectList">
      <uni-swipe-action>
        <!-- 滑动操作项 -->
        <uni-swipe-action-item v-for="item in collectList" :key="item.id" class="collect-swipe">
          <!-- 商品信息 -->
          <view class="goods">
            <navigator :url="goodsDetailUrl(item.goods_id)" hover-class="none" class="navigator">
              <image mode="aspectFill" class="picture" :src="formatSrc(item.picture)"></image>
              <view class="meta">
                <view class="name">{{ item.name }}</view>
                <view class="price">
                  <text class="current-price">{{ formatPrice(item.price) }}</text>
                  <text v-if="item.join_price" class="join-price">{{
                    formatPrice(item.join_price)
                  }}</text>
                </view>
              </view>
            </navigator>
            <view
              v-if="item.tenant_store?.id"
              class="store-entry ellipsis"
              @tap.stop="goStore(item.tenant_store.id)"
            >
              {{ item.tenant_store.name }}<text class="store-arrow">&gt;</text>
            </view>
          </view>
          <!-- 右侧删除按钮 -->
          <template #right>
            <view class="collect-swipe-right">
              <button @click="onDeleteCollect(item.id)" class="button delete-button">删除</button>
            </view>
          </template>
        </uni-swipe-action-item>
      </uni-swipe-action>
    </view>
    <!-- 收藏空状态 -->
    <EmptyState
      v-else
      image="/static/images/empty_collect.png"
      text="还没有收藏商品哦"
      min-height="60vh"
      button-text="去首页看看"
      @action="goIndex"
    />
  </scroll-view>
</template>

<style lang="scss">
page {
  height: 100%;
  background-color: #f4f4f4;
}

// 滚动容器
.scroll-view {
  flex: 1;
  background-color: #f7f7f8;
}

// 购物车列表
.collect-list {
  padding: 0 20rpx;

  // 购物车商品
  .goods {
    display: flex;
    padding: 20rpx;
    border-radius: 10rpx;
    background-color: #fff;
    position: relative;

    .navigator {
      display: block;
      width: 100%;

      .navigator-wrap {
        display: flex;
        width: 100%;
        align-items: flex-start;
        min-width: 0;
      }
    }

    .picture {
      flex-shrink: 0;
      width: 170rpx;
      height: 170rpx;
    }

    .meta {
      box-sizing: border-box;
      height: 170rpx;
      min-width: 0;
      flex: 1;
      display: flex;
      flex-direction: column;
      margin-left: 20rpx;
      padding-bottom: 44rpx;
    }

    .store-entry {
      position: absolute;
      z-index: 1;
      left: 210rpx;
      right: 20rpx;
      bottom: 20rpx;
      line-height: 32rpx;
      color: #555;
      font-size: 24rpx;
    }

    .store-arrow {
      margin-left: 8rpx;
      color: #999;
    }

    .name {
      display: -webkit-box;
      overflow: hidden;
      line-height: 36rpx;
      font-size: 26rpx;
      color: #444;
      -webkit-box-orient: vertical;
      -webkit-line-clamp: 2;
    }

    .price {
      margin-top: auto;
      display: flex;
      align-items: center;
      gap: 8rpx;
      font-size: 26rpx;

      .current-price {
        color: #cf4444;

        &::before {
          content: '￥';
          font-size: 80%;
        }
      }

      .join-price {
        color: #999;
        text-decoration: line-through;
        font-size: 20rpx;
        position: relative;
        top: 2rpx;

        &::before {
          content: '￥';
          font-size: 80%;
        }
      }
    }
  }

  .collect-swipe {
    display: block;
    margin: 20rpx 0;
  }

  .collect-swipe-right {
    display: flex;
    height: 100%;

    .button {
      display: flex;
      justify-content: center;
      align-items: center;
      width: 50px;
      padding: 6px;
      line-height: 1.5;
      color: #fff;
      font-size: 26rpx;
      border-radius: 0;
    }

    .delete-button {
      background-color: #cf4444;
    }
  }
}
</style>
