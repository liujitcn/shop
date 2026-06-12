<script setup lang="ts">
import type { AiAssistantAction } from '@/rpc/base/v1/ai_assistant_message'
import type { AppTreeOptionResponse_Option } from '@/rpc/common/v1/common'
import { formatPrice, formatSrc } from '@/utils/index'
import OrderListBlock from './OrderListBlock.vue'

type AssistantFlowBlock = {
  type: string
  [key: string]: any
}

type AddressFormStepKey = 'receiver' | 'contact' | 'address' | 'detail'

type AddressFormStep = {
  key: AddressFormStepKey
  label: string
  shortLabel: string
  hint: string
  placeholder: string
}

type FlowMessage = {
  blocks: AssistantFlowBlock[]
  flowReveal?: Record<string, number>
}

const props = defineProps<{
  message: FlowMessage
  addressFormSteps: AddressFormStep[]
  addressAreaTree: AppTreeOptionResponse_Option[]
}>()

const emit = defineEmits<{
  'flow-action': [action?: AiAssistantAction, label?: string]
  'sku-num-change': [sku: AssistantFlowBlock, delta: number]
  'sku-submit': [block: AssistantFlowBlock, sku: AssistantFlowBlock]
  'create-address-guide': [block: AssistantFlowBlock]
  'address-submit': [block: AssistantFlowBlock]
  'address-region-change': [
    block: AssistantFlowBlock,
    event: Parameters<UniHelper.RegionPickerOnChange>[0],
  ]
  'address-city-change': [
    block: AssistantFlowBlock,
    event: Parameters<UniHelper.UniDataPickerOnChange>[0],
  ]
  'review-submit': [block: AssistantFlowBlock]
  'review-score-change': [form: AssistantFlowBlock, key: string, delta: number]
}>()

const reviewScores = [
  ['goods_score', '商品'],
  ['package_score', '包装'],
  ['delivery_score', '配送'],
]

function visibleFlowList(block: AssistantFlowBlock, blockIndex: number, field: string) {
  const list = Array.isArray(block[field]) ? block[field] : []
  if (!props.message.flowReveal) {
    return list
  }
  const count = props.message.flowReveal[buildFlowRevealKey(blockIndex, field)] ?? list.length
  return list.slice(0, count)
}

function resolveFlowScrollHeight(values: unknown, maxHeight: number, rowHeight: number) {
  const count = Array.isArray(values) ? values.length : 0
  if (count <= 0) {
    return '0rpx'
  }
  const gapHeight = Math.max(0, count - 1) * 14
  return `${Math.min(maxHeight, count * rowHeight + gapHeight)}rpx`
}

function isAddressFormFieldFilled(block: AssistantFlowBlock, key: AddressFormStepKey) {
  const form = block.form || {}
  if (key === 'contact') {
    return /^1[3-9]\d{9}$/.test(String(form.contact || ''))
  }
  if (key === 'address') {
    return Boolean(form.address?.length || form.address_name?.length)
  }
  return Boolean(form[key])
}

function resolveAddressActiveStepKey(block: AssistantFlowBlock) {
  return (
    props.addressFormSteps.find((item) => !isAddressFormFieldFilled(block, item.key))?.key ?? ''
  )
}

function resolveAddressStepIndex(block: AssistantFlowBlock) {
  const activeStepKey = resolveAddressActiveStepKey(block)
  if (!activeStepKey) {
    return props.addressFormSteps.length
  }
  return props.addressFormSteps.findIndex((item) => item.key === activeStepKey) + 1
}

function resolveAddressStepClass(block: AssistantFlowBlock, key: AddressFormStepKey) {
  if (isAddressFormFieldFilled(block, key)) {
    return 'is-done'
  }
  return resolveAddressActiveStepKey(block) === key ? 'is-active' : ''
}

function resolveAddressGuideHint(block: AssistantFlowBlock) {
  const activeStepKey = resolveAddressActiveStepKey(block)
  if (!activeStepKey) {
    return '信息已填完，确认无误后保存地址。'
  }
  const step = props.addressFormSteps.find((item) => item.key === activeStepKey)
  return step ? `第 ${resolveAddressStepIndex(block)} 步：${step.hint}` : ''
}

function isAddressFormReady(block: AssistantFlowBlock) {
  return props.addressFormSteps.every((item) => isAddressFormFieldFilled(block, item.key))
}

function forwardFlowAction(action?: AiAssistantAction, label?: string) {
  emit('flow-action', action, label)
}

function buildFlowRevealKey(blockIndex: number, field: string) {
  return `${blockIndex}:${field}`
}

function resolveSimpleItemImage(item: AssistantFlowBlock) {
  if (item.picture) {
    return item.picture
  }
  if (Array.isArray(item.pictures)) {
    return item.pictures.find(Boolean) || ''
  }
  return ''
}
</script>

<template>
  <view v-if="message.blocks.length" class="flow-block-list">
    <view v-for="(block, blockIndex) in message.blocks" :key="blockIndex" class="flow-block">
      <view v-if="block.title" class="flow-title">{{ block.title }}</view>

      <view v-if="block.type === 'goods_list'" class="flow-goods-list">
        <view v-if="!block.goods?.length" class="flow-empty">暂时没有推荐商品</view>
        <scroll-view
          v-else
          class="flow-scroll-list flow-goods-scroll"
          scroll-y
          :show-scrollbar="false"
          :style="{ height: resolveFlowScrollHeight(block.goods, 500, 150) }"
        >
          <view
            v-for="goods in visibleFlowList(block, blockIndex, 'goods')"
            :key="goods.id"
            class="flow-goods-card"
            :class="{ 'flow-reveal-item': message.flowReveal }"
            @tap="emit('flow-action', goods.action, `选择商品：${goods.name || ''}`)"
          >
            <image
              v-if="goods.picture"
              class="flow-goods-image"
              mode="aspectFill"
              :src="formatSrc(goods.picture)"
            />
            <view class="flow-goods-info">
              <view class="flow-goods-name">{{ goods.name }}</view>
              <view class="flow-goods-desc">{{ goods.desc || '精选推荐商品' }}</view>
              <view class="flow-price">¥{{ formatPrice(Number(goods.price || 0)) }}</view>
            </view>
            <button class="flow-mini-button" hover-class="none">选规格</button>
          </view>
        </scroll-view>
      </view>

      <view v-else-if="block.type === 'sku_selector'" class="flow-sku-panel">
        <view class="flow-goods-card is-static">
          <image
            v-if="block.goods?.picture"
            class="flow-goods-image"
            mode="aspectFill"
            :src="formatSrc(block.goods.picture)"
          />
          <view class="flow-goods-info">
            <view class="flow-goods-name">{{ block.goods?.name }}</view>
            <view class="flow-goods-desc">{{ block.goods?.desc }}</view>
          </view>
        </view>
        <view v-if="!block.skus?.length" class="flow-empty">暂时没有可选规格</view>
        <scroll-view
          v-else
          class="flow-scroll-list flow-sku-scroll"
          scroll-y
          :show-scrollbar="false"
          :style="{ height: resolveFlowScrollHeight(block.skus, 430, 100) }"
        >
          <view
            v-for="sku in visibleFlowList(block, blockIndex, 'skus')"
            :key="sku.sku_code"
            class="flow-sku-row"
            :class="{ 'flow-reveal-item': message.flowReveal }"
          >
            <view class="flow-sku-info">
              <view class="flow-sku-name">{{ sku.spec_text || sku.sku_code }}</view>
              <view class="flow-price">¥{{ formatPrice(Number(sku.price || 0)) }}</view>
            </view>
            <view class="flow-stepper">
              <button
                class="flow-stepper-button"
                hover-class="none"
                @tap="emit('sku-num-change', sku, -1)"
              >
                -
              </button>
              <text class="flow-stepper-value">{{ sku.num }}</text>
              <button
                class="flow-stepper-button"
                hover-class="none"
                @tap="emit('sku-num-change', sku, 1)"
              >
                +
              </button>
            </view>
            <button
              class="flow-primary-button"
              hover-class="none"
              @tap="emit('sku-submit', block, sku)"
            >
              确认
            </button>
          </view>
        </scroll-view>
      </view>

      <view v-else-if="block.type === 'cart_list'" class="flow-cart-list">
        <view v-if="!block.carts?.length" class="flow-empty">购物车里暂时没有商品</view>
        <scroll-view
          v-else
          class="flow-scroll-list flow-goods-scroll"
          scroll-y
          :show-scrollbar="false"
          :style="{ height: resolveFlowScrollHeight(block.carts, 500, 136) }"
        >
          <view
            v-for="cart in visibleFlowList(block, blockIndex, 'carts')"
            :key="cart.id"
            class="flow-goods-card"
            :class="{ 'flow-reveal-item': message.flowReveal }"
          >
            <image
              v-if="cart.picture"
              class="flow-goods-image"
              mode="aspectFill"
              :src="formatSrc(cart.picture)"
            />
            <view class="flow-goods-info">
              <view class="flow-goods-name">{{ cart.name }}</view>
              <view class="flow-goods-desc">{{ cart.spec_text || cart.sku_code }}</view>
              <view class="flow-price">¥{{ formatPrice(Number(cart.price || 0)) }}</view>
            </view>
            <view class="flow-cart-meta">
              <view class="flow-cart-num">x{{ cart.num || 1 }}</view>
              <view v-if="cart.checked" class="flow-status-pill">已选</view>
            </view>
          </view>
        </scroll-view>
      </view>

      <view v-else-if="block.type === 'simple_list'" class="flow-simple-list">
        <image
          v-if="block.banner"
          class="flow-simple-banner"
          mode="aspectFill"
          :src="formatSrc(block.banner)"
        />
        <view v-if="!block.items?.length" class="flow-empty">暂时没有可展示内容</view>
        <template v-else>
          <view
            v-for="item in visibleFlowList(block, blockIndex, 'items')"
            :key="item.id || item.title"
            class="flow-simple-item"
            :class="{
              'is-clickable': item.action,
              'flow-reveal-item': message.flowReveal,
            }"
            @tap="item.action && emit('flow-action', item.action, item.title || '继续')"
          >
            <image
              v-if="resolveSimpleItemImage(item)"
              class="flow-simple-image"
              mode="aspectFill"
              :src="formatSrc(resolveSimpleItemImage(item))"
            />
            <view class="flow-simple-main">
              <view class="flow-simple-title">{{ item.title }}</view>
              <view v-if="item.desc" class="flow-simple-desc">{{ item.desc }}</view>
            </view>
            <button v-if="item.action" class="flow-mini-button" hover-class="none">查看</button>
          </view>
        </template>
      </view>

      <view v-else-if="block.type === 'profile_panel'" class="flow-profile-panel">
        <view class="flow-profile-head" v-if="block.avatar || block.pictures?.[0]">
          <image
            class="flow-profile-avatar"
            mode="aspectFill"
            :src="formatSrc(block.avatar || block.pictures?.[0])"
          />
        </view>
        <view v-for="field in block.fields || []" :key="field.label" class="flow-line">
          <text class="flow-line-sub">{{ field.label }}</text>
          <text class="flow-line-main is-right">{{ field.value }}</text>
        </view>
      </view>

      <view v-else-if="block.type === 'order_preview'" class="flow-order-preview">
        <view v-for="goods in block.goods" :key="goods.sku_code" class="flow-line">
          <text class="flow-line-main">{{ goods.name }}</text>
          <text class="flow-line-sub">x{{ goods.num }}</text>
        </view>
        <view class="flow-summary">
          <view class="flow-line">
            <text>商品金额</text>
            <text>¥{{ formatPrice(Number(block.summary?.total_money || 0)) }}</text>
          </view>
          <view class="flow-line is-strong">
            <text>应付</text>
            <text>¥{{ formatPrice(Number(block.summary?.pay_money || 0)) }}</text>
          </view>
        </view>
      </view>

      <view v-else-if="block.type === 'address_selector'" class="flow-address-list">
        <view v-if="!block.addresses?.length" class="flow-address-empty-guide">
          <view class="flow-address-empty-title">还没有收货地址</view>
          <view class="flow-address-empty-desc">
            我可以带你一步一步新增，先填收货人，再选地区和详细地址。
          </view>
          <button
            class="flow-primary-button is-wide"
            hover-class="none"
            @tap="emit('create-address-guide', block)"
          >
            开始新增地址
          </button>
        </view>
        <view
          v-for="address in visibleFlowList(block, blockIndex, 'addresses')"
          :key="address.id"
          class="flow-address-card"
          :class="{
            'is-selectable': address.action,
            'flow-reveal-item': message.flowReveal,
          }"
          @tap="
            address.action &&
            emit('flow-action', address.action, `选择地址：${address.receiver || ''}`)
          "
        >
          <view class="flow-address-main">
            <view class="flow-line is-strong">
              <text>{{ address.receiver }}</text>
              <text>{{ address.contact }}</text>
            </view>
            <view class="flow-address-text">
              {{ (address.address || []).join(' ') }} {{ address.detail }}
            </view>
          </view>
          <view v-if="address.action" class="flow-address-select">选择</view>
        </view>
      </view>

      <view v-else-if="block.type === 'address_form'" class="flow-form is-address">
        <view class="flow-address-guide-head">
          <view>
            <view class="flow-address-guide-title">新增收货地址</view>
            <view class="flow-address-guide-hint">{{ resolveAddressGuideHint(block) }}</view>
          </view>
          <view class="flow-address-guide-step">
            {{ resolveAddressStepIndex(block) }}/{{ addressFormSteps.length }}
          </view>
        </view>
        <view class="flow-address-steps">
          <view
            v-for="(step, stepIndex) in addressFormSteps"
            :key="step.key"
            class="flow-address-step"
            :class="resolveAddressStepClass(block, step.key)"
          >
            <view class="flow-address-step-dot">{{ stepIndex + 1 }}</view>
            <view class="flow-address-step-label">{{ step.shortLabel }}</view>
          </view>
        </view>
        <view class="flow-form-hint">也可以直接把完整收货信息发给我，我会自动帮你填入。</view>
        <view class="flow-address-field" :class="resolveAddressStepClass(block, 'receiver')">
          <view class="flow-address-field-label">1. 收货人</view>
          <input
            v-model="block.form.receiver"
            class="flow-input is-large"
            :placeholder="addressFormSteps[0].placeholder"
            placeholder-class="flow-placeholder"
          />
        </view>
        <view class="flow-address-field" :class="resolveAddressStepClass(block, 'contact')">
          <view class="flow-address-field-label">2. 手机号</view>
          <input
            v-model="block.form.contact"
            class="flow-input is-large"
            :maxlength="11"
            :placeholder="addressFormSteps[1].placeholder"
            placeholder-class="flow-placeholder"
          />
        </view>
        <view class="flow-address-field" :class="resolveAddressStepClass(block, 'address')">
          <view class="flow-address-field-label">3. 所在地区</view>
          <!-- #ifdef MP-WEIXIN -->
          <picker
            class="flow-picker is-large"
            mode="region"
            :value="block.form.address_name"
            @change="emit('address-region-change', block, $event)"
          >
            <view v-if="block.form.address_name?.length" class="flow-picker-text">
              {{ block.form.address_name.join('-') }}
            </view>
            <view v-else class="flow-placeholder">{{ addressFormSteps[2].placeholder }}</view>
          </picker>
          <!-- #endif -->
          <!-- #ifdef H5 || APP-PLUS -->
          <view class="flow-picker is-data-picker is-large">
            <uni-data-picker
              v-model="block.form.address"
              :localdata="addressAreaTree"
              :placeholder="addressFormSteps[2].placeholder"
              popup-title="请选择城市"
              :clear-icon="false"
              @change="emit('address-city-change', block, $event)"
            />
          </view>
          <!-- #endif -->
        </view>
        <view class="flow-address-field" :class="resolveAddressStepClass(block, 'detail')">
          <view class="flow-address-field-label">4. 详细地址</view>
          <input
            v-model="block.form.detail"
            class="flow-input is-large"
            :placeholder="addressFormSteps[3].placeholder"
            placeholder-class="flow-placeholder"
          />
        </view>
        <button
          class="flow-primary-button is-wide"
          :class="{ 'is-disabled': !isAddressFormReady(block) }"
          hover-class="none"
          @tap="emit('address-submit', block)"
        >
          保存地址
        </button>
      </view>

      <view
        v-else-if="block.type === 'selected_address'"
        class="flow-address-card is-static is-selected"
      >
        <view class="flow-address-main">
          <view class="flow-address-badge">已选地址</view>
          <view class="flow-line is-strong">
            <text>{{ block.address?.receiver }}</text>
            <text>{{ block.address?.contact }}</text>
          </view>
          <view class="flow-address-text">
            {{ (block.address?.address || []).join(' ') }} {{ block.address?.detail }}
          </view>
        </view>
      </view>

      <view v-else-if="block.type === 'confirm_order'" class="flow-action-panel">
        <view class="flow-desc">{{ block.desc }}</view>
        <button
          class="flow-primary-button is-wide"
          hover-class="none"
          @tap="emit('flow-action', block.action, '确认下单')"
        >
          确认下单
        </button>
      </view>

      <view v-else-if="block.type === 'payment_panel'" class="flow-action-panel">
        <view class="flow-desc">订单号：{{ block.order_id }}</view>
        <button
          class="flow-primary-button is-wide"
          hover-class="none"
          @tap="emit('flow-action', block.action, '发起支付')"
        >
          发起支付
        </button>
      </view>

      <view v-else-if="block.type === 'payment_result'" class="flow-action-panel">
        <view class="flow-desc">支付已发起，请按系统提示完成支付。</view>
      </view>

      <OrderListBlock
        v-else-if="block.type === 'order_list'"
        :orders="visibleFlowList(block, blockIndex, 'orders')"
        :reveal="Boolean(message.flowReveal)"
        @flow-action="forwardFlowAction"
      />

      <view v-else-if="block.type === 'pending_review_list'" class="flow-goods-list">
        <view v-if="!block.goods?.length" class="flow-empty">暂时没有待评价商品</view>
        <view
          v-for="goods in visibleFlowList(block, blockIndex, 'goods')"
          :key="`${goods.order_id}:${goods.goods_id}:${goods.sku_code}`"
          class="flow-goods-card"
          :class="{ 'flow-reveal-item': message.flowReveal }"
        >
          <image
            v-if="goods.goods_picture"
            class="flow-goods-image"
            mode="aspectFill"
            :src="formatSrc(goods.goods_picture)"
          />
          <view class="flow-goods-info">
            <view class="flow-goods-name">{{ goods.goods_name }}</view>
            <view class="flow-goods-desc">{{ goods.sku_desc || goods.desc }}</view>
          </view>
          <button
            class="flow-mini-button"
            hover-class="none"
            @tap="emit('flow-action', goods.action, `评价：${goods.goods_name || ''}`)"
          >
            评价
          </button>
        </view>
      </view>

      <view v-else-if="block.type === 'review_form'" class="flow-form">
        <view class="flow-goods-name">{{ block.goods?.goods_name }}</view>
        <textarea
          v-model="block.form.content"
          class="flow-textarea"
          auto-height
          maxlength="300"
          placeholder="写下真实使用感受"
          placeholder-class="flow-placeholder"
        />
        <view v-for="score in reviewScores" :key="score[0]" class="flow-score-row">
          <text>{{ score[1] }}</text>
          <view class="flow-stepper">
            <button
              class="flow-stepper-button"
              hover-class="none"
              @tap="emit('review-score-change', block.form, score[0], -1)"
            >
              -
            </button>
            <text class="flow-stepper-value">{{ block.form[score[0]] }}</text>
            <button
              class="flow-stepper-button"
              hover-class="none"
              @tap="emit('review-score-change', block.form, score[0], 1)"
            >
              +
            </button>
          </view>
        </view>
        <button
          class="flow-primary-button is-wide"
          hover-class="none"
          @tap="emit('review-submit', block)"
        >
          提交评价
        </button>
      </view>

      <view v-else-if="block.type === 'order_logistics'" class="flow-logistics">
        <view class="flow-line is-strong">
          <text class="flow-line-main">订单 {{ block.order?.order_no || block.order?.id }}</text>
          <text class="flow-line-sub">¥{{ formatPrice(Number(block.order?.pay_money || 0)) }}</text>
        </view>
        <view v-if="block.address" class="flow-address-text">
          {{ block.address.receiver }} {{ block.address.contact }}
          {{ (block.address.address || []).join(' ') }} {{ block.address.detail }}
        </view>
        <view v-if="block.logistics" class="flow-logistics-box">
          <view class="flow-desc">
            {{ block.logistics.name || '物流信息' }} {{ block.logistics.no || '' }}
          </view>
          <view
            v-for="detail in block.logistics.detail || []"
            :key="`${detail.time}-${detail.text}`"
            class="flow-timeline"
          >
            <view class="flow-timeline-time">{{ detail.time }}</view>
            <view class="flow-timeline-text">{{ detail.text }}</view>
          </view>
        </view>
        <button
          v-if="block.action"
          class="flow-primary-button is-wide"
          hover-class="none"
          @tap="emit('flow-action', block.action, '确认收货')"
        >
          确认收货
        </button>
      </view>

      <view
        v-else-if="block.type === 'success' || block.type === 'notice'"
        class="flow-action-panel"
      >
        <view class="flow-desc">{{ block.desc }}</view>
      </view>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.flow-block-list {
  margin-top: 20rpx;
}

.flow-block {
  max-width: 100%;
  padding: 18rpx;
  border-radius: 10rpx;
  background-color: #f6f7f9;
  box-sizing: border-box;
  overflow: hidden;
}

.flow-block + .flow-block {
  margin-top: 16rpx;
}

.flow-title {
  margin-bottom: 14rpx;
  color: #333;
  font-size: 26rpx;
  font-weight: 600;
  line-height: 34rpx;
}

.flow-empty,
.flow-desc {
  color: #898b94;
  font-size: 24rpx;
  line-height: 36rpx;
}

.flow-scroll-list {
  box-sizing: border-box;
}

.flow-goods-scroll,
.flow-sku-scroll {
  padding-right: 4rpx;
}

.flow-goods-card,
.flow-address-card,
.flow-simple-item,
.flow-profile-panel {
  display: flex;
  align-items: center;
  gap: 16rpx;
  min-width: 0;
  max-width: 100%;
  padding: 16rpx;
  border-radius: 10rpx;
  background-color: #fff;
  box-sizing: border-box;
}

.flow-reveal-item {
  animation: flow-reveal-in 180ms ease-out both;
}

@keyframes flow-reveal-in {
  from {
    opacity: 0;
    transform: translateY(10rpx);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.flow-goods-card + .flow-goods-card,
.flow-address-card + .flow-address-card,
.flow-sku-row + .flow-sku-row,
.flow-simple-item + .flow-simple-item {
  margin-top: 14rpx;
}

.flow-goods-card.is-static,
.flow-address-card.is-static {
  align-items: flex-start;
}

.flow-address-card {
  border: 2rpx solid transparent;
}

.flow-address-card.is-selectable {
  padding: 18rpx;
  border-color: rgba(39, 186, 155, 0.28);
  background-color: #f8fffd;
}

.flow-address-card.is-selected {
  border-color: #27ba9b;
  background-color: #f3fffb;
}

.flow-address-main {
  flex: 1;
  min-width: 0;
}

.flow-address-select {
  flex-shrink: 0;
  min-width: 76rpx;
  height: 48rpx;
  padding: 0 18rpx;
  border-radius: 8rpx;
  box-sizing: border-box;
  color: #fff;
  font-size: 22rpx;
  font-weight: 600;
  line-height: 48rpx;
  text-align: center;
  background-color: #27ba9b;
}

.flow-address-badge {
  display: inline-flex;
  height: 34rpx;
  padding: 0 12rpx;
  margin-bottom: 10rpx;
  border-radius: 6rpx;
  color: #13876f;
  font-size: 21rpx;
  line-height: 34rpx;
  background-color: rgba(39, 186, 155, 0.12);
}

.flow-goods-image {
  flex-shrink: 0;
  width: 104rpx;
  height: 104rpx;
  border-radius: 8rpx;
  background-color: #eef0f3;
}

.flow-goods-info,
.flow-sku-info,
.flow-simple-main {
  flex: 1;
  min-width: 0;
}

.flow-goods-name,
.flow-sku-name,
.flow-simple-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #333;
  font-size: 25rpx;
  font-weight: 600;
  line-height: 34rpx;
}

.flow-goods-desc,
.flow-address-text,
.flow-simple-desc {
  margin-top: 6rpx;
  color: #777;
  font-size: 22rpx;
  line-height: 32rpx;
}

.flow-price {
  margin-top: 8rpx;
  color: #cf4444;
  font-size: 25rpx;
  font-weight: 600;
  line-height: 34rpx;
}

.flow-cart-meta {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 8rpx;
}

.flow-cart-num {
  color: #333;
  font-size: 24rpx;
  font-weight: 600;
  line-height: 32rpx;
}

.flow-status-pill {
  height: 34rpx;
  padding: 0 12rpx;
  border-radius: 6rpx;
  color: #16806d;
  font-size: 21rpx;
  line-height: 34rpx;
  background-color: #e8f8f4;
}

.flow-simple-list,
.flow-cart-list {
  min-width: 0;
}

.flow-simple-banner {
  width: 100%;
  height: 180rpx;
  margin-bottom: 14rpx;
  border-radius: 10rpx;
  background-color: #eef0f3;
}

.flow-simple-item {
  background-color: #fff;
}

.flow-simple-item.is-clickable {
  border: 2rpx solid rgba(39, 186, 155, 0.2);
  background-color: #f8fffd;
}

.flow-simple-image {
  flex-shrink: 0;
  width: 88rpx;
  height: 88rpx;
  border-radius: 8rpx;
  background-color: #eef0f3;
}

.flow-profile-panel {
  flex-direction: column;
  align-items: stretch;
  background-color: #fff;
}

.flow-profile-head {
  display: flex;
  justify-content: center;
  padding-bottom: 14rpx;
  margin-bottom: 4rpx;
}

.flow-profile-avatar {
  width: 108rpx;
  height: 108rpx;
  border-radius: 54rpx;
  background-color: #eef0f3;
}

.flow-mini-button,
.flow-primary-button,
.flow-stepper-button {
  padding: 0;
  margin: 0;
  border-radius: 0;
  background: transparent;
  line-height: normal;

  &::after {
    border: none;
  }
}

.flow-mini-button {
  flex-shrink: 0;
  width: 104rpx;
  height: 52rpx;
  border-radius: 8rpx;
  color: #fff;
  font-size: 23rpx;
  line-height: 52rpx;
  background-color: #27ba9b;
}

.flow-primary-button {
  flex-shrink: 0;
  width: 112rpx;
  height: 56rpx;
  border-radius: 8rpx;
  color: #fff;
  font-size: 24rpx;
  line-height: 56rpx;
  background-color: #27ba9b;
}

.flow-primary-button.is-wide {
  width: 100%;
  margin-top: 18rpx;
}

.flow-primary-button.is-disabled {
  background-color: #9ddfcc;
}

.flow-sku-row {
  display: flex;
  align-items: center;
  gap: 14rpx;
  padding: 16rpx;
  border-radius: 10rpx;
  background-color: #fff;
}

.flow-stepper {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  height: 52rpx;
  border-radius: 8rpx;
  background-color: #f0f2f5;
}

.flow-stepper-button {
  width: 48rpx;
  height: 52rpx;
  color: #333;
  font-size: 28rpx;
  line-height: 52rpx;
}

.flow-stepper-value {
  min-width: 42rpx;
  color: #333;
  font-size: 24rpx;
  line-height: 52rpx;
  text-align: center;
}

.flow-order-preview,
.flow-action-panel,
.flow-form,
.flow-logistics {
  padding: 16rpx;
  border-radius: 10rpx;
  background-color: #fff;
  box-sizing: border-box;
}

.flow-form.is-address {
  padding: 20rpx;
}

.flow-line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
  min-width: 0;
  color: #666;
  font-size: 23rpx;
  line-height: 34rpx;
}

.flow-line + .flow-line {
  margin-top: 10rpx;
}

.flow-line.is-strong {
  color: #333;
  font-weight: 600;
}

.flow-line-main {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.flow-line-sub {
  flex-shrink: 0;
  color: #898b94;
}

.flow-summary {
  margin-top: 16rpx;
  padding-top: 14rpx;
  border-top: 1rpx solid #edf0f2;
}

.flow-input,
.flow-textarea,
.flow-picker {
  width: 100%;
  padding: 16rpx;
  border-radius: 8rpx;
  box-sizing: border-box;
  color: #333;
  font-size: 24rpx;
  line-height: 36rpx;
  background-color: #f6f7f9;
}

.flow-input.is-large,
.flow-picker.is-large {
  min-height: 84rpx;
  padding: 20rpx 22rpx;
  border: 2rpx solid #edf0f2;
  border-radius: 12rpx;
  color: #111;
  font-size: 29rpx;
  line-height: 44rpx;
  background-color: #fff;
}

.flow-picker.is-large {
  display: flex;
  align-items: center;
}

.flow-input + .flow-input,
.flow-input + .flow-picker,
.flow-picker + .flow-input,
.flow-textarea {
  margin-top: 12rpx;
}

.flow-form-hint {
  margin-bottom: 12rpx;
  color: #898b94;
  font-size: 22rpx;
  line-height: 32rpx;
}

.flow-address-guide-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16rpx;
  margin-bottom: 18rpx;
}

.flow-address-guide-title {
  color: #222;
  font-size: 28rpx;
  font-weight: 700;
  line-height: 38rpx;
}

.flow-address-guide-hint {
  margin-top: 6rpx;
  color: #666;
  font-size: 23rpx;
  line-height: 34rpx;
}

.flow-address-guide-step {
  flex-shrink: 0;
  min-width: 72rpx;
  height: 44rpx;
  padding: 0 14rpx;
  border-radius: 22rpx;
  color: #16806d;
  font-size: 22rpx;
  font-weight: 600;
  line-height: 44rpx;
  text-align: center;
  background-color: #e8f8f4;
  box-sizing: border-box;
}

.flow-address-steps {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10rpx;
  margin-bottom: 18rpx;
}

.flow-address-step {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8rpx;
  color: #9aa0aa;
}

.flow-address-step-dot {
  width: 42rpx;
  height: 42rpx;
  border: 2rpx solid #dde2e8;
  border-radius: 50%;
  box-sizing: border-box;
  font-size: 22rpx;
  font-weight: 600;
  line-height: 38rpx;
  text-align: center;
  background-color: #fff;
}

.flow-address-step-label {
  font-size: 21rpx;
  line-height: 28rpx;
}

.flow-address-step.is-active {
  color: #16806d;
}

.flow-address-step.is-active .flow-address-step-dot {
  border-color: #27ba9b;
  color: #27ba9b;
  background-color: #e8f8f4;
}

.flow-address-step.is-done {
  color: #27ba9b;
}

.flow-address-step.is-done .flow-address-step-dot {
  border-color: #27ba9b;
  color: #fff;
  background-color: #27ba9b;
}

.flow-address-field {
  padding: 16rpx;
  border: 2rpx solid #edf0f2;
  border-radius: 14rpx;
  background-color: #f9fafb;
}

.flow-address-field + .flow-address-field {
  margin-top: 14rpx;
}

.flow-address-field.is-active {
  border-color: rgba(39, 186, 155, 0.55);
  background-color: #f7fffc;
}

.flow-address-field.is-done {
  border-color: rgba(39, 186, 155, 0.22);
}

.flow-address-field-label {
  margin-bottom: 12rpx;
  color: #333;
  font-size: 25rpx;
  font-weight: 600;
  line-height: 34rpx;
}

.flow-address-empty-guide {
  padding: 22rpx;
  border: 2rpx dashed rgba(39, 186, 155, 0.38);
  border-radius: 14rpx;
  background-color: #f8fffd;
}

.flow-address-empty-title {
  color: #222;
  font-size: 27rpx;
  font-weight: 700;
  line-height: 38rpx;
}

.flow-address-empty-desc {
  margin-top: 8rpx;
  color: #666;
  font-size: 24rpx;
  line-height: 36rpx;
}

.flow-picker {
  min-height: 68rpx;
}

.flow-picker.is-data-picker {
  padding: 0;
  background-color: transparent;
}

.flow-picker.is-data-picker.is-large {
  min-height: 84rpx;
  padding: 0 22rpx;
  border: 2rpx solid #edf0f2;
  border-radius: 12rpx;
  background-color: #fff;
}

.flow-picker-text {
  color: #333;
}

.flow-textarea {
  min-height: 132rpx;
}

.flow-placeholder {
  color: #b8bcc5;
}

.flow-score-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 14rpx;
  color: #333;
  font-size: 24rpx;
  line-height: 36rpx;
}

.flow-logistics-box {
  margin-top: 16rpx;
  padding-top: 14rpx;
  border-top: 1rpx solid #edf0f2;
}

.flow-timeline {
  margin-top: 12rpx;
}

.flow-timeline-time {
  color: #898b94;
  font-size: 21rpx;
  line-height: 30rpx;
}

.flow-timeline-text {
  margin-top: 4rpx;
  color: #333;
  font-size: 23rpx;
  line-height: 34rpx;
}
</style>
