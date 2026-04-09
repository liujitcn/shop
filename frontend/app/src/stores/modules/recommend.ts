import { defRecommendService } from '@/api/app/recommend'
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useUserStore } from './user'

const RECOMMEND_ANONYMOUS_ID_HEADER = 'X-Recommend-Anonymous-Id'

/** 推荐匿名主体 store，负责缓存匿名 ID 并为 API 组装请求头。 */
export const useRecommendStore = defineStore(
  'recommend',
  () => {
    // 未登录用户的推荐匿名标识，持久化后可跨页面复用。
    const anonymousId = ref(0)

    /** 获取匿名推荐主体，已登录用户直接返回 0 表示不使用匿名身份。 */
    const getAnonymousId = async (): Promise<number> => {
      const userStore = useUserStore()
      if (userStore.userInfo) {
        return 0
      }

      if (anonymousId.value) {
        return anonymousId.value
      }

      const actor = await defRecommendService.RecommendAnonymousActor({})
      anonymousId.value = actor.value || 0
      return anonymousId.value
    }

    /** 登录成功后把匿名推荐主体绑定到当前用户。 */
    const bindAnonymousActor = async (): Promise<void> => {
      if (!anonymousId.value) {
        return
      }

      await defRecommendService.BindRecommendAnonymousActor({})
    }

    /** 统一生成推荐请求头，避免业务侧重复拼接 header。 */
    const buildAnonymousHeader = (): Record<string, string> => {
      const userStore = useUserStore()
      if (userStore.userInfo) {
        return {}
      }

      const currentAnonymousId = anonymousId.value
      if (!currentAnonymousId) {
        return {}
      }

      return {
        [RECOMMEND_ANONYMOUS_ID_HEADER]: String(currentAnonymousId),
      }
    }

    return {
      anonymousId,
      getAnonymousId,
      bindAnonymousActor,
      buildAnonymousHeader,
    }
  },
  {
    persist: {
      storage: {
        getItem(key) {
          return uni.getStorageSync(key)
        },
        setItem(key, value) {
          uni.setStorageSync(key, value)
        },
      },
    },
  },
)
