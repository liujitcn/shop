function getCurrentRouteUrl() {
  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1]
  if (!currentPage?.route) {
    return ''
  }

  let params: Record<string, string> = {}
  const miniPage = currentPage as { options?: Record<string, string> }
  const routePage = currentPage as { $vm?: { $route?: { query?: Record<string, string> } } }

  // 微信小程序通过页面实例 options 读取参数。
  // #ifdef MP-WEIXIN
  params = miniPage.options || {}
  // #endif

  // H5 和 APP 通过 Vue 路由读取参数。
  // #ifdef H5 || APP-PLUS
  if (routePage.$vm?.$route) {
    params = routePage.$vm.$route.query || {}
  }
  // #endif

  const query = Object.keys(params)
    .map((key) => `${key}=${encodeURIComponent(params[key])}`)
    .join('&')

  return query ? `/${currentPage.route}?${query}` : `/${currentPage.route}`
}

export function saveCurrentRoute() {
  const currentRoute = getCurrentRouteUrl()
  if (!currentRoute || currentRoute.startsWith('/pages/login/login')) {
    uni.removeStorageSync('lastRoute')
    return
  }
  uni.setStorageSync('lastRoute', currentRoute)
}

export function navigateToLogin() {
  saveCurrentRoute()

  uni.navigateTo({
    url: '/pages/login/login',
    fail: () => {
      uni.reLaunch({
        url: '/pages/login/login',
      })
    },
  })
}
