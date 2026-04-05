<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import type { PageUserCollectRequest, UserCollect } from '@/rpc/app/user_collect'
import { defUserCollectService } from '@/api/app/user_collect'
import { formatSrc, formatPrice } from '@/utils'
// 分页参数
const pageParams: Required<PageUserCollectRequest> = {
  pageNum: 1,
  pageSize: 10,
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
  const list = res.list || []
  collectList.value.push(...list)
  // 分页条件
  if (collectList.value.length < res.total) {
    // 页码累加
    pageParams.pageNum++
  } else {
    finish.value = true
  }

  showCollectList.value = collectList.value.length > 0
}

// 组件挂载完毕
onLoad(async () => {
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
        await defUserCollectService.DeleteUserCollect({ value: id + '' })
        // 删除成功，界面中删除订单
        const index = collectList.value.findIndex((v) => v.id === id)
        collectList.value.splice(index, 1)
      }
    },
  })
}

// 切换首页
const goIndex = () => {
  uni.switchTab({ url: '/pages/index/index' }).then((r) => {
    console.log(r)
  })
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
            <navigator
              :url="`/pages/goods/goods?id=${item.goodsId}`"
              hover-class="none"
              class="navigator"
            >
              <image mode="aspectFill" class="picture" :src="formatSrc(item.picture)"></image>
              <view class="meta">
                <view class="name ellipsis">{{ item.name }}</view>
                <view class="desc ellipsis">{{ item.desc }}</view>
                <view class="price">
                  <text class="current-price">{{ formatPrice(item.price) }}</text>
                  <text v-if="item.joinPrice" class="join-price">{{
                    formatPrice(item.joinPrice)
                  }}</text>
                </view>
              </view>
            </navigator>
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
    <!-- 购物车空状态 -->
    <view class="collect-blank" v-else>
      <image src="/static/images/blank.png" class="image" />
      <text class="text">还没有收藏商品哦</text>
      <button class="button" @tap="goIndex()">去首页看看</button>
    </view>
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
      display: flex;
    }

    .picture {
      width: 170rpx;
      height: 170rpx;
    }

    .meta {
      flex: 1;
      display: flex;
      flex-direction: column;
      justify-content: space-between;
      margin-left: 20rpx;
    }

    .name {
      height: 72rpx;
      font-size: 26rpx;
      color: #444;
    }

    .desc {
      line-height: 1.8;
      padding: 0 15rpx;
      font-size: 24rpx;
      align-self: flex-start;
      border-radius: 4rpx;
      color: #888;
      background-color: #f7f7f8;
    }

    .price {
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

// 空状态
.collect-blank {
  display: flex;
  justify-content: center;
  align-items: center;
  flex-direction: column;
  height: 60vh;
  .image {
    width: 400rpx;
    height: 281rpx;
  }
  .text {
    color: #444;
    font-size: 26rpx;
    margin: 20rpx 0;
  }
  .button {
    width: 240rpx !important;
    height: 60rpx;
    line-height: 60rpx;
    margin-top: 20rpx;
    font-size: 26rpx;
    border-radius: 60rpx;
    color: #fff;
    background-color: #27ba9b;
  }
}
</style>
