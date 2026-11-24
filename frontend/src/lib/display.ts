import { writable } from 'svelte/store';

export type FontSize = 'small' | 'normal' | 'large';
export type FontColor = 'default' | 'warm' | 'cool' | 'contrast';

const FONT_SIZE_KEY = 'ay_font_size';
const FONT_COLOR_KEY = 'ay_font_color';

function getInitialFontSize(): FontSize {
  try {
    const saved = localStorage.getItem(FONT_SIZE_KEY);
    if (saved === 'small' || saved === 'normal' || saved === 'large') return saved;
  } catch {}
  return 'normal';
}

function getInitialFontColor(): FontColor {
  try {
    const saved = localStorage.getItem(FONT_COLOR_KEY);
    if (saved === 'default' || saved === 'warm' || saved === 'cool' || saved === 'contrast') return saved;
  } catch {}
  return 'default';
}

export const fontSize = writable<FontSize>(getInitialFontSize());
export const fontColor = writable<FontColor>(getInitialFontColor());

fontSize.subscribe((size) => {
  if (typeof document !== 'undefined') {
    document.documentElement.dataset.fontSize = size;
    try {
      localStorage.setItem(FONT_SIZE_KEY, size);
    } catch {}
  }
});

fontColor.subscribe((color) => {
  if (typeof document !== 'undefined') {
    document.documentElement.dataset.fontColor = color;
    try {
      localStorage.setItem(FONT_COLOR_KEY, color);
    } catch {}
  }
});
