<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { computed, onBeforeUnmount, ref } from 'vue'
import { defAiAssistantMessageService } from '@/api/base/ai_assistant_message'
import { defAiAssistantSessionService } from '@/api/base/ai_assistant_session'
import type {
  AiAssistantAttachment,
  AiAssistantMessage,
  AiAssistantSession,
  AiAssistantTool,
} from '@/rpc/base/v1/ai_assistant_session'
import { AiAssistantMessageStatus, Terminal } from '@/rpc/common/v1/enum'
import { uploadFileList } from '@/utils/file'
import { formatSrc } from '@/utils/index'
import {
  type AiAssistantStreamEvent,
  type AiAssistantStreamPayload,
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
  tokenTotal: number
  firstTokenMs: number
  durationMs: number
  localOnly?: boolean
  streamKey?: string
  speaking?: boolean
}

type SubmitPayload = {
  text: string
  attachments: AiAssistantAttachment[]
}

type StreamTask = {
  controller: AbortController
  finished: boolean
}

const THINKING_MESSAGE_CONTENT = '正在整理回复'
const LOCAL_USER_MESSAGE_PREFIX = 'assistant-user-local'
const PENDING_MESSAGE_ID = 'pending'
const MAX_ATTACHMENT_COUNT = 6
const AI_ASSISTANT_TERMINAL = Terminal.TERMINAL_APP

const systemInfo = uni.getSystemInfoSync()
const { safeAreaInsets } = systemInfo
const composerBottom = `${Math.max(safeAreaInsets?.bottom || 0, 9)}px`
const drawerTopPadding = `${(safeAreaInsets?.top || 0) + 12}px`
const windowWidth = systemInfo.windowWidth || systemInfo.screenWidth || 375
const windowHeight = systemInfo.windowHeight || systemInfo.screenHeight || 667
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
const uploadingAttachment = ref(false)
const sendingSessionMap = ref<Record<string, boolean>>({})
const sessions = ref<AiAssistantSession[]>([])
const messages = ref<Record<string, ChatMessageItem[]>>({})
const selectedAttachments = ref<AiAssistantAttachment[]>([])
const runningStreamTaskMap = new Map<string, StreamTask>()
const pendingDeltaMap = new Map<string, AiAssistantStreamPayload>()
let speakingMessageKey = ''
let pendingDeltaTimer = 0
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
  clearPendingDelta()
})

/** 打开或收起历史会话抽屉。 */
const toggleSessionDrawer = () => {
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
  }
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

/** 上传图片附件，发送时随消息一起提交。 */
const handleAttachment = async () => {
  if (uploadingAttachment.value || currentSessionSending.value) {
    return
  }
  if (selectedAttachments.value.length >= MAX_ATTACHMENT_COUNT) {
    uni.showToast({ icon: 'none', title: `最多上传 ${MAX_ATTACHMENT_COUNT} 个附件` })
    return
  }

  try {
    const result = await uni.chooseImage({
      count: MAX_ATTACHMENT_COUNT - selectedAttachments.value.length,
      sizeType: ['compressed'],
      sourceType: ['album', 'camera'],
    })
    const filePaths = Array.isArray(result.tempFilePaths)
      ? result.tempFilePaths
      : [result.tempFilePaths].filter(Boolean)
    if (!filePaths.length) {
      return
    }
    uploadingAttachment.value = true
    const files = await uploadFileList('assistant', filePaths)
    const nextAttachments = files.map<AiAssistantAttachment>((file, index) => ({
      id: file.url || `${file.name}-${index}`,
      name: file.name,
      size: 0,
      url: file.url,
      mime_type: resolveMimeType(file.name),
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
  const payload = {
    text,
    attachments: [...selectedAttachments.value],
  }
  inputText.value = ''
  selectedAttachments.value = []
  await sendAiAssistantPayload(payload)
}

const formatTools = (tools: AiAssistantTool[]) => {
  return tools.map((item) => item.title || item.name).join(' · ')
}

const formatRuntime = (item: ChatMessageItem) => {
  const duration = item.durationMs ? `${(item.durationMs / 1000).toFixed(1)}s` : '生成中'
  return `${item.tokenTotal} Token · 首字 ${item.firstTokenMs}ms · 总耗时 ${duration}`
}

/** 预览消息附件图片，二期附件统一按图片附件处理。 */
const previewAttachment = (
  attachment: AiAssistantAttachment,
  attachments: AiAssistantAttachment[],
) => {
  const urls = attachments.map((item) => formatSrc(item.url)).filter(Boolean)
  const current = formatSrc(attachment.url)
  if (!current || !urls.length) {
    uni.showToast({ icon: 'none', title: '附件地址为空' })
    return
  }
  uni.previewImage({
    current,
    urls,
  })
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
  setSessionSending(sessionID, true)
  await runAiAssistantTask(sessionID, payload)
  return true
}

/** 后台执行 AI 助手消息请求。 */
async function runAiAssistantTask(sessionID: string, payload: SubmitPayload) {
  let task: StreamTask | undefined
  try {
    const canStream =
      typeof fetch === 'function' &&
      typeof ReadableStream !== 'undefined' &&
      typeof AbortController !== 'undefined'
    if (!canStream) {
      const response = await defAiAssistantMessageService.SendAiAssistantMessage({
        session_id: sessionID,
        content: payload.text,
        attachments: payload.attachments,
      })
      messages.value[sessionID] = replacePendingMessages(
        messages.value[sessionID] ?? [],
        normalizeMessageList(response.messages),
      )
      if (response.session) {
        upsertSession(normalizeSession(response.session))
      }
      return
    }

    const controller = new AbortController()
    task = { controller, finished: false }
    runningStreamTaskMap.set(sessionID, task)
    const response = await defAiAssistantMessageService.StreamAiAssistantMessage(
      {
        session_id: sessionID,
        content: payload.text,
        attachments: payload.attachments,
      },
      { signal: controller.signal },
    )
    if (!response.body) {
      throw new Error('AI 助手流式响应为空')
    }
    await readAiAssistantEventStream(
      response.body,
      (event) => handleAiAssistantStreamEvent(event, task),
      controller.signal,
    )
    if (!task.finished && !controller.signal.aborted) {
      throw new Error('AI 助手流式响应未完整返回')
    }
  } catch (error) {
    if (task?.controller.signal.aborted) {
      return
    }
    messages.value[sessionID] = markThinkingMessageFailed(messages.value[sessionID] ?? [], {
      sessionID,
    })
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
  if (payload.session) {
    upsertSession(normalizeSession(payload.session))
  }
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
    return
  }
  messages.value[sessionID] = markStreamingError(
    ensureStreamingMessage(messages.value[sessionID] ?? [], payload),
    payload,
  )
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
  task.controller.abort()
  runningStreamTaskMap.delete(sessionID)
  setSessionSending(sessionID, false)
}

function cancelAllStreamTasks() {
  Array.from(runningStreamTaskMap.keys()).forEach((sessionID) => cancelSessionStreamTask(sessionID))
}

function markAllMessagesSpeaking(messageKey?: string) {
  Object.keys(messages.value).forEach((sessionID) => {
    messages.value[sessionID] = (messages.value[sessionID] ?? []).map((item) => ({
      ...item,
      speaking: Boolean(messageKey && item.key === messageKey),
    }))
  })
}

function resolveMimeType(name: string) {
  const extension = name.split('.').pop()?.toLowerCase() ?? ''
  if (['jpg', 'jpeg'].includes(extension)) {
    return 'image/jpeg'
  }
  if (['png', 'gif', 'webp'].includes(extension)) {
    return `image/${extension}`
  }
  return ''
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

              <text class="bubble-content">{{ item.content }}</text>

              <view v-if="item.attachments.length" class="attachment-list">
                <view
                  v-for="attachment in item.attachments"
                  :key="attachment.id || attachment.url || attachment.name"
                  class="attachment-card"
                  @tap="previewAttachment(attachment, item.attachments)"
                >
                  <view class="attachment-icon">📎</view>
                  <view class="attachment-info">
                    <view class="attachment-name">{{ attachment.name }}</view>
                    <view class="attachment-meta">{{ attachment.mime_type || '附件' }}</view>
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
            <view class="session-title">{{ session.title }}</view>
            <view class="session-summary">{{ session.summary }}</view>
          </view>
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
</style>
