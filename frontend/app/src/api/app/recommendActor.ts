import { http } from '@/utils/http'
import type { Empty } from '@/rpc/google/protobuf/empty'

export const RECOMMEND_ANONYMOUS_ACTOR_KEY = 'recommend_anonymous_actor'
export const RECOMMEND_ANONYMOUS_ID_HEADER = 'X-Recommend-Anonymous-Id'

interface Int64Value {
  value?: number
}

export const getCachedRecommendAnonymousId = (): number => {
  const cachedActor = uni.getStorageSync(RECOMMEND_ANONYMOUS_ACTOR_KEY) as Int64Value | undefined
  return cachedActor?.value || 0
}

export const buildRecommendAnonymousHeader = (anonymousId?: number): Record<string, string> => {
  const currentAnonymousId = anonymousId || 0
  if (!currentAnonymousId) {
    return {}
  }
  return {
    [RECOMMEND_ANONYMOUS_ID_HEADER]: String(currentAnonymousId),
  }
}

export const bindRecommendAnonymousActor = async (): Promise<void> => {
  const anonymousId = getCachedRecommendAnonymousId()
  if (!anonymousId) {
    return
  }

  await http<Empty>({
    url: '/app/recommend/actor/bind',
    method: 'POST',
    header: buildRecommendAnonymousHeader(anonymousId),
    data: {},
  })
}
