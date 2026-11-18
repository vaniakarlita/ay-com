import { writable } from 'svelte/store';

export const googleDraft = writable<{ name: string; email: string } | null>(null);