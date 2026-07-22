<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { computed, nextTick, onBeforeUnmount, ref } from 'vue'
import { defBaseAreaService } from '@/api/system/base_area'
import { defAiMessageService, StreamAiMessageByChunkedRequest } from '@/api/base/ai_message'
import { defAiSessionService } from '@/api/base/ai_session'
import { defAiToolService } from '@/api/base/ai_tool'
import type { AiAction } from '@/rpc/base/v1/ai_message'
import type { AppTreeOptionResponse_Option } from '@/rpc/common/v1/common'
import type { AiAttachment, AiMessage, AiSession } from '@/rpc/base/v1/ai_session'
import type { AiShortcut, AiToolCall } from '@/rpc/base/v1/ai_tool'
import { AiMessageStatus, Terminal } from '@/rpc/common/v1/enum'
import { uploadFile } from '@/utils/file'
import { formatSrc } from '@/utils/index'
import { appendOrderPaymentRedirectUrl, redirectToOrderPayment } from '@/utils/navigation'
import Composer from './components/Composer.vue'
import FlowBlocks from './components/FlowBlocks.vue'
import SessionDrawer from './components/SessionDrawer.vue'
import WelcomePanel from './components/WelcomePanel.vue'
import {
  type AiStreamEvent,
  type AiStreamPayload,
  createAiEventStreamTextParser,
  parseAiEventStreamText,
  readAiEventStream,
} from './stream'

type ChatRole = 'user' | 'ai'

type MessageActionKey = 'retry' | 'speak' | 'copy' | 'delete' | 'edit' | 'branch'
type SessionActionKey = 'rename' | 'delete'
type OperationIcon =
  | 'refresh'
  | 'copy-document'
  | 'delete'
  | 'edit-pen'
  | 'branch-action'
  | 'speak-action'

type MessageAction = {
  key: MessageActionKey
  icon: OperationIcon
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

type ChatMessageItem = AiMessage & {
  key: string
  messageID: string
  role: ChatRole
  content: string
  status: AiMessageStatus
  tools: AiToolCall[]
  model: string
  replySource: string
  fallback: boolean
  fallbackReason: string
  flow: string
  step: string
  blocksJson: string
  blocks: AIFlowBlock[]
  tokenTotal: number
  firstTokenMs: number
  durationMs: number
  localOnly?: boolean
  streamKey?: string
  flowReveal?: Record<string, number>
  speaking?: boolean
}

type SubmitPayload = {
  text: string
  attachments: AiAttachment[]
  action?: AiAction
}

type AIFlowBlock = {
  type: string
  [key: string]: any
}

type StreamTask = {
  abort: () => void
  aborted: boolean
  finished: boolean
  success?: boolean
}

type AttachmentFileCandidate = {
  name: string
  path: string
  size: number
  extension: string
  mimeType: string
}

type AddressFormStepKey = 'receiver' | 'contact' | 'address' | 'detail'

type MenuButtonRect = {
  top: number
  bottom: number
  left: number
  right: number
  width: number
  height: number
}

const THINKING_MESSAGE_CONTENT = '正在回复'
const LOCAL_USER_MESSAGE_PREFIX = 'ai-user-local'
const PENDING_MESSAGE_ID = 'pending'
const SUBMITTED_ADDRESS_FORM_BLOCKS_KEY = 'ai_submitted_address_form_blocks'
const SUBMITTED_ADDRESS_FORM_BLOCK_LIMIT = 200
const MAX_ATTACHMENT_COUNT = 6
const STARTER_PROMPT_PAGE_SIZE = 4
const AI_TERMINAL = Terminal.TERMINAL_APP
const FLOW_REVEAL_INTERVAL_MS = 90
const FLOW_REVEAL_CLEANUP_MS = 240
const MOBILE_PHONE_PATTERN = /^1[3-9]\d{9}$/
const addressFormSteps: {
  key: AddressFormStepKey
  label: string
  shortLabel: string
  hint: string
  placeholder: string
}[] = [
  {
    key: 'receiver',
    label: '收货人',
    shortLabel: '姓名',
    hint: '先填写收货人姓名',
    placeholder: '请输入收货人姓名',
  },
  {
    key: 'contact',
    label: '手机号',
    shortLabel: '电话',
    hint: '再填写收货人手机号',
    placeholder: '请输入手机号',
  },
  {
    key: 'address',
    label: '所在地区',
    shortLabel: '地区',
    hint: '选择省市区',
    placeholder: '请选择省/市/区',
  },
  {
    key: 'detail',
    label: '详细地址',
    shortLabel: '门牌',
    hint: '最后填写街道、楼栋和门牌号',
    placeholder: '例如：海淀路 1 号 2 单元 301',
  },
]
const imageAttachmentExtensions = ['jpg', 'jpeg', 'png', 'gif', 'webp']
const documentAttachmentExtensions = [
  'pdf',
  'doc',
  'docx',
  'xls',
  'xlsx',
  'ppt',
  'pptx',
  'txt',
  'json',
  'csv',
  'xml',
  'md',
  'markdown',
]
const supportedAttachmentExtensions = [
  ...imageAttachmentExtensions,
  ...documentAttachmentExtensions,
]
const attachmentMimeMap: Record<string, string> = {
  jpg: 'image/jpeg',
  jpeg: 'image/jpeg',
  png: 'image/png',
  gif: 'image/gif',
  webp: 'image/webp',
  pdf: 'application/pdf',
  doc: 'application/msword',
  docx: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
  xls: 'application/vnd.ms-excel',
  xlsx: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
  ppt: 'application/vnd.ms-powerpoint',
  pptx: 'application/vnd.openxmlformats-officedocument.presentationml.presentation',
  txt: 'text/plain',
  json: 'application/json',
  csv: 'text/csv',
  xml: 'application/xml',
  md: 'text/plain',
  markdown: 'text/plain',
}

const systemInfo = uni.getSystemInfoSync()
const { safeAreaInsets } = systemInfo
const composerBottom = `${Math.max(safeAreaInsets?.bottom || 0, 9)}px`
const windowWidth = systemInfo.windowWidth || systemInfo.screenWidth || 375
const windowHeight = systemInfo.windowHeight || systemInfo.screenHeight || 667
const menuButtonRect = resolveMenuButtonRect()
const statusBarHeight = systemInfo.statusBarHeight || safeAreaInsets?.top || 0
const navRowHeightValue = menuButtonRect
  ? menuButtonRect.height + Math.max(menuButtonRect.top - statusBarHeight, 0) * 2
  : 44
const navTopPadding = `${statusBarHeight}px`
const navHeight = `${statusBarHeight + navRowHeightValue}px`
const navRowHeight = `${navRowHeightValue}px`
const navButtonSize = `${menuButtonRect?.height || 32}px`
const navSideWidth = `${Math.max(menuButtonRect ? windowWidth - menuButtonRect.left : 88, 88)}px`
const drawerTopPadding = `${statusBarHeight + 12}px`
const showSessionDrawer = ref(false)
const activeSessionID = ref('')
const inputText = ref('')
const isRecording = ref(false)
const starterPromptGroupIndex = ref(0)
const sessionKeyword = ref('')
const showRenameDialog = ref(false)
const renamingSessionID = ref('')
const renamingTitle = ref('')
const editingMessageKey = ref('')
const editingContent = ref('')
const actionMessageKey = ref('')
const actionSessionID = ref('')
const ignoredTapSessionID = ref('')
const loadingSessions = ref(false)
const loadingSessionID = ref('')
const loadingShortcuts = ref(false)
const chatBottomAnchor = ref('')
const uploadingAttachment = ref(false)
const showTextPreview = ref(false)
const loadingTextPreview = ref(false)
const textPreviewTitle = ref('')
const textPreviewContent = ref('')
const sendingSessionMap = ref<Record<string, boolean>>({})
const sessions = ref<AiSession[]>([])
const starterShortcuts = ref<AiShortcut[]>([])
const messages = ref<Record<string, ChatMessageItem[]>>({})
const selectedAttachments = ref<AiAttachment[]>([])
const addressAreaTree = ref<AppTreeOptionResponse_Option[]>([])
const runningStreamTaskMap = new Map<string, StreamTask>()
const pendingDeltaMap = new Map<string, AiStreamPayload>()
const handledPaymentBlockSet = new Set<string>()
const flowRevealTimerMap = new Map<string, number>()
const submittedAddressFormBlockSet = new Set(readSubmittedAddressFormBlockKeys())
let speakingMessageKey = ''
let pendingDeltaTimer = 0
let loadingAddressAreaTree = false
const operationPoint = ref({
  x: Math.round(windowWidth / 2),
  y: Math.round(windowHeight / 2),
})

const filteredSessions = computed(() => {
  const keyword = sessionKeyword.value
  if (!keyword) {
    return sessions.value
  }
  return sessions.value.filter(
    (item) => item.title.includes(keyword) || item.summary.includes(keyword),
  )
})

const currentMessages = computed(() => messages.value[activeSessionID.value] ?? [])
const hasMessages = computed(() => currentMessages.value.length > 0)
const currentSessionSending = computed(() => isSessionSending(activeSessionID.value))
const activeFlowMessageID = computed(() => {
  const list = currentMessages.value.filter(
    (item) =>
      item.role === 'ai' && !item.localOnly && item.status !== AiMessageStatus.GENERATING_AAMS,
  )
  return list[list.length - 1]?.messageID ?? ''
})
const starterPromptPageCount = computed(() => {
  return Math.max(1, Math.ceil(starterShortcuts.value.length / STARTER_PROMPT_PAGE_SIZE))
})
const canRefreshStarterPrompts = computed(
  () => starterShortcuts.value.length > STARTER_PROMPT_PAGE_SIZE,
)
const starterPrompts = computed(() => {
  const pageIndex = starterPromptGroupIndex.value % starterPromptPageCount.value
  const start = pageIndex * STARTER_PROMPT_PAGE_SIZE
  return starterShortcuts.value.slice(start, start + STARTER_PROMPT_PAGE_SIZE)
})
const aiGreetingPeriod = computed(() => {
  const hour = new Date().getHours()
  if (hour < 11) {
    return '上午'
  }
  if (hour < 14) {
    return '中午'
  }
  if (hour < 18) {
    return '下午'
  }
  return '晚上'
})
const aiGreetingMessage = computed(() => {
  return `尊敬的用户您好：❤️${aiGreetingPeriod.value}浪漫时光别有风味，请问有什么可以帮您的~🐥`
})

const actionMessage = computed(() => {
  return currentMessages.value.find((item) => item.key === actionMessageKey.value)
})

const actionSession = computed(() => {
  return sessions.value.find((item) => item.id === actionSessionID.value)
})

const composerPlaceholder = computed(() => {
  if (isRecording.value) {
    return '正在听...'
  }
  if (uploadingAttachment.value) {
    return '附件上传中...'
  }
  return hasMessages.value ? '继续追问或补充预算' : '问问商城 AI 助手'
})

const isSubmitDisabled = computed(() => {
  return (
    uploadingAttachment.value ||
    currentSessionSending.value ||
    isRecording.value ||
    (!inputText.value && selectedAttachments.value.length === 0)
  )
})

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
  const titleHeight = actionMessage.value || actionSession.value ? 30 : 0
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

const isThinkingMessage = (item: ChatMessageItem) => {
  return item.role === 'ai' && item.status === AiMessageStatus.GENERATING_AAMS
}

const lastEditableUserMessageKey = computed(() => {
  const list = currentMessages.value
    .filter((item) => item.role === 'user' && !item.localOnly)
    .sort((left, right) => resolveTimestamp(left.created_at) - resolveTimestamp(right.created_at))
  const lastMessage = list[list.length - 1]
  if (!lastMessage || lastMessage.status === AiMessageStatus.GENERATING_AAMS) {
    return ''
  }
  return lastMessage.key
})

/** 首次打开时加载移动端会话列表。 */
onLoad(() => {
  void loadAiShortcuts()
  void ensureSessionsLoaded()
})

onBeforeUnmount(() => {
  cancelAllStreamTasks()
  cancelAllFlowRevealTimers()
  clearPendingDelta()
})

/** 打开或收起历史会话抽屉。 */
const toggleSessionDrawer = () => {
  closeOperationSheet()
  showSessionDrawer.value = !showSessionDrawer.value
}

/** 切换当前会话并加载消息。 */
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
  if (!messages.value[sessionID]?.length || !isSessionSending(sessionID)) {
    void loadMessages(sessionID)
    return
  }
  scrollChatToBottom()
}

/** 新建会话，并切换到新会话。 */
const createSession = async () => {
  try {
    const sessionID = await createRemoteSession()
    if (!sessionID) {
      return
    }
    activeSessionID.value = sessionID
    messages.value[sessionID] = []
    sessionKeyword.value = ''
    showSessionDrawer.value = false
    uni.showToast({ icon: 'none', title: '已创建新会话' })
  } catch (error) {
    showError(error, '创建会话失败')
  }
}

/** 打开会话重命名弹窗。 */
const openRenameSession = (session: AiSession) => {
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

/** 保存会话名称，并同步后端。 */
const confirmRenameSession = async () => {
  if (!renamingTitle.value) {
    uni.showToast({ icon: 'none', title: '请输入会话名称' })
    return
  }

  try {
    const response = await defAiSessionService.UpdateAiSession({
      id: renamingSessionID.value,
      title: renamingTitle.value,
    })
    upsertSession(normalizeSession(response.session))
    cancelRenameSession()
    uni.showToast({ icon: 'none', title: '会话已重命名' })
  } catch (error) {
    showError(error, '重命名失败')
  }
}

/** 删除会话，并自动切换到剩余会话。 */
const deleteSession = async (sessionID: string) => {
  const session = sessions.value.find((item) => item.id === sessionID)
  const result = await uni.showModal({
    title: '删除会话',
    content: `是否删除「${session?.title || '当前会话'}」？`,
    confirmText: '删除',
    confirmColor: '#cf4444',
  })
  if (!result.confirm) {
    return
  }

  try {
    await defAiSessionService.DeleteAiSession({ id: sessionID })
    cancelSessionStreamTask(sessionID)
    cancelSessionFlowRevealTimers(sessionID)
    sessions.value = sessions.value.filter((item) => item.id !== sessionID)
    delete messages.value[sessionID]
    if (activeSessionID.value === sessionID) {
      activeSessionID.value = sessions.value[0]?.id ?? ''
    }
    const nextSessionID = await ensureActiveSession()
    if (nextSessionID) {
      await loadMessages(nextSessionID)
    }
    uni.showToast({ icon: 'none', title: '已删除会话' })
  } catch (error) {
    showError(error, '删除会话失败')
  }
}

/** 复制当前消息正文。 */
const copyMessage = (item: ChatMessageItem) => {
  uni.setClipboardData({
    data: item.content,
    success: () => {
      uni.showToast({ icon: 'none', title: '消息已复制' })
    },
  })
}

/** 删除当前轮消息，并同步后端。 */
const deleteMessage = async (item: ChatMessageItem) => {
  const sessionID = activeSessionID.value
  if (!sessionID) {
    return
  }

  try {
    if (item.localOnly) {
      messages.value[sessionID] = (messages.value[sessionID] ?? []).filter((message) => {
        if (item.role === 'user') {
          return !message.localOnly
        }
        return message.key !== item.key
      })
      editingMessageKey.value = ''
      editingContent.value = ''
      closeOperationSheet()
      uni.showToast({ icon: 'none', title: '消息已删除' })
      return
    }

    if (!item.localOnly) {
      await defAiMessageService.DeleteAiMessage({
        session_id: sessionID,
        message_id: item.messageID,
      })
    }
    messages.value[sessionID] = (messages.value[sessionID] ?? []).filter(
      (message) => message.messageID !== item.messageID,
    )
    editingMessageKey.value = ''
    editingContent.value = ''
    closeOperationSheet()
    uni.showToast({ icon: 'none', title: '消息已删除' })
  } catch (error) {
    showError(error, '删除消息失败')
  }
}

/** 开始编辑用户消息正文。 */
const startEditMessage = (item: ChatMessageItem) => {
  if (!isLastEditableUserMessage(item)) {
    return
  }
  editingMessageKey.value = item.key
  editingContent.value = item.content
}

/** 取消当前消息编辑。 */
const cancelEditMessage = () => {
  editingMessageKey.value = ''
  editingContent.value = ''
}

/** 保存用户消息编辑，并重新生成助手输出。 */
const saveEditMessage = async (item: ChatMessageItem) => {
  if (!editingContent.value) {
    uni.showToast({ icon: 'none', title: '请输入消息内容' })
    return
  }

  const sessionID = activeSessionID.value
  if (!sessionID || currentSessionSending.value || item.role !== 'user' || item.localOnly) {
    return
  }

  setSessionSending(sessionID, true)
  try {
    const updatedList = currentMessages.value.map((message) => {
      if (message.messageID !== item.messageID || message.role !== 'user') {
        return message
      }
      return {
        ...message,
        content: editingContent.value,
        input_content: {
          kind: message.input_content?.kind || 'text',
          content: editingContent.value,
        },
      }
    })
    messages.value[sessionID] = markAIMessageRegenerating(updatedList, sessionID, item.messageID)

    const response = await defAiMessageService.UpdateAiMessage({
      session_id: sessionID,
      message_id: item.messageID,
      content: editingContent.value,
    })
    messages.value[sessionID] = replacePendingMessages(
      messages.value[sessionID] ?? [],
      normalizeMessageList(response.messages),
    )
    if (response.session) {
      upsertSession(normalizeSession(response.session))
    }
    cancelEditMessage()
    closeOperationSheet()
    uni.showToast({ icon: 'none', title: '消息已更新' })
  } catch (error) {
    await loadMessages(sessionID, { force: true })
    showError(error, '更新消息失败')
  } finally {
    setSessionSending(sessionID, false)
  }
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
  if (editingMessageKey.value === item.key) {
    return
  }
  if (!resolveMessageActions(item).length && !hasRuntimeBrief(item)) {
    return
  }
  resolvePressPoint(event)
  actionMessageKey.value = item.key
  actionSessionID.value = ''
}

/** 长按会话后展示移动端会话操作菜单。 */
const openSessionActionSheet = (session: AiSession, event?: PressEvent) => {
  ignoredTapSessionID.value = session.id
  resolvePressPoint(event)
  actionSessionID.value = session.id
  actionMessageKey.value = ''
}

const isLastEditableUserMessage = (item: ChatMessageItem) => {
  return item.role === 'user' && !item.localOnly && item.key === lastEditableUserMessageKey.value
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
  return item.role === 'ai' && Boolean(item.tokenTotal || item.durationMs)
}

const resolveMessageActions = (item: ChatMessageItem): MessageAction[] => {
  if (item.status === AiMessageStatus.GENERATING_AAMS) {
    return []
  }

  const copyAction: MessageAction = { key: 'copy', icon: 'copy-document', label: '复制' }
  const deleteAction: MessageAction = { key: 'delete', icon: 'delete', label: '删除', danger: true }
  const editAction: MessageAction = { key: 'edit', icon: 'edit-pen', label: '编辑' }

  if (item.role === 'user') {
    const actions = isLastEditableUserMessage(item)
      ? [editAction, copyAction, deleteAction]
      : [copyAction, deleteAction]
    if (item.status === AiMessageStatus.FAILED_AAMS) {
      return [{ key: 'retry', icon: 'refresh', label: '重新发送' }, ...actions]
    }
    return item.localOnly ? [copyAction, deleteAction] : actions
  }

  const aiActions: MessageAction[] = [{ key: 'retry', icon: 'refresh', label: '重新生成' }]
  if (item.status === AiMessageStatus.SUCCESS_AAMS) {
    aiActions.push({ key: 'branch', icon: 'branch-action', label: '创建分支' })
    aiActions.push({
      key: 'speak',
      icon: 'speak-action',
      label: item.speaking ? '停止朗读' : '朗读',
    })
  }
  return [...aiActions, copyAction, deleteAction]
}

const handleMessageAction = async (action: MessageActionKey, item: ChatMessageItem) => {
  if (action === 'copy') {
    copyMessage(item)
    return
  }
  if (action === 'delete') {
    await deleteMessage(item)
    return
  }
  if (action === 'edit') {
    startEditMessage(item)
    return
  }
  if (action === 'branch') {
    await branchMessage(item)
    return
  }
  if (action === 'speak') {
    toggleSpeakMessage(item)
    return
  }
  await retryMessage(item)
}

const showRuntimeDetail = (item: ChatMessageItem) => {
  uni.showModal({
    title: '运行明细',
    content: formatRuntime(item),
    showCancel: false,
    confirmText: '知道了',
  })
}

const closeOperationSheet = () => {
  actionMessageKey.value = ''
  actionSessionID.value = ''
  ignoredTapSessionID.value = ''
}

const handleMessageOperation = async (action: MessageActionKey, item: ChatMessageItem) => {
  closeOperationSheet()
  await handleMessageAction(action, item)
}

const handleRuntimeOperation = (item: ChatMessageItem) => {
  closeOperationSheet()
  showRuntimeDetail(item)
}

const handleSelectedMessageOperation = (action: MessageActionKey) => {
  if (!actionMessage.value) {
    return
  }
  void handleMessageOperation(action, actionMessage.value)
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
  void deleteSession(session.id)
}

/** 返回上一页，缺少页面栈时回到首页。 */
const navigateBack = () => {
  const pages = getCurrentPages()
  if (pages.length > 1) {
    uni.navigateBack()
    return
  }
  uni.switchTab({ url: '/pages/index/index' })
}

/** 上传助手附件，H5 与微信小程序共用同一套类型白名单。 */
const handleAttachment = async () => {
  if (uploadingAttachment.value || currentSessionSending.value) {
    return
  }
  if (selectedAttachments.value.length >= MAX_ATTACHMENT_COUNT) {
    uni.showToast({ icon: 'none', title: `最多上传 ${MAX_ATTACHMENT_COUNT} 个附件` })
    return
  }

  try {
    const attachmentFiles = await chooseAttachmentFiles(
      MAX_ATTACHMENT_COUNT - selectedAttachments.value.length,
    )
    if (!attachmentFiles.length) {
      return
    }
    uploadingAttachment.value = true
    const files = await Promise.all(attachmentFiles.map((item) => uploadFile('ai', item.path)))
    const nextAttachments = files.map<AiAttachment>((file, index) => ({
      id: file.url || `${file.name}-${index}`,
      name: attachmentFiles[index]?.name || file.name,
      size: attachmentFiles[index]?.size || 0,
      url: file.url,
      mime_type: attachmentFiles[index]?.mimeType || resolveMimeType(file.name),
    }))
    selectedAttachments.value = [...selectedAttachments.value, ...nextAttachments].slice(
      0,
      MAX_ATTACHMENT_COUNT,
    )
    if (nextAttachments.length) {
      uni.showToast({ icon: 'none', title: `已上传 ${nextAttachments.length} 个附件` })
    }
  } catch (error) {
    if (isUserCancelError(error)) {
      return
    }
    showError(error, '附件上传失败')
  } finally {
    uploadingAttachment.value = false
  }
}

/** 语音入口先保留移动端状态反馈。 */
const handleToggleRecord = () => {
  isRecording.value = !isRecording.value
  uni.showToast({
    icon: 'none',
    title: isRecording.value ? '正在识别语音' : '已停止语音输入',
  })
}

/** 发送消息并同步服务端回复。 */
const handleSend = async () => {
  if (isSubmitDisabled.value) {
    return
  }
  const text = inputText.value || (selectedAttachments.value.length ? '请结合附件内容继续分析' : '')
  if (!selectedAttachments.value.length && (await fillActiveAddressFormFromText(text))) {
    inputText.value = ''
    scrollChatToBottom()
    uni.showToast({ icon: 'none', title: '已填入地址表单，请确认后保存' })
    return
  }
  const payload = {
    text,
    attachments: [...selectedAttachments.value],
  }
  inputText.value = ''
  selectedAttachments.value = []
  await sendAiPayload(payload)
}

/** 使用空态快捷入口直接开始对话。 */
const handleStarterPrompt = async (shortcut: AiShortcut) => {
  if (
    loadingSessions.value ||
    uploadingAttachment.value ||
    currentSessionSending.value ||
    isRecording.value
  ) {
    return
  }
  inputText.value = ''
  selectedAttachments.value = []
  await sendAiPayload({
    text: shortcut.prompt || shortcut.title,
    attachments: [],
    action: buildStarterPromptAction(shortcut),
  })
}

/** 轮换本地已加载的空态快捷入口。 */
const refreshStarterPrompts = () => {
  if (!canRefreshStarterPrompts.value) {
    return
  }
  starterPromptGroupIndex.value = (starterPromptGroupIndex.value + 1) % starterPromptPageCount.value
}

/** 提交助手流程动作，保持流程在聊天内闭环。 */
const handleFlowAction = async (action?: AiAction, label?: string) => {
  if (!action || currentSessionSending.value) {
    return false
  }
  if (!isCurrentFlowAction(action)) {
    showExpiredFlowActionToast()
    return false
  }
  const nextAction = enrichFlowAction(action)
  return sendAiPayload({
    text: label || resolveFlowActionLabel(nextAction),
    attachments: [],
    action: nextAction,
  })
}

/** 提交规格选择动作。 */
const submitSkuSelection = (block: AIFlowBlock, sku: AIFlowBlock) => {
  const payload = parseActionPayload(block.action?.payload_json)
  payload.sku_code = sku.sku_code
  payload.num = Number(sku.num || 1)
  const action = buildFlowAction(block.action, payload)
  void handleFlowAction(action, `选择规格：${sku.spec_text || sku.sku_code}`)
}

/** 调整规格数量。 */
const changeSkuNum = (sku: AIFlowBlock, delta: number) => {
  const current = Number(sku.num || 1)
  const inventory = Number(sku.inventory || 0)
  const next = Math.max(1, current + delta)
  sku.num = inventory > 0 ? Math.min(next, inventory) : next
}

/** 提交新增地址表单。 */
const submitAddressForm = async (block: AIFlowBlock) => {
  if (!isCurrentFlowAction(block.action)) {
    showExpiredFlowActionToast()
    return
  }
  const form = block.form || {}
  if (!form.receiver || !form.contact || !form.address?.length || !form.detail) {
    uni.showToast({ icon: 'none', title: '请补全收货地址' })
    return
  }
  if (!isValidMobilePhone(form.contact)) {
    uni.showToast({ icon: 'none', title: '手机号格式不正确' })
    return
  }
  const payload = parseActionPayload(block.action?.payload_json)
  payload.user_address = {
    receiver: form.receiver,
    contact: form.contact,
    address: form.address,
    address_name: form.address_name || form.address,
    detail: form.detail,
    is_default: Boolean(form.is_default),
  }
  const action = buildFlowAction(block.action, payload)
  if (await handleFlowAction(action, '新增收货地址')) {
    removeAddressFormBlock(block)
  }
}

/** 从无地址状态进入新增地址流程。 */
const handleCreateAddressGuide = async (block: AIFlowBlock) => {
  if (currentSessionSending.value) {
    return
  }
  if (!isCurrentFlowAction(block.action)) {
    showExpiredFlowActionToast()
    return
  }
  await handleFlowAction(block.action, '新增收货地址')
}

/** 提交评价表单。 */
const submitReviewForm = (block: AIFlowBlock) => {
  if (!isCurrentFlowAction(block.action)) {
    showExpiredFlowActionToast()
    return
  }
  const form = block.form || {}
  if (!form.content) {
    uni.showToast({ icon: 'none', title: '请输入评价内容' })
    return
  }
  const payload = parseActionPayload(block.action?.payload_json)
  payload.content = form.content
  payload.goods_score = Number(form.goods_score || 5)
  payload.package_score = Number(form.package_score || 5)
  payload.delivery_score = Number(form.delivery_score || 5)
  payload.is_anonymous = Boolean(form.is_anonymous)
  payload.img = []
  const action = buildFlowAction(block.action, payload)
  void handleFlowAction(action, '提交评价')
}

/** 调整评价分数。 */
const changeReviewScore = (form: AIFlowBlock, key: string, delta: number) => {
  form[key] = Math.min(5, Math.max(1, Number(form[key] || 5) + delta))
}

/** 选择省市区，微信使用系统区域选择器，其他端使用项目行政区域树。 */
const onAddressRegionChange = (
  block: AIFlowBlock,
  ev: Parameters<UniHelper.RegionPickerOnChange>[0],
) => {
  block.form.address_name = ev.detail.value
  block.form.address = ev.detail.code
}

/** 选择 H5/App 行政区域树。 */
const onAddressCityChange = (
  block: AIFlowBlock,
  ev: Parameters<UniHelper.UniDataPickerOnChange>[0],
) => {
  const values = ev.detail.value || []
  block.form.address = values.map((item) => String(item.value))
  block.form.address_name = values.map((item) => String(item.text))
}

const formatTools = (tools: AiToolCall[]) => {
  return tools.map((item) => item.title || item.name).join(' · ')
}

const formatRuntime = (item: ChatMessageItem) => {
  const duration = item.durationMs ? `${(item.durationMs / 1000).toFixed(1)}s` : '生成中'
  return `${item.tokenTotal} Token · 首字 ${item.firstTokenMs}ms · 总耗时 ${duration}`
}

/** 预览消息附件，图片走图片预览，文档走平台文档预览。 */
const previewAttachment = async (attachment: AiAttachment, attachments: AiAttachment[]) => {
  const current = formatSrc(attachment.url)
  if (!current) {
    uni.showToast({ icon: 'none', title: '附件地址为空' })
    return
  }
  if (isImageAttachment(attachment)) {
    const urls = attachments
      .filter((item) => isImageAttachment(item))
      .map((item) => formatSrc(item.url))
      .filter(Boolean)
    uni.previewImage({
      current,
      urls: urls.length ? urls : [current],
    })
    return
  }
  await openDocumentAttachment(attachment, current)
}

/** 一次性加载当前终端可用的快捷助手入口。 */
async function loadAiShortcuts() {
  if (loadingShortcuts.value) {
    return
  }

  loadingShortcuts.value = true
  try {
    const response = await defAiToolService.ListAiShortcut({
      terminal: AI_TERMINAL,
    })
    starterShortcuts.value = normalizeStarterShortcuts(response.shortcuts)
    starterPromptGroupIndex.value = 0
  } catch (error) {
    starterShortcuts.value = []
    showError(error, '加载快捷助手失败')
  } finally {
    loadingShortcuts.value = false
  }
}

/** 加载移动端 AI 助手会话列表。 */
async function ensureSessionsLoaded() {
  if (loadingSessions.value || sessions.value.length > 0) {
    return
  }

  loadingSessions.value = true
  try {
    const response = await defAiSessionService.ListAiSession({
      terminal: AI_TERMINAL,
    })
    sessions.value = normalizeSessionList(response.sessions)
    const sessionID = await ensureActiveSession()
    if (sessionID) {
      await loadMessages(sessionID)
    }
  } catch (error) {
    showError(error, '加载会话失败')
  } finally {
    loadingSessions.value = false
  }
}

/** 加载指定会话消息记录。 */
async function loadMessages(sessionID: string, options?: { force?: boolean }) {
  if (!sessionID) {
    return
  }
  if (!options?.force && isSessionSending(sessionID) && messages.value[sessionID]?.length) {
    return
  }

  loadingSessionID.value = sessionID
  try {
    const response = await defAiMessageService.ListAiMessage({
      session_id: sessionID,
    })
    if (loadingSessionID.value !== sessionID) {
      return
    }
    messages.value[sessionID] = normalizeMessageList(response.messages)
    if (activeSessionID.value === sessionID) {
      scrollChatToBottom()
    }
  } catch (error) {
    if (loadingSessionID.value === sessionID) {
      messages.value[sessionID] = []
    }
    showError(error, '加载消息失败')
  } finally {
    if (loadingSessionID.value === sessionID) {
      loadingSessionID.value = ''
    }
  }
}

/** 保证当前存在可用会话。 */
async function ensureActiveSession() {
  if (!activeSessionID.value && sessions.value.length > 0) {
    activeSessionID.value = sessions.value[0].id
  }
  if (activeSessionID.value) {
    return activeSessionID.value
  }

  activeSessionID.value = (await createRemoteSession()) ?? ''
  return activeSessionID.value
}

/** 创建新的远端会话。 */
async function createRemoteSession(options?: { title?: string }) {
  const response = await defAiSessionService.CreateAiSession({
    title: options?.title || '新对话',
    terminal: AI_TERMINAL,
  })
  const session = normalizeSession(response.session)
  upsertSession(session)
  return session.id
}

function resolveMenuButtonRect() {
  const wechatRuntime = globalThis as typeof globalThis & {
    wx?: {
      getMenuButtonBoundingClientRect?: () => MenuButtonRect
    }
  }
  try {
    const rect = wechatRuntime.wx?.getMenuButtonBoundingClientRect?.()
    if (!rect?.width || !rect.height) {
      return null
    }
    return rect
  } catch {
    return null
  }
}

/** 发送消息，H5 优先流式，其他端不支持流式时退回完整响应。 */
async function sendAiPayload(payload: SubmitPayload) {
  const sessionID = await ensureActiveSession()
  if (!sessionID || isSessionSending(sessionID)) {
    return false
  }

  const localUserMessage = createLocalUserMessage(payload)
  const thinkingMessage = createThinkingMessage({ sessionID })
  messages.value[sessionID] = sortMessages([
    ...(messages.value[sessionID] ?? []),
    localUserMessage,
    thinkingMessage,
  ])
  scrollChatToBottom()
  setSessionSending(sessionID, true)
  return runAiTask(sessionID, payload)
}

/** 后台执行 AI 助手消息请求。 */
async function runAiTask(sessionID: string, payload: SubmitPayload) {
  let task: StreamTask | undefined
  const request = {
    session_id: sessionID,
    content: payload.text,
    attachments: payload.attachments,
    action: payload.action,
  }
  try {
    let handledByStream = false

    // #ifdef MP-WEIXIN
    const parser = createAiEventStreamTextParser((event) => handleAiStreamEvent(event, task))
    const chunkedTask = StreamAiMessageByChunkedRequest(request, {
      onChunk: (chunkText) => parser.push(chunkText),
    })
    task = {
      aborted: false,
      finished: false,
      abort() {
        task!.aborted = true
        chunkedTask.abort()
      },
    }
    runningStreamTaskMap.set(sessionID, task)
    handledByStream = true
    await chunkedTask.promise
    parser.flush()
    if (!task.finished && !task.aborted) {
      throw new Error('AI 助手流式响应未完整返回')
    }
    // #endif

    // #ifdef H5
    if (
      !handledByStream &&
      typeof fetch === 'function' &&
      typeof ReadableStream !== 'undefined' &&
      typeof AbortController !== 'undefined'
    ) {
      const controller = new AbortController()
      task = {
        aborted: false,
        finished: false,
        abort() {
          task!.aborted = true
          controller.abort()
        },
      }
      runningStreamTaskMap.set(sessionID, task)
      const response = await defAiMessageService.StreamAiMessage(request, {
        signal: controller.signal,
      })
      if (!response.body) {
        throw new Error('AI 助手流式响应为空')
      }
      await readAiEventStream(
        response.body,
        (event) => handleAiStreamEvent(event, task),
        controller.signal,
      )
      if (!task.finished && !task.aborted) {
        throw new Error('AI 助手流式响应未完整返回')
      }
      handledByStream = true
    }
    // #endif

    if (!handledByStream) {
      const response = await defAiMessageService.SendAiMessage(request)
      const nextMessages = normalizeNonStreamMessages(response)
      if (!nextMessages.length) {
        throw new Error('AI 助手响应为空')
      }
      const success = hasSuccessfulAiMessages(nextMessages)
      messages.value[sessionID] = replacePendingMessages(
        messages.value[sessionID] ?? [],
        nextMessages,
      )
      scrollChatToBottom()
      if (response.session) {
        upsertSession(normalizeSession(response.session))
      }
      handlePaymentBlocks(nextMessages)
      return success
    }
    return Boolean(task?.success)
  } catch (error) {
    if (task?.aborted) {
      return false
    }
    messages.value[sessionID] = markThinkingMessageFailed(messages.value[sessionID] ?? [], {
      sessionID,
    })
    scrollChatToBottom()
    showError(error, 'AI 助手请求失败')
    return false
  } finally {
    if (task && runningStreamTaskMap.get(sessionID) === task) {
      runningStreamTaskMap.delete(sessionID)
    }
    setSessionSending(sessionID, false)
  }
}

/** 重试失败消息或重新生成助手输出。 */
async function retryMessage(item: ChatMessageItem) {
  const sessionID = activeSessionID.value
  if (!sessionID || isSessionSending(sessionID)) {
    return
  }

  try {
    if (item.localOnly) {
      const payload = resolveLocalRetryPayload(item)
      if (!payload) {
        uni.showToast({ icon: 'none', title: '未找到可重新发送的消息' })
        return
      }
      messages.value[sessionID] = (messages.value[sessionID] ?? []).filter(
        (message) => !message.localOnly,
      )
      await sendAiPayload(payload)
      return
    }

    setSessionSending(sessionID, true)
    let response
    if (item.role === 'user') {
      if (item.status !== AiMessageStatus.FAILED_AAMS) {
        uni.showToast({ icon: 'none', title: '只有发送失败的消息可以重新发送' })
        return
      }
      response = await defAiMessageService.RetryAiUserMessage({
        session_id: sessionID,
        message_id: item.messageID,
      })
    } else {
      messages.value[sessionID] = markAIMessageRegenerating(
        messages.value[sessionID] ?? [],
        sessionID,
        item.messageID,
      )
      response = await defAiMessageService.RegenerateAiMessage({
        session_id: sessionID,
        message_id: item.messageID,
      })
    }
    messages.value[sessionID] = replacePendingMessages(
      messages.value[sessionID] ?? [],
      normalizeMessageList(response.messages),
    )
    if (response.session) {
      upsertSession(normalizeSession(response.session))
    }
    uni.showToast({ icon: 'none', title: item.role === 'user' ? '已重新发送' : '已重新生成' })
  } catch (error) {
    if (item.role !== 'user') {
      await loadMessages(sessionID, { force: true })
    }
    showError(error, '重新生成失败')
  } finally {
    if (!item.localOnly) {
      setSessionSending(sessionID, false)
    }
  }
}

/** 从当前消息创建分支会话。 */
async function branchMessage(item: ChatMessageItem) {
  const sourceSessionID = activeSessionID.value
  if (!sourceSessionID || item.localOnly) {
    return
  }
  try {
    const response = await defAiSessionService.CreateAiSessionBranch({
      source_session_id: sourceSessionID,
      anchor_message_id: item.messageID,
      title: buildBranchSessionTitle(item),
      terminal: AI_TERMINAL,
    })
    const branchSession = normalizeSession(response.session)
    upsertSession(branchSession)
    messages.value[branchSession.id] = normalizeMessageList(response.messages)
    activeSessionID.value = branchSession.id
    showSessionDrawer.value = false
    scrollChatToBottom()
    uni.showToast({ icon: 'none', title: '已创建分支会话' })
  } catch (error) {
    showError(error, '创建分支失败')
  }
}

/** 朗读入口，移动端先保持与管理端一致的操作态反馈。 */
function toggleSpeakMessage(item: ChatMessageItem) {
  if (item.role === 'user') {
    return
  }
  const key = item.key
  speakingMessageKey = speakingMessageKey === key ? '' : key
  markAllMessagesSpeaking(speakingMessageKey)
  uni.showToast({ icon: 'none', title: speakingMessageKey ? '开始朗读' : '已停止朗读' })
}

/** 移除已选择但尚未发送的附件。 */
function removeSelectedAttachment(attachment: AiAttachment) {
  selectedAttachments.value = selectedAttachments.value.filter(
    (item) =>
      (item.id || item.url || item.name) !== (attachment.id || attachment.url || attachment.name),
  )
}

async function chooseAttachmentFiles(limit: number) {
  let selectedFiles: AttachmentFileCandidate[] = []
  // #ifdef MP-WEIXIN
  selectedFiles = await chooseWechatAttachmentFiles(limit)
  // #endif
  // #ifdef H5
  selectedFiles = await chooseH5AttachmentFiles(limit)
  // #endif
  // #ifndef MP-WEIXIN
  // #ifndef H5
  uni.showToast({ icon: 'none', title: '当前端暂不支持附件' })
  // #endif
  // #endif
  const supportedFiles = selectedFiles.filter((item) =>
    isSupportedAttachmentExtension(item.extension),
  )
  if (selectedFiles.length && !supportedFiles.length) {
    uni.showToast({ icon: 'none', title: '请选择支持的文件类型' })
  }
  return supportedFiles.slice(0, limit)
}

async function chooseWechatAttachmentFiles(limit: number) {
  const result = await wx.chooseMessageFile({
    count: limit,
    type: 'all',
    extension: supportedAttachmentExtensions,
  })
  return result.tempFiles.map((item) =>
    normalizeAttachmentFile(item.path, item.name, Number(item.size || 0)),
  )
}

async function chooseH5AttachmentFiles(limit: number) {
  const result = await uni.chooseFile({
    count: limit,
    type: 'all',
    extension: supportedAttachmentExtensions.map((item) => `.${item}`),
  })
  const tempFilePaths = Array.isArray(result.tempFilePaths)
    ? result.tempFilePaths
    : [result.tempFilePaths].filter(Boolean)
  const tempFiles = Array.isArray(result.tempFiles) ? result.tempFiles : [result.tempFiles]
  return tempFilePaths
    .map((path, index) => {
      const item = tempFiles[index]
      const file = item as File & { path?: string; name?: string; size?: number }
      return normalizeAttachmentFile(
        path || file.path || '',
        file.name || '',
        Number(file.size || 0),
      )
    })
    .filter((item) => item.path)
}

/** 处理 AI 助手流式事件。 */
function handleAiStreamEvent(event: AiStreamEvent, task?: StreamTask) {
  if (event.event === 'delta') {
    handleAiDelta(event.payload)
    return
  }
  if (event.event === 'finish') {
    handleAiFinish(event.payload, task)
    return
  }
  handleAiError(event.payload, task)
}

function handleAiDelta(payload: AiStreamPayload) {
  if (!payload.delta) {
    return
  }
  queueAiDelta(payload)
}

function handleAiFinish(payload: AiStreamPayload, task?: StreamTask) {
  const sessionID = payload.session_id
  if (!sessionID) {
    return
  }
  if (task) {
    task.finished = true
  }
  flushAiDelta()
  const nextMessages = normalizeMessageList(payload.messages)
  if (task) {
    task.success = hasSuccessfulAiMessages(nextMessages)
  }
  const current = messages.value[sessionID] ?? []
  const streamKey = payload.message_id ? buildStreamMessageKey(sessionID, payload.message_id) : ''
  const hasLocalStreamingMessages = current.some(
    (item) => item.localOnly && item.streamKey === streamKey,
  )
  messages.value[sessionID] =
    nextMessages.length || !hasLocalStreamingMessages
      ? replacePendingMessages(current, nextMessages, payload)
      : current
  const revealMessageKey = nextMessages.find(
    (item) => item.role === 'ai' && item.messageID === payload.message_id,
  )?.key
  if (revealMessageKey) {
    startFlowReveal(sessionID, revealMessageKey)
  }
  scrollChatToBottom()
  if (payload.session) {
    upsertSession(normalizeSession(payload.session))
  }
  handlePaymentBlocks(nextMessages)
}

function handleAiError(payload: AiStreamPayload, task?: StreamTask) {
  const sessionID = payload.session_id
  if (!sessionID) {
    return
  }
  if (task) {
    task.finished = true
    task.success = false
  }
  flushAiDelta()
  const nextMessages = normalizeMessageList(payload.messages)
  if (nextMessages.length) {
    messages.value[sessionID] = replacePendingMessages(
      messages.value[sessionID] ?? [],
      nextMessages,
      payload,
    )
    scrollChatToBottom()
    return
  }
  messages.value[sessionID] = markStreamingError(
    ensureStreamingMessage(messages.value[sessionID] ?? [], payload),
    payload,
  )
  scrollChatToBottom()
}

/** 合并同一时刻的流式分片，降低移动端频繁渲染压力。 */
function queueAiDelta(payload: AiStreamPayload) {
  const sessionID = payload.session_id
  const messageID = payload.message_id
  if (!sessionID || !messageID || !messages.value[sessionID]) {
    return
  }

  const key = buildStreamMessageKey(sessionID, messageID)
  const cachedPayload = pendingDeltaMap.get(key)
  pendingDeltaMap.set(key, {
    ...payload,
    delta: `${cachedPayload?.delta ?? ''}${payload.delta ?? ''}`,
  })

  if (pendingDeltaTimer) {
    return
  }
  pendingDeltaTimer = setTimeout(() => {
    pendingDeltaTimer = 0
    flushAiDelta()
  }, 32) as unknown as number
}

function flushAiDelta() {
  if (pendingDeltaTimer) {
    clearTimeout(pendingDeltaTimer)
    pendingDeltaTimer = 0
  }
  if (!pendingDeltaMap.size) {
    return
  }
  const payloadList = Array.from(pendingDeltaMap.values())
  pendingDeltaMap.clear()
  for (const payload of payloadList) {
    const sessionID = payload.session_id
    if (!sessionID || !messages.value[sessionID]) {
      continue
    }
    messages.value[sessionID] = appendStreamingDelta(
      ensureStreamingMessage(messages.value[sessionID] ?? [], payload),
      payload,
    )
    scrollChatToBottom()
  }
}

function clearPendingDelta() {
  if (pendingDeltaTimer) {
    clearTimeout(pendingDeltaTimer)
    pendingDeltaTimer = 0
  }
  pendingDeltaMap.clear()
}

function normalizeSession(session?: Partial<AiSession> | null): AiSession {
  return {
    id: String(session?.id ?? ''),
    title: String(session?.title ?? '新对话'),
    summary: String(session?.summary ?? ''),
    updated_at: session?.updated_at,
    terminal: Number(session?.terminal ?? AI_TERMINAL),
  }
}

function normalizeSessionList(list?: AiSession[] | null) {
  if (!Array.isArray(list)) {
    return []
  }
  return list.map((item) => normalizeSession(item)).filter((item) => item.id)
}

function normalizeMessageList(list?: AiMessage[] | null) {
  if (!Array.isArray(list)) {
    return []
  }
  return sortMessages(
    list
      .filter(Boolean)
      .flatMap((item) => [mapMessageItem(item, 'user'), mapMessageItem(item, 'ai')]),
  )
}

function hasSuccessfulAiMessages(list: ChatMessageItem[]) {
  return list.some((item) => item.status === AiMessageStatus.SUCCESS_AAMS)
}

function removeAddressFormBlock(block: AIFlowBlock) {
  const sessionID = activeSessionID.value
  const list = messages.value[sessionID] ?? []
  const message = list.find((item) => item.blocks.includes(block))
  if (!sessionID || !message) {
    return
  }

  const blockIndex = message.blocks.findIndex((item) => item === block)
  if (blockIndex < 0) {
    return
  }

  markAddressFormBlockSubmitted(message.messageID, blockIndex, block)
  messages.value[sessionID] = list.map((item) => {
    if (item.key !== message.key) {
      return item
    }
    return {
      ...item,
      blocks: item.blocks.filter((_, index) => index !== blockIndex),
    }
  })
  scrollChatToBottom()
}

function isValidMobilePhone(value: unknown) {
  return MOBILE_PHONE_PATTERN.test(String(value ?? ''))
}

function isSubmittedAddressFormBlock(messageID: string, blockIndex: number, block: AIFlowBlock) {
  return (
    block.type === 'address_form' &&
    submittedAddressFormBlockSet.has(buildAddressFormBlockKey(messageID, blockIndex, block))
  )
}

function markAddressFormBlockSubmitted(messageID: string, blockIndex: number, block: AIFlowBlock) {
  if (!messageID) {
    return
  }
  submittedAddressFormBlockSet.add(buildAddressFormBlockKey(messageID, blockIndex, block))
  writeSubmittedAddressFormBlockKeys(Array.from(submittedAddressFormBlockSet))
}

function buildAddressFormBlockKey(messageID: string, blockIndex: number, block: AIFlowBlock) {
  const action = block.action || {}
  return [
    messageID,
    blockIndex,
    action.flow || '',
    action.step || '',
    action.type || '',
    action.payload_json || '',
  ].join('|')
}

function readSubmittedAddressFormBlockKeys() {
  try {
    const value = uni.getStorageSync(SUBMITTED_ADDRESS_FORM_BLOCKS_KEY)
    return Array.isArray(value) ? value.map((item) => String(item)).filter(Boolean) : []
  } catch {
    return []
  }
}

function writeSubmittedAddressFormBlockKeys(keys: string[]) {
  try {
    uni.setStorageSync(
      SUBMITTED_ADDRESS_FORM_BLOCKS_KEY,
      keys.slice(-SUBMITTED_ADDRESS_FORM_BLOCK_LIMIT),
    )
  } catch {
    // 本地记录失败只影响旧表单隐藏，不影响地址新增主流程。
  }
}

function normalizeNonStreamMessages(response: unknown) {
  const jsonResponse = response as { messages?: AiMessage[] }
  if (Array.isArray(jsonResponse?.messages)) {
    return normalizeMessageList(jsonResponse.messages)
  }

  const events = parseAiEventStreamText(response)
  const finishEvent = [...events].reverse().find((item) => item.event === 'finish')
  if (finishEvent) {
    return normalizeMessageList(finishEvent.payload.messages)
  }
  const errorEvent = [...events].reverse().find((item) => item.event === 'error')
  if (errorEvent) {
    throw new Error('AI 助手请求失败')
  }
  return []
}

function parseFlowBlocks(raw?: string, messageID?: string) {
  if (!raw) {
    return []
  }
  try {
    const blocks = JSON.parse(raw)
    if (!Array.isArray(blocks)) {
      return []
    }
    return blocks
      .filter((item): item is AIFlowBlock => Boolean(item?.type))
      .filter((item, index) => !isSubmittedAddressFormBlock(messageID || '', index, item))
      .map((item) => normalizeFlowBlock(item))
  } catch {
    return []
  }
}

function normalizeFlowBlock(block: AIFlowBlock) {
  if (block.type === 'sku_selector') {
    block.skus = Array.isArray(block.skus)
      ? block.skus.map((item: AIFlowBlock) => ({ ...item, num: Number(item.num || 1) }))
      : []
  }
  if (block.type === 'address_form' && !block.form) {
    block.form = {
      receiver: '',
      contact: '',
      address: [],
      address_name: [],
      detail: '',
      is_default: true,
    }
  }
  if (block.type === 'address_form') {
    block.form.address = Array.isArray(block.form.address) ? block.form.address : []
    block.form.address_name = Array.isArray(block.form.address_name) ? block.form.address_name : []
    void ensureAddressAreaTreeLoaded()
  }
  if (block.type === 'review_form' && !block.form) {
    block.form = {
      content: '',
      goods_score: 5,
      package_score: 5,
      delivery_score: 5,
      is_anonymous: false,
    }
  }
  return block
}

function markFlowBlocksDisabled(blocks: AIFlowBlock[], messageID: string) {
  return blocks.map((block) => markFlowValueDisabled(block, messageID)) as AIFlowBlock[]
}

function markFlowValueDisabled(value: unknown, messageID: string): unknown {
  if (Array.isArray(value)) {
    value.forEach((item) => markFlowValueDisabled(item, messageID))
    return value
  }
  if (!value || typeof value !== 'object') {
    return value
  }
  const current = value as Record<string, any>
  if (current.action?.type) {
    current.disabled = !isFlowActionFromMessage(current.action, messageID)
  }
  Object.values(current).forEach((item) => markFlowValueDisabled(item, messageID))
  return current
}

function mapMessageItem(message: AiMessage, role: ChatRole): ChatMessageItem {
  const inputContent = {
    kind: message.input_content?.kind || 'text',
    content: message.input_content?.content ?? '',
  }
  const outputContent = {
    kind: message.output_content?.kind || 'text',
    content: message.output_content?.content ?? '',
    reply_source: message.output_content?.reply_source ?? '',
    model: message.output_content?.model ?? '',
    fallback: Boolean(message.output_content?.fallback),
    fallback_reason: message.output_content?.fallback_reason ?? '',
    flow: message.output_content?.flow ?? '',
    step: message.output_content?.step ?? '',
    blocks_json: message.output_content?.blocks_json ?? '',
  }
  const status = Number(message.status ?? AiMessageStatus.SUCCESS_AAMS)
  return {
    ...message,
    key: `${message.id}:${role}`,
    messageID: message.id,
    role,
    content: role === 'user' ? inputContent.content : outputContent.content,
    input_content: inputContent,
    output_content: outputContent,
    attachments: Array.isArray(message.attachments) ? message.attachments : [],
    status,
    token: {
      input: Number(message.token?.input ?? 0),
      output: Number(message.token?.output ?? 0),
      cache: Number(message.token?.cache ?? 0),
      total: Number(message.token?.total ?? 0),
    },
    tools: Array.isArray(message.tools) ? message.tools : [],
    model: role === 'ai' ? outputContent.model : '',
    replySource: role === 'ai' ? outputContent.reply_source : '',
    fallback: role === 'ai' && outputContent.fallback,
    fallbackReason: role === 'ai' ? outputContent.fallback_reason : '',
    flow: role === 'ai' ? outputContent.flow : '',
    step: role === 'ai' ? outputContent.step : '',
    blocksJson: role === 'ai' ? outputContent.blocks_json : '',
    blocks:
      role === 'ai'
        ? markFlowBlocksDisabled(parseFlowBlocks(outputContent.blocks_json, message.id), message.id)
        : [],
    tokenTotal: Number(message.token?.total ?? 0),
    firstTokenMs: Number(message.first_token_ms ?? 0),
    durationMs: Number(message.duration_ms ?? 0),
  }
}

function createLocalUserMessage(payload: SubmitPayload): ChatMessageItem {
  const now = Date.now()
  const message = mapMessageItem(
    {
      id: `${LOCAL_USER_MESSAGE_PREFIX}-${now}`,
      input_content: { kind: 'text', content: payload.text },
      output_content: undefined,
      attachments: payload.attachments,
      created_at: {
        seconds: Math.floor(now / 1000),
        nanos: (now % 1000) * 1_000_000,
      },
      status: AiMessageStatus.GENERATING_AAMS,
      token: { input: 0, output: 0, cache: 0, total: 0 },
      tools: [],
      first_token_ms: 0,
      duration_ms: 0,
    },
    'user',
  )
  message.localOnly = true
  message.status = AiMessageStatus.GENERATING_AAMS
  return message
}

function createThinkingMessage(options?: {
  sessionID?: string
  messageID?: string
}): ChatMessageItem {
  const now = Date.now()
  const streamKey = options?.sessionID
    ? buildStreamMessageKey(options.sessionID, options.messageID || PENDING_MESSAGE_ID)
    : undefined
  const message = mapMessageItem(
    {
      id: streamKey || `ai-thinking-${now}`,
      input_content: undefined,
      output_content: {
        kind: 'text',
        content: THINKING_MESSAGE_CONTENT,
        reply_source: '',
        model: '',
        fallback: false,
        fallback_reason: '',
        flow: '',
        step: '',
        blocks_json: '',
      },
      attachments: [],
      created_at: {
        seconds: Math.floor(now / 1000),
        nanos: (now % 1000) * 1_000_000,
      },
      status: AiMessageStatus.GENERATING_AAMS,
      token: { input: 0, output: 0, cache: 0, total: 0 },
      tools: [],
      first_token_ms: 0,
      duration_ms: 0,
    },
    'ai',
  )
  message.localOnly = true
  message.streamKey = streamKey
  return message
}

function buildStreamMessageKey(sessionID: string, messageID: string) {
  return `${sessionID}:${messageID}`
}

function buildPendingStreamMessageKey(sessionID: string) {
  return buildStreamMessageKey(sessionID, PENDING_MESSAGE_ID)
}

function ensureStreamingMessage(current: ChatMessageItem[], payload: AiStreamPayload) {
  const sessionID = payload.session_id
  const messageID = payload.message_id
  if (!sessionID || !messageID) {
    return current
  }

  const streamKey = buildStreamMessageKey(sessionID, messageID)
  if (current.some((item) => item.streamKey === streamKey)) {
    return current
  }

  const pendingStreamKey = buildPendingStreamMessageKey(sessionID)
  const next = current.map((item) =>
    item.streamKey === pendingStreamKey
      ? { ...item, id: messageID, messageID, key: `${messageID}:ai`, streamKey }
      : item,
  )
  if (next.some((item) => item.streamKey === streamKey)) {
    return next
  }

  return sortMessages([...next, createThinkingMessage({ sessionID, messageID })])
}

function appendStreamingDelta(current: ChatMessageItem[], payload: AiStreamPayload) {
  if (!payload.delta) {
    return current
  }
  const streamKey = buildStreamMessageKey(payload.session_id, payload.message_id)
  return current.map((item) => {
    if (item.streamKey !== streamKey || item.role === 'user') {
      return item
    }
    const baseContent = item.content === THINKING_MESSAGE_CONTENT ? '' : item.content
    return {
      ...item,
      content: `${baseContent}${payload.delta}`,
      status: AiMessageStatus.GENERATING_AAMS,
    }
  })
}

function replacePendingMessages(
  current: ChatMessageItem[],
  nextMessages: ChatMessageItem[],
  payload?: AiStreamPayload,
) {
  const sessionID = payload?.session_id ?? ''
  const streamKey = payload?.message_id ? buildStreamMessageKey(sessionID, payload.message_id) : ''
  const pendingStreamKey = sessionID ? buildPendingStreamMessageKey(sessionID) : ''
  const stableMessages = current.filter((item) => {
    if (!item.localOnly) {
      return true
    }
    if (payload?.message_id && item.role === 'user') {
      return !nextMessages.some(
        (message) => message.role === 'user' && message.messageID === payload.message_id,
      )
    }
    if (!streamKey) {
      return false
    }
    return item.streamKey !== streamKey && item.streamKey !== pendingStreamKey
  })
  const messageMap = new Map<string, ChatMessageItem>()
  for (const item of stableMessages) {
    messageMap.set(item.key, item)
  }
  for (const item of nextMessages) {
    messageMap.set(item.key, item)
  }
  return sortMessages(Array.from(messageMap.values()))
}

function markThinkingMessageFailed(
  current: ChatMessageItem[],
  options?: { sessionID?: string; messageID?: string },
) {
  const streamKey =
    options?.sessionID && options.messageID
      ? buildStreamMessageKey(options.sessionID, options.messageID)
      : ''
  return current.map((item) => {
    if (!item.localOnly) {
      return item
    }
    if (streamKey && item.streamKey !== streamKey) {
      return item
    }
    return {
      ...item,
      status: AiMessageStatus.FAILED_AAMS,
      content:
        item.role === 'ai' ? '这次回复没有成功返回，你可以直接重试刚才的问题。' : item.content,
    }
  })
}

function markStreamingError(current: ChatMessageItem[], payload: AiStreamPayload) {
  const streamKey = buildStreamMessageKey(payload.session_id, payload.message_id)
  return current.map((item) => {
    if (!item.localOnly || item.streamKey !== streamKey) {
      return item
    }
    return {
      ...item,
      status: AiMessageStatus.FAILED_AAMS,
      content: '这次回复没有成功返回，你可以直接重试刚才的问题。',
    }
  })
}

function markAIMessageRegenerating(
  current: ChatMessageItem[],
  sessionID: string,
  messageID: string,
) {
  const streamKey = buildStreamMessageKey(sessionID, messageID)
  return current.map((item) => {
    if (item.messageID !== messageID || item.role === 'user') {
      return item
    }
    return {
      ...item,
      content: THINKING_MESSAGE_CONTENT,
      fallback: false,
      fallbackReason: '',
      token: { input: 0, output: 0, cache: 0, total: 0 },
      tokenTotal: 0,
      tools: [],
      firstTokenMs: 0,
      durationMs: 0,
      status: AiMessageStatus.GENERATING_AAMS,
      streamKey,
    }
  })
}

function sortMessages(list: ChatMessageItem[]) {
  return [...list].sort((left, right) => {
    const leftTime = resolveTimestamp(left.created_at)
    const rightTime = resolveTimestamp(right.created_at)
    if (leftTime === rightTime) {
      if (left.role !== right.role) {
        return left.role === 'user' ? -1 : 1
      }
      return left.messageID.localeCompare(right.messageID, 'zh-Hans-CN', { numeric: true })
    }
    return leftTime - rightTime
  })
}

function resolveTimestamp(timestamp?: { seconds?: number; nanos?: number }) {
  if (!timestamp) {
    return 0
  }
  const seconds = Number(timestamp.seconds ?? 0)
  const nanos = Number(timestamp.nanos ?? 0)
  return seconds * 1000 + Math.floor(nanos / 1_000_000)
}

function resolveLocalRetryPayload(item: ChatMessageItem): SubmitPayload | undefined {
  if (item.role === 'user') {
    return { text: item.content, attachments: item.attachments ?? [] }
  }
  const list = sortMessages(messages.value[activeSessionID.value] ?? [])
  const targetIndex = list.findIndex((message) => message.key === item.key)
  const endIndex = targetIndex >= 0 ? targetIndex - 1 : list.length - 1
  for (let index = endIndex; index >= 0; index--) {
    const message = list[index]
    if (message.localOnly && message.role === 'user') {
      return { text: message.content, attachments: message.attachments ?? [] }
    }
  }
  return undefined
}

function buildBranchSessionTitle(item: ChatMessageItem) {
  const content = String(item.input_content?.content || item.content || '新对话').replace(
    /\s+/g,
    ' ',
  )
  return `分支：${content.slice(0, 18) || '新对话'}`
}

function parseActionPayload(raw?: string) {
  if (!raw) {
    return {} as Record<string, any>
  }
  try {
    return JSON.parse(raw) as Record<string, any>
  } catch {
    return {} as Record<string, any>
  }
}

function normalizeStarterShortcuts(list?: AiShortcut[] | null) {
  return [...(list || [])]
    .filter((item) => Boolean(item?.key && (item.title || item.prompt)))
    .map((item) => ({
      ...item,
      title: item.title || item.prompt,
      prompt: item.prompt || item.title,
      action: normalizeStarterShortcutAction(item.action),
      required_tools: Array.isArray(item.required_tools) ? item.required_tools : [],
      sort: Number(item.sort || 0),
    }))
    .sort((prev, next) => prev.sort - next.sort)
}

function normalizeStarterShortcutAction(action?: AiShortcut['action']) {
  if (!action?.type) {
    return undefined
  }
  return {
    flow: action.flow || '',
    step: action.step || '',
    type: action.type,
    payload_json: action.payload_json || '{}',
  }
}

function buildStarterPromptAction(shortcut: AiShortcut) {
  const action = normalizeStarterShortcutAction(shortcut.action)
  if (!action) {
    return undefined
  }
  return buildFlowAction(action, parseActionPayload(action.payload_json))
}

function buildFlowAction(action?: Partial<AiAction>, payload?: Record<string, any>) {
  if (!action?.type) {
    return undefined
  }
  return {
    flow: action.flow || '',
    step: action.step || '',
    type: action.type,
    payload_json: JSON.stringify(payload || {}),
    source_message_id: action.source_message_id || '',
    action_id: action.action_id || '',
    flow_version: Number(action.flow_version || 0),
  }
}

function isCurrentFlowAction(action?: Partial<AiAction>) {
  if (!action?.type) {
    return false
  }
  return isFlowActionFromMessage(action, activeFlowMessageID.value)
}

function isFlowActionFromMessage(action: Partial<AiAction>, messageID: string) {
  if (!messageID || !action.source_message_id || !action.action_id) {
    return false
  }
  return action.source_message_id === messageID && String(action.flow_version || '') === messageID
}

function showExpiredFlowActionToast() {
  uni.showToast({ icon: 'none', title: '这一步已过期，请从最新消息继续操作' })
}

function enrichFlowAction(action: AiAction) {
  if (action.type !== 'start_payment') {
    return action
  }
  const payload = parseActionPayload(action.payload_json)
  let platform = 'jsapi'
  // #ifdef H5
  platform = 'h5'
  // #endif
  // #ifdef APP-PLUS
  platform = 'app'
  // #endif
  // #ifdef MP-WEIXIN
  platform = 'jsapi'
  // #endif
  payload.platform = platform
  return buildFlowAction(action, payload) || action
}

function resolveFlowActionLabel(action: AiAction) {
  const labelMap: Record<string, string> = {
    select_goods: '选择商品',
    select_sku: '确认规格',
    create_address: '新增收货地址',
    select_address: '选择收货地址',
    confirm_order: '确认下单',
    start_payment: '发起支付',
    open_review_form: '写评价',
    submit_review: '提交评价',
    view_order: '查看订单',
    receive_order: '确认收货',
    view_goods_category: '查看分类商品',
    view_shop_hot_item: '查看热门专区',
  }
  return labelMap[action.type] || '继续'
}

async function fillActiveAddressFormFromText(text: string) {
  const content = text.replace(/\s+/g, ' ').trim()
  if (!content) {
    return false
  }

  const target = findActiveAddressFormBlock()
  if (!target) {
    return false
  }

  const form = target.block.form
  const phone = content.match(/1[3-9]\d{9}/)?.[0] ?? ''
  if (phone) {
    form.contact = phone
  }

  const receiver = extractAddressField(content, ['收货人', '联系人', '姓名'])
  if (receiver) {
    form.receiver = receiver
  }

  const withoutPhone = phone ? content.replace(phone, ' ') : content
  const region = await matchAddressRegionFromText(withoutPhone)
  if (region) {
    form.address = region.codes
    form.address_name = region.names
  }

  const detailSource = extractAddressField(content, ['详细地址', '地址'], 80) || withoutPhone
  const detailText = region
    ? removeRegionNames(detailSource, region.names)
    : detailSource.replace(/[，,]/g, ' ')
  const parts = detailText
    .replace(receiver || '', ' ')
    .replace(/收货人|联系人|姓名|手机号码|手机号|电话|联系方式|详细地址|地址/g, ' ')
    .split(/\s+/)
    .map((item) => item.trim())
    .filter(Boolean)
  if (!form.receiver && parts.length > 1 && parts[0].length <= 8) {
    form.receiver = parts.shift()
  }
  if (!form.detail && parts.length) {
    form.detail = parts.join(' ')
  }

  return Boolean(form.receiver || form.contact || form.address?.length || form.detail)
}

function extractAddressField(text: string, labels: string[], maxLength = 12) {
  for (const label of labels) {
    const pattern = new RegExp(`${label}[:：\\s]*([^，,\\s]+)`)
    const matched = text.match(pattern)?.[1]?.trim() ?? ''
    if (matched) {
      return matched.slice(0, maxLength)
    }
  }
  return ''
}

function findActiveAddressFormBlock() {
  const sessionID = activeSessionID.value
  const list = messages.value[sessionID] ?? []
  for (let index = list.length - 1; index >= 0; index--) {
    const message = list[index]
    if (message.role !== 'ai' || message.messageID !== activeFlowMessageID.value) {
      continue
    }
    const block = [...message.blocks]
      .reverse()
      .find((item) => item.type === 'address_form' && isCurrentFlowAction(item.action))
    if (block) {
      return { message, block }
    }
  }
  return undefined
}

async function matchAddressRegionFromText(text: string) {
  await ensureAddressAreaTreeLoaded()
  const normalizedText = normalizeAddressText(text)
  for (const province of addressAreaTree.value) {
    for (const city of province.children || []) {
      for (const area of city.children || []) {
        const names = [province.text, city.text, area.text].filter(Boolean)
        if (names.every((name) => normalizedText.includes(normalizeAddressText(name)))) {
          return {
            codes: [province.value, city.value, area.value].map((item) => String(item)),
            names,
          }
        }
      }
    }
  }
  return undefined
}

async function ensureAddressAreaTreeLoaded() {
  if (addressAreaTree.value.length || loadingAddressAreaTree) {
    return
  }
  loadingAddressAreaTree = true
  try {
    const response = await defBaseAreaService.TreeBaseArea({})
    addressAreaTree.value = response.areas || []
  } catch {
    addressAreaTree.value = []
  } finally {
    loadingAddressAreaTree = false
  }
}

function removeRegionNames(text: string, names: string[]) {
  return names
    .reduce((result, name) => result.split(name).join(' '), text)
    .replace(/[，,]/g, ' ')
    .trim()
}

function normalizeAddressText(value: string) {
  return value.replace(/\s+/g, '').replace(/特别行政区|自治区|省|市|区|县/g, '')
}

function startFlowReveal(sessionID: string, messageKey: string) {
  cancelFlowRevealTimer(sessionID, messageKey)

  const message = (messages.value[sessionID] ?? []).find((item) => item.key === messageKey)
  const targets = message ? resolveFlowRevealTargets(message) : []
  if (!message || !targets.length) {
    return
  }

  const initialReveal = targets.reduce<Record<string, number>>((result, target) => {
    result[buildFlowRevealKey(target.blockIndex, target.field)] = Math.min(1, target.total)
    return result
  }, {})
  updateMessageFlowReveal(sessionID, messageKey, initialReveal)

  const tick = () => {
    const current = (messages.value[sessionID] ?? []).find((item) => item.key === messageKey)
    const currentTargets = current ? resolveFlowRevealTargets(current) : []
    if (!current || !currentTargets.length) {
      flowRevealTimerMap.delete(buildFlowRevealTaskKey(sessionID, messageKey))
      return
    }

    const nextReveal = { ...(current.flowReveal ?? {}) }
    let hasMore = false
    for (const target of currentTargets) {
      const key = buildFlowRevealKey(target.blockIndex, target.field)
      const currentCount = Number(nextReveal[key] ?? 0)
      if (currentCount < target.total) {
        nextReveal[key] = currentCount + 1
        hasMore = true
      } else {
        nextReveal[key] = target.total
      }
    }
    updateMessageFlowReveal(sessionID, messageKey, nextReveal)
    scrollChatToBottom()

    const taskKey = buildFlowRevealTaskKey(sessionID, messageKey)
    if (hasMore) {
      flowRevealTimerMap.set(
        taskKey,
        setTimeout(tick, FLOW_REVEAL_INTERVAL_MS) as unknown as number,
      )
      return
    }

    flowRevealTimerMap.set(
      taskKey,
      setTimeout(() => {
        updateMessageFlowReveal(sessionID, messageKey, undefined)
        flowRevealTimerMap.delete(taskKey)
      }, FLOW_REVEAL_CLEANUP_MS) as unknown as number,
    )
  }

  flowRevealTimerMap.set(
    buildFlowRevealTaskKey(sessionID, messageKey),
    setTimeout(tick, FLOW_REVEAL_INTERVAL_MS) as unknown as number,
  )
}

function resolveFlowRevealTargets(item: ChatMessageItem) {
  const targets: { blockIndex: number; field: string; total: number }[] = []
  item.blocks.forEach((block, blockIndex) => {
    const field = resolveFlowRevealField(block)
    const total = field && Array.isArray(block[field]) ? block[field].length : 0
    if (field && total > 1) {
      targets.push({ blockIndex, field, total })
    }
  })
  return targets
}

function resolveFlowRevealField(block: AIFlowBlock) {
  if (block.type === 'goods_list' || block.type === 'pending_review_list') {
    return 'goods'
  }
  if (block.type === 'sku_selector') {
    return 'skus'
  }
  if (block.type === 'address_selector') {
    return 'addresses'
  }
  if (block.type === 'order_list') {
    return 'orders'
  }
  if (block.type === 'cart_list') {
    return 'carts'
  }
  if (block.type === 'simple_list') {
    return 'items'
  }
  return ''
}

function updateMessageFlowReveal(
  sessionID: string,
  messageKey: string,
  flowReveal?: Record<string, number>,
) {
  const list = messages.value[sessionID]
  if (!list?.length) {
    return
  }
  messages.value[sessionID] = list.map((item) => {
    if (item.key !== messageKey) {
      return item
    }
    return { ...item, flowReveal }
  })
}

function buildFlowRevealKey(blockIndex: number, field: string) {
  return `${blockIndex}:${field}`
}

function buildFlowRevealTaskKey(sessionID: string, messageKey: string) {
  return `${sessionID}:${messageKey}`
}

function splitAddressText(value: string) {
  return String(value)
    .split(/[\s/，,]+/)
    .filter(Boolean)
}

function handlePaymentBlocks(list: ChatMessageItem[]) {
  for (const item of list) {
    if (item.role !== 'ai') {
      continue
    }
    for (const block of item.blocks) {
      if (block.type === 'payment_result') {
        executePaymentBlock(block)
      }
    }
  }
}

/** 执行一次 AI 支付结果块，并通过支付键避免重复调起。 */
function executePaymentBlock(block: AIFlowBlock) {
  const payData = block.pay_data || {}
  const tradeID = Number(block.trade_id || 0)
  const platform = String(block.platform || 'jsapi')
  const paymentKey = `${tradeID}:${platform}:${payData.time_stamp || payData.h5_url || ''}`
  if (!tradeID || handledPaymentBlockSet.has(paymentKey)) {
    return
  }
  handledPaymentBlockSet.add(paymentKey)

  if (platform === 'h5' || platform === 'app') {
    openFlowH5PayUrl(String(payData.h5_url || ''), tradeID)
    return
  }

  // #ifdef MP-WEIXIN
  uni.requestPayment({
    provider: 'wxpay',
    nonceStr: String(payData.nonce_str || ''),
    package: String(payData.package || ''),
    paySign: String(payData.pay_sign || ''),
    timeStamp: String(payData.time_stamp || ''),
    signType: 'RSA',
    complete: () => {
      void redirectToOrderPayment(tradeID)
    },
  })
  // #endif

  // #ifndef MP-WEIXIN
  uni.showToast({ icon: 'none', title: '当前端暂不支持该支付方式' })
  // #endif
}

/** 打开 AI 返回的 H5 支付链接，并衔接当前交易的支付结果页。 */
function openFlowH5PayUrl(url: string, tradeID: number) {
  if (!url) {
    uni.showToast({ icon: 'none', title: '支付链接为空' })
    return
  }
  // #ifdef H5
  window.location.href = appendOrderPaymentRedirectUrl(url, tradeID)
  // #endif
  // #ifdef APP-PLUS
  plus.runtime.openURL(url)
  void redirectToOrderPayment(tradeID)
  // #endif
}

function upsertSession(session: AiSession) {
  if (!session.id) {
    return
  }
  const nextList = sessions.value.filter((item) => item.id !== session.id)
  nextList.unshift(session)
  sessions.value = nextList.sort((left, right) => {
    return resolveTimestamp(right.updated_at) - resolveTimestamp(left.updated_at)
  })
}

function isSessionSending(sessionID: string) {
  return Boolean(sessionID && sendingSessionMap.value[sessionID])
}

function setSessionSending(sessionID: string, sending: boolean) {
  if (!sessionID) {
    return
  }
  const nextMap = { ...sendingSessionMap.value }
  if (sending) {
    nextMap[sessionID] = true
  } else {
    delete nextMap[sessionID]
  }
  sendingSessionMap.value = nextMap
}

function cancelSessionStreamTask(sessionID: string) {
  const task = runningStreamTaskMap.get(sessionID)
  if (!task) {
    return
  }
  task.finished = true
  task.abort()
  runningStreamTaskMap.delete(sessionID)
  setSessionSending(sessionID, false)
}

function cancelFlowRevealTimer(sessionID: string, messageKey: string) {
  const taskKey = buildFlowRevealTaskKey(sessionID, messageKey)
  const timer = flowRevealTimerMap.get(taskKey)
  if (!timer) {
    return
  }
  clearTimeout(timer)
  flowRevealTimerMap.delete(taskKey)
}

function cancelSessionFlowRevealTimers(sessionID: string) {
  Array.from(flowRevealTimerMap.keys()).forEach((taskKey) => {
    if (!taskKey.startsWith(`${sessionID}:`)) {
      return
    }
    const timer = flowRevealTimerMap.get(taskKey)
    if (timer) {
      clearTimeout(timer)
    }
    flowRevealTimerMap.delete(taskKey)
  })
}

function cancelAllFlowRevealTimers() {
  Array.from(flowRevealTimerMap.values()).forEach((timer) => clearTimeout(timer))
  flowRevealTimerMap.clear()
}

function cancelAllStreamTasks() {
  Array.from(runningStreamTaskMap.keys()).forEach((sessionID) => cancelSessionStreamTask(sessionID))
}

function scrollChatToBottom() {
  if (!activeSessionID.value) {
    return
  }
  void nextTick(() => {
    chatBottomAnchor.value = ''
    void nextTick(() => {
      chatBottomAnchor.value = 'chat-bottom-anchor'
      setTimeout(() => {
        chatBottomAnchor.value = ''
        void nextTick(() => {
          chatBottomAnchor.value = 'chat-bottom-anchor'
        })
      }, 80)
    })
  })
}

function markAllMessagesSpeaking(messageKey?: string) {
  Object.keys(messages.value).forEach((sessionID) => {
    messages.value[sessionID] = (messages.value[sessionID] ?? []).map((item) => ({
      ...item,
      speaking: Boolean(messageKey && item.key === messageKey),
    }))
  })
}

function normalizeAttachmentFile(
  path: string,
  name: string,
  size: number,
): AttachmentFileCandidate {
  const fileName = name || resolveFileName(path)
  const extension = resolveFileExtension(fileName || path)
  return {
    name: fileName || '未命名附件',
    path,
    size,
    extension,
    mimeType: resolveMimeType(fileName || path),
  }
}

function resolveFileName(path: string) {
  return path.split(/[\\/]/).pop()?.split('?')[0] || ''
}

function resolveFileExtension(name: string) {
  const cleanName = name.split('?')[0].split('#')[0]
  const index = cleanName.lastIndexOf('.')
  if (index < 0 || index === cleanName.length - 1) {
    return ''
  }
  return cleanName.slice(index + 1).toLowerCase()
}

function isSupportedAttachmentExtension(extension: string) {
  return supportedAttachmentExtensions.includes(extension)
}

function isImageAttachment(attachment: AiAttachment) {
  const mimeType = attachment.mime_type || resolveMimeType(attachment.name || attachment.url)
  return mimeType.startsWith('image/')
}

function isDocumentPreviewAttachment(attachment: AiAttachment) {
  const extension = resolveFileExtension(attachment.name || attachment.url)
  return ['pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx'].includes(extension)
}

function isTextPreviewAttachment(attachment: AiAttachment) {
  const extension = resolveFileExtension(attachment.name || attachment.url)
  return ['txt', 'json', 'csv', 'xml', 'md', 'markdown'].includes(extension)
}

function resolveAttachmentIcon(attachment: AiAttachment) {
  const extension = resolveFileExtension(attachment.name || attachment.url)
  if (imageAttachmentExtensions.includes(extension)) {
    return '图'
  }
  if (extension === 'pdf') {
    return 'PDF'
  }
  if (['doc', 'docx'].includes(extension)) {
    return 'W'
  }
  if (['xls', 'xlsx'].includes(extension)) {
    return 'X'
  }
  if (['ppt', 'pptx'].includes(extension)) {
    return 'P'
  }
  return '文'
}

function formatAttachmentMeta(attachment: AiAttachment) {
  const extension = resolveFileExtension(attachment.name || attachment.url)
  const labelMap: Record<string, string> = {
    jpg: '图片',
    jpeg: '图片',
    png: '图片',
    gif: '图片',
    webp: '图片',
    pdf: 'PDF 文档',
    doc: 'Word 文档',
    docx: 'Word 文档',
    xls: 'Excel 表格',
    xlsx: 'Excel 表格',
    ppt: 'PPT 文档',
    pptx: 'PPT 文档',
    txt: '文本',
    json: 'JSON',
    csv: 'CSV',
    xml: 'XML',
    md: 'Markdown',
    markdown: 'Markdown',
  }
  const typeLabel = labelMap[extension] || attachment.mime_type || '附件'
  return attachment.size ? `${typeLabel} · ${formatAttachmentSize(attachment.size)}` : typeLabel
}

function formatAttachmentSize(size: number) {
  if (size >= 1024 * 1024) {
    return `${(size / 1024 / 1024).toFixed(1)}MB`
  }
  if (size >= 1024) {
    return `${Math.ceil(size / 1024)}KB`
  }
  return `${size}B`
}

async function openDocumentAttachment(attachment: AiAttachment, url: string) {
  if (isTextPreviewAttachment(attachment)) {
    await openTextAttachmentPreview(attachment, url)
    return
  }
  if (!isDocumentPreviewAttachment(attachment)) {
    uni.showToast({ icon: 'none', title: '该文件暂不支持预览' })
    return
  }
  // #ifdef MP-WEIXIN
  const downloadResult = await uni.downloadFile({ url })
  if (downloadResult.statusCode !== 200) {
    uni.showToast({ icon: 'none', title: '文件下载失败' })
    return
  }
  await wx.openDocument({
    filePath: downloadResult.tempFilePath,
    fileType: resolveFileExtension(
      attachment.name || attachment.url,
    ) as WechatMiniprogram.OpenDocumentOption['fileType'],
    showMenu: true,
  })
  // #endif
  // #ifdef H5
  window.open(url, '_blank')
  // #endif
}

async function openTextAttachmentPreview(attachment: AiAttachment, url: string) {
  loadingTextPreview.value = true
  showTextPreview.value = true
  textPreviewTitle.value = attachment.name || '文本附件'
  textPreviewContent.value = '正在加载...'
  try {
    // #ifdef MP-WEIXIN
    const downloadResult = await uni.downloadFile({ url })
    if (downloadResult.statusCode !== 200) {
      throw new Error('文件下载失败')
    }
    const fileSystemManager = wx.getFileSystemManager()
    textPreviewContent.value = fileSystemManager.readFileSync(
      downloadResult.tempFilePath,
      'utf-8',
    ) as string
    // #endif
    // #ifdef H5
    const response = await fetch(url)
    if (!response.ok) {
      throw new Error('文件加载失败')
    }
    textPreviewContent.value = await response.text()
    // #endif
  } catch (error) {
    textPreviewContent.value = error instanceof Error ? error.message : '文件加载失败'
  } finally {
    loadingTextPreview.value = false
  }
}

function closeTextPreview() {
  showTextPreview.value = false
  loadingTextPreview.value = false
  textPreviewTitle.value = ''
  textPreviewContent.value = ''
}

function resolveMimeType(name: string) {
  return attachmentMimeMap[resolveFileExtension(name)] || ''
}

function isUserCancelError(error: unknown) {
  const message = error instanceof Error ? error.message : String(error ?? '')
  return message.includes('cancel') || message.includes('取消')
}

function showError(error: unknown, fallback: string) {
  const message = error instanceof Error ? error.message : fallback
  uni.showToast({ icon: 'none', title: message || fallback })
}
</script>

<template>
  <view class="ai-page">
    <view class="ai-navbar" :style="{ paddingTop: navTopPadding, height: navHeight }">
      <view class="ai-navbar__left" :style="{ width: navSideWidth }">
        <button
          class="nav-back-button"
          :style="{ width: navButtonSize, height: navButtonSize }"
          hover-class="none"
          @tap="navigateBack"
        >
          <view class="nav-back-icon"></view>
        </button>
        <button
          class="history-button ai-session-button"
          :style="{ width: navButtonSize, height: navButtonSize }"
          hover-class="none"
          @tap="toggleSessionDrawer"
        >
          <view class="history-icon">
            <view></view>
            <view></view>
            <view></view>
          </view>
        </button>
      </view>
      <view
        class="ai-navbar__title"
        :style="{
          left: navSideWidth,
          right: navSideWidth,
          height: navRowHeight,
          lineHeight: navRowHeight,
        }"
      >
        AI 助手
      </view>
      <view class="ai-navbar__right" :style="{ width: navSideWidth }"></view>
    </view>
    <scroll-view
      class="ai-body"
      scroll-y
      scroll-with-animation
      :scroll-into-view="chatBottomAnchor"
      :show-scrollbar="false"
    >
      <template v-if="!hasMessages">
        <WelcomePanel
          :greeting-message="aiGreetingMessage"
          :loading="loadingSessions || loadingShortcuts"
          :shortcuts="starterPrompts"
          :can-refresh="canRefreshStarterPrompts"
          @refresh="refreshStarterPrompts"
          @shortcut-tap="handleStarterPrompt"
        />
      </template>

      <view v-else class="chat-list">
        <view
          v-for="item in currentMessages"
          :key="item.key"
          class="message-row"
          :class="item.role === 'user' ? 'is-user' : 'is-ai'"
        >
          <view class="message-stack" :class="item.role === 'user' ? 'is-user' : 'is-ai'">
            <view
              class="bubble"
              :class="[
                item.role === 'ai' ? 'ai-bubble' : '',
                item.status === AiMessageStatus.GENERATING_AAMS ? 'is-streaming' : '',
              ]"
              @longpress="openMessageActionSheet(item, $event)"
            >
              <view v-if="item.role === 'ai'" class="reply-meta">
                <text class="reply-tag">{{
                  item.fallback ? '降级回复' : item.replySource === 'llm' ? '模型回复' : '系统回复'
                }}</text>
                <text v-if="item.model" class="reply-model">{{ item.model }}</text>
              </view>

              <view
                v-if="isThinkingMessage(item) && item.content === THINKING_MESSAGE_CONTENT"
                class="typing-content"
              >
                <text>{{ item.content }}</text>
                <view class="typing-dots">
                  <view class="typing-dot"></view>
                  <view class="typing-dot"></view>
                  <view class="typing-dot"></view>
                </view>
              </view>
              <view v-else class="bubble-content">{{ item.content }}</view>

              <FlowBlocks
                :message="item"
                :active-flow-message-id="activeFlowMessageID"
                :address-form-steps="addressFormSteps"
                :address-area-tree="addressAreaTree"
                @flow-action="handleFlowAction"
                @sku-num-change="changeSkuNum"
                @sku-submit="submitSkuSelection"
                @create-address-guide="handleCreateAddressGuide"
                @address-submit="submitAddressForm"
                @address-region-change="onAddressRegionChange"
                @address-city-change="onAddressCityChange"
                @review-submit="submitReviewForm"
                @review-score-change="changeReviewScore"
              />

              <view v-if="item.attachments.length" class="attachment-list">
                <view
                  v-for="attachment in item.attachments"
                  :key="attachment.id || attachment.url || attachment.name"
                  class="attachment-card"
                  @tap="previewAttachment(attachment, item.attachments)"
                >
                  <view class="attachment-icon">{{ resolveAttachmentIcon(attachment) }}</view>
                  <view class="attachment-info">
                    <view class="attachment-name">{{ attachment.name }}</view>
                    <view class="attachment-meta">{{ formatAttachmentMeta(attachment) }}</view>
                  </view>
                </view>
              </view>

              <view v-if="item.tools.length" class="tool-row"
                >已调用：{{ formatTools(item.tools) }}</view
              >
            </view>
            <view v-if="editingMessageKey === item.key" class="message-edit">
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
        <view id="chat-bottom-anchor" class="chat-bottom-anchor"></view>
      </view>
    </scroll-view>

    <view
      v-if="hasMessages && (starterPrompts.length || loadingShortcuts)"
      class="thread-prompt-panel"
    >
      <view class="thread-prompt-head">
        <view class="thread-prompt-title">快捷操作</view>
        <button
          v-if="canRefreshStarterPrompts"
          class="thread-prompt-refresh"
          hover-class="none"
          @tap="refreshStarterPrompts"
        >
          <text>换一换</text>
          <uni-icons type="refresh" size="22" color="#00a96b" />
        </button>
      </view>
      <view v-if="loadingShortcuts" class="thread-prompt-empty">正在加载快捷助手...</view>
      <scroll-view v-else class="thread-prompt-scroll" scroll-x :show-scrollbar="false">
        <view class="thread-prompt-list">
          <button
            v-for="(shortcut, shortcutIndex) in starterPrompts"
            :key="shortcut.key || shortcut.title"
            class="thread-prompt-item"
            :class="{
              'is-disabled':
                loadingSessions || currentSessionSending || uploadingAttachment || isRecording,
            }"
            :disabled="
              loadingSessions || currentSessionSending || uploadingAttachment || isRecording
            "
            hover-class="none"
            @tap="handleStarterPrompt(shortcut)"
          >
            <text class="thread-prompt-mark">{{ shortcutIndex + 1 }}</text>
            <text class="thread-prompt-text">{{ shortcut.title }}</text>
          </button>
        </view>
      </scroll-view>
    </view>

    <Composer
      v-model="inputText"
      :attachments="selectedAttachments"
      :placeholder="composerPlaceholder"
      :bottom="composerBottom"
      :recording="isRecording"
      :sending="currentSessionSending"
      :disabled="isSubmitDisabled"
      @attach="handleAttachment"
      @record="handleToggleRecord"
      @send="handleSend"
      @remove-attachment="removeSelectedAttachment"
    />

    <SessionDrawer
      v-model:keyword="sessionKeyword"
      :open="showSessionDrawer"
      :top-padding="drawerTopPadding"
      :loading="loadingSessions"
      :sessions="filteredSessions"
      :active-session-id="activeSessionID"
      @close="toggleSessionDrawer"
      @create="createSession"
      @select="selectSession"
      @action="openSessionActionSheet"
    />

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
              class="operation-icon-symbol"
              :class="[`is-${action.icon}`, { 'is-danger': action.danger }]"
            >
              <view class="operation-icon-core"></view>
            </view>
            <text class="operation-label">{{ action.label }}</text>
          </button>
          <button
            v-if="hasRuntimeBrief(actionMessage)"
            class="operation-item"
            hover-class="none"
            @tap="handleSelectedRuntimeOperation"
          >
            <view class="operation-icon-symbol is-data-analysis">
              <view class="operation-icon-core"></view>
            </view>
            <text class="operation-label">运行明细</text>
            <text class="operation-runtime">
              {{ formatTokenBrief(actionMessage) }} {{ formatDurationBrief(actionMessage) }}
            </text>
          </button>
        </template>
        <template v-else-if="actionSession">
          <view class="operation-title">{{ actionSession.title }}</view>
          <button class="operation-item" hover-class="none" @tap="handleSessionOperation('rename')">
            <view class="operation-icon-symbol is-edit-pen">
              <view class="operation-icon-core"></view>
            </view>
            <text class="operation-label">重命名</text>
          </button>
          <button
            class="operation-item is-danger"
            hover-class="none"
            @tap="handleSessionOperation('delete')"
          >
            <view class="operation-icon-symbol is-delete is-danger">
              <view class="operation-icon-core"></view>
            </view>
            <text class="operation-label">删除会话</text>
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

    <view v-if="showTextPreview" class="text-preview-mask" @tap="closeTextPreview">
      <view class="text-preview-dialog" @tap.stop>
        <view class="text-preview-head">
          <view class="text-preview-title">{{ textPreviewTitle }}</view>
          <button class="text-preview-close" hover-class="none" @tap="closeTextPreview">
            关闭
          </button>
        </view>
        <scroll-view class="text-preview-body" scroll-y :show-scrollbar="false">
          <text class="text-preview-content">
            {{ loadingTextPreview ? '正在加载...' : textPreviewContent }}
          </text>
        </scroll-view>
      </view>
    </view>
  </view>
</template>

<style lang="scss">
page {
  height: 100%;
  overflow: hidden;
  background-color: #f6f6f6;
}

.ai-page {
  position: relative;
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  overflow: hidden;
  color: #333;
  background-color: #f6f6f6;
  box-sizing: border-box;
}

.history-button,
.nav-back-button,
.thread-prompt-refresh,
.thread-prompt-item,
.operation-item,
.message-edit-button,
.rename-button,
.text-preview-close {
  padding: 0;
  margin: 0;
  border-radius: 0;
  background: transparent;
  line-height: normal;

  &::after {
    border: none;
  }
}

.ai-navbar {
  position: relative;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  padding-right: 18rpx;
  padding-left: 18rpx;
  border-bottom: 1rpx solid #e9e9e9;
  background-color: #fff;
  box-sizing: border-box;
}

.ai-navbar__left,
.ai-navbar__right {
  z-index: 1;
  display: flex;
  align-items: center;
  flex-shrink: 0;
  min-width: 0;
}

.ai-navbar__left {
  justify-content: flex-start;
  gap: 18rpx;
}

.ai-navbar__right {
  justify-content: flex-end;
}

.ai-navbar__title {
  position: absolute;
  bottom: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #111;
  font-size: 32rpx;
  font-weight: 600;
  text-align: center;
}

.nav-back-button {
  display: flex;
  align-items: center;
  justify-content: center;
}

.nav-back-icon {
  width: 22rpx;
  height: 22rpx;
  border-bottom: 4rpx solid #111;
  border-left: 4rpx solid #111;
  transform: rotate(45deg);
}

.history-button {
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8rpx;
  color: #27ba9b;
  font-size: 24rpx;
  background-color: transparent;
}

.history-icon {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 6rpx;
  width: 30rpx;
  height: 32rpx;
}

.history-icon view {
  width: 30rpx;
  height: 4rpx;
  border-radius: 4rpx;
  background-color: #333;
}

.ai-session-button {
  background-color: #fff;
}

.ai-body {
  flex: 1;
  width: 100%;
  min-height: 0;
  padding: 44rpx 28rpx 24rpx;
  box-sizing: border-box;
  background-color: #f6f6f6;
}

.chat-list {
  padding-bottom: 36rpx;
}

.thread-prompt-panel {
  flex-shrink: 0;
  padding: 24rpx 24rpx 20rpx;
  margin: 0 28rpx 10rpx;
  border: 1rpx solid #e7ecef;
  border-radius: 10rpx;
  background-color: #fff;
  box-shadow: 0 10rpx 28rpx rgba(15, 23, 42, 0.05);
  box-sizing: border-box;
}

.thread-prompt-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  margin-bottom: 18rpx;
}

.thread-prompt-title {
  color: #111;
  font-size: 28rpx;
  font-weight: 700;
  line-height: 38rpx;
}

.thread-prompt-refresh {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 10rpx;
  height: 46rpx;
  color: #00a96b;
  font-size: 24rpx;
  line-height: 46rpx;
}

.thread-prompt-scroll {
  width: 100%;
  white-space: nowrap;
}

.thread-prompt-list {
  display: inline-flex;
  gap: 14rpx;
  min-width: 100%;
}

.thread-prompt-item {
  display: inline-flex;
  align-items: center;
  gap: 12rpx;
  max-width: 360rpx;
  height: 70rpx;
  padding: 0 18rpx 0 14rpx;
  border: 1rpx solid #dbe9e5;
  border-radius: 8rpx;
  color: #111;
  text-align: left;
  background-color: #f8fcfb;
  box-sizing: border-box;
  vertical-align: top;
}

.thread-prompt-item.is-disabled {
  opacity: 0.55;
}

.thread-prompt-mark {
  flex-shrink: 0;
  width: 34rpx;
  height: 34rpx;
  border-radius: 8rpx;
  color: #00a96b;
  font-size: 20rpx;
  font-weight: 700;
  line-height: 34rpx;
  text-align: center;
  background-color: #e7f7f2;
}

.thread-prompt-text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 25rpx;
  line-height: 34rpx;
}

.thread-prompt-empty {
  padding: 12rpx 0;
  color: #8d929c;
  font-size: 24rpx;
  line-height: 34rpx;
}

.chat-bottom-anchor {
  height: 1rpx;
}

.message-row {
  display: flex;
  margin-bottom: 28rpx;
}

.message-row.is-user {
  justify-content: flex-end;
}

.message-row.is-ai {
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
  overflow: visible;
}

.ai-bubble {
  width: 100%;
  color: #333;
  background-color: #fff;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.03);
}

.ai-bubble.is-streaming {
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
  display: block;
  min-height: 42rpx;
  max-width: 100%;
  overflow: visible;
  line-height: 42rpx;
  white-space: pre-wrap;
  overflow-wrap: break-word;
  word-break: break-word;
  word-wrap: break-word;
}

.typing-content {
  display: flex;
  align-items: center;
  min-height: 42rpx;
  color: #333;
  font-size: 28rpx;
  line-height: 42rpx;
}

.typing-dots {
  display: flex;
  align-items: center;
  gap: 7rpx;
  margin-left: 12rpx;
}

.typing-dot {
  width: 8rpx;
  height: 8rpx;
  border-radius: 50%;
  background-color: #27ba9b;
  animation: typing-dot-bounce 1.2s infinite ease-in-out;
}

.typing-dot:nth-child(2) {
  animation-delay: 0.16s;
}

.typing-dot:nth-child(3) {
  animation-delay: 0.32s;
}

@keyframes typing-dot-bounce {
  0%,
  80%,
  100% {
    opacity: 0.35;
    transform: translateY(0);
  }

  40% {
    opacity: 1;
    transform: translateY(-5rpx);
  }
}

.attachment-list {
  margin-top: 22rpx;
}

.attachment-card {
  display: flex;
  align-items: center;
  gap: 16rpx;
  padding: 18rpx;
  border-radius: 10rpx;
  background-color: #f6f7f9;
}

.attachment-card + .attachment-card {
  margin-top: 16rpx;
}

.attachment-icon {
  flex-shrink: 0;
  width: 56rpx;
  height: 56rpx;
  border-radius: 8rpx;
  color: #27ba9b;
  font-size: 28rpx;
  line-height: 56rpx;
  text-align: center;
  background-color: #e8f8f4;
}

.attachment-info {
  flex: 1;
  min-width: 0;
}

.attachment-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #333;
  font-size: 24rpx;
  font-weight: 600;
  line-height: 32rpx;
}

.attachment-meta {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-top: 4rpx;
  color: #898b94;
  font-size: 20rpx;
  line-height: 28rpx;
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

.operation-label {
  min-width: 0;
}

.operation-runtime {
  flex: 1;
  min-width: 0;
  color: #999;
  font-size: 22rpx;
  text-align: right;
}

.operation-icon-symbol {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 42rpx;
  height: 42rpx;
}

.operation-icon-core {
  width: 32rpx;
  height: 32rpx;
  background-repeat: no-repeat;
  background-position: center;
  background-size: 100% 100%;
}

.operation-icon-symbol.is-refresh .operation-icon-core {
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 1024 1024' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath fill='%2364748b' d='M771.776 794.88A384 384 0 0 1 128 512h64a320 320 0 0 0 555.712 216.448H654.72a32 32 0 1 1 0-64h149.056a32 32 0 0 1 32 32v148.928a32 32 0 1 1-64 0v-50.56zM276.288 295.616h92.992a32 32 0 0 1 0 64H220.16a32 32 0 0 1-32-32V178.56a32 32 0 0 1 64 0v50.56A384 384 0 0 1 896.128 512h-64a320 320 0 0 0-555.776-216.384z'/%3E%3C/svg%3E");
}

.operation-icon-symbol.is-copy-document .operation-icon-core {
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 1024 1024' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath fill='%2364748b' d='M768 832a128 128 0 0 1-128 128H192A128 128 0 0 1 64 832V384a128 128 0 0 1 128-128v64a64 64 0 0 0-64 64v448a64 64 0 0 0 64 64h448a64 64 0 0 0 64-64z'/%3E%3Cpath fill='%2364748b' d='M384 128a64 64 0 0 0-64 64v448a64 64 0 0 0 64 64h448a64 64 0 0 0 64-64V192a64 64 0 0 0-64-64zm0-64h448a128 128 0 0 1 128 128v448a128 128 0 0 1-128 128H384a128 128 0 0 1-128-128V192A128 128 0 0 1 384 64'/%3E%3C/svg%3E");
}

.operation-icon-symbol.is-delete .operation-icon-core {
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 1024 1024' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath fill='%2364748b' d='M160 256H96a32 32 0 0 1 0-64h256V95.936a32 32 0 0 1 32-32h256a32 32 0 0 1 32 32V192h256a32 32 0 1 1 0 64h-64v672a32 32 0 0 1-32 32H192a32 32 0 0 1-32-32zm448-64v-64H416v64zM224 896h576V256H224zm192-128a32 32 0 0 1-32-32V416a32 32 0 0 1 64 0v320a32 32 0 0 1-32 32m192 0a32 32 0 0 1-32-32V416a32 32 0 0 1 64 0v320a32 32 0 0 1-32 32'/%3E%3C/svg%3E");
}

.operation-icon-symbol.is-delete.is-danger .operation-icon-core {
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 1024 1024' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath fill='%23cf4444' d='M160 256H96a32 32 0 0 1 0-64h256V95.936a32 32 0 0 1 32-32h256a32 32 0 0 1 32 32V192h256a32 32 0 1 1 0 64h-64v672a32 32 0 0 1-32 32H192a32 32 0 0 1-32-32zm448-64v-64H416v64zM224 896h576V256H224zm192-128a32 32 0 0 1-32-32V416a32 32 0 0 1 64 0v320a32 32 0 0 1-32 32m192 0a32 32 0 0 1-32-32V416a32 32 0 0 1 64 0v320a32 32 0 0 1-32 32'/%3E%3C/svg%3E");
}

.operation-icon-symbol.is-edit-pen .operation-icon-core {
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 1024 1024' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath fill='%2364748b' d='m199.04 672.64 193.984 112 224-387.968-193.92-112-224 388.032zm-23.872 60.16 32.896 148.288 144.896-45.696zM455.04 229.248l193.92 112 56.704-98.112-193.984-112zM104.32 708.8l384-665.024 304.768 175.936L409.152 884.8h.064l-248.448 78.336zm384 254.272v-64h448v64z'/%3E%3C/svg%3E");
}

.operation-icon-symbol.is-data-analysis .operation-icon-core {
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 1024 1024' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath fill='%2364748b' d='m665.216 768 110.848 192h-73.856L591.36 768H433.024L322.176 960H248.32l110.848-192H160a32 32 0 0 1-32-32V192H64a32 32 0 0 1 0-64h896a32 32 0 1 1 0 64h-64v544a32 32 0 0 1-32 32zM832 192H192v512h640zM352 448a32 32 0 0 1 32 32v64a32 32 0 0 1-64 0v-64a32 32 0 0 1 32-32m160-64a32 32 0 0 1 32 32v128a32 32 0 0 1-64 0V416a32 32 0 0 1 32-32m160-64a32 32 0 0 1 32 32v192a32 32 0 1 1-64 0V352a32 32 0 0 1 32-32'/%3E%3C/svg%3E");
}

.operation-icon-symbol.is-branch-action .operation-icon-core {
  background-image: url("data:image/svg+xml,%3Csvg viewBox='96 160 832 704' xmlns='http://www.w3.org/2000/svg' fill='none'%3E%3Cpath d='M256 832V192' stroke='%2364748b' stroke-width='64' stroke-linecap='round'/%3E%3Cpath d='M256 192 128 320M256 192l128 128' stroke='%2364748b' stroke-width='64' stroke-linecap='round' stroke-linejoin='round'/%3E%3Cpath d='M384 640c256 0 384-176 384-448' stroke='%2364748b' stroke-width='64' stroke-linecap='round'/%3E%3Cpath d='M768 192 640 320M768 192l128 128' stroke='%2364748b' stroke-width='64' stroke-linecap='round' stroke-linejoin='round'/%3E%3C/svg%3E");
}

.operation-icon-symbol.is-speak-action .operation-icon-core {
  background-image: url("data:image/svg+xml,%3Csvg viewBox='80 160 864 704' xmlns='http://www.w3.org/2000/svg' fill='none'%3E%3Cpath d='M128 400h144L512 224v576L272 624H128z' stroke='%2364748b' stroke-width='64' stroke-linejoin='round'/%3E%3Cpath d='M640 352a224 224 0 0 1 0 320' stroke='%2364748b' stroke-width='64' stroke-linecap='round'/%3E%3Cpath d='M768 224a384 384 0 0 1 0 576' stroke='%2364748b' stroke-width='64' stroke-linecap='round'/%3E%3C/svg%3E");
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

.text-preview-mask {
  position: absolute;
  inset: 0;
  z-index: 32;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 42rpx;
  background-color: rgba(0, 0, 0, 0.32);
  box-sizing: border-box;
}

.text-preview-dialog {
  display: flex;
  flex-direction: column;
  width: 100%;
  max-height: 78%;
  border-radius: 10rpx;
  background-color: #fff;
  overflow: hidden;
}

.text-preview-head {
  display: flex;
  align-items: center;
  gap: 18rpx;
  padding: 24rpx 26rpx;
  border-bottom: 1rpx solid #eef0f3;
}

.text-preview-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #333;
  font-size: 28rpx;
  font-weight: 600;
  line-height: 40rpx;
}

.text-preview-close {
  width: 96rpx;
  height: 52rpx;
  border-radius: 8rpx;
  color: #27ba9b;
  font-size: 24rpx;
  line-height: 52rpx;
  background-color: #e8f8f4;
}

.text-preview-body {
  min-height: 260rpx;
  max-height: 760rpx;
  padding: 24rpx;
  box-sizing: border-box;
}

.text-preview-content {
  white-space: pre-wrap;
  word-break: break-word;
  color: #333;
  font-size: 24rpx;
  line-height: 38rpx;
}
</style>
