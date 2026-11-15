import { writable } from 'svelte/store';

const ACCESS_KEY = 'accessToken';
const REFRESH_KEY = 'refreshToken';

function read(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function write(key: string, value: string | null) {
  try {
    if (value === null) localStorage.removeItem(key);
    else localStorage.setItem(key, value);
  } catch {
    return;
  }
}

export const tokens = {
  get access() {
    return read(ACCESS_KEY);
  },
  get refresh() {
    return read(REFRESH_KEY);
  },
  set(access: string, refresh: string) {
    write(ACCESS_KEY, access);
    write(REFRESH_KEY, refresh);
  },
  clear() {
    write(ACCESS_KEY, null);
    write(REFRESH_KEY, null);
  },
};

export const isLoggedIn = writable<boolean>(Boolean(tokens.refresh));

export function saveSession(access: string, refresh: string) {
  tokens.set(access, refresh);
  isLoggedIn.set(true);
}

export function clearSession() {
  tokens.clear();
  isLoggedIn.set(false);
}