<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { computed, nextTick, onBeforeUnmount, ref } from 'vue'
import { defBaseAreaService } from '@/api/app/base_area'
import {
  defAiAssistantMessageService,
  StreamAiAssistantMessageByChunkedRequest,
} from '@/api/base/ai_assistant_message'
import { defAiAssistantSessionService } from '@/api/base/ai_assistant_session'
import type { AiAssistantAction } from '@/rpc/base/v1/ai_assistant_message'
import type { AppTreeOptionResponse_Option } from '@/rpc/common/v1/common'
import type {
  AiAssistantAttachment,
  AiAssistantMessage,
  AiAssistantSession,
  AiAssistantTool,
} from '@/rpc/base/v1/ai_assistant_session'
import { AiAssistantMessageStatus, Terminal } from '@/rpc/common/v1/enum'
import { uploadFile } from '@/utils/file'
import { formatPrice, formatSrc } from '@/utils/index'
import {
  type AiAssistantStreamEvent,
  type AiAssistantStreamPayload,
  createAiAssistantEventStreamTextParser,
  parseAiAssistantEventStreamText,
  readAiAssistantEventStream,
} from './stream'

type ChatRole = 'user' | 'assistant'

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

type ChatMessageItem = AiAssistantMessage & {
  key: string
  messageID: string
  role: ChatRole
  content: string
  status: AiAssistantMessageStatus
  tools: AiAssistantTool[]
  model: string
  replySource: string
  fallback: boolean
  fallbackReason: string
  flow: string
  step: string
  blocksJson: string
  blocks: AssistantFlowBlock[]
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
  attachments: AiAssistantAttachment[]
  action?: AiAssistantAction
}

type AssistantFlowBlock = {
  type: string
  [key: string]: any
}

type StreamTask = {
  abort: () => void
  aborted: boolean
  finished: boolean
}

type AttachmentFileCandidate = {
  name: string
  path: string
  size: number
  extension: string
  mimeType: string
}

const THINKING_MESSAGE_CONTENT = '正在回复'
const LOCAL_USER_MESSAGE_PREFIX = 'assistant-user-local'
const PENDING_MESSAGE_ID = 'pending'
const MAX_ATTACHMENT_COUNT = 6
const AI_ASSISTANT_TERMINAL = Terminal.TERMINAL_APP
const FLOW_REVEAL_INTERVAL_MS = 90
const FLOW_REVEAL_CLEANUP_MS = 240
const starterPrompts = ['帮我推荐商品']
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
const navTopPadding = `${safeAreaInsets?.top || 0}px`
const composerBottom = `${Math.max(safeAreaInsets?.bottom || 0, 9)}px`
const windowWidth = systemInfo.windowWidth || systemInfo.screenWidth || 375
const windowHeight = systemInfo.windowHeight || systemInfo.screenHeight || 667
const drawerTopPadding = `${(safeAreaInsets?.top || 0) + 12}px`
const showSessionDrawer = ref(false)
const activeSessionID = ref('')
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
const loadingSessions = ref(false)
const loadingSessionID = ref('')
const chatBottomAnchor = ref('')
const uploadingAttachment = ref(false)
const showTextPreview = ref(false)
const loadingTextPreview = ref(false)
const textPreviewTitle = ref('')
const textPreviewContent = ref('')
const sendingSessionMap = ref<Record<string, boolean>>({})
const sessions = ref<AiAssistantSession[]>([])
const messages = ref<Record<string, ChatMessageItem[]>>({})
const selectedAttachments = ref<AiAssistantAttachment[]>([])
const addressAreaTree = ref<AppTreeOptionResponse_Option[]>([])
const runningStreamTaskMap = new Map<string, StreamTask>()
const pendingDeltaMap = new Map<string, AiAssistantStreamPayload>()
const handledPaymentBlockSet = new Set<string>()
const flowRevealTimerMap = new Map<string, number>()
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
  return item.role === 'assistant' && item.status === AiAssistantMessageStatus.GENERATING_AAMS
}

const lastEditableUserMessageKey = computed(() => {
  const list = currentMessages.value
    .filter((item) => item.role === 'user' && !item.localOnly)
    .sort((left, right) => resolveTimestamp(left.created_at) - resolveTimestamp(right.created_at))
  const lastMessage = list[list.length - 1]
  if (!lastMessage || lastMessage.status === AiAssistantMessageStatus.GENERATING_AAMS) {
    return ''
  }
  return lastMessage.key
})

/** 首次打开时加载移动端会话列表。 */
onLoad(() => {
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

/** 保存会话名称，并同步后端。 */
const confirmRenameSession = async () => {
  if (!renamingTitle.value) {
    uni.showToast({ icon: 'none', title: '请输入会话名称' })
    return
  }

  try {
    const response = await defAiAssistantSessionService.UpdateAiAssistantSession({
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
    await defAiAssistantSessionService.DeleteAiAssistantSession({ id: sessionID })
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
      await defAiAssistantMessageService.DeleteAiAssistantMessage({
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
    messages.value[sessionID] = markAssistantMessageRegenerating(
      updatedList,
      sessionID,
      item.messageID,
    )

    const response = await defAiAssistantMessageService.UpdateAiAssistantMessage({
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
const openSessionActionSheet = (session: AiAssistantSession, event?: PressEvent) => {
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
  return item.role === 'assistant' && Boolean(item.tokenTotal || item.durationMs)
}

const resolveMessageActions = (item: ChatMessageItem): MessageAction[] => {
  if (item.status === AiAssistantMessageStatus.GENERATING_AAMS) {
    return []
  }

  const copyAction: MessageAction = { key: 'copy', icon: 'copy-document', label: '复制' }
  const deleteAction: MessageAction = { key: 'delete', icon: 'delete', label: '删除', danger: true }
  const editAction: MessageAction = { key: 'edit', icon: 'edit-pen', label: '编辑' }

  if (item.role === 'user') {
    const actions = isLastEditableUserMessage(item)
      ? [editAction, copyAction, deleteAction]
      : [copyAction, deleteAction]
    if (item.status === AiAssistantMessageStatus.FAILED_AAMS) {
      return [{ key: 'retry', icon: 'refresh', label: '重新发送' }, ...actions]
    }
    return item.localOnly ? [copyAction, deleteAction] : actions
  }

  const assistantActions: MessageAction[] = [{ key: 'retry', icon: 'refresh', label: '重新生成' }]
  if (item.status === AiAssistantMessageStatus.SUCCESS_AAMS) {
    assistantActions.push({ key: 'branch', icon: 'branch-action', label: '创建分支' })
    assistantActions.push({
      key: 'speak',
      icon: 'speak-action',
      label: item.speaking ? '停止朗读' : '朗读',
    })
  }
  return [...assistantActions, copyAction, deleteAction]
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
    const files = await Promise.all(
      attachmentFiles.map((item) => uploadFile('assistant', item.path)),
    )
    const nextAttachments = files.map<AiAssistantAttachment>((file, index) => ({
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
  await sendAiAssistantPayload(payload)
}

/** 使用空态快捷提示直接开始对话。 */
const handleStarterPrompt = async (text: string) => {
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
  await sendAiAssistantPayload({ text, attachments: [] })
}

/** 提交助手流程动作，保持流程在聊天内闭环。 */
const handleFlowAction = async (action?: AiAssistantAction, label?: string) => {
  if (!action || currentSessionSending.value) {
    return
  }
  const nextAction = enrichFlowAction(action)
  await sendAiAssistantPayload({
    text: label || resolveFlowActionLabel(nextAction),
    attachments: [],
    action: nextAction,
  })
}

/** 提交规格选择动作。 */
const submitSkuSelection = (block: AssistantFlowBlock, sku: AssistantFlowBlock) => {
  const payload = parseActionPayload(block.action?.payload_json)
  payload.sku_code = sku.sku_code
  payload.num = Number(sku.num || 1)
  const action = buildFlowAction(block.action, payload)
  void handleFlowAction(action, `选择规格：${sku.spec_text || sku.sku_code}`)
}

/** 调整规格数量。 */
const changeSkuNum = (sku: AssistantFlowBlock, delta: number) => {
  const current = Number(sku.num || 1)
  const inventory = Number(sku.inventory || 0)
  const next = Math.max(1, current + delta)
  sku.num = inventory > 0 ? Math.min(next, inventory) : next
}

/** 提交新增地址表单。 */
const submitAddressForm = (block: AssistantFlowBlock) => {
  const form = block.form || {}
  if (!form.receiver || !form.contact || !form.address?.length || !form.detail) {
    uni.showToast({ icon: 'none', title: '请补全收货地址' })
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
  void handleFlowAction(action, '新增收货地址')
}

/** 提交评价表单。 */
const submitReviewForm = (block: AssistantFlowBlock) => {
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
const changeReviewScore = (form: AssistantFlowBlock, key: string, delta: number) => {
  form[key] = Math.min(5, Math.max(1, Number(form[key] || 5) + delta))
}

/** 选择省市区，微信使用系统区域选择器，其他端使用项目行政区域树。 */
const onAddressRegionChange = (
  block: AssistantFlowBlock,
  ev: Parameters<UniHelper.RegionPickerOnChange>[0],
) => {
  block.form.address_name = ev.detail.value
  block.form.address = ev.detail.code
}

/** 选择 H5/App 行政区域树。 */
const onAddressCityChange = (
  block: AssistantFlowBlock,
  ev: Parameters<UniHelper.UniDataPickerOnChange>[0],
) => {
  const values = ev.detail.value || []
  block.form.address = values.map((item) => String(item.value))
  block.form.address_name = values.map((item) => String(item.text))
}

const formatTools = (tools: AiAssistantTool[]) => {
  return tools.map((item) => item.title || item.name).join(' · ')
}

const formatRuntime = (item: ChatMessageItem) => {
  const duration = item.durationMs ? `${(item.durationMs / 1000).toFixed(1)}s` : '生成中'
  return `${item.tokenTotal} Token · 首字 ${item.firstTokenMs}ms · 总耗时 ${duration}`
}

const formatSessionTime = (session: AiAssistantSession) => {
  const timestamp = resolveTimestamp(session.updated_at)
  if (!timestamp) {
    return ''
  }
  const date = new Date(timestamp)
  const now = new Date()
  const isToday =
    date.getFullYear() === now.getFullYear() &&
    date.getMonth() === now.getMonth() &&
    date.getDate() === now.getDate()
  const month = `${date.getMonth() + 1}`.padStart(2, '0')
  const day = `${date.getDate()}`.padStart(2, '0')
  const hour = `${date.getHours()}`.padStart(2, '0')
  const minute = `${date.getMinutes()}`.padStart(2, '0')

  if (isToday) {
    return `${hour}:${minute}`
  }
  return `${month}-${day}`
}

/** 预览消息附件，图片走图片预览，文档走平台文档预览。 */
const previewAttachment = async (
  attachment: AiAssistantAttachment,
  attachments: AiAssistantAttachment[],
) => {
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

/** 加载移动端 AI 助手会话列表。 */
async function ensureSessionsLoaded() {
  if (loadingSessions.value || sessions.value.length > 0) {
    return
  }

  loadingSessions.value = true
  try {
    const response = await defAiAssistantSessionService.ListAiAssistantSessions({
      terminal: AI_ASSISTANT_TERMINAL,
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
    const response = await defAiAssistantMessageService.ListAiAssistantMessages({
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
  const response = await defAiAssistantSessionService.CreateAiAssistantSession({
    title: options?.title || '新对话',
    terminal: AI_ASSISTANT_TERMINAL,
  })
  const session = normalizeSession(response.session)
  upsertSession(session)
  return session.id
}

/** 发送消息，H5 优先流式，其他端不支持流式时退回完整响应。 */
async function sendAiAssistantPayload(payload: SubmitPayload) {
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
  await runAiAssistantTask(sessionID, payload)
  return true
}

/** 后台执行 AI 助手消息请求。 */
async function runAiAssistantTask(sessionID: string, payload: SubmitPayload) {
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
    const parser = createAiAssistantEventStreamTextParser((event) =>
      handleAiAssistantStreamEvent(event, task),
    )
    const chunkedTask = StreamAiAssistantMessageByChunkedRequest(request, {
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
      const response = await defAiAssistantMessageService.StreamAiAssistantMessage(request, {
        signal: controller.signal,
      })
      if (!response.body) {
        throw new Error('AI 助手流式响应为空')
      }
      await readAiAssistantEventStream(
        response.body,
        (event) => handleAiAssistantStreamEvent(event, task),
        controller.signal,
      )
      if (!task.finished && !task.aborted) {
        throw new Error('AI 助手流式响应未完整返回')
      }
      handledByStream = true
    }
    // #endif

    if (!handledByStream) {
      const response = await defAiAssistantMessageService.SendAiAssistantMessage(request)
      const nextMessages = normalizeNonStreamMessages(response)
      if (!nextMessages.length) {
        throw new Error('AI 助手响应为空')
      }
      messages.value[sessionID] = replacePendingMessages(
        messages.value[sessionID] ?? [],
        nextMessages,
      )
      scrollChatToBottom()
      if (response.session) {
        upsertSession(normalizeSession(response.session))
      }
      handlePaymentBlocks(nextMessages)
    }
  } catch (error) {
    if (task?.aborted) {
      return
    }
    messages.value[sessionID] = markThinkingMessageFailed(messages.value[sessionID] ?? [], {
      sessionID,
    })
    scrollChatToBottom()
    showError(error, 'AI 助手请求失败')
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
      await sendAiAssistantPayload(payload)
      return
    }

    setSessionSending(sessionID, true)
    let response
    if (item.role === 'user') {
      if (item.status !== AiAssistantMessageStatus.FAILED_AAMS) {
        uni.showToast({ icon: 'none', title: '只有发送失败的消息可以重新发送' })
        return
      }
      response = await defAiAssistantMessageService.RetryAiAssistantUserMessage({
        session_id: sessionID,
        message_id: item.messageID,
      })
    } else {
      messages.value[sessionID] = markAssistantMessageRegenerating(
        messages.value[sessionID] ?? [],
        sessionID,
        item.messageID,
      )
      response = await defAiAssistantMessageService.RegenerateAiAssistantMessage({
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
    const response = await defAiAssistantSessionService.CreateAiAssistantSessionBranch({
      source_session_id: sourceSessionID,
      anchor_message_id: item.messageID,
      title: buildBranchSessionTitle(item),
      terminal: AI_ASSISTANT_TERMINAL,
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
function removeSelectedAttachment(attachment: AiAssistantAttachment) {
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
function handleAiAssistantStreamEvent(event: AiAssistantStreamEvent, task?: StreamTask) {
  if (event.event === 'delta') {
    handleAiAssistantDelta(event.payload)
    return
  }
  if (event.event === 'finish') {
    handleAiAssistantFinish(event.payload, task)
    return
  }
  handleAiAssistantError(event.payload, task)
}

function handleAiAssistantDelta(payload: AiAssistantStreamPayload) {
  if (!payload.delta) {
    return
  }
  queueAiAssistantDelta(payload)
}

function handleAiAssistantFinish(payload: AiAssistantStreamPayload, task?: StreamTask) {
  const sessionID = payload.session_id
  if (!sessionID) {
    return
  }
  if (task) {
    task.finished = true
  }
  flushAiAssistantDelta()
  const nextMessages = normalizeMessageList(payload.messages)
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
    (item) => item.role === 'assistant' && item.messageID === payload.message_id,
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

function handleAiAssistantError(payload: AiAssistantStreamPayload, task?: StreamTask) {
  const sessionID = payload.session_id
  if (!sessionID) {
    return
  }
  if (task) {
    task.finished = true
  }
  flushAiAssistantDelta()
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
function queueAiAssistantDelta(payload: AiAssistantStreamPayload) {
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
    flushAiAssistantDelta()
  }, 32) as unknown as number
}

function flushAiAssistantDelta() {
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

function normalizeSession(session?: Partial<AiAssistantSession> | null): AiAssistantSession {
  return {
    id: String(session?.id ?? ''),
    title: String(session?.title ?? '新对话'),
    summary: String(session?.summary ?? ''),
    updated_at: session?.updated_at,
    terminal: Number(session?.terminal ?? AI_ASSISTANT_TERMINAL),
  }
}

function normalizeSessionList(list?: AiAssistantSession[] | null) {
  if (!Array.isArray(list)) {
    return []
  }
  return list.map((item) => normalizeSession(item)).filter((item) => item.id)
}

function normalizeMessageList(list?: AiAssistantMessage[] | null) {
  if (!Array.isArray(list)) {
    return []
  }
  return sortMessages(
    list
      .filter(Boolean)
      .flatMap((item) => [mapMessageItem(item, 'user'), mapMessageItem(item, 'assistant')]),
  )
}

function normalizeNonStreamMessages(response: unknown) {
  const jsonResponse = response as { messages?: AiAssistantMessage[] }
  if (Array.isArray(jsonResponse?.messages)) {
    return normalizeMessageList(jsonResponse.messages)
  }

  const events = parseAiAssistantEventStreamText(response)
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

function parseFlowBlocks(raw?: string) {
  if (!raw) {
    return []
  }
  try {
    const blocks = JSON.parse(raw)
    if (!Array.isArray(blocks)) {
      return []
    }
    return blocks
      .filter((item): item is AssistantFlowBlock => Boolean(item?.type))
      .map((item) => normalizeFlowBlock(item))
  } catch {
    return []
  }
}

function normalizeFlowBlock(block: AssistantFlowBlock) {
  if (block.type === 'sku_selector') {
    block.skus = Array.isArray(block.skus)
      ? block.skus.map((item: AssistantFlowBlock) => ({ ...item, num: Number(item.num || 1) }))
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

function mapMessageItem(message: AiAssistantMessage, role: ChatRole): ChatMessageItem {
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
  const status = Number(message.status ?? AiAssistantMessageStatus.SUCCESS_AAMS)
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
    model: role === 'assistant' ? outputContent.model : '',
    replySource: role === 'assistant' ? outputContent.reply_source : '',
    fallback: role === 'assistant' && outputContent.fallback,
    fallbackReason: role === 'assistant' ? outputContent.fallback_reason : '',
    flow: role === 'assistant' ? outputContent.flow : '',
    step: role === 'assistant' ? outputContent.step : '',
    blocksJson: role === 'assistant' ? outputContent.blocks_json : '',
    blocks: role === 'assistant' ? parseFlowBlocks(outputContent.blocks_json) : [],
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
      status: AiAssistantMessageStatus.GENERATING_AAMS,
      token: { input: 0, output: 0, cache: 0, total: 0 },
      tools: [],
      first_token_ms: 0,
      duration_ms: 0,
    },
    'user',
  )
  message.localOnly = true
  message.status = AiAssistantMessageStatus.GENERATING_AAMS
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
      id: streamKey || `assistant-thinking-${now}`,
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
      status: AiAssistantMessageStatus.GENERATING_AAMS,
      token: { input: 0, output: 0, cache: 0, total: 0 },
      tools: [],
      first_token_ms: 0,
      duration_ms: 0,
    },
    'assistant',
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

function ensureStreamingMessage(current: ChatMessageItem[], payload: AiAssistantStreamPayload) {
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
      ? { ...item, id: messageID, messageID, key: `${messageID}:assistant`, streamKey }
      : item,
  )
  if (next.some((item) => item.streamKey === streamKey)) {
    return next
  }

  return sortMessages([...next, createThinkingMessage({ sessionID, messageID })])
}

function appendStreamingDelta(current: ChatMessageItem[], payload: AiAssistantStreamPayload) {
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
      status: AiAssistantMessageStatus.GENERATING_AAMS,
    }
  })
}

function replacePendingMessages(
  current: ChatMessageItem[],
  nextMessages: ChatMessageItem[],
  payload?: AiAssistantStreamPayload,
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
      status: AiAssistantMessageStatus.FAILED_AAMS,
      content:
        item.role === 'assistant'
          ? '这次回复没有成功返回，你可以直接重试刚才的问题。'
          : item.content,
    }
  })
}

function markStreamingError(current: ChatMessageItem[], payload: AiAssistantStreamPayload) {
  const streamKey = buildStreamMessageKey(payload.session_id, payload.message_id)
  return current.map((item) => {
    if (!item.localOnly || item.streamKey !== streamKey) {
      return item
    }
    return {
      ...item,
      status: AiAssistantMessageStatus.FAILED_AAMS,
      content: '这次回复没有成功返回，你可以直接重试刚才的问题。',
    }
  })
}

function markAssistantMessageRegenerating(
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
      status: AiAssistantMessageStatus.GENERATING_AAMS,
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

function buildFlowAction(action?: Partial<AiAssistantAction>, payload?: Record<string, any>) {
  if (!action?.type) {
    return undefined
  }
  return {
    flow: action.flow || '',
    step: action.step || '',
    type: action.type,
    payload_json: JSON.stringify(payload || {}),
  }
}

function enrichFlowAction(action: AiAssistantAction) {
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

function resolveFlowActionLabel(action: AiAssistantAction) {
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
  }
  return labelMap[action.type] || '继续'
}

function resolveFlowScrollHeight(values: unknown, maxHeight: number, rowHeight: number) {
  const count = Array.isArray(values) ? values.length : 0
  if (count <= 0) {
    return '0rpx'
  }
  const gapHeight = Math.max(0, count - 1) * 14
  return `${Math.min(maxHeight, count * rowHeight + gapHeight)}rpx`
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
    if (message.role !== 'assistant') {
      continue
    }
    const block = [...message.blocks].reverse().find((item) => item.type === 'address_form')
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
    const response = await defBaseAreaService.TreeBaseAreas({})
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

function visibleFlowList(
  item: ChatMessageItem,
  block: AssistantFlowBlock,
  blockIndex: number,
  field: string,
) {
  const list = Array.isArray(block[field]) ? block[field] : []
  if (!item.flowReveal) {
    return list
  }
  const count = item.flowReveal[buildFlowRevealKey(blockIndex, field)] ?? list.length
  return list.slice(0, count)
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

function resolveFlowRevealField(block: AssistantFlowBlock) {
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
    if (item.role !== 'assistant') {
      continue
    }
    for (const block of item.blocks) {
      if (block.type === 'payment_result') {
        executePaymentBlock(block)
      }
    }
  }
}

function executePaymentBlock(block: AssistantFlowBlock) {
  const payData = block.pay_data || {}
  const orderID = Number(block.order_id || 0)
  const platform = String(block.platform || 'jsapi')
  const paymentKey = `${orderID}:${platform}:${payData.time_stamp || payData.h5_url || ''}`
  if (!orderID || handledPaymentBlockSet.has(paymentKey)) {
    return
  }
  handledPaymentBlockSet.add(paymentKey)

  if (platform === 'h5' || platform === 'app') {
    openFlowH5PayUrl(String(payData.h5_url || ''))
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
    success: () => {
      uni.showToast({ icon: 'success', title: '支付完成' })
    },
    fail: () => {
      uni.showToast({ icon: 'none', title: '支付未完成' })
    },
  })
  // #endif

  // #ifndef MP-WEIXIN
  uni.showToast({ icon: 'none', title: '当前端暂不支持该支付方式' })
  // #endif
}

function openFlowH5PayUrl(url: string) {
  if (!url) {
    uni.showToast({ icon: 'none', title: '支付链接为空' })
    return
  }
  // #ifdef H5
  window.location.href = url
  // #endif
  // #ifdef APP-PLUS
  plus.runtime.openURL(url)
  // #endif
}

function upsertSession(session: AiAssistantSession) {
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

function isImageAttachment(attachment: AiAssistantAttachment) {
  const mimeType = attachment.mime_type || resolveMimeType(attachment.name || attachment.url)
  return mimeType.startsWith('image/')
}

function isDocumentPreviewAttachment(attachment: AiAssistantAttachment) {
  const extension = resolveFileExtension(attachment.name || attachment.url)
  return ['pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx'].includes(extension)
}

function isTextPreviewAttachment(attachment: AiAssistantAttachment) {
  const extension = resolveFileExtension(attachment.name || attachment.url)
  return ['txt', 'json', 'csv', 'xml', 'md', 'markdown'].includes(extension)
}

function resolveAttachmentIcon(attachment: AiAssistantAttachment) {
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

function formatAttachmentMeta(attachment: AiAssistantAttachment) {
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

async function openDocumentAttachment(attachment: AiAssistantAttachment, url: string) {
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

async function openTextAttachmentPreview(attachment: AiAssistantAttachment, url: string) {
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
  <view class="assistant-page">
    <view class="assistant-navbar" :style="{ paddingTop: navTopPadding }">
      <view class="assistant-navbar__left">
        <button class="nav-back-button" hover-class="none" @tap="navigateBack">
          <view class="nav-back-icon"></view>
        </button>
        <button
          class="history-button assistant-session-button"
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
      <view class="assistant-navbar__title">AI 助手</view>
      <view class="assistant-navbar__right"></view>
    </view>
    <scroll-view
      class="assistant-body"
      scroll-y
      scroll-with-animation
      :scroll-into-view="chatBottomAnchor"
      :show-scrollbar="false"
    >
      <template v-if="!hasMessages">
        <view class="empty-panel">
          <view class="empty-title">AI 助手</view>
          <view class="empty-desc">输入文字，或使用语音说出你的问题。</view>
          <view v-if="!loadingSessions" class="starter-prompts">
            <button
              v-for="starterPrompt in starterPrompts"
              :key="starterPrompt"
              class="starter-prompt"
              hover-class="none"
              @tap="handleStarterPrompt(starterPrompt)"
            >
              <text>{{ starterPrompt }}</text>
              <uni-icons type="paperplane" size="16" color="#27ba9b" />
            </button>
          </view>
        </view>

        <view class="empty-note">
          {{
            loadingSessions ? '正在加载会话...' : '对话会按当前用户保存，可在历史会话中继续追问。'
          }}
        </view>
      </template>

      <view v-else class="chat-list">
        <view
          v-for="item in currentMessages"
          :key="item.key"
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

              <view v-if="item.blocks.length" class="flow-block-list">
                <view
                  v-for="(block, blockIndex) in item.blocks"
                  :key="blockIndex"
                  class="flow-block"
                >
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
                        v-for="goods in visibleFlowList(item, block, blockIndex, 'goods')"
                        :key="goods.id"
                        class="flow-goods-card"
                        :class="{ 'flow-reveal-item': item.flowReveal }"
                        @tap="handleFlowAction(goods.action, `选择商品：${goods.name || ''}`)"
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
                          <view class="flow-price"
                            >¥{{ formatPrice(Number(goods.price || 0)) }}</view
                          >
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
                        v-for="sku in visibleFlowList(item, block, blockIndex, 'skus')"
                        :key="sku.sku_code"
                        class="flow-sku-row"
                        :class="{ 'flow-reveal-item': item.flowReveal }"
                      >
                        <view class="flow-sku-info">
                          <view class="flow-sku-name">{{ sku.spec_text || sku.sku_code }}</view>
                          <view class="flow-price">¥{{ formatPrice(Number(sku.price || 0)) }}</view>
                        </view>
                        <view class="flow-stepper">
                          <button
                            class="flow-stepper-button"
                            hover-class="none"
                            @tap="changeSkuNum(sku, -1)"
                          >
                            -
                          </button>
                          <text class="flow-stepper-value">{{ sku.num }}</text>
                          <button
                            class="flow-stepper-button"
                            hover-class="none"
                            @tap="changeSkuNum(sku, 1)"
                          >
                            +
                          </button>
                        </view>
                        <button
                          class="flow-primary-button"
                          hover-class="none"
                          @tap="submitSkuSelection(block, sku)"
                        >
                          确认
                        </button>
                      </view>
                    </scroll-view>
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
                    <view v-if="!block.addresses?.length" class="flow-empty">还没有收货地址</view>
                    <view
                      v-for="address in visibleFlowList(item, block, blockIndex, 'addresses')"
                      :key="address.id"
                      class="flow-address-card is-selectable"
                      :class="{ 'flow-reveal-item': item.flowReveal }"
                      @tap="handleFlowAction(address.action, `选择地址：${address.receiver || ''}`)"
                    >
                      <view class="flow-address-check"></view>
                      <view class="flow-address-main">
                        <view class="flow-line is-strong">
                          <text>{{ address.receiver }}</text>
                          <text>{{ address.contact }}</text>
                        </view>
                        <view class="flow-address-text">
                          {{ (address.address || []).join(' ') }} {{ address.detail }}
                        </view>
                      </view>
                      <view class="flow-address-select">选择</view>
                    </view>
                  </view>

                  <view v-else-if="block.type === 'address_form'" class="flow-form">
                    <view class="flow-form-hint">可直接发送完整收货信息，我会先填入表单。</view>
                    <input
                      v-model="block.form.receiver"
                      class="flow-input"
                      placeholder="收货人"
                      placeholder-class="flow-placeholder"
                    />
                    <input
                      v-model="block.form.contact"
                      class="flow-input"
                      placeholder="手机号"
                      placeholder-class="flow-placeholder"
                    />
                    <!-- #ifdef MP-WEIXIN -->
                    <picker
                      class="flow-picker"
                      mode="region"
                      :value="block.form.address_name"
                      @change="onAddressRegionChange(block, $event)"
                    >
                      <view v-if="block.form.address_name?.length" class="flow-picker-text">
                        {{ block.form.address_name.join('-') }}
                      </view>
                      <view v-else class="flow-placeholder">请选择省/市/区</view>
                    </picker>
                    <!-- #endif -->
                    <!-- #ifdef H5 || APP-PLUS -->
                    <view class="flow-picker is-data-picker">
                      <uni-data-picker
                        v-model="block.form.address"
                        :localdata="addressAreaTree"
                        placeholder="请选择省/市/区"
                        popup-title="请选择城市"
                        :clear-icon="false"
                        @change="onAddressCityChange(block, $event)"
                      />
                    </view>
                    <!-- #endif -->
                    <input
                      v-model="block.form.detail"
                      class="flow-input"
                      placeholder="详细地址"
                      placeholder-class="flow-placeholder"
                    />
                    <button
                      class="flow-primary-button is-wide"
                      hover-class="none"
                      @tap="submitAddressForm(block)"
                    >
                      保存地址
                    </button>
                  </view>

                  <view
                    v-else-if="block.type === 'selected_address'"
                    class="flow-address-card is-static is-selected"
                  >
                    <view class="flow-address-check is-active"></view>
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
                      @tap="handleFlowAction(block.action, '确认下单')"
                    >
                      确认下单
                    </button>
                  </view>

                  <view v-else-if="block.type === 'payment_panel'" class="flow-action-panel">
                    <view class="flow-desc">订单号：{{ block.order_id }}</view>
                    <button
                      class="flow-primary-button is-wide"
                      hover-class="none"
                      @tap="handleFlowAction(block.action, '发起支付')"
                    >
                      发起支付
                    </button>
                  </view>

                  <view v-else-if="block.type === 'payment_result'" class="flow-action-panel">
                    <view class="flow-desc">支付已发起，请按系统提示完成支付。</view>
                  </view>

                  <view v-else-if="block.type === 'order_list'" class="flow-order-list">
                    <view v-if="!block.orders?.length" class="flow-empty">暂时没有相关订单</view>
                    <view
                      v-for="order in visibleFlowList(item, block, blockIndex, 'orders')"
                      :key="order.id"
                      class="flow-order-card"
                      :class="{ 'flow-reveal-item': item.flowReveal }"
                    >
                      <view class="flow-line is-strong">
                        <text>订单 {{ order.order_no || order.id }}</text>
                        <text>¥{{ formatPrice(Number(order.pay_money || 0)) }}</text>
                      </view>
                      <view
                        v-for="goods in order.goods || []"
                        :key="goods.sku_code"
                        class="flow-line"
                      >
                        <text class="flow-line-main">{{ goods.name }}</text>
                        <text class="flow-line-sub">x{{ goods.num }}</text>
                      </view>
                      <button
                        class="flow-primary-button is-wide"
                        hover-class="none"
                        @tap="handleFlowAction(order.action, resolveFlowActionLabel(order.action))"
                      >
                        {{ order.action?.type === 'start_payment' ? '继续支付' : '查看详情' }}
                      </button>
                    </view>
                  </view>

                  <view v-else-if="block.type === 'pending_review_list'" class="flow-goods-list">
                    <view v-if="!block.goods?.length" class="flow-empty">暂时没有待评价商品</view>
                    <view
                      v-for="goods in visibleFlowList(item, block, blockIndex, 'goods')"
                      :key="`${goods.order_id}:${goods.goods_id}:${goods.sku_code}`"
                      class="flow-goods-card"
                      :class="{ 'flow-reveal-item': item.flowReveal }"
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
                        @tap="handleFlowAction(goods.action, `评价：${goods.goods_name || ''}`)"
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
                    <view
                      v-for="score in [
                        ['goods_score', '商品'],
                        ['package_score', '包装'],
                        ['delivery_score', '配送'],
                      ]"
                      :key="score[0]"
                      class="flow-score-row"
                    >
                      <text>{{ score[1] }}</text>
                      <view class="flow-stepper">
                        <button
                          class="flow-stepper-button"
                          hover-class="none"
                          @tap="changeReviewScore(block.form, score[0], -1)"
                        >
                          -
                        </button>
                        <text class="flow-stepper-value">{{ block.form[score[0]] }}</text>
                        <button
                          class="flow-stepper-button"
                          hover-class="none"
                          @tap="changeReviewScore(block.form, score[0], 1)"
                        >
                          +
                        </button>
                      </view>
                    </view>
                    <button
                      class="flow-primary-button is-wide"
                      hover-class="none"
                      @tap="submitReviewForm(block)"
                    >
                      提交评价
                    </button>
                  </view>

                  <view v-else-if="block.type === 'order_logistics'" class="flow-logistics">
                    <view class="flow-line is-strong">
                      <text>订单 {{ block.order?.order_no || block.order?.id }}</text>
                      <text>¥{{ formatPrice(Number(block.order?.pay_money || 0)) }}</text>
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
                      @tap="handleFlowAction(block.action, '确认收货')"
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

    <view class="composer" :style="{ paddingBottom: composerBottom }">
      <view class="composer-main">
        <button class="attach-button" hover-class="none" @tap="handleAttachment">
          <uni-icons type="plusempty" size="30" color="#111" />
        </button>
        <view class="composer-card">
          <view v-if="selectedAttachments.length" class="composer-attachments">
            <view
              v-for="attachment in selectedAttachments"
              :key="attachment.id || attachment.url || attachment.name"
              class="composer-attachment"
              @tap="removeSelectedAttachment(attachment)"
            >
              {{ attachment.name }} ×
            </view>
          </view>
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
            :class="{ 'is-disabled': isSubmitDisabled, 'is-sending': currentSessionSending }"
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
        <view v-if="loadingSessions" class="session-empty">正在加载会话...</view>
        <view
          v-for="session in filteredSessions"
          :key="session.id"
          class="session-item"
          :class="{ 'is-active': session.id === activeSessionID }"
          @tap="selectSession(session.id)"
          @longpress="openSessionActionSheet(session, $event)"
        >
          <view class="session-content">
            <view class="session-row">
              <view class="session-title">{{ session.title }}</view>
              <view class="session-time">{{ formatSessionTime(session) }}</view>
            </view>
            <view class="session-summary">{{ session.summary || '暂无摘要' }}</view>
          </view>
          <button
            class="session-more"
            hover-class="none"
            @tap.stop="openSessionActionSheet(session, $event)"
          >
            <view></view>
            <view></view>
            <view></view>
          </button>
        </view>
        <view v-if="!loadingSessions && !filteredSessions.length" class="session-empty"
          >没有匹配的会话</view
        >
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

.history-button,
.nav-back-button,
.attach-button,
.voice-button,
.send-button,
.session-create,
.session-more,
.operation-item,
.message-edit-button,
.rename-button,
.text-preview-close,
.starter-prompt {
  padding: 0;
  margin: 0;
  border-radius: 0;
  background: transparent;
  line-height: normal;

  &::after {
    border: none;
  }
}

.assistant-navbar {
  position: relative;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  min-height: 88rpx;
  padding-right: 20rpx;
  padding-left: 20rpx;
  border-bottom: 1rpx solid #f0f0f0;
  background-color: #fff;
  box-sizing: border-box;
}

.assistant-navbar__left,
.assistant-navbar__right {
  z-index: 1;
  display: flex;
  align-items: center;
  min-width: 230rpx;
}

.assistant-navbar__left {
  justify-content: flex-start;
  gap: 8rpx;
}

.assistant-navbar__right {
  justify-content: flex-end;
}

.assistant-navbar__title {
  position: absolute;
  left: 230rpx;
  right: 230rpx;
  bottom: 0;
  height: 88rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #111;
  font-size: 32rpx;
  font-weight: 600;
  line-height: 88rpx;
  text-align: center;
}

.nav-back-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56rpx;
  height: 56rpx;
}

.nav-back-icon {
  width: 18rpx;
  height: 18rpx;
  border-bottom: 3rpx solid #111;
  border-left: 3rpx solid #111;
  transform: rotate(45deg);
}

.history-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56rpx;
  height: 56rpx;
  border-radius: 8rpx;
  color: #27ba9b;
  font-size: 24rpx;
  line-height: 56rpx;
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

.assistant-session-button {
  background-color: #fff;
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

.starter-prompts {
  display: flex;
  justify-content: center;
  margin-top: 30rpx;
}

.starter-prompt {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10rpx;
  max-width: 100%;
  min-height: 64rpx;
  padding: 0 24rpx;
  border: 1rpx solid #d9f1ec;
  border-radius: 8rpx;
  color: #16806d;
  font-size: 26rpx;
  line-height: 36rpx;
  background-color: #f2fbf8;
  box-sizing: border-box;
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
  overflow: visible;
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

.flow-block-list {
  margin-top: 20rpx;
}

.flow-block {
  padding: 18rpx;
  border-radius: 10rpx;
  background-color: #f6f7f9;
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
.flow-order-card {
  display: flex;
  align-items: center;
  gap: 16rpx;
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
.flow-order-card + .flow-order-card,
.flow-sku-row + .flow-sku-row {
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

.flow-address-check {
  flex-shrink: 0;
  width: 30rpx;
  height: 30rpx;
  border: 3rpx solid #27ba9b;
  border-radius: 50%;
  box-sizing: border-box;
  background-color: #fff;
}

.flow-address-check.is-active {
  border-width: 8rpx;
  background-color: #27ba9b;
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
.flow-sku-info {
  flex: 1;
  min-width: 0;
}

.flow-goods-name,
.flow-sku-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #333;
  font-size: 25rpx;
  font-weight: 600;
  line-height: 34rpx;
}

.flow-goods-desc,
.flow-address-text {
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
}

.flow-line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
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

.flow-picker {
  min-height: 68rpx;
}

.flow-picker.is-data-picker {
  padding: 0;
  background-color: transparent;
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
  flex-wrap: wrap;
  gap: 12rpx;
  min-height: 92rpx;
  padding: 10rpx 12rpx 10rpx 28rpx;
  border-radius: 46rpx;
  background-color: #fff;
  box-shadow: 0 10rpx 28rpx rgba(15, 23, 42, 0.08);
  box-sizing: border-box;
}

.composer-attachments {
  display: flex;
  flex-wrap: wrap;
  gap: 8rpx;
  width: 100%;
  padding-top: 4rpx;
}

.composer-attachment {
  max-width: 240rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding: 6rpx 12rpx;
  border-radius: 8rpx;
  color: #16806d;
  font-size: 20rpx;
  line-height: 28rpx;
  background-color: #e8f8f4;
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
  min-height: 46rpx;
  max-height: 138rpx;
  padding: 15rpx 0;
  box-sizing: border-box;
  color: #333;
  font-size: 32rpx;
  line-height: 46rpx;
  overflow-y: auto;
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
  right: 0;
  bottom: 0;
  left: 560rpx;
  z-index: 20;
  background-color: rgba(0, 0, 0, 0.18);
}

.session-drawer {
  display: flex;
  flex-direction: column;
  position: absolute;
  top: 0;
  left: 0;
  bottom: 0;
  z-index: 21;
  width: 560rpx;
  padding: 0 24rpx 36rpx;
  background-color: #fff;
  box-shadow: none;
  box-sizing: border-box;
  transform: translateX(-100%);
  transition: transform 0.2s ease;
}

.session-drawer.is-open {
  box-shadow: 24rpx 0 60rpx rgba(0, 0, 0, 0.12);
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
  padding: 22rpx 16rpx 22rpx 22rpx;
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

.session-row {
  display: flex;
  align-items: center;
  gap: 14rpx;
}

.session-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #333;
  font-size: 26rpx;
  font-weight: 600;
  line-height: 34rpx;
}

.session-time {
  flex-shrink: 0;
  color: #a6abb3;
  font-size: 20rpx;
  line-height: 30rpx;
}

.session-summary {
  margin-top: 10rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #777;
  font-size: 22rpx;
  line-height: 32rpx;
}

.session-more {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 5rpx;
  width: 56rpx;
  height: 56rpx;
  border-radius: 50%;
  background-color: transparent;
}

.session-more view {
  width: 6rpx;
  height: 6rpx;
  border-radius: 50%;
  background-color: #9ca3af;
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
