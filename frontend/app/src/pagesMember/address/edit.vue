<script setup lang="ts">
import { defUserAddressService } from '@/api/app/user_address'
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { defBaseAreaService } from '@/api/app/base_area'
import type { AppTreeOptionResponse_Option } from '@/rpc/common/common'
import type { UserAddressForm } from '@/rpc/app/user_address'
// 表单数据
const form = ref<UserAddressForm>({
  /** 用户地址ID */
  id: 0,
  /** 联系人 */
  receiver: '',
  /** 联系方式 */
  contact: '',
  /** 省市区 */
  address: [],
  /** 详细地址 */
  detail: '',
  /** 省市区 */
  addressName: [],
  /** 状态 */
  isDefault: false,
})

// 获取页面参数
const query = defineProps<{
  id?: string
}>()

const localData = ref<AppTreeOptionResponse_Option[]>([])
// 页面加载
onLoad(async () => {
  // #ifdef H5 || APP-PLUS
  const tree = await defBaseAreaService.TreeBaseArea({})
  localData.value = tree.list || []
  // #endif
  if (query.id) {
    // 发送请求
    const res = await defUserAddressService.GetUserAddress({
      value: Number(query.id),
    })
    // 把数据合并到表单中
    Object.assign(form.value, res)
  }
})

// 动态设置标题
uni.setNavigationBarTitle({ title: query.id ? '修改地址' : '新建地址' })

// 收集所在地区
const onRegionChange: UniHelper.RegionPickerOnChange = (ev) => {
  // 省市区(前端展示)
  form.value.addressName = ev.detail.value
  // 省市区(后端参数)
  Object.assign(form.value, { address: ev.detail.code })
}

// 收集是否默认收货地址
const onSwitchChange: UniHelper.SwitchOnChange = (ev) => {
  form.value.isDefault = ev.detail.value
}

// 定义校验规则
const rules: UniHelper.UniFormsRules = {
  receiver: {
    rules: [{ required: true, errorMessage: '请输入收货人姓名' }],
  },
  contact: {
    rules: [
      { required: true, errorMessage: '请输入联系方式' },
      { pattern: /^1[3-9]\d{9}$/, errorMessage: '手机号格式不正确' },
    ],
  },
  address: {
    rules: [{ required: true, errorMessage: '请选择所在地区' }],
  },
  detail: {
    rules: [{ required: true, errorMessage: '请选择详细地址' }],
  },
}

// 表单组件实例
const formRef = ref<UniHelper.UniFormsInstance>()

// 提交表单
const onSubmit = async () => {
  try {
    // 表单校验
    await formRef.value?.validate?.()
    // 校验通过后再发送请求
    if (query.id) {
      // 修改地址请求
      await defUserAddressService.UpdateUserAddress(form.value)
    } else {
      // 新建地址请求
      await defUserAddressService.CreateUserAddress(form.value)
    }
    // 成功提示
    await uni.showToast({ icon: 'success', title: query.id ? '修改成功' : '添加成功' })
    // 返回上一页
    setTimeout(() => {
      uni.navigateBack()
    }, 400)
  } catch (error) {
    await uni.showToast({ icon: 'error', title: '请填写完整信息' })
  }
}

// #ifdef H5 || APP-PLUS
const onCityChange: UniHelper.UniDataPickerOnChange = (ev) => {
  const address = ev.detail.value.map((item) => item.value)
  // 收集后端所需的 code 数据
  Object.assign(form.value, { address: address })
}
// #endif
</script>

<template>
  <view class="content">
    <uni-forms :rules="rules" :model="form" ref="formRef">
      <!-- 表单内容 -->
      <uni-forms-item name="receiver" class="form-item">
        <text class="label"><text class="is-required">*</text>收货人</text>
        <input class="input" placeholder="请填写收货人姓名" v-model="form.receiver" />
      </uni-forms-item>
      <uni-forms-item name="contact" class="form-item">
        <text class="label"><text class="is-required">*</text>手机号码</text>
        <input
          class="input"
          placeholder="请填写收货人手机号码"
          :maxlength="11"
          v-model="form.contact"
        />
      </uni-forms-item>
      <uni-forms-item name="address" class="form-item">
        <text class="label"><text class="is-required">*</text>所在地区</text>
        <!-- #ifdef MP-WEIXIN -->
        <picker @change="onRegionChange" class="picker" mode="region" :value="form.address">
          <view v-if="form.addressName.length">{{ form.addressName.join('-') }}</view>
          <view v-else class="placeholder">请选择省/市/区(县)</view>
        </picker>
        <!-- #endif -->

        <!-- #ifdef H5 || APP-PLUS -->
        <uni-data-picker
          :localdata="localData"
          placeholder="请选择地址"
          popup-title="请选择城市"
          @change="onCityChange"
          :clear-icon="false"
          v-model="form.address"
        />
        <!-- #endif -->
      </uni-forms-item>
      <uni-forms-item name="detail" class="form-item">
        <text class="label"><text class="is-required">*</text>详细地址</text>
        <input class="input" placeholder="街道、楼牌号等信息" v-model="form.detail" />
      </uni-forms-item>
      <view class="form-item">
        <label class="label"><text class="is-required">*</text>设为默认地址</label>
        <switch @change="onSwitchChange" class="switch" color="#27ba9b" :checked="form.isDefault" />
      </view>
    </uni-forms>
  </view>
  <!-- 提交按钮 -->
  <button @tap="onSubmit" class="button">保存并使用</button>
</template>

<style lang="scss">
// 深度选择器修改 uni-data-picker 组件样式
:deep(.selected-area) {
  flex: 0 1 auto;
  height: auto;
}

page {
  background-color: #f4f4f4;
}

.content {
  margin: 20rpx 20rpx 0;
  padding: 0 20rpx;
  border-radius: 20rpx;
  background-color: #fff;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.04);

  .form-item,
  .uni-forms-item {
    display: flex;
    align-items: center;
    min-height: 96rpx;
    padding: 25rpx 10rpx;
    background-color: #fff;
    font-size: 28rpx;
    border-bottom: 1rpx solid #ddd;
    position: relative;
    margin-bottom: 0;

    // 调整 uni-forms 样式
    .uni-forms-item__content {
      display: flex;
    }

    .uni-forms-item__error {
      margin-left: 200rpx;
    }

    &:last-child {
      border: none;
    }

    .label {
      width: 200rpx;
      color: #333;
    }

    .input {
      flex: 1;
      display: block;
      height: 46rpx;
    }

    .switch {
      position: absolute;
      right: -20rpx;
      transform: scale(0.8);
    }

    .picker {
      flex: 1;
    }

    .placeholder {
      color: #808080;
    }
  }
}

.button {
  height: 80rpx;
  margin: 30rpx 20rpx;
  color: #fff;
  border-radius: 80rpx;
  font-size: 30rpx;
  background-color: #27ba9b;
}
</style>
