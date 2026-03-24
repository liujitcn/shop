// 访问 token 缓存的 key
const ACCESS_TOKEN_KEY = "access_token";
// 访问 token 缓存的 key的有效期
const ACCESS_TOKEN_EXPIRES_IN = "expiresIn";
// 刷新 token 缓存的 key
const REFRESH_TOKEN_KEY = "refresh_token";

function getToken(): string {
  return localStorage.getItem(ACCESS_TOKEN_KEY) || "";
}

function setToken(token: string) {
  localStorage.setItem(ACCESS_TOKEN_KEY, token);
}

function getTokenExpiresIn(): number {
  const s = localStorage.getItem(ACCESS_TOKEN_EXPIRES_IN) || "";
  return Number(s);
}

function setTokenExpiresIn(expiresIn: number) {
  // 获取当前时间（毫秒）
  const d = new Date().getTime() + expiresIn * 1000;
  localStorage.setItem(ACCESS_TOKEN_EXPIRES_IN, String(d));
}

function getRefreshToken(): string {
  return localStorage.getItem(REFRESH_TOKEN_KEY) || "";
}

function setRefreshToken(token: string) {
  localStorage.setItem(REFRESH_TOKEN_KEY, token);
}

function clearToken() {
  localStorage.removeItem(ACCESS_TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
  localStorage.removeItem(ACCESS_TOKEN_EXPIRES_IN);
}

export {
  getToken,
  setToken,
  clearToken,
  getRefreshToken,
  setRefreshToken,
  setTokenExpiresIn,
  getTokenExpiresIn,
};
