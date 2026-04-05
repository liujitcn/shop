<script setup lang="ts">
import { defUserAddressService } from '@/api/app/user_address'
import { useAddressStore, useUserStore } from '@/stores'
import type { UserAddress } from '@/rpc/app/user_address'
import { onShow } from '@dcloudio/uni-app'
import { ref, onMounted } from 'vue' // 获取会员信息
const userStore = useUserStore()

// 修改地址
const addressStore = useAddressStore()

// 获取收货地址列表数据
const addressList = ref<UserAddress[]>([])
const selectedAddressId = ref<number | null>(null)
const getUserAddressData = async () => {
  const res = await defUserAddressService.ListUserAddress({})
  addressList.value = res.list || []
  if (addressStore.selectedAddress) {
    selectedAddressId.value = addressStore.selectedAddress.id
  }
}

// 修改收货地址
const onChangeAddress = (item: UserAddress) => {
  selectedAddressId.value = item.id
  addressStore.changeSelectedAddress(item)
}

// 初始化调用(页面显示)
onShow(() => {
  if (userStore.userInfo) {
    getUserAddressData()
  }
})
onMounted(() => {
  if (userStore.userInfo) {
    getUserAddressData()
  }
})
// 子调父
const emit = defineEmits<{
  (event: 'close'): void
}>()
</script>

<template>
  <view class="address-panel">
    <!-- 关闭按钮 -->
    <text class="close icon-close" @tap="emit('close')"></text>
    <!-- 标题 -->
    <view class="title">配送至</view>
    <!-- 内容 -->
    <view class="content">
      <view
        class="item"
        v-for="item in addressList"
        :key="item.id"
        @tap="onChangeAddress(item)"
        :class="{ selected: item.id === selectedAddressId }"
      >
        <view class="user">{{ item.receiver }} {{ item.contact }}</view>
        <view class="address">{{ item.address.join('-') }}-{{ item.detail }}</view>
        <text class="icon icon-checked" v-if="item.id === selectedAddressId"></text>
      </view>
    </view>
    <view class="footer">
      <view class="button primary">
        <navigator
          hover-class="none"
          :url="userStore.userInfo ? '/pagesMember/address/edit' : '/pages/login/login'"
          >新建地址</navigator
        >
      </view>
    </view>
  </view>
</template>

<style lang="scss">
.address-panel {
  padding: 0 30rpx;
  border-radius: 20rpx 20rpx 0 0;
  position: relative;
  background-color: #fff;
}

.title {
  line-height: 1;
  padding: 40rpx 0;
  text-align: center;
  font-size: 32rpx;
  font-weight: normal;
  border-bottom: 1rpx solid #ddd;
  color: #444;
}

.close {
  position: absolute;
  right: 24rpx;
  top: 24rpx;
}

.content {
  min-height: 300rpx;
  max-height: 540rpx;
  overflow: auto;
  padding: 20rpx;
  .item {
    padding: 30rpx 50rpx 30rpx 60rpx;
    background-size: 40rpx;
    background-repeat: no-repeat;
    background-position: 0 center;
    background-image: url(@/static/images/locate.png);
    position: relative;
  }
  .icon {
    color: #999;
    font-size: 40rpx;
    transform: translateY(-50%);
    position: absolute;
    top: 50%;
    right: 0;
  }
  // 新增选中样式
  .item.selected {
    background-color: #f5f5f5;
    border-radius: 20rpx;
  }

  // 调整图标位置
  .icon-checked {
    position: absolute;
    right: 20rpx;
    top: 50%;
    transform: translateY(-50%);
    color: #27ba9b;
    font-size: 40rpx;
  }

  // 调整项布局
  .item {
    position: relative;
    padding-right: 80rpx;
  }
  .icon-ring {
    color: #444;
  }
  .user {
    font-size: 28rpx;
    color: #444;
    font-weight: 500;
  }
  .address {
    font-size: 26rpx;
    color: #666;
  }
}

.footer {
  display: flex;
  justify-content: space-between;
  padding: 20rpx 0 40rpx;
  font-size: 28rpx;
  color: #444;

  .button {
    flex: 1;
    height: 72rpx;
    text-align: center;
    line-height: 72rpx;
    margin: 0 20rpx;
    color: #fff;
    border-radius: 72rpx;
  }

  .primary {
    color: #fff;
    background-color: #27ba9b;
  }

  .secondary {
    background-color: #ffa868;
  }
}
</style>
