export function getToken(): string | null {
  return localStorage.getItem("token");
}

export function setToken(token: string) {
  localStorage.setItem("token", token);
}

export function removeToken() {
  localStorage.removeItem("token");
}

export function formatTime(time: string): string {
  if (!time) return "-";
  return new Date(time).toLocaleString("zh-CN");
}
