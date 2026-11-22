import { clearSession, saveSession, tokens } from './session';

const BASE: string = import.meta.env.VITE_API_URL ?? 'http://localhost:8080';

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

type Options = {
  method?: string;
  body?: unknown;
  auth?: boolean;
};

let refreshing: Promise<boolean> | null = null;

async function refreshTokens(): Promise<boolean> {
  const refreshToken = tokens.refresh;
  if (!refreshToken) return false;
  try {
    const res = await fetch(`${BASE}/api/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refreshToken }),
    });
    if (!res.ok) return false;
    const data = await res.json();
    saveSession(data.accessToken, data.refreshToken);
    return true;
  } catch {
    return false;
  }
}

function call(path: string, { method = 'GET', body, auth = false }: Options): Promise<Response> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (auth && tokens.access) headers.Authorization = `Bearer ${tokens.access}`;
  return fetch(`${BASE}${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });
}

export async function request<T>(path: string, options: Options = {}): Promise<T> {
  let res: Response;
  try {
    res = await call(path, options);
  } catch {
    throw new ApiError(0, 'Cannot reach the server. Please try again.');
  }

  if (res.status === 401 && options.auth) {
    refreshing ??= refreshTokens().finally(() => {
      refreshing = null;
    });
    if (await refreshing) {
      try {
        res = await call(path, options);
      } catch {
        throw new ApiError(0, 'Cannot reach the server. Please try again.');
      }
    } else {
      clearSession();
    }
  }

  const data = await res.json().catch(() => null);
  if (!res.ok) {
    throw new ApiError(res.status, data?.error ?? 'Something went wrong. Please try again.');
  }
  return data as T;
}

export type RegisterPayload = {
  name: string;
  username: string;
  email: string;
  password: string;
  gender: string;
  dateOfBirth: string;
  profilePictureUrl: string;
  bannerUrl: string;
  securityQuestion: string;
  securityAnswer: string;
  newsletter: boolean;
};

type Tokens = { accessToken: string; refreshToken: string };
type Message = { message: string };

export type CurrentUser = {
  userId: string;
  name?: string;
  username?: string;
  profilePictureUrl?: string;
  isVerified?: boolean;
};

type GoogleResult = {
  registered: boolean;
  accessToken: string;
  refreshToken: string;
  email: string;
  name: string;
};

export const authApi = {
  register: (body: RegisterPayload) =>
    request<{ userId: string; message: string }>('/api/auth/register', { method: 'POST', body }),
  verify: (email: string, code: string) =>
    request<Message>('/api/auth/verify', { method: 'POST', body: { email, code } }),
  resendCode: (email: string) =>
    request<Message>('/api/auth/resend-code', { method: 'POST', body: { email } }),
  login: (email: string, password: string) =>
    request<Tokens>('/api/auth/login', { method: 'POST', body: { email, password } }),
  me: () => request<CurrentUser>('/api/me', { auth: true }),
  forgotQuestion: (email: string) =>
    request<{ question: string }>('/api/auth/forgot/question', { method: 'POST', body: { email } }),
  forgotVerify: (email: string, answer: string) =>
    request<{ resetToken: string }>('/api/auth/forgot/verify', { method: 'POST', body: { email, answer } }),
  forgotReset: (email: string, resetToken: string, newPassword: string) =>
    request<Message>('/api/auth/forgot/reset', { method: 'POST', body: { email, resetToken, newPassword } }),
  google: (idToken: string) =>
    request<GoogleResult>('/api/auth/google', { method: 'POST', body: { idToken } }),
  async logout() {
    const refreshToken = tokens.refresh;
    try {
      if (refreshToken) {
        await request<Message>('/api/auth/logout', { method: 'POST', body: { refreshToken } });
      }
    } finally {
      clearSession();
    }
  },
};

export type ThreadItem = {
  id: string;
  userId: string;
  authorName: string;
  authorUsername: string;
  authorAvatarUrl: string;
  authorIsVerified: boolean;
  content: string;
  replyPermission: string;
  categories: string[];
  communityId: string;
  communityName: string;
  isAd: boolean;
  scheduledAt: string;
  createdAt: string;
  likeCount: number;
  replyCount: number;
  repostCount: number;
  bookmarkCount: number;
  likedByMe: boolean;
  repostedByMe: boolean;
  bookmarkedByMe: boolean;
  repostedByName?: string;
  parentId?: string;
};

export type CategoryItem = {
  id: string;
  name: string;
};

export type FeedResponse = {
  threads: ThreadItem[];
  nextCursor: string;
  hasMore: boolean;
};

export type CreateThreadPayload = {
  content: string;
  replyPermission?: string;
  categories?: string[];
  communityId?: string;
  scheduledAt?: string;
  parentId?: string;
};

export type RepliesResponse = {
  replies: ThreadItem[];
  nextCursor: string;
  hasMore: boolean;
};

export type InteractionResponse = {
  success: boolean;
  likeCount: number;
  repostCount: number;
  bookmarkCount: number;
  likedByMe: boolean;
  repostedByMe: boolean;
  bookmarkedByMe: boolean;
};

export type UserSuggestion = {
  id: string;
  name: string;
  username: string;
  profilePictureUrl: string;
  followerCount: number;
  isFollowing: boolean;
};

export type UserProfileResponse = {
  id: string;
  name: string;
  username: string;
  email: string;
  bio: string;
  profilePictureUrl: string;
  bannerUrl: string;
  followerCount: number;
  followingCount: number;
  isFollowing: boolean;
  isSelf: boolean;
};

export type FollowListResponse = {
  users: Array<{
    id: string;
    name: string;
    username: string;
    profilePictureUrl: string;
    isFollowing: boolean;
  }>;
  nextCursor: string;
  hasMore: boolean;
};

export const userApi = {
  follow: (id: string) =>
    request<{ success: boolean; message: string }>(`/api/users/${id}/follow`, {
      method: 'POST',
      auth: true,
    }),
  unfollow: (id: string) =>
    request<{ success: boolean; message: string }>(`/api/users/${id}/follow`, {
      method: 'DELETE',
      auth: true,
    }),
  getProfile: (username: string) =>
    request<UserProfileResponse>(`/api/users/${username}`, { auth: true }),
  getFollowers: (id: string, cursor = '', limit = 20) => {
    const params = new URLSearchParams();
    if (cursor) params.set('cursor', cursor);
    if (limit) params.set('limit', limit.toString());
    const q = params.toString() ? `?${params.toString()}` : '';
    return request<FollowListResponse>(`/api/users/${id}/followers${q}`, { auth: true });
  },
  getFollowing: (id: string, cursor = '', limit = 20) => {
    const params = new URLSearchParams();
    if (cursor) params.set('cursor', cursor);
    if (limit) params.set('limit', limit.toString());
    const q = params.toString() ? `?${params.toString()}` : '';
    return request<FollowListResponse>(`/api/users/${id}/following${q}`, { auth: true });
  },
  getSuggestions: () =>
    request<UserSuggestion[]>('/api/users/suggestions', { auth: true }),
};

export const threadApi = {
  create: (body: CreateThreadPayload) =>
    request<ThreadItem>('/api/threads', { method: 'POST', body, auth: true }),
  feed: (cursor = '', limit = 20, type = 'for-you') => {
    const params = new URLSearchParams();
    if (cursor) params.set('cursor', cursor);
    if (limit) params.set('limit', limit.toString());
    if (type) params.set('type', type);
    const q = params.toString() ? `?${params.toString()}` : '';
    return request<FeedResponse>(`/api/threads/feed${q}`, { auth: true });
  },
  delete: (id: string) =>
    request<{ success: boolean; message: string }>(`/api/threads/${id}`, {
      method: 'DELETE',
      auth: true,
    }),
  categories: () => request<CategoryItem[]>('/api/categories'),
  like: (id: string) =>
    request<InteractionResponse>(`/api/threads/${id}/like`, { method: 'POST', auth: true }),
  unlike: (id: string) =>
    request<InteractionResponse>(`/api/threads/${id}/like`, { method: 'DELETE', auth: true }),
  repost: (id: string) =>
    request<InteractionResponse>(`/api/threads/${id}/repost`, { method: 'POST', auth: true }),
  unrepost: (id: string) =>
    request<InteractionResponse>(`/api/threads/${id}/repost`, { method: 'DELETE', auth: true }),
  bookmark: (id: string) =>
    request<InteractionResponse>(`/api/threads/${id}/bookmark`, { method: 'POST', auth: true }),
  unbookmark: (id: string) =>
    request<InteractionResponse>(`/api/threads/${id}/bookmark`, { method: 'DELETE', auth: true }),
  bookmarks: (query = '', cursor = '', limit = 20) => {
    const params = new URLSearchParams();
    if (query) params.set('q', query);
    if (cursor) params.set('cursor', cursor);
    if (limit) params.set('limit', limit.toString());
    const q = params.toString() ? `?${params.toString()}` : '';
    return request<FeedResponse>(`/api/bookmarks${q}`, { auth: true });
  },
  getById: (id: string) =>
    request<ThreadItem>(`/api/threads/${id}`, { auth: true }),
  replies: (id: string, cursor = '', limit = 20) => {
    const params = new URLSearchParams();
    if (cursor) params.set('cursor', cursor);
    if (limit) params.set('limit', limit.toString());
    const q = params.toString() ? `?${params.toString()}` : '';
    return request<RepliesResponse>(`/api/threads/${id}/replies${q}`, { auth: true });
  },
};

export async function uploadImage(file: File, kind: 'profile' | 'banner'): Promise<string> {
  const form = new FormData();
  form.append('kind', kind);
  form.append('file', file);

  let res: Response;
  try {
    res = await fetch(`${BASE}/api/uploads/image`, { method: 'POST', body: form });
  } catch {
    throw new ApiError(0, 'Cannot reach the server. Please try again.');
  }

  const data = await res.json().catch(() => null);
  if (!res.ok) {
    throw new ApiError(res.status, data?.error ?? 'Image upload failed. Please try again.');
  }
  return data.url as string;
}