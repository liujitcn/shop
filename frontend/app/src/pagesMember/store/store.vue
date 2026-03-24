<script setup lang="ts">
import { defUserStoreService } from '@/api/app/user_store'
import { onLoad } from '@dcloudio/uni-app'
import type { FileInfo } from "@/rpc/base/file";
import { computed, ref } from 'vue'
import { defBaseAreaService } from '@/api/app/base_area'
import type { AppTreeOptionResponse_Option } from '@/rpc/common/common'
import { UserStoreStatus } from '@/rpc/common/enum'
import type { UserStore, UserStoreForm } from '@/rpc/app/user_store'
import { getFileInfo, multiUploadFile, uploadFileList } from '@/utils/file.ts'

const imageStyles = {
  width: 100,
  height: 100,
  border: {
    color: '#eee',
    width: '1px',
    style: 'solid',
    radius: '3px',
  },
}

const imgMaxSize = ref(1024 * 1024)
const picture = computed(() => {
  let arr: FileInfo[] = []
  form.value.picture?.map((url) => {
    const fileInfo = getFileInfo(url)
    arr.push(fileInfo)
  })
  return arr
})
const businessLicense = computed(() => {
  let arr: FileInfo[] = []
  form.value.businessLicense?.map((url) => {
    const fileInfo = getFileInfo(url)
    arr.push(fileInfo)
  })
  return arr
})

// 表单数据
const detail = ref<UserStore>({
  /** 用户门店ID */
  id: 0,
  /** 门店名称 */
  name: '',
  /** 省市区 */
  address: [],
  /** 详细地址 */
  detail: '',
  /** 门店照片 */
  picture: [],
  /** 营业执照 */
  businessLicense: [],
  /** 省市区 */
  addressName: [],
  /** 状态 */
  status: UserStoreStatus.UNKNOWN_USS,
  /** 备注名 */
  remark: '',
})

// 表单数据
const form = ref<UserStoreForm>({
  /** 用户门店ID */
  id: 0,
  /** 门店名称 */
  name: '',
  /** 省市区 */
  address: [],
  /** 详细地址 */
  detail: '',
  /** 门店照片 */
  picture: [],
  /** 营业执照 */
  businessLicense: [],
  /** 省市区 */
  addressName: [],
})

const localData = ref<AppTreeOptionResponse_Option[]>([])
// 页面加载
onLoad(async () => {
  // #ifdef H5 || APP-PLUS
  const tree = await defBaseAreaService.TreeBaseArea({})
  localData.value = tree.list || []
  // #endif
  // 发送请求
  detail.value = await defUserStoreService.GetUserStore({})
  // 把数据合并到表单中
  Object.assign(form.value, detail.value)
})

// 图片选择处理
const handlePictureSelect = async (files: any) => {
  const tempFiles = files.tempFiles
  // #ifdef H5 || APP-PLUS
  const validFiles = tempFiles.filter((file: any) => file.size <= imgMaxSize.value)
  if (validFiles.length !== tempFiles.length) {
    await uni.showToast({
      title: '请上传小于1M的照片',
      icon: 'none',
      duration: 1500,
    })
  }
  if (validFiles.length === 0) {
    return
  }
  const multiRes = await multiUploadFile('store', validFiles)
  multiRes.map((file) => {
    form.value.picture.push(file.url)
  })
  // #endif
  // #ifdef MP-WEIXIN
  let filePaths: string[] = []
  let hasOversize = false
  tempFiles.map((file: any) => {
    const { path, size } = file
    if (size > imgMaxSize.value) {
      hasOversize = true
      return
    }
    filePaths.push(path)
  })
  if (hasOversize) {
    await uni.showToast({
      title: '请上传小于1M的照片',
      icon: 'none',
      duration: 1500,
    })
  }
  if (filePaths.length === 0) {
    return
  }
  const res = await uploadFileList('store', filePaths)
  res.map((file) => {
    form.value.picture.push(file.url)
  })
  // #endif
}

// 图片删除处理
const handlePictureDelete = (file: any) => {
  const tempFilePath = file.tempFilePath
  const index = form.value.picture.findIndex((item) => tempFilePath.endsWith(item))
  if (index !== -1) {
    form.value.picture.splice(index, 1) // 同步更新 fileList
  }
}

// 营业执照选择
const handleBusinessLicenseSelect = async (files: any) => {
  const tempFiles = files.tempFiles
  // #ifdef H5 || APP-PLUS
  const validFiles = tempFiles.filter((file: any) => file.size <= imgMaxSize.value)
  if (validFiles.length !== tempFiles.length) {
    await uni.showToast({
      title: '请上传小于1M的照片',
      icon: 'none',
      duration: 1500,
    })
  }
  if (validFiles.length === 0) {
    return
  }
  const multiRes = await multiUploadFile('store', validFiles)
  multiRes.map((file) => {
    form.value.businessLicense.push(file.url)
  })
  // #endif
  // #ifdef MP-WEIXIN
  let filePaths: string[] = []
  let hasOversize = false
  tempFiles.map((file: any) => {
    const { path, size } = file
    if (size > imgMaxSize.value) {
      hasOversize = true
      return
    }
    filePaths.push(path)
  })
  if (hasOversize) {
    await uni.showToast({
      title: '请上传小于1M的照片',
      icon: 'none',
      duration: 1500,
    })
  }
  if (filePaths.length === 0) {
    return
  }
  const res = await uploadFileList('store', filePaths)
  res.map((file) => {
    form.value.businessLicense.push(file.url)
  })
  // #endif
}

// 营业执照删除
const handleBusinessLicenseDelete = (file: any) => {
  const tempFilePath = file.tempFilePath
  const index = form.value.businessLicense.findIndex((item) => tempFilePath.endsWith(item))
  if (index !== -1) {
    form.value.businessLicense.splice(index, 1) // 同步更新 fileList
  }
}

// 收集所在地区
const onRegionChange: UniHelper.RegionPickerOnChange = (ev) => {
  // 省市区(前端展示)
  form.value.addressName = ev.detail.value
  // 省市区(后端参数)
  Object.assign(form.value, { address: ev.detail.code })
}

// 定义校验规则
const rules: UniHelper.UniFormsRules = {
  name: {
    rules: [{ required: true, errorMessage: '请输入门店名称' }],
  },
  address: {
    rules: [{ required: true, errorMessage: '请选择所在地区' }],
  },
  detail: {
    rules: [{ required: true, errorMessage: '请选择详细地址' }],
  },
  picture: {
    rules: [{ required: true, errorMessage: '请上传门店照片' }],
  },
  businessLicense: {
    rules: [{ required: true, errorMessage: '请上传营业执照' }],
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
    if (form.value.id) {
      // 修改门店请求
      await defUserStoreService.UpdateUserStore(form.value)
    } else {
      // 新建门店请求
      await defUserStoreService.CreateUserStore(form.value)
    }
    // 成功提示
    await uni.showToast({ icon: 'success', title: '保存成功' })
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
  <!-- 新增审核状态提示 -->
  <view v-if="detail.status && detail.status !== UserStoreStatus.APPROVED" class="audit-status">
    <view v-if="detail.status === UserStoreStatus.PENDING_REVIEW" class="status-item auditing">
      <uni-icons type="info" size="16" color="#fff" />
      <text>资料审核中，请耐心等待...</text>
    </view>
    <view v-if="detail.status === UserStoreStatus.FAILED_REVIEW" class="status-item rejected">
      <uni-icons type="close" size="16" color="#fff" />
      <text>审核未通过：{{ detail.remark || '请修改后重新提交' }}</text>
    </view>
  </view>
  <view class="content">
    <uni-forms ref="formRef" :rules="rules" :model="form">
      <!-- 表单内容 -->
      <uni-forms-item name="receiver" class="form-item">
        <text class="label"><text class="is-required">*</text>门店名称</text>
        <input
          v-model="form.name"
          class="input"
          :disabled="detail.status === UserStoreStatus.PENDING_REVIEW"
          placeholder="请填写门店名称"
        />
      </uni-forms-item>
      <uni-forms-item name="countyCode" class="form-item">
        <text class="label"><text class="is-required">*</text>所在地区</text>
        <!-- #ifdef MP-WEIXIN -->
        <picker
          class="picker"
          mode="region"
          :value="form.address"
          :disabled="detail.status === UserStoreStatus.PENDING_REVIEW"
          @change="onRegionChange"
        >
          <view v-if="form.addressName.length">{{ form.addressName.join('-') }}</view>
          <view v-else class="placeholder">请选择省/市/区(县)</view>
        </picker>
        <!-- #endif -->

        <!-- #ifdef H5 || APP-PLUS -->
        <uni-data-picker
          v-model="form.address"
          :localdata="localData"
          placeholder="请选择地址"
          popup-title="请选择城市"
          :clear-icon="false"
          :readonly="detail.status === UserStoreStatus.PENDING_REVIEW"
          @change="onCityChange"
        />
        <!-- #endif -->
      </uni-forms-item>
      <uni-forms-item name="address" class="form-item">
        <text class="label"><text class="is-required">*</text>详细地址</text>
        <input
          v-model="form.detail"
          class="input"
          placeholder="街道、楼牌号等信息"
          :disabled="detail.status === UserStoreStatus.PENDING_REVIEW"
        />
      </uni-forms-item>
      <!-- 门店照片（多图） -->
      <uni-forms-item name="picture" class="form-item">
        <text class="label"><text class="is-required">*</text>门店照片</text>
        <uni-file-picker
          v-model="picture"
          fileMediatype="image"
          mode="grid"
          limit="9"
          :image-styles="imageStyles"
          :readonly="detail.status === UserStoreStatus.PENDING_REVIEW"
          @select="handlePictureSelect"
          @delete="handlePictureDelete"
        />
      </uni-forms-item>
      <!-- 营业执照（单图） -->
      <uni-forms-item name="businessLicense" class="form-item">
        <text class="label"><text class="is-required">*</text>营业执照</text>
        <uni-file-picker
          v-model="businessLicense"
          fileMediatype="image"
          mode="grid"
          limit="9"
          :image-styles="imageStyles"
          :readonly="detail.status === UserStoreStatus.PENDING_REVIEW"
          @select="handleBusinessLicenseSelect"
          @delete="handleBusinessLicenseDelete"
        />
      </uni-forms-item>
    </uni-forms>
  </view>
  <!-- 提交按钮 -->
  <button v-if="detail.status !== UserStoreStatus.PENDING_REVIEW" class="button" @tap="onSubmit">
    保存
  </button>
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
.audit-status {
  padding: 20rpx 30rpx;
  .status-item {
    display: flex;
    align-items: center;
    padding: 20rpx;
    border-radius: 8rpx;
    color: #fff;
    font-size: 26rpx;

    &.auditing {
      background: #f0ad4e;
    }
    &.rejected {
      background: #dd524d;
    }

    uni-icons {
      margin-right: 10rpx;
    }
  }
}
.content {
  margin: 20rpx 20rpx 0;
  padding: 0 20rpx;
  border-radius: 10rpx;
  background-color: #fff;

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
