import { writable } from 'svelte/store';
import type { ThreadItem } from '../api';

export const isCreateModalOpen = writable<boolean>(false);
export const latestCreatedThread = writable<ThreadItem | null>(null);

export function openCreateModal() {
  isCreateModalOpen.set(true);
}

export function closeCreateModal() {
  isCreateModalOpen.set(false);
}

export function notifyThreadCreated(thread: ThreadItem) {
  latestCreatedThread.set(thread);
}
