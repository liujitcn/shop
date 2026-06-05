<script setup lang="ts">
import type {
  AiAssistantMessage,
  AiAssistantSession,
  AiAssistantTool,
} from '@/rpc/base/v1/ai_assistant_session'
import { AiAssistantMessageStatus, Terminal } from '@/rpc/common/v1/enum'
import { computed, ref } from 'vue'

type ChatRole = 'user' | 'assistant'

type ProductCard = {
  id: string
  name: string
  tag: string
  price: string
}

type MessageActionKey = 'retry' | 'speak' | 'copy' | 'delete' | 'edit' | 'branch'
type SessionActionKey = 'rename' | 'delete'

type MessageAction = {
  key: MessageActionKey
  icon: string
  label: string
  danger?: boolean
}

type PressPoint = {
  clientX?: number
  clientY?: number
  pageX?: number
  pageY?: number
  x?: number
  y?: number
}

type PressEvent = {
  detail?: {
    x?: number
    y?: number
  }
  touches?: PressPoint[]
  changedTouches?: PressPoint[]
}

type ChatMessageItem = {
  id: string
  messageID: string
  role: ChatRole
  content: string
  status: AiAssistantMessageStatus
  tools: AiAssistantTool[]
  model: string
  replySource: string
  tokenTotal: number
  firstTokenMs: number
  durationMs: number
  products: ProductCard[]
}

const systemInfo = uni.getSystemInfoSync()
const { safeAreaInsets } = systemInfo
const composerBottom = `${Math.max(safeAreaInsets?.bottom || 0, 9)}px`
const drawerTopPadding = `${(safeAreaInsets?.top || 0) + 12}px`
const windowWidth = systemInfo.windowWidth || systemInfo.screenWidth || 375
const windowHeight = systemInfo.windowHeight || systemInfo.screenHeight || 667
const showSessionDrawer = ref(false)
const activeSessionID = ref('1001')
const inputText = ref('')
const isRecording = ref(false)
const sessionKeyword = ref('')
const showRenameDialog = ref(false)
const renamingSessionID = ref('')
const renamingTitle = ref('')
const editingMessageKey = ref('')
const editingContent = ref('')
const actionMessageKey = ref('')
const actionSessionID = ref('')
const ignoredTapSessionID = ref('')
const operationPoint = ref({
  x: Math.round(windowWidth / 2),
  y: Math.round(windowHeight / 2),
})

const mockSessions = ref<AiAssistantSession[]>([
  {
    id: '1001',
    title: '618 送礼选品',
    summary: '围绕预算、时效和评价筛选',
    updated_at: undefined,
    terminal: Terminal.TERMINAL_APP,
  },
  {
    id: '1002',
    title: '订单和售后咨询',
    summary: '查物流、退款进度',
    updated_at: undefined,
    terminal: Terminal.TERMINAL_APP,
  },
  {
    id: '1003',
    title: '商品对比',
    summary: '规格、价格、服务',
    updated_at: undefined,
    terminal: Terminal.TERMINAL_APP,
  },
  {
    id: '1004',
    title: '水果礼盒配送',
    summary: '确认同城仓、次日达和收货时间',
    updated_at: undefined,
    terminal: Terminal.TERMINAL_APP,
  },
  {
    id: '1005',
    title: '售后政策整理',
    summary: '退换货条件、服务承诺和凭证准备',
    updated_at: undefined,
    terminal: Terminal.TERMINAL_APP,
  },
  {
    id: '1006',
    title: '购物车凑单',
    summary: '满减门槛、优惠券和组合推荐',
    updated_at: undefined,
    terminal: Terminal.TERMINAL_APP,
  },
  {
    id: '1007',
    title: '评价摘要',
    summary: '提炼近期好评、差评和风险点',
    updated_at: undefined,
    terminal: Terminal.TERMINAL_APP,
  },
  {
    id: '1008',
    title: '订单物流查询',
    summary: '跟进发货节点和配送异常',
    updated_at: undefined,
    terminal: Terminal.TERMINAL_APP,
  },
  {
    id: '1009',
    title: '家用小电器推荐',
    summary: '按预算、品牌、售后筛选',
    updated_at: undefined,
    terminal: Terminal.TERMINAL_APP,
  },
])

const recommendedProducts: ProductCard[] = [
  {
    id: 'goods-1',
    name: '时令水果礼盒',
    tag: '次日达 · 评价稳定',
    price: '¥199',
  },
  {
    id: 'goods-2',
    name: '便携按摩仪',
    tag: '售后无忧',
    price: '¥269',
  },
]

const mockMessages = ref<Record<string, AiAssistantMessage[]>>({
  '1001': [
    {
      id: '5001',
      input_content: {
        kind: 'text',
        content: '预算 300 元以内，适合送长辈的礼物？',
      },
      output_content: {
        kind: 'markdown',
        content: '优先看库存稳定、售后简单、评价波动小的商品。下面这两类更适合送长辈：',
        reply_source: 'llm',
        model: 'gpt-4.1-mini',
        fallback: false,
        fallback_reason: '',
      },
      attachments: [],
      created_at: undefined,
      status: AiAssistantMessageStatus.SUCCESS_AAMS,
      token: {
        input: 1280,
        output: 462,
        cache: 256,
        total: 1998,
      },
      tools: [
        {
          type: 'function',
          name: 'app_v1_recommend_service_recommend_goods',
          title: '商品推荐',
          status: 'success',
          input: '{"scene":"PROFILE","page_size":3}',
          output: '{"goods_infos":[{"name":"时令水果礼盒"},{"name":"便携按摩仪"}]}',
        },
        {
          type: 'function',
          name: 'app_v1_goods_info_service_page_goods_info',
          title: '商品检索',
          status: 'success',
          input: '{"keyword":"礼盒","page_size":5}',
          output: '{"total":24}',
        },
      ],
      first_token_ms: 680,
      duration_ms: 3200,
    },
    {
      id: '5002',
      input_content: {
        kind: 'text',
        content: '如果对方更看重配送速度和售后，应该怎么排序？',
      },
      output_content: {
        kind: 'markdown',
        content:
          '建议把筛选优先级调整为：本地仓库存、服务标签、近期评价、退换政策。价格只作为最后一层约束，避免为了低价牺牲履约稳定性。',
        reply_source: 'llm',
        model: 'gpt-4.1-mini',
        fallback: false,
        fallback_reason: '',
      },
      attachments: [],
      created_at: undefined,
      status: AiAssistantMessageStatus.SUCCESS_AAMS,
      token: {
        input: 820,
        output: 214,
        cache: 0,
        total: 1034,
      },
      tools: [
        {
          type: 'function',
          name: 'app_v1_goods_info_service_page_goods_info',
          title: '商品检索',
          status: 'success',
          input: '{"delivery":"next_day","service":"after_sale"}',
          output: '{"total":12}',
        },
      ],
      first_token_ms: 430,
      duration_ms: 1800,
    },
    {
      id: '5003',
      input_content: {
        kind: 'text',
        content: '帮我总结成下单前确认清单。',
      },
      output_content: {
        kind: 'markdown',
        content:
          '下单前建议确认：\n1. 商品是否支持次日达或预约配送；\n2. 是否有明确退换说明；\n3. 近 30 天评价是否集中出现包装、破损、延迟问题；\n4. 收货地址是否在服务范围内；\n5. 礼盒类商品是否需要备注祝福卡或发票。',
        reply_source: 'llm',
        model: 'gpt-4.1-mini',
        fallback: false,
        fallback_reason: '',
      },
      attachments: [],
      created_at: undefined,
      status: AiAssistantMessageStatus.SUCCESS_AAMS,
      token: {
        input: 940,
        output: 310,
        cache: 0,
        total: 1250,
      },
      tools: [],
      first_token_ms: 520,
      duration_ms: 0,
    },
  ],
  '1002': [],
  '1003': [],
  '1004': [],
  '1005': [],
  '1006': [],
  '1007': [],
  '1008': [],
  '1009': [],
})

const filteredSessions = computed(() => {
  if (!sessionKeyword.value) {
    return mockSessions.value
  }
  return mockSessions.value.filter(
    (item) =>
      item.title.includes(sessionKeyword.value) || item.summary.includes(sessionKeyword.value),
  )
})

const currentMessages = computed(() => mockMessages.value[activeSessionID.value] ?? [])

const chatMessages = computed<ChatMessageItem[]>(() => {
  return currentMessages.value.flatMap((item) => [
    {
      id: `${item.id}:user`,
      messageID: item.id,
      role: 'user',
      content: item.input_content?.content ?? '',
      status: item.status,
      tools: [],
      model: '',
      replySource: '',
      tokenTotal: 0,
      firstTokenMs: 0,
      durationMs: 0,
      products: [],
    },
    {
      id: `${item.id}:assistant`,
      messageID: item.id,
      role: 'assistant',
      content: item.output_content?.content ?? '',
      status: item.status,
      tools: item.tools,
      model: item.output_content?.model ?? '',
      replySource: item.output_content?.reply_source ?? '',
      tokenTotal: item.token?.total ?? 0,
      firstTokenMs: item.first_token_ms,
      durationMs: item.duration_ms,
      products: item.id === '5001' ? recommendedProducts : [],
    },
  ])
})

const hasMessages = computed(() => chatMessages.value.length > 0)

const actionMessage = computed(() => {
  return chatMessages.value.find((item) => item.id === actionMessageKey.value)
})

const actionSession = computed(() => {
  return mockSessions.value.find((item) => item.id === actionSessionID.value)
})

const composerPlaceholder = computed(() => {
  if (isRecording.value) {
    return '正在听...'
  }
  return hasMessages.value ? '继续追问或补充预算' : '问问商城 AI 助手'
})

const isSubmitDisabled = computed(() => inputText.value.length === 0 || isRecording.value)

const messageOperationActions = computed(() => {
  if (!actionMessage.value) {
    return []
  }
  return resolveMessageActions(actionMessage.value)
})

const operationMenuCount = computed(() => {
  if (actionMessage.value) {
    return messageOperationActions.value.length + (hasRuntimeBrief(actionMessage.value) ? 1 : 0)
  }
  return actionSession.value ? 2 : 0
})

const operationSheetStyle = computed(() => {
  const menuWidth = Math.min(224, windowWidth - 32)
  const titleHeight = actionSession.value ? 30 : 0
  const itemHeight = 40
  const menuHeight = titleHeight + operationMenuCount.value * itemHeight + 12
  const horizontalInset = 12
  const verticalInset = 12
  const topGap = 10

  const left = Math.min(
    Math.max(operationPoint.value.x - menuWidth / 2, horizontalInset),
    windowWidth - menuWidth - horizontalInset,
  )
  const top = Math.min(
    Math.max(operationPoint.value.y + topGap, verticalInset),
    windowHeight - menuHeight - verticalInset,
  )

  return {
    left: `${left}px`,
    top: `${top}px`,
    width: `${menuWidth}px`,
  }
})

const lastEditableUserMessageID = computed(() => {
  const lastMessage = currentMessages.value[currentMessages.value.length - 1]
  if (!lastMessage || lastMessage.status === AiAssistantMessageStatus.GENERATING_AAMS) {
    return ''
  }
  return lastMessage.id
})

/** 打开或收起历史会话抽屉。 */
const toggleSessionDrawer = () => {
  showSessionDrawer.value = !showSessionDrawer.value
}

/** 切换当前静态会话，用于验证对话态和空态。 */
const selectSession = (sessionID: string) => {
  if (ignoredTapSessionID.value === sessionID) {
    ignoredTapSessionID.value = ''
    return
  }
  activeSessionID.value = sessionID
  showSessionDrawer.value = false
  editingMessageKey.value = ''
  editingContent.value = ''
  closeOperationSheet()
}

/** 新建静态会话，并切换到新会话。 */
const createSession = () => {
  const id = `new-${Date.now()}`
  mockSessions.value = [
    {
      id,
      title: '新对话',
      summary: '还没有消息',
      updated_at: undefined,
      terminal: Terminal.TERMINAL_APP,
    },
    ...mockSessions.value,
  ]
  mockMessages.value[id] = []
  activeSessionID.value = id
  sessionKeyword.value = ''
  showSessionDrawer.value = false
  uni.showToast({
    icon: 'none',
    title: '已创建新会话',
  })
}

/** 打开会话重命名弹窗。 */
const openRenameSession = (session: AiAssistantSession) => {
  renamingSessionID.value = session.id
  renamingTitle.value = session.title
  showRenameDialog.value = true
}

/** 取消会话重命名。 */
const cancelRenameSession = () => {
  showRenameDialog.value = false
  renamingSessionID.value = ''
  renamingTitle.value = ''
}

/** 保存会话名称，静态页仅更新本地会话列表。 */
const confirmRenameSession = () => {
  if (!renamingTitle.value) {
    uni.showToast({
      icon: 'none',
      title: '请输入会话名称',
    })
    return
  }

  mockSessions.value = mockSessions.value.map((item) =>
    item.id === renamingSessionID.value ? { ...item, title: renamingTitle.value } : item,
  )
  cancelRenameSession()
  uni.showToast({
    icon: 'none',
    title: '会话已重命名',
  })
}

/** 删除静态会话，保留至少一条会话用于页面展示。 */
const deleteSession = (sessionID: string) => {
  if (mockSessions.value.length <= 1) {
    uni.showToast({
      icon: 'none',
      title: '至少保留一个会话',
    })
    return
  }

  const nextSessions = mockSessions.value.filter((item) => item.id !== sessionID)
  mockSessions.value = nextSessions
  delete mockMessages.value[sessionID]

  if (activeSessionID.value === sessionID) {
    activeSessionID.value = nextSessions[0].id
  }

  uni.showToast({
    icon: 'none',
    title: '已删除会话',
  })
}

/** 复制当前消息正文。 */
const copyMessage = (item: ChatMessageItem) => {
  uni.setClipboardData({
    data: item.content,
    success: () => {
      uni.showToast({
        icon: 'none',
        title: '消息已复制',
      })
    },
  })
}

/** 删除当前轮消息。 */
const deleteMessage = (item: ChatMessageItem) => {
  mockMessages.value[activeSessionID.value] = currentMessages.value.filter(
    (message) => message.id !== item.messageID,
  )
  editingMessageKey.value = ''
  editingContent.value = ''
  closeOperationSheet()
  uni.showToast({
    icon: 'none',
    title: '消息已删除',
  })
}

/** 开始编辑用户消息正文。 */
const startEditMessage = (item: ChatMessageItem) => {
  if (!isLastEditableUserMessage(item)) {
    return
  }
  editingMessageKey.value = item.id
  editingContent.value = item.content
}

/** 取消当前消息编辑。 */
const cancelEditMessage = () => {
  editingMessageKey.value = ''
  editingContent.value = ''
}

/** 保存用户消息编辑，并模拟重新生成助手输出。 */
const saveEditMessage = (item: ChatMessageItem) => {
  if (!editingContent.value) {
    uni.showToast({
      icon: 'none',
      title: '请输入消息内容',
    })
    return
  }

  mockMessages.value[activeSessionID.value] = currentMessages.value.map((message) => {
    if (message.id !== item.messageID) {
      return message
    }
    return {
      ...message,
      input_content: {
        kind: message.input_content?.kind || 'text',
        content: editingContent.value,
      },
      output_content: message.output_content
        ? {
            ...message.output_content,
            content:
              '已根据修改后的问题重新整理结果。静态页先展示编辑后的消息流，后续接入接口后会重新生成真实回复。',
          }
        : message.output_content,
      status: AiAssistantMessageStatus.SUCCESS_AAMS,
    }
  })
  cancelEditMessage()
  closeOperationSheet()
  uni.showToast({
    icon: 'none',
    title: '消息已更新',
  })
}

/** 长按气泡后展示移动端消息操作菜单。 */
const resolvePressPoint = (event?: PressEvent) => {
  const point = event?.changedTouches?.[0] || event?.touches?.[0]
  operationPoint.value = {
    x:
      point?.clientX ?? point?.pageX ?? point?.x ?? event?.detail?.x ?? Math.round(windowWidth / 2),
    y:
      point?.clientY ??
      point?.pageY ??
      point?.y ??
      event?.detail?.y ??
      Math.round(windowHeight / 2),
  }
}

const openMessageActionSheet = (item: ChatMessageItem, event?: PressEvent) => {
  if (editingMessageKey.value === item.id) {
    return
  }
  if (!resolveMessageActions(item).length && !hasRuntimeBrief(item)) {
    return
  }
  resolvePressPoint(event)
  actionMessageKey.value = item.id
  actionSessionID.value = ''
}

/** 长按会话后展示移动端会话操作菜单。 */
const openSessionActionSheet = (session: AiAssistantSession, event?: PressEvent) => {
  ignoredTapSessionID.value = session.id
  resolvePressPoint(event)
  actionSessionID.value = session.id
  actionMessageKey.value = ''
}

const isLastEditableUserMessage = (item: ChatMessageItem) => {
  return item.role === 'user' && item.messageID === lastEditableUserMessageID.value
}

const formatTokenBrief = (item: ChatMessageItem) => {
  if (!item.tokenTotal) {
    return ''
  }
  if (item.tokenTotal >= 1000) {
    return `${(item.tokenTotal / 1000).toFixed(1)}K`
  }
  return `${item.tokenTotal}`
}

const formatDurationBrief = (item: ChatMessageItem) => {
  if (!item.durationMs) {
    return '生成中'
  }
  return `${(item.durationMs / 1000).toFixed(2)}s`
}

const hasRuntimeBrief = (item: ChatMessageItem) => {
  return item.role === 'assistant' && Boolean(item.tokenTotal || item.durationMs)
}

const resolveMessageActions = (item: ChatMessageItem): MessageAction[] => {
  if (item.status === AiAssistantMessageStatus.GENERATING_AAMS) {
    return []
  }

  const copyAction: MessageAction = { key: 'copy', icon: 'copy', label: '复制' }
  const deleteAction: MessageAction = { key: 'delete', icon: 'trash', label: '删除', danger: true }
  const editAction: MessageAction = { key: 'edit', icon: 'compose', label: '编辑' }

  if (item.role === 'user') {
    const actions = isLastEditableUserMessage(item) ? [editAction, copyAction] : [copyAction]
    if (item.status === AiAssistantMessageStatus.FAILED_AAMS) {
      return [{ key: 'retry', icon: 'refresh', label: '重新发送' }, ...actions, deleteAction]
    }
    return [...actions, deleteAction]
  }

  const assistantActions: MessageAction[] = [{ key: 'retry', icon: 'refresh', label: '重新生成' }]
  if (item.status === AiAssistantMessageStatus.SUCCESS_AAMS) {
    assistantActions.push({ key: 'branch', icon: 'branch', label: '创建分支' })
    assistantActions.push({ key: 'speak', icon: 'sound', label: '朗读' })
  }
  return [...assistantActions, copyAction, deleteAction]
}

const handleMessageAction = (action: MessageActionKey, item: ChatMessageItem) => {
  if (action === 'copy') {
    copyMessage(item)
    return
  }
  if (action === 'delete') {
    deleteMessage(item)
    return
  }
  if (action === 'edit') {
    startEditMessage(item)
    return
  }

  const actionTitle: Record<MessageActionKey, string> = {
    retry: item.role === 'user' ? '后续接入重新发送' : '后续接入重新生成',
    speak: '后续接入语音朗读',
    copy: '消息已复制',
    delete: '消息已删除',
    edit: '编辑消息',
    branch: '后续接入分支会话',
  }
  uni.showToast({
    icon: 'none',
    title: actionTitle[action],
  })
}

const showRuntimeDetail = (item: ChatMessageItem) => {
  uni.showToast({
    icon: 'none',
    title: formatRuntime(item),
  })
}

const closeOperationSheet = () => {
  actionMessageKey.value = ''
  actionSessionID.value = ''
  ignoredTapSessionID.value = ''
}

const handleMessageOperation = (action: MessageActionKey, item: ChatMessageItem) => {
  closeOperationSheet()
  handleMessageAction(action, item)
}

const handleRuntimeOperation = (item: ChatMessageItem) => {
  closeOperationSheet()
  showRuntimeDetail(item)
}

const handleSelectedMessageOperation = (action: MessageActionKey) => {
  if (!actionMessage.value) {
    return
  }
  handleMessageOperation(action, actionMessage.value)
}

const handleSelectedRuntimeOperation = () => {
  if (!actionMessage.value) {
    return
  }
  handleRuntimeOperation(actionMessage.value)
}

const handleSessionOperation = (action: SessionActionKey) => {
  const session = actionSession.value
  if (!session) {
    return
  }

  closeOperationSheet()
  if (action === 'rename') {
    openRenameSession(session)
    return
  }
  deleteSession(session.id)
}

/** 附件入口先保留静态交互，后续接入上传服务。 */
const handleAttachment = () => {
  uni.showToast({
    icon: 'none',
    title: '后续接入附件上传',
  })
}

/** 语音入口先提供移动端状态反馈，后续接入语音识别。 */
const handleToggleRecord = () => {
  isRecording.value = !isRecording.value
  uni.showToast({
    icon: 'none',
    title: isRecording.value ? '正在识别语音' : '已停止语音输入',
  })
}

/** 静态发送入口，保留后续接入真实消息接口的位置。 */
const handleSend = () => {
  if (isSubmitDisabled.value) {
    return
  }
  uni.showToast({
    icon: 'none',
    title: '静态页面，后续接入发送接口',
  })
  inputText.value = ''
}

const formatTools = (tools: AiAssistantTool[]) => {
  return tools.map((item) => item.title || item.name).join(' · ')
}

const formatRuntime = (item: ChatMessageItem) => {
  const duration = item.durationMs ? `${(item.durationMs / 1000).toFixed(1)}s` : '生成中'
  return `${item.tokenTotal} Token · 首字 ${item.firstTokenMs}ms · 总耗时 ${duration}`
}

/** 返回上一页，无历史栈时回到我的页面。 */
const onNavigateBack = () => {
  const pages = getCurrentPages()
  if (pages.length > 1) {
    uni.navigateBack()
    return
  }
  uni.switchTab({ url: '/pages/my/my' })
}
</script>

<template>
  <view class="assistant-page">
    <view class="assistant-header" :style="{ paddingTop: `${safeAreaInsets?.top || 0}px` }">
      <view class="assistant-nav">
        <view class="back-button icon-left" @tap="onNavigateBack"></view>
        <view class="page-title">AI 助手</view>
        <button class="history-button" hover-class="none" @tap="toggleSessionDrawer">会话</button>
      </view>
    </view>

    <scroll-view class="assistant-body" scroll-y :show-scrollbar="false">
      <template v-if="!hasMessages">
        <view class="empty-panel">
          <view class="empty-title">AI 助手</view>
          <view class="empty-desc">输入文字，或使用语音说出你的问题。</view>
        </view>

        <view class="empty-note">
          对话内容会在接入接口后按当前用户会话保存；这里先用于确认页面结构和输入方式。
        </view>
      </template>

      <view v-else class="chat-list">
        <view
          v-for="item in chatMessages"
          :key="item.id"
          class="message-row"
          :class="item.role === 'user' ? 'is-user' : 'is-assistant'"
        >
          <view class="message-stack" :class="item.role === 'user' ? 'is-user' : 'is-assistant'">
            <view
              class="bubble"
              :class="[
                item.role === 'assistant' ? 'assistant-bubble' : '',
                item.status === AiAssistantMessageStatus.GENERATING_AAMS ? 'is-streaming' : '',
              ]"
              @longpress="openMessageActionSheet(item, $event)"
            >
              <view v-if="item.role === 'assistant'" class="reply-meta">
                <text class="reply-tag">{{
                  item.replySource === 'llm' ? '模型回复' : '系统回复'
                }}</text>
                <text v-if="item.model" class="reply-model">{{ item.model }}</text>
              </view>

              <text class="bubble-content">{{ item.content }}</text>

              <view v-if="item.products.length" class="product-list">
                <view v-for="product in item.products" :key="product.id" class="product-card">
                  <view class="product-img"></view>
                  <view class="product-info">
                    <view class="product-name">{{ product.name }}</view>
                    <view class="product-tag">{{ product.tag }}</view>
                    <view class="product-footer">
                      <text class="price">{{ product.price }}</text>
                      <text class="view-product">查看</text>
                    </view>
                  </view>
                </view>
              </view>

              <view v-if="item.tools.length" class="tool-row"
                >已调用：{{ formatTools(item.tools) }}</view
              >
            </view>
            <view v-if="editingMessageKey === item.id" class="message-edit">
              <textarea
                v-model="editingContent"
                class="message-edit-input"
                auto-height
                maxlength="500"
                placeholder="编辑消息内容"
                placeholder-class="message-edit-placeholder"
              />
              <view class="message-edit-actions">
                <button class="message-edit-button" hover-class="none" @tap="cancelEditMessage">
                  取消
                </button>
                <button
                  class="message-edit-button is-primary"
                  hover-class="none"
                  @tap="saveEditMessage(item)"
                >
                  保存
                </button>
              </view>
            </view>
          </view>
        </view>
      </view>
    </scroll-view>

    <view class="composer" :style="{ paddingBottom: composerBottom }">
      <view class="composer-main">
        <button class="attach-button" hover-class="none" @tap="handleAttachment">
          <uni-icons type="plusempty" size="30" color="#111" />
        </button>
        <view class="composer-card">
          <textarea
            v-model="inputText"
            class="composer-input"
            auto-height
            :maxlength="500"
            :placeholder="composerPlaceholder"
            placeholder-class="composer-placeholder"
          />
          <button
            class="voice-button"
            :class="{ active: isRecording }"
            hover-class="none"
            @tap="handleToggleRecord"
          >
            <uni-icons type="mic" size="27" :color="isRecording ? '#27ba9b' : '#666'" />
          </button>
          <button
            class="send-button"
            :class="{ 'is-disabled': isSubmitDisabled }"
            :disabled="isSubmitDisabled"
            hover-class="none"
            @tap="handleSend"
          >
            <uni-icons type="paperplane" size="27" :color="isSubmitDisabled ? '#666' : '#27ba9b'" />
          </button>
        </view>
      </view>
    </view>

    <view v-if="showSessionDrawer" class="session-mask" @tap="toggleSessionDrawer"></view>
    <view
      class="session-drawer"
      :class="{ 'is-open': showSessionDrawer }"
      :style="{ paddingTop: drawerTopPadding }"
    >
      <view class="session-drawer__head">
        <view class="session-drawer__title">历史会话</view>
        <button class="session-create" hover-class="none" @tap="createSession">
          <uni-icons type="plusempty" size="16" color="#27ba9b" />
          <text>新建</text>
        </button>
      </view>
      <view class="session-search">
        <uni-icons type="search" size="16" color="#898b94" />
        <input
          v-model="sessionKeyword"
          class="session-search-input"
          confirm-type="search"
          placeholder="搜索会话"
          placeholder-class="session-search-placeholder"
        />
      </view>
      <scroll-view class="session-list" scroll-y :show-scrollbar="false">
        <view
          v-for="session in filteredSessions"
          :key="session.id"
          class="session-item"
          :class="{ 'is-active': session.id === activeSessionID }"
          @tap="selectSession(session.id)"
          @longpress="openSessionActionSheet(session, $event)"
        >
          <view class="session-content">
            <view class="session-title">{{ session.title }}</view>
            <view class="session-summary">{{ session.summary }}</view>
          </view>
        </view>
        <view v-if="!filteredSessions.length" class="session-empty">没有匹配的会话</view>
      </scroll-view>
    </view>

    <view v-if="actionMessage || actionSession" class="operation-mask" @tap="closeOperationSheet">
      <view class="operation-sheet" :style="operationSheetStyle" @tap.stop>
        <template v-if="actionMessage">
          <view class="operation-title">
            {{ actionMessage.role === 'user' ? '用户消息' : '助手消息' }}
          </view>
          <button
            v-for="action in messageOperationActions"
            :key="action.key"
            class="operation-item"
            :class="{ 'is-danger': action.danger }"
            hover-class="none"
            @tap="handleSelectedMessageOperation(action.key)"
          >
            <view
              v-if="action.icon === 'copy' || action.icon === 'branch'"
              class="operation-icon-symbol"
              :class="`is-${action.icon}`"
            ></view>
            <uni-icons
              v-else
              :type="action.icon"
              size="22"
              :color="action.danger ? '#cf4444' : '#5f6673'"
            />
            <text>{{ action.label }}</text>
          </button>
          <button
            v-if="hasRuntimeBrief(actionMessage)"
            class="operation-item"
            hover-class="none"
            @tap="handleSelectedRuntimeOperation"
          >
            <view class="operation-icon-symbol is-runtime"></view>
            <text>运行明细</text>
            <text class="operation-runtime">
              {{ formatTokenBrief(actionMessage) }} {{ formatDurationBrief(actionMessage) }}
            </text>
          </button>
        </template>
        <template v-else-if="actionSession">
          <view class="operation-title">{{ actionSession.title }}</view>
          <button class="operation-item" hover-class="none" @tap="handleSessionOperation('rename')">
            <uni-icons type="compose" size="22" color="#5f6673" />
            <text>重命名</text>
          </button>
          <button
            class="operation-item is-danger"
            hover-class="none"
            @tap="handleSessionOperation('delete')"
          >
            <uni-icons type="trash" size="22" color="#cf4444" />
            <text>删除会话</text>
          </button>
        </template>
      </view>
    </view>

    <view v-if="showRenameDialog" class="rename-mask" @tap="cancelRenameSession">
      <view class="rename-dialog" @tap.stop>
        <view class="rename-title">重命名会话</view>
        <input
          v-model="renamingTitle"
          class="rename-input"
          maxlength="30"
          placeholder="请输入会话名称"
          placeholder-class="rename-placeholder"
        />
        <view class="rename-actions">
          <button class="rename-button" hover-class="none" @tap="cancelRenameSession">取消</button>
          <button class="rename-button is-primary" hover-class="none" @tap="confirmRenameSession">
            确定
          </button>
        </view>
      </view>
    </view>
  </view>
</template>

<style lang="scss">
page {
  height: 100%;
  overflow: hidden;
  background-color: #f4f4f4;
}

.assistant-page {
  position: relative;
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  overflow: hidden;
  color: #333;
  background-color: #f4f4f4;
  box-sizing: border-box;
}

.assistant-header {
  flex-shrink: 0;
  width: 100%;
  background-color: #fff;
}

.assistant-nav {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 88rpx;
  padding: 0 24rpx;
  border-bottom: 1rpx solid #eee;
}

.back-button {
  width: 64rpx;
  height: 64rpx;
  color: #333;
  font-size: 44rpx;
  line-height: 64rpx;
}

.page-title {
  position: absolute;
  left: 50%;
  color: #333;
  font-size: 32rpx;
  font-weight: 600;
  line-height: 88rpx;
  transform: translateX(-50%);
}

.history-button,
.attach-button,
.voice-button,
.send-button,
.session-create,
.operation-item,
.message-edit-button,
.rename-button {
  padding: 0;
  margin: 0;
  border-radius: 0;
  background: transparent;
  line-height: normal;

  &::after {
    border: none;
  }
}

.history-button {
  width: 108rpx;
  height: 60rpx;
  border-radius: 60rpx;
  color: #27ba9b;
  font-size: 26rpx;
  line-height: 60rpx;
  text-align: center;
  background-color: #e8f8f4;
}

.assistant-body {
  flex: 1;
  width: 100%;
  min-height: 0;
  padding: 20rpx;
  box-sizing: border-box;
  background-color: #f4f4f4;
}

.empty-panel,
.empty-note {
  border-radius: 10rpx;
  background-color: #fff;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.03);
}

.empty-panel {
  padding: 56rpx 36rpx 48rpx;
  text-align: center;
}

.empty-title {
  color: #333;
  font-size: 34rpx;
  font-weight: 600;
  line-height: 44rpx;
}

.empty-desc {
  margin-top: 14rpx;
  color: #898b94;
  font-size: 26rpx;
  line-height: 40rpx;
}

.empty-note {
  margin-top: 20rpx;
  padding: 26rpx 24rpx;
  color: #898b94;
  font-size: 24rpx;
  line-height: 38rpx;
}

.chat-list {
  padding-bottom: 36rpx;
}

.message-row {
  display: flex;
  margin-bottom: 28rpx;
}

.message-row.is-user {
  justify-content: flex-end;
}

.message-row.is-assistant {
  justify-content: flex-start;
}

.bubble {
  max-width: 660rpx;
  padding: 24rpx 26rpx;
  border-radius: 10rpx;
  color: #fff;
  font-size: 28rpx;
  line-height: 42rpx;
  background-color: #27ba9b;
  box-sizing: border-box;
}

.assistant-bubble {
  width: 100%;
  color: #333;
  background-color: #fff;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.03);
}

.assistant-bubble.is-streaming {
  box-shadow: inset 0 0 0 1rpx #c7eee4;
}

.reply-meta {
  display: flex;
  align-items: center;
  gap: 14rpx;
  margin-bottom: 16rpx;
}

.reply-tag {
  padding: 4rpx 14rpx;
  border-radius: 8rpx;
  color: #27ba9b;
  font-size: 20rpx;
  line-height: 28rpx;
  background-color: #e8f8f4;
}

.reply-model {
  color: #999;
  font-size: 20rpx;
  line-height: 28rpx;
}

.bubble-content {
  white-space: pre-wrap;
  word-break: break-word;
}

.product-list {
  margin-top: 22rpx;
}

.product-card {
  display: flex;
  gap: 20rpx;
  padding: 18rpx;
  border-radius: 10rpx;
  background-color: #f6f7f9;
}

.product-card + .product-card {
  margin-top: 16rpx;
}

.product-img {
  flex-shrink: 0;
  width: 116rpx;
  height: 116rpx;
  border-radius: 8rpx;
  background-color: #ececec;
}

.product-info {
  flex: 1;
  min-width: 0;
}

.product-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #333;
  font-size: 26rpx;
  font-weight: 600;
  line-height: 34rpx;
}

.product-tag {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-top: 12rpx;
  padding: 4rpx 12rpx;
  border-radius: 18rpx;
  color: #ad7a12;
  font-size: 20rpx;
  line-height: 28rpx;
  background-color: #fff7e8;
}

.product-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 14rpx;
}

.price {
  color: #cf4444;
  font-size: 26rpx;
  font-weight: 700;
  line-height: 32rpx;
}

.view-product {
  padding: 6rpx 16rpx;
  border-radius: 8rpx;
  color: #27ba9b;
  font-size: 20rpx;
  line-height: 28rpx;
  background-color: #e8f8f4;
}

.tool-row {
  margin-top: 20rpx;
  padding: 16rpx 18rpx;
  border-radius: 8rpx;
  color: #898b94;
  font-size: 22rpx;
  line-height: 32rpx;
  background-color: #f6f7f9;
}

.message-edit {
  margin-top: 18rpx;
  padding-top: 18rpx;
  border-top: 1rpx solid #f0f0f0;
}

.message-edit-input {
  width: 100%;
  min-height: 112rpx;
  padding: 18rpx;
  border-radius: 10rpx;
  box-sizing: border-box;
  color: #333;
  font-size: 26rpx;
  line-height: 40rpx;
  background-color: #f7f7f8;
}

.message-edit-placeholder {
  color: #b8bcc5;
}

.message-edit-actions {
  display: flex;
  justify-content: flex-end;
  gap: 14rpx;
  margin-top: 14rpx;
}

.message-edit-button {
  width: 104rpx;
  height: 52rpx;
  border-radius: 8rpx;
  color: #6b7280;
  font-size: 24rpx;
  line-height: 52rpx;
  background-color: #f6f7f9;
}

.message-edit-button.is-primary {
  color: #fff;
  background-color: #27ba9b;
}

.composer {
  flex-shrink: 0;
  width: 100%;
  padding: 18rpx 22rpx 20rpx;
  overflow: hidden;
  background-color: transparent;
  box-sizing: border-box;
}

.attach-button {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 92rpx;
  height: 92rpx;
  border-radius: 50%;
  background-color: #fff;
  box-shadow: 0 10rpx 28rpx rgba(15, 23, 42, 0.08);
}

.composer-card {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 12rpx;
  min-height: 92rpx;
  padding: 10rpx 12rpx 10rpx 28rpx;
  border-radius: 46rpx;
  background-color: #fff;
  box-shadow: 0 10rpx 28rpx rgba(15, 23, 42, 0.08);
  box-sizing: border-box;
}

.composer-main {
  display: flex;
  align-items: center;
  gap: 18rpx;
  box-sizing: border-box;
}

.composer-input {
  flex: 1;
  min-width: 0;
  max-height: 118rpx;
  padding: 15rpx 0;
  box-sizing: border-box;
  color: #333;
  font-size: 32rpx;
  line-height: 46rpx;
  background-color: transparent;
}

.composer-placeholder {
  color: #8a8a8a;
  font-size: 32rpx;
}

.voice-button,
.send-button {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64rpx;
  height: 64rpx;
  border-radius: 50%;
  background-color: transparent;
}

.voice-button.active {
  background-color: #e8f8f4;
}

.send-button.is-disabled {
  background-color: transparent;
}

.session-mask {
  position: absolute;
  top: 0;
  right: 560rpx;
  bottom: 0;
  left: 0;
  z-index: 20;
  background-color: rgba(0, 0, 0, 0.18);
}

.session-drawer {
  display: flex;
  flex-direction: column;
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  z-index: 21;
  width: 560rpx;
  padding: 0 24rpx 36rpx;
  background-color: #fff;
  box-shadow: none;
  box-sizing: border-box;
  transform: translateX(100%);
  transition: transform 0.2s ease;
}

.session-drawer.is-open {
  box-shadow: -24rpx 0 60rpx rgba(0, 0, 0, 0.12);
  transform: translateX(0);
}

.session-drawer__head {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 76rpx;
  margin-bottom: 18rpx;
}

.session-drawer__title {
  color: #333;
  font-size: 32rpx;
  font-weight: 700;
  line-height: 40rpx;
}

.session-create {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4rpx;
  width: 112rpx;
  height: 52rpx;
  border-radius: 8rpx;
  color: #27ba9b;
  font-size: 24rpx;
  line-height: 52rpx;
  background-color: #e8f8f4;
}

.session-search {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 10rpx;
  height: 64rpx;
  padding: 0 18rpx;
  margin-bottom: 18rpx;
  border-radius: 10rpx;
  background-color: #f6f7f9;
  box-sizing: border-box;
}

.session-search-input {
  flex: 1;
  min-width: 0;
  color: #333;
  font-size: 24rpx;
}

.session-search-placeholder {
  color: #b8bcc5;
}

.session-list {
  flex: 1;
  min-height: 0;
}

.session-item {
  display: flex;
  align-items: center;
  gap: 12rpx;
  padding: 24rpx;
  border: 1rpx solid #eee;
  border-radius: 10rpx;
  background-color: #fff;
}

.session-item + .session-item {
  margin-top: 18rpx;
}

.session-item.is-active {
  border-color: #c7eee4;
  background-color: #e8f8f4;
}

.session-content {
  flex: 1;
  min-width: 0;
}

.session-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #333;
  font-size: 26rpx;
  font-weight: 600;
  line-height: 34rpx;
}

.session-summary {
  margin-top: 10rpx;
  color: #777;
  font-size: 22rpx;
  line-height: 32rpx;
}

.session-empty {
  padding: 60rpx 0;
  color: #999;
  font-size: 24rpx;
  text-align: center;
}

.operation-mask {
  position: absolute;
  inset: 0;
  z-index: 28;
  background-color: rgba(0, 0, 0, 0.05);
}

.operation-sheet {
  position: absolute;
  padding: 10rpx 0;
  border: 1rpx solid #f0f0f0;
  border-radius: 10rpx;
  background-color: #fff;
  box-shadow: 0 16rpx 44rpx rgba(15, 23, 42, 0.16);
  box-sizing: border-box;
}

.operation-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding: 8rpx 24rpx 10rpx;
  border-bottom: 1rpx solid #f5f5f5;
  color: #999;
  font-size: 22rpx;
  line-height: 30rpx;
}

.operation-item {
  display: flex;
  align-items: center;
  gap: 18rpx;
  width: 100%;
  height: 76rpx;
  padding: 0 24rpx;
  color: #333;
  font-size: 26rpx;
  line-height: 76rpx;
  text-align: left;
  box-sizing: border-box;
}

.operation-item.is-danger {
  color: #cf4444;
}

.operation-runtime {
  flex: 1;
  min-width: 0;
  color: #999;
  font-size: 22rpx;
  text-align: right;
}

.operation-icon-symbol {
  position: relative;
  flex-shrink: 0;
  width: 42rpx;
  height: 42rpx;
  color: #5f6673;
}

.operation-icon-symbol.is-copy::before,
.operation-icon-symbol.is-copy::after {
  position: absolute;
  width: 22rpx;
  height: 22rpx;
  border: 4rpx solid currentColor;
  border-radius: 6rpx;
  content: '';
}

.operation-icon-symbol.is-copy::before {
  top: 15rpx;
  left: 7rpx;
}

.operation-icon-symbol.is-copy::after {
  top: 7rpx;
  left: 15rpx;
  background-color: #fff;
}

.operation-icon-symbol.is-branch::before,
.operation-icon-symbol.is-branch::after {
  position: absolute;
  content: '';
}

.operation-icon-symbol.is-branch::before {
  top: 22rpx;
  left: 8rpx;
  width: 26rpx;
  height: 4rpx;
  background-color: currentColor;
  transform: rotate(-38deg);
  transform-origin: right center;
}

.operation-icon-symbol.is-branch::after {
  top: 7rpx;
  right: 5rpx;
  width: 15rpx;
  height: 15rpx;
  border-top: 4rpx solid currentColor;
  border-right: 4rpx solid currentColor;
  transform: rotate(-8deg);
}

.operation-icon-symbol.is-runtime::before {
  position: absolute;
  top: 10rpx;
  left: 6rpx;
  width: 32rpx;
  height: 24rpx;
  border: 4rpx solid currentColor;
  border-radius: 4rpx;
  content: '';
}

.operation-icon-symbol.is-runtime::after {
  position: absolute;
  top: 17rpx;
  left: 15rpx;
  width: 5rpx;
  height: 13rpx;
  border-radius: 4rpx;
  background-color: currentColor;
  box-shadow:
    8rpx -5rpx 0 currentColor,
    16rpx -9rpx 0 currentColor;
  content: '';
}

.rename-mask {
  position: absolute;
  inset: 0;
  z-index: 30;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40rpx;
  background-color: rgba(0, 0, 0, 0.28);
  box-sizing: border-box;
}

.rename-dialog {
  width: 100%;
  padding: 32rpx 28rpx 26rpx;
  border-radius: 10rpx;
  background-color: #fff;
  box-sizing: border-box;
}

.rename-title {
  color: #333;
  font-size: 32rpx;
  font-weight: 600;
  line-height: 42rpx;
}

.rename-input {
  width: 100%;
  height: 76rpx;
  padding: 0 20rpx;
  margin-top: 26rpx;
  border-radius: 10rpx;
  box-sizing: border-box;
  color: #333;
  font-size: 28rpx;
  background-color: #f7f7f8;
}

.rename-placeholder {
  color: #b8bcc5;
}

.rename-actions {
  display: flex;
  justify-content: flex-end;
  gap: 16rpx;
  margin-top: 28rpx;
}

.rename-button {
  width: 120rpx;
  height: 60rpx;
  border-radius: 8rpx;
  color: #6b7280;
  font-size: 26rpx;
  line-height: 60rpx;
  background-color: #f6f7f9;
}

.rename-button.is-primary {
  color: #fff;
  background-color: #27ba9b;
}
</style>
