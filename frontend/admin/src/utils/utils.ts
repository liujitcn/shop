/**
 * JSON 格式化显示。
 */
export function formatJson(str: string) {
  try {
    return JSON.stringify(JSON.parse(str), null, 2);
  } catch {
    return str;
  }
}

/**
 * 将后端分单位金额转换为元字符串。
 */
export function formatPrice(price?: number) {
  if (!price) return "0.00";
  return (price / 100).toFixed(2);
}

/**
 * 按静态资源域名补齐图片地址。
 */
export function formatSrc(src: string) {
  if (!src) return src;
  if (!src.startsWith("http") && !src.startsWith("https")) {
    return import.meta.env.VITE_APP_STATIC_URL + src;
  }
  return src;
}

/** 解析商品 H5 预览所需的站点根地址，开发环境优先复用代理目标。 */
function resolveGoodsH5Origin() {
  const apiUrl = import.meta.env.VITE_API_URL || import.meta.env.VITE_APP_API_URL || "";
  if (/^https?:\/\//.test(apiUrl)) {
    return apiUrl.replace(/\/api\/?$/, "");
  }

  const proxyConfig = import.meta.env.VITE_PROXY;
  const proxyList =
    typeof proxyConfig === "string"
      ? (() => {
          try {
            return JSON.parse(proxyConfig);
          } catch {
            return [];
          }
        })()
      : proxyConfig;

  if (Array.isArray(proxyList)) {
    const matchedProxy = proxyList.find(item => Array.isArray(item) && ["/shop", "/api"].includes(item[0])) ?? proxyList[0];
    if (Array.isArray(matchedProxy) && typeof matchedProxy[1] === "string" && /^https?:\/\//.test(matchedProxy[1])) {
      return matchedProxy[1].replace(/\/$/, "");
    }
  }

  return window.location.origin;
}

/** 按商品ID生成商城 H5 商品详情预览地址。 */
export function buildGoodsH5PreviewUrl(goodsId?: string | number) {
  const id = String(goodsId ?? "").trim();
  if (!id) return "";
  return `${resolveGoodsH5Origin()}/app#/pages/goods/goods?id=${encodeURIComponent(id)}`;
}
