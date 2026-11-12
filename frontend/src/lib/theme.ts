import { writable } from 'svelte/store';

export type Theme = 'light' | 'dark';

function getInitialTheme(): Theme {
  try {
    return localStorage.getItem('theme') === 'dark' ? 'dark' : 'light';
  } catch {
    return 'light';
  }
}

export const theme = writable<Theme>(getInitialTheme());

theme.subscribe((value) => {
  document.documentElement.dataset.theme = value;
  try {
    localStorage.setItem('theme', value);
  } catch {
  }
});

export function toggleTheme() {
  theme.update((t) => (t === 'light' ? 'dark' : 'light'));
}