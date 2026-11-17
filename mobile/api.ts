import * as SecureStore from 'expo-secure-store';
import { Platform } from 'react-native';

const SERVER_KEY = 'api_base_url';
const defaultBaseUrl = (process.env.EXPO_PUBLIC_API_URL || (Platform.OS === 'android' ? 'http://10.0.2.2:8080' : 'http://localhost:8080')).replace(/\/+$/, '');

// 可运行时修改：局域网 IP、公网隧道或正式域名；修改后写入 SecureStore，重启仍生效。
export let apiBaseUrl = defaultBaseUrl;
let refreshPromise: Promise<string | null> | null = null;

export function normalizeServer(value: string): string | null {
  const trimmed = value.trim();
  if (!trimmed) return null;
  const withScheme = /^https?:\/\//i.test(trimmed) ? trimmed : `http://${trimmed}`;
  try {
    const parsed = new URL(withScheme);
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') return null;
    if (!parsed.hostname) return null;
    const path = parsed.pathname.replace(/\/+$/, '');
    return `${parsed.protocol}//${parsed.host}${path}`;
  } catch {
    return null;
  }
}

// 带超时的 fetch：地址不可达时 SYN 可能被静默丢弃，没有超时会让界面无限等待。
export async function fetchWithTimeout(url: string, init: RequestInit = {}, timeoutMs = 8000): Promise<Response> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    return await fetch(url, { ...init, signal: controller.signal });
  } finally {
    clearTimeout(timer);
  }
}

export function isTimeoutError(error: unknown): boolean {
  return !!error && typeof error === 'object' && (error as { name?: string }).name === 'AbortError';
}

export async function initApiBase(): Promise<void> {
  try {
    const saved = await SecureStore.getItemAsync(SERVER_KEY);
    const normalized = saved ? normalizeServer(saved) : null;
    if (normalized) apiBaseUrl = normalized;
  } catch {
    // SecureStore 不可用时沿用打包时的默认地址。
  }
}

export async function setApiBase(value: string): Promise<string | null> {
  const normalized = normalizeServer(value);
  if (!normalized) return null;
  apiBaseUrl = normalized;
  try {
    await SecureStore.setItemAsync(SERVER_KEY, normalized);
  } catch {
    // 写入失败不影响本次会话使用。
  }
  return normalized;
}

export async function resetApiBase(): Promise<void> {
  apiBaseUrl = defaultBaseUrl;
  try {
    await SecureStore.deleteItemAsync(SERVER_KEY);
  } catch {
    // 忽略。
  }
}

async function renewSession(): Promise<string | null> {
  if (refreshPromise) return refreshPromise;
  refreshPromise = (async () => {
    try {
      const refresh = await SecureStore.getItemAsync('refresh_token');
      if (!refresh) return null;
      const response = await fetchWithTimeout(`${apiBaseUrl}/api/v1/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refresh }),
      });
      if (!response.ok) {
        if (response.status === 401 || response.status === 403) {
          await SecureStore.deleteItemAsync('access_token');
          await SecureStore.deleteItemAsync('refresh_token');
        }
        return null;
      }
      const session = (await response.json()) as { access_token: string; refresh_token: string };
      await SecureStore.setItemAsync('access_token', session.access_token);
      await SecureStore.setItemAsync('refresh_token', session.refresh_token);
      return session.access_token;
    } catch {
      return null;
    } finally {
      refreshPromise = null;
    }
  })();
  return refreshPromise;
}

export async function apiFetch(path: string, initialToken: string, init: RequestInit = {}): Promise<Response> {
  const token = (await SecureStore.getItemAsync('access_token')) || initialToken;
  const headers = new Headers(init.headers);
  headers.set('Authorization', `Bearer ${token}`);
  const timeout = init.body instanceof FormData ? 45_000 : 15_000;
  const response = await fetchWithTimeout(`${apiBaseUrl}${path}`, { ...init, headers }, timeout);
  if (response.status !== 401) return response;
  const renewed = await renewSession();
  if (!renewed) return response;
  const retryHeaders = new Headers(init.headers);
  retryHeaders.set('Authorization', `Bearer ${renewed}`);
  return fetchWithTimeout(`${apiBaseUrl}${path}`, { ...init, headers: retryHeaders }, timeout);
}
