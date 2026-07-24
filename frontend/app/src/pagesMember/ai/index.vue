<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'
import { defAiMessageService } from '@/api/base/ai_message'
import { defAiSessionService } from '@/api/base/ai_session'
import type { AiMessage, AiSession } from '@/rpc/base/v1/ai_session'
import { AiMessageStatus, Terminal } from '@/rpc/common/v1/enum'
import { useUserStore } from '@/stores'
import { navigateToLogin } from '@/utils/navigation'

const userStore = useUserStore()
const sessions = ref<AiSession[]>([])
const activeSessionID = ref('')
const messages = ref<AiMessage[]>([])
const input = ref('')
const loading = ref(false)
const sending = ref(false)

const activeSession = computed(() =>
  sessions.value.find((item) => item.id === activeSessionID.value),
)

onLoad(() => {
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin()
    return
  }
  void loadSessions()
})

const loadSessions = async () => {
  loading.value = true
  try {
    const response = await defAiSessionService.ListAiSession({
      terminal: Terminal.TERMINAL_APP,
    })
    sessions.value = response.sessions || []
    if (sessions.value.length) {
      await selectSession(sessions.value[0].id)
    } else {
      await createSession()
    }
  } catch {
    uni.showToast({ icon: 'none', title: '会话加载失败' })
  } finally {
    loading.value = false
  }
}

const createSession = async () => {
  const response = await defAiSessionService.CreateAiSession({
    title: '新会话',
    terminal: Terminal.TERMINAL_APP,
  })
  if (!response.session) {
    return
  }
  sessions.value = [response.session, ...sessions.value]
  activeSessionID.value = response.session.id
  messages.value = []
}

const selectSession = async (sessionID: string) => {
  activeSessionID.value = sessionID
  try {
    const response = await defAiSessionService.ListAiMessage({ session_id: sessionID })
    messages.value = response.messages || []
  } catch {
    uni.showToast({ icon: 'none', title: '消息加载失败' })
  }
}

const textOf = (message: AiMessage) => {
  if (message.input_content?.content) {
    return message.input_content.content
  }
  return message.output_content?.content || ''
}

const isUserMessage = (message: AiMessage) => Boolean(message.input_content?.content)

const send = async () => {
  const content = input.value.trim()
  if (!content || !activeSessionID.value || sending.value) {
    return
  }

  input.value = ''
  sending.value = true
  try {
    const response = await defAiMessageService.SendAiMessage({
      session_id: activeSessionID.value,
      content,
      attachments: [],
      action: undefined,
    })
    messages.value = [...messages.value, ...(response.messages || [])]
    if (response.session) {
      sessions.value = sessions.value.map((item) =>
        item.id === response.session?.id ? response.session : item,
      )
    }
  } catch {
    input.value = content
    uni.showToast({ icon: 'none', title: '发送失败，请稍后重试' })
  } finally {
    sending.value = false
  }
}

const messageStatus = (message: AiMessage) => {
  if (message.status === AiMessageStatus.FAILED_AAMS) {
    return '生成失败'
  }
  if (message.status === AiMessageStatus.GENERATING_AAMS) {
    return '生成中'
  }
  return ''
}
</script>

<template>
  <view class="page">
    <view class="toolbar">
      <picker
        v-if="sessions.length"
        class="session-picker"
        :range="sessions"
        range-key="title"
        :value="
          Math.max(
            sessions.findIndex((item) => item.id === activeSessionID),
            0,
          )
        "
        @change="selectSession(sessions[$event.detail.value].id)"
      >
        <text>{{ activeSession?.title || '选择会话' }}</text>
        <text class="picker-arrow">⌄</text>
      </picker>
      <button class="new-button" :disabled="loading || sending" @tap="createSession">新建</button>
    </view>

    <scroll-view scroll-y class="messages">
      <view v-if="!messages.length && !loading" class="empty">
        <text class="empty-title">开始一段新对话</text>
        <text class="empty-desc">输入问题，使用基础 AI 能力。</text>
      </view>
      <view
        v-for="message in messages"
        :key="message.id"
        class="message"
        :class="{ 'message--user': isUserMessage(message) }"
      >
        <text class="message-text">{{ textOf(message) }}</text>
        <text v-if="messageStatus(message)" class="message-status">{{
          messageStatus(message)
        }}</text>
      </view>
    </scroll-view>

    <view class="composer">
      <textarea
        v-model="input"
        class="input"
        :disabled="sending"
        maxlength="4000"
        placeholder="输入消息"
        :adjust-position="true"
        @confirm="send"
      />
      <button class="send-button" :disabled="sending || !input.trim()" @tap="send">
        {{ sending ? '发送中' : '发送' }}
      </button>
    </view>
  </view>
</template>

<style lang="scss">
page {
  background: #f5f7f8;
}

.page {
  display: flex;
  flex-direction: column;
  height: 100vh;
  padding: 24rpx 28rpx;
  box-sizing: border-box;
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 72rpx;
}

.session-picker {
  display: flex;
  align-items: center;
  color: #26332f;
  font-size: 30rpx;
  font-weight: 600;
}

.picker-arrow {
  margin-left: 10rpx;
  color: #8a9691;
}

.new-button,
.send-button {
  margin: 0;
  border: 0;
  background: #2f9f87;
  color: #fff;
  font-size: 24rpx;
}

.new-button {
  padding: 0 24rpx;
  border-radius: 28rpx;
  line-height: 56rpx;
}

.new-button::after,
.send-button::after {
  border: 0;
}

.messages {
  flex: 1;
  min-height: 0;
  padding: 24rpx 0;
  box-sizing: border-box;
}

.empty {
  padding-top: 220rpx;
  text-align: center;
}

.empty-title,
.empty-desc {
  display: block;
}

.empty-title {
  color: #26332f;
  font-size: 34rpx;
  font-weight: 600;
}

.empty-desc {
  margin-top: 16rpx;
  color: #8a9691;
  font-size: 26rpx;
}

.message {
  max-width: 86%;
  margin: 0 auto 20rpx 0;
  padding: 22rpx 24rpx;
  border-radius: 16rpx 16rpx 16rpx 4rpx;
  background: #fff;
  color: #26332f;
  font-size: 28rpx;
  line-height: 1.6;
}

.message--user {
  margin-right: 0;
  margin-left: auto;
  border-radius: 16rpx 16rpx 4rpx 16rpx;
  background: #2f9f87;
  color: #fff;
}

.message-text,
.message-status {
  display: block;
  white-space: pre-wrap;
  word-break: break-word;
}

.message-status {
  margin-top: 8rpx;
  color: #c55a5a;
  font-size: 22rpx;
}

.message--user .message-status {
  color: rgba(255, 255, 255, 0.8);
}

.composer {
  display: flex;
  align-items: flex-end;
  gap: 16rpx;
  padding: 18rpx 0 calc(18rpx + env(safe-area-inset-bottom));
}

.input {
  flex: 1;
  min-height: 76rpx;
  max-height: 220rpx;
  padding: 18rpx 22rpx;
  border-radius: 16rpx;
  background: #fff;
  box-sizing: border-box;
  color: #26332f;
  font-size: 28rpx;
}

.send-button {
  min-width: 116rpx;
  padding: 0 18rpx;
  border-radius: 16rpx;
  line-height: 76rpx;
}

.send-button[disabled],
.new-button[disabled] {
  opacity: 0.5;
}
</style>
