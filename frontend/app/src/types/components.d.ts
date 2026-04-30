import XtxSwiper from '@/components/XtxSwiper.vue'
import XtxGuess from '@/components/XtxGuess.vue'
import XtxEmptyState from '@/components/XtxEmptyState.vue'

declare module 'vue' {
  export interface GlobalComponents {
    XtxSwiper: typeof XtxSwiper
    XtxGuess: typeof XtxGuess
    XtxEmptyState: typeof XtxEmptyState
  }
}

// 组件实例类型
export type XtxGuessInstance = InstanceType<typeof XtxGuess>
export type XtxSwiperInstance = InstanceType<typeof XtxSwiper>
