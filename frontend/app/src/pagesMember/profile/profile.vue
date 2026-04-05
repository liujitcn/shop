<script setup lang="ts">
import { defAuthService } from '@/api/app/auth'

import { useUserStore } from '@/stores'
import type { UserInfo } from '@/rpc/app/auth'
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import type { ListBaseDictResponse_DictItem } from '@/rpc/app/base_dict'
import { defBaseDictService } from '@/api/app/base_dict'
import { formatSrc } from '@/utils'

const userStore = useUserStore()

// 获取屏幕边界到安全区域距离
const { safeAreaInsets } = uni.getSystemInfoSync()

const imgMaxSize = ref(1024 * 1024)
const staticApiBasePath = import.meta.env.VITE_APP_STATIC_API || '/shop'

// 获取个人信息，修改个人信息需提供初始值
const userInfo = ref({} as UserInfo)
const getUserData = async () => {
  const res = await defAuthService.GetUserInfo({})
  userInfo.value = res
  // 同步 Store 的头像和昵称，用于我的页面展示
  userStore.userInfo!.userName = res.userName
  userStore.userInfo!.nickName = res.nickName
  userStore.userInfo!.gender = res.gender
  userStore.userInfo!.phone = res.phone
  userStore.userInfo!.avatar = res.avatar
}

const genderList = ref<ListBaseDictResponse_DictItem[]>([])

const getDictData = async () => {
  const gender = 'base_user_gender'
  const res = await defBaseDictService.ListBaseDict({
    value: gender,
  })
  const list = res.list || []
  list.map((item) => {
    switch (item.code) {
      case gender:
        genderList.value = item.items || []
        break
    }
  })
}

onLoad(() => {
  Promise.all([getUserData(), getDictData()])
})
// 修改头像
const onAvatarChange = async () => {
  // 调用拍照/选择图片
  // 选择图片条件编译
  // #ifdef H5 || APP-PLUS
  // 微信小程序从基础库 2.21.0 开始， wx.chooseImage 停止维护，请使用 uni.chooseMedia 代替
  uni.chooseImage({
    count: 1,
    success: async (res: any) => {
      const { path, size } = res.tempFiles[0]
      if (size > imgMaxSize.value) {
        await uni.showToast({
          title: '请上传小于1M的照片',
          icon: 'none',
          duration: 1500,
        })
        return
      }
      // 上传
      await uploadFile(path)
    },
  })
  // #endif

  // #ifdef MP-WEIXIN
  // uni.chooseMedia 仅支持微信小程序端
  uni.chooseMedia({
    // 文件个数
    count: 1,
    // 文件类型
    mediaType: ['image'],
    success: async (res: any) => {
      // 本地路径
      const { tempFilePath, size } = res.tempFiles[0]
      if (size > imgMaxSize.value) {
        await uni.showToast({
          title: '请上传小于1M的照片',
          icon: 'none',
          duration: 1500,
        })
        return
      }
      await uploadFile(tempFilePath)
    },
  })
  // #endif
}

// 文件上传-兼容小程序端、H5端、App端
const uploadFile = async (file: string) => {
  // 文件上传
  uni.uploadFile({
    url: '/file',
    name: 'file',
    filePath: file,
    formData: {
      basePath: staticApiBasePath,
      fileType: 'avatar',
    },
    success: async (res) => {
      if (res.statusCode === 200) {
        const fileInfo = JSON.parse(res.data)
        // 更新用户头像
        userInfo.value.avatar = fileInfo.url
        // 更新用户信息
        await defAuthService.UpdateUserInfo(userInfo.value)
        userStore.userInfo!.avatar = userInfo.value.avatar
        await uni.showToast({ icon: 'success', title: '更新成功' })
      } else {
        await uni.showToast({ icon: 'error', title: '上传头像失败' })
      }
    },
  })
}

// 修改性别
const onGenderChange: UniHelper.RadioGroupOnChange = (ev) => {
  userInfo.value.gender = Number(ev.detail.value)
}

// #ifdef MP-WEIXIN
// 新增授权手机号处理
const onGetPhoneNumber: UniHelper.ButtonOnGetphonenumber = async (e) => {
  if (e.detail.errMsg !== 'getPhoneNumber:ok') return
  const res = await defAuthService.PhoneAuth({ code: e.detail.code || '' })
  userInfo.value.phone = res.phone
  await uni.showToast({ icon: 'success', title: '授权成功' })
}

// #endif

// 点击保存提交表单
const onSubmit = async () => {
  const { nickName, gender } = userInfo.value
  await defAuthService.UpdateUserInfo({
    nickName: nickName,
    gender: gender,
    avatar: userInfo.value.avatar,
  })
  // 更新Store昵称
  userStore.userInfo!.nickName = nickName
  userStore.userInfo!.gender = gender
  await uni.showToast({ icon: 'success', title: '保存成功' })
  setTimeout(() => {
    uni.navigateBack()
  }, 400)
}
</script>

<template>
  <view class="viewport">
    <!-- 导航栏 -->
    <view class="navbar" :style="{ paddingTop: safeAreaInsets?.top + 'px' }">
      <navigator open-type="navigateBack" class="back icon-left" hover-class="none"></navigator>
      <view class="title">个人信息</view>
    </view>
    <view class="avatar">
      <view @tap="onAvatarChange" class="avatar-content">
        <image
          v-if="userInfo?.avatar"
          class="image"
          :src="formatSrc(userInfo?.avatar)"
          mode="aspectFill"
        />
        <image v-else class="image" src="@/static/images/avatar.png" mode="aspectFill"></image>
        <text class="text">点击修改头像</text>
      </view>
    </view>
    <!-- 表单 -->
    <view class="form">
      <!-- 表单内容 -->
      <view class="form-content">
        <view class="form-item" v-if="userInfo?.userName">
          <text class="label">账号</text>
          <text class="account placeholder">{{ userInfo?.userName }}</text>
        </view>
        <!-- #ifdef MP-WEIXIN -->
        <!-- 手机号 -->
        <view class="form-item">
          <text class="label">手机号</text>
          <view class="input">
            <text v-if="userInfo.phone" class="account">{{ userInfo.phone }}</text>
            <button
              v-else
              class="auth-button"
              open-type="getPhoneNumber"
              @getphonenumber="onGetPhoneNumber"
            >
              微信授权手机号
            </button>
          </view>
        </view>
        <!-- #endif -->
        <view class="form-item">
          <text class="label">昵称</text>
          <input class="input" type="text" placeholder="请填写昵称" v-model="userInfo.nickName" />
        </view>
        <view class="form-item">
          <text class="label">性别</text>
          <radio-group @change="onGenderChange">
            <label class="radio" v-for="(item, index) in genderList" :key="index">
              <radio
                :value="item.value"
                color="#27ba9b"
                :checked="userInfo?.gender === Number(item.value)"
              />
              {{ item.label }}
            </label>
          </radio-group>
        </view>
      </view>
      <!-- 提交按钮 -->
      <button @tap="onSubmit" class="form-button">保 存</button>
    </view>
  </view>
</template>

<style lang="scss">
page {
  background-color: #f4f4f4;
}

.viewport {
  display: flex;
  flex-direction: column;
  height: 100%;
  background-image: url(@/static/images/navigator_bg.png);
  background-size: auto 420rpx;
  background-repeat: no-repeat;
}

// 导航栏
.navbar {
  position: relative;

  .title {
    height: 40px;
    display: flex;
    justify-content: center;
    align-items: center;
    font-size: 16px;
    font-weight: 500;
    color: #fff;
  }

  .back {
    position: absolute;
    height: 40px;
    width: 40px;
    left: 0;
    font-size: 20px;
    color: #fff;
    display: flex;
    justify-content: center;
    align-items: center;
  }
}

// 头像
.avatar {
  text-align: center;
  width: 100%;
  height: 260rpx;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;

  .image {
    width: 160rpx;
    height: 160rpx;
    border-radius: 50%;
    background-color: #eee;
  }

  .text {
    display: block;
    padding-top: 20rpx;
    line-height: 1;
    font-size: 26rpx;
    color: #fff;
  }
}

// 表单
.form {
  background-color: #f4f4f4;

  &-content {
    margin: 20rpx 20rpx 0;
    padding: 0 20rpx;
    border-radius: 20rpx;
    background-color: #fff;
    box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.04);
  }

  &-item {
    display: flex;
    height: 96rpx;
    line-height: 46rpx;
    padding: 25rpx 10rpx;
    background-color: #fff;
    font-size: 28rpx;
    border-bottom: 1rpx solid #ddd;

    &:last-child {
      border: none;
    }

    .label {
      width: 180rpx;
      color: #333;
    }

    .account {
      color: #666;
    }

    .input {
      flex: 1;
      display: block;
      height: 46rpx;
    }

    .radio {
      margin-right: 20rpx;
    }

    .picker {
      flex: 1;
    }
    .placeholder {
      color: #808080;
    }
  }

  &-button {
    height: 80rpx;
    text-align: center;
    line-height: 80rpx;
    margin: 30rpx 20rpx;
    color: #fff;
    border-radius: 80rpx;
    font-size: 30rpx;
    background-color: #27ba9b;
  }
}
.auth-button {
  height: 60rpx;
  line-height: 60rpx;
  margin: 0;
  padding: 0 20rpx;
  font-size: 26rpx;
  color: #27ba9b;
  border: 1rpx solid #27ba9b;
  border-radius: 30rpx;
  background: none;

  &::after {
    border: none;
  }
}
</style>
