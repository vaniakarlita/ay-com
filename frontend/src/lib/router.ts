import { writable } from 'svelte/store';

function getPath(): string {
  return window.location.pathname;
}

export const path = writable<string>(getPath());

export function navigate(to: string, replace = false) {
  if (to === window.location.pathname) return;
  if (replace) {
    window.history.replaceState({}, '', to);
  } else {
    window.history.pushState({}, '', to);
  }
  path.set(to);
}

window.addEventListener('popstate', () => path.set(getPath()));