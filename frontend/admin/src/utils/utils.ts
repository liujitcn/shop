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

/** 按商品ID生成商城 H5 商品详情预览相对地址，避免展示 IP 和端口。 */
export function buildGoodsH5PreviewUrl(goodsId?: string | number) {
  const id = String(goodsId ?? "").trim();
  if (!id) return "";
  return `/app#/pages/goods/goods?id=${encodeURIComponent(id)}`;
}
