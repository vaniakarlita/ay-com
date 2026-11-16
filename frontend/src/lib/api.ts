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

export const authApi = {
  register: (body: RegisterPayload) =>
    request<{ userId: string; message: string }>('/api/auth/register', { method: 'POST', body }),
  verify: (email: string, code: string) =>
    request<Message>('/api/auth/verify', { method: 'POST', body: { email, code } }),
  resendCode: (email: string) =>
    request<Message>('/api/auth/resend-code', { method: 'POST', body: { email } }),
  login: (email: string, password: string) =>
    request<Tokens>('/api/auth/login', { method: 'POST', body: { email, password } }),
  me: () => request<{ userId: string }>('/api/me', { auth: true }),
  forgotQuestion: (email: string) =>
    request<{ question: string }>('/api/auth/forgot/question', { method: 'POST', body: { email } }),
  forgotVerify: (email: string, answer: string) =>
    request<{ resetToken: string }>('/api/auth/forgot/verify', { method: 'POST', body: { email, answer } }),
  forgotReset: (email: string, resetToken: string, newPassword: string) =>
    request<Message>('/api/auth/forgot/reset', { method: 'POST', body: { email, resetToken, newPassword } }),
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