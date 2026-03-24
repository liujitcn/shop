// 商城服务
export interface ShopService {
  /** 主键id */
  id: number
  /** 标签 */
  label: string
  /** 值 */
  value: string
  /** 排序 */
  sort: number
  /** 状态 */
  status: number
  /** 创建时间 */
  createdAt: string
  /** 更新时间 */
  updatedAt: string
}

// 商城服务表单
export interface ShopServiceForm {
  /** 主键id */
  id?: number
  /** 标签 */
  label: string
  /** 值 */
  value: string
  /** 排序 */
  sort: number
  /** 状态 */
  status: number
} 