import ShopSwiper from '@/components/ShopSwiper.vue'
import GoodsGuess from '@/components/GoodsGuess.vue'
import EmptyState from '@/components/EmptyState.vue'
import GoodsHero from '@/components/GoodsHero.vue'
import GoodsActionBar from '@/components/GoodsActionBar.vue'

declare module 'vue' {
  export interface GlobalComponents {
    ShopSwiper: typeof ShopSwiper
    GoodsGuess: typeof GoodsGuess
    EmptyState: typeof EmptyState
    GoodsHero: typeof GoodsHero
    GoodsActionBar: typeof GoodsActionBar
  }
}

// 组件实例类型
export type GoodsGuessInstance = InstanceType<typeof GoodsGuess>
export type ShopSwiperInstance = InstanceType<typeof ShopSwiper>
