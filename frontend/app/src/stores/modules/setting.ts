import { defConfigService } from '@/api/base/config'
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { BaseConfigSite } from '@/rpc/common/enum'

export const useSettingStore = defineStore('setting', () => {
  const data = ref<Map<string, string>>()

  const getData = (key: string): string | undefined => {
    return data.value?.get(key)
  }

  const loadData = async () => {
    const res = await defConfigService.GetConfig({
      site: BaseConfigSite.APP,
    })
    data.value = new Map<string, string>()
    res.data?.forEach((item) => {
      data.value?.set(item.key, item.value)
    })
  }

  return {
    getData,
    loadData,
  }
})
