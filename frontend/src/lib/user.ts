import { writable } from 'svelte/store';
import { authApi, type CurrentUser } from './api';

export const currentUser = writable<CurrentUser | null>(null);

export async function loadCurrentUser() {
  try {
    currentUser.set(await authApi.me());
  } catch {
    currentUser.set(null);
  }
}