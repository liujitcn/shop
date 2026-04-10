<script setup lang="ts">
import { defRecommendService } from '@/api/app/recommend'
import { defOrderService } from '@/api/app/order_info.ts'
import { useGuessList } from '@/composables'
import { onLoad } from '@dcloudio/uni-app'
import { useRecommendStore } from '@/stores'
import { RecommendGoodsActionType, RecommendScene } from '@/rpc/common/enum'

// 获取页面参数
const query = defineProps<{
  id: string
}>()

// 猜你喜欢
const { guessRef, onScrollToLower } = useGuessList()
const recommendStore = useRecommendStore()

// 页面加载
onLoad(() => {
  void (async () => {
    const res = await defOrderService.GetOrderInfoById({
      value: Number(query.id),
    })
    const goodsItems =
      res.order?.goods.map((item) => ({
        goodsId: item.goodsId,
        goodsNum: item.num,
        recommendContext: {
          scene: item.scene,
          requestId: item.requestId,
          position: item.position,
        },
      })) || []
    if (goodsItems.length === 0) {
      return
    }
    await recommendStore.getAnonymousId()
    await defRecommendService.RecommendGoodsActionReport({
      eventType: RecommendGoodsActionType.RECOMMEND_GOODS_ACTION_ORDER_PAY,
      goodsItems,
    })
  })()
})
</script>

<template>
  <scroll-view enable-back-to-top class="viewport" scroll-y @scrolltolower="onScrollToLower">
    <!-- 订单状态 -->
    <view class="overview">
      <view class="status icon-checked">支付成功</view>
      <view class="tips">提示: 请后续关注发货信息，有问题及时联系</view>
      <view class="buttons">
        <navigator
          hover-class="none"
          class="button navigator"
          url="/pages/index/index"
          open-type="switchTab"
        >
          返回首页
        </navigator>
        <navigator
          hover-class="none"
          class="button navigator"
          :url="`/pagesOrder/detail/detail?id=${query.id}&internal=true`"
          open-type="redirect"
        >
          查看订单
        </navigator>
      </view>
    </view>

    <!-- 猜你喜欢 -->
    <XtxGuess
      ref="guessRef"
      title="顺手再带两件"
      :scene="RecommendScene.ORDER_PAID"
      :order-id="Number(query.id)"
    />
  </scroll-view>
</template>

<style lang="scss">
page {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.viewport {
  background-color: #f7f7f8;
}

.overview {
  line-height: 1;
  padding: 50rpx 0;
  color: #fff;
  background-color: #27ba9b;

  .tips {
    width: 70%;
    font-size: 24rpx;
    text-align: center;
    line-height: 1.5;
    margin: 60rpx auto;
  }

  .status {
    font-size: 36rpx;
    font-weight: 500;
    text-align: center;
  }

  .status::before {
    display: block;
    font-size: 110rpx;
    margin-bottom: 20rpx;
  }

  .buttons {
    height: 60rpx;
    line-height: 60rpx;
    display: flex;
    justify-content: center;
    align-items: center;
    margin-top: 60rpx;
  }

  .button {
    text-align: center;
    margin: 0 10rpx;
    font-size: 28rpx;
    color: #fff;

    &:first-child {
      width: 200rpx;
      border-radius: 64rpx;
      border: 1rpx solid #fff;
    }
  }
}
</style>
