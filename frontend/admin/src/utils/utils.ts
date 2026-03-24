/**
 * JSON 格式化显示
 * @param str
 */
export const formatJson = (str: string) => {
  try {
    return JSON.stringify(JSON.parse(str), null, 2);
  } catch {
    return str;
  }
};

/**
 * 金额格式化函数
 * @param price 金额
 */
export const formatPrice = (price: number | undefined) => {
  if (!price) {
    return "0.00";
  }
  return (price / 100).toFixed(2);
};

/**
 * 图片地址格式化函数
 * @param src 图片地址
 */
export const formatSrc = (src: string) => {
  if (!src) {
    return src;
  }
  if (!src.startsWith("http") && !src.startsWith("https")) {
    return import.meta.env.VITE_APP_STATIC_URL + src;
  } else {
    return src;
  }
};
