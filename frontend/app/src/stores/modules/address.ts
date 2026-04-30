import type { UserAddress } from '@/rpc/app/v1/user_address'
import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useAddressStore = defineStore('address', () => {
  const selectedAddress = ref<UserAddress>()

  const changeSelectedAddress = (val: UserAddress) => {
    selectedAddress.value = val
  }

  return {
    selectedAddress,
    changeSelectedAddress,
  }
})
