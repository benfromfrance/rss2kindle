import type {
  Article,
  ArticlePage,
  Category,
  Collection,
  Counts,
  Delivery,
  Feed,
  Settings,
  StatusResponse,
} from '../types';

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
    credentials: 'same-origin',
  });

  if (res.status === 401) {
    window.dispatchEvent(new CustomEvent('auth-required'));
  }

  if (!res.ok) {
    let errMessage = `Erreur HTTP ${res.status}`;
    try {
      const data = await res.json();
      if (data.error) errMessage = data.error;
    } catch {
      // ignore
    }
    throw new Error(errMessage);
  }

  return res.json();
}

export const api = {
  // Auth
  login: (password: string) =>
    request<{ authenticated: boolean }>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ password }),
    }),
  logout: () =>
    request<{ authenticated: boolean }>('/api/auth/logout', { method: 'POST' }),
  checkAuth: () =>
    request<{ authenticated: boolean }>('/api/auth/me'),

  // Catégories
  listCategories: () => request<Category[]>('/api/categories'),
  createCategory: (name: string) =>
    request<Category>('/api/categories', {
      method: 'POST',
      body: JSON.stringify({ name }),
    }),
  updateCategory: (id: number, name: string, position: number) =>
    request<{ ok: boolean }>(`/api/categories/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ name, position }),
    }),
  deleteCategory: (id: number) =>
    request<{ ok: boolean }>(`/api/categories/${id}`, { method: 'DELETE' }),

  // Flux
  listFeeds: () => request<Feed[]>('/api/feeds'),
  getFeed: (id: number) => request<Feed>(`/api/feeds/${id}`),
  createFeed: (data: {
    url: string;
    title?: string;
    category_id?: number | null;
    full_text?: boolean;
    include_in_digest?: boolean;
    max_articles?: number;
  }) =>
    request<Feed>('/api/feeds', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  updateFeed: (id: number, data: Partial<Feed>) =>
    request<{ ok: boolean }>(`/api/feeds/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  deleteFeed: (id: number) =>
    request<{ ok: boolean }>(`/api/feeds/${id}`, { method: 'DELETE' }),
  validateFeed: (url: string) =>
    request<{
      title: string;
      feed_url: string;
      site_url: string;
      article_count: number;
      samples: Array<{
        title: string;
        url: string;
        author: string;
        published_at: string;
        summary: string;
      }>;
    }>('/api/feeds/validate', {
      method: 'POST',
      body: JSON.stringify({ url }),
    }),
  refreshFeed: (id: number) =>
    request<{ new_articles: number }>(`/api/feeds/${id}/refresh`, {
      method: 'POST',
    }),
  refreshAllFeeds: () =>
    request<{ message: string }>('/api/feeds/refresh-all', { method: 'POST' }),

  // OPML
  importOPML: async (file: File): Promise<{ imported: number; message: string }> => {
    const formData = new FormData();
    formData.append('file', file);
    const res = await fetch('/api/opml/import', {
      method: 'POST',
      body: formData,
      credentials: 'same-origin',
    });
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      throw new Error(data.error || 'Échec import OPML');
    }
    return res.json();
  },

  // Articles
  listArticles: (params: {
    view?: string;
    feed_id?: number;
    category_id?: number;
    collection_id?: number;
    q?: string;
    cursor?: string;
    limit?: number;
  }) => {
    const search = new URLSearchParams();
    if (params.view) search.set('view', params.view);
    if (params.feed_id) search.set('feed_id', String(params.feed_id));
    if (params.category_id) search.set('category_id', String(params.category_id));
    if (params.collection_id) search.set('collection_id', String(params.collection_id));
    if (params.q) search.set('q', params.q);
    if (params.cursor) search.set('cursor', params.cursor);
    if (params.limit) search.set('limit', String(params.limit));
    return request<ArticlePage>(`/api/articles?${search.toString()}`);
  },
  getArticle: (id: number) => request<Article>(`/api/articles/${id}`),
  patchArticle: (id: number, data: { is_read?: boolean; is_starred?: boolean }) =>
    request<{ ok: boolean }>(`/api/articles/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    }),
  bulkArticles: (data: {
    action: 'mark_read' | 'mark_unread' | 'star' | 'unstar' | 'add_collection' | 'remove_collection';
    article_ids: number[];
    collection_id?: number;
  }) =>
    request<{ affected: number }>('/api/articles/bulk', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  extractFullText: (id: number) =>
    request<{ full_content_html: string }>(`/api/articles/${id}/fulltext`, {
      method: 'POST',
    }),
  markAllRead: (feed_id?: number, category_id?: number) =>
    request<{ affected: number }>('/api/articles/mark-all-read', {
      method: 'POST',
      body: JSON.stringify({ feed_id, category_id }),
    }),

  // Compteurs
  getCounts: () => request<Counts>('/api/counts'),

  // Collections
  listCollections: () => request<Collection[]>('/api/collections'),
  getCollection: (id: number) => request<Collection>(`/api/collections/${id}`),
  createCollection: (name: string) =>
    request<Collection>('/api/collections', {
      method: 'POST',
      body: JSON.stringify({ name }),
    }),
  updateCollection: (id: number, name: string) =>
    request<{ ok: boolean }>(`/api/collections/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ name }),
    }),
  deleteCollection: (id: number) =>
    request<{ ok: boolean }>(`/api/collections/${id}`, { method: 'DELETE' }),
  collectionItems: (
    id: number,
    action: 'add' | 'remove' | 'reorder',
    article_ids: number[]
  ) =>
    request<{ ok: boolean }>(`/api/collections/${id}/items`, {
      method: 'POST',
      body: JSON.stringify({ action, article_ids }),
    }),

  // Livraisons
  listDeliveries: (limit = 50) =>
    request<Delivery[]>(`/api/deliveries?limit=${limit}`),
  getDelivery: (id: number) => request<Delivery>(`/api/deliveries/${id}`),
  createDelivery: (data: {
    kind: 'digest' | 'selection' | 'collection' | 'starred';
    title?: string;
    article_ids?: number[];
    collection_id?: number;
    send?: boolean;
  }) =>
    request<Delivery>('/api/deliveries', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  resendDelivery: (id: number, regenerate = false) =>
    request<Delivery>(`/api/deliveries/${id}/resend`, {
      method: 'POST',
      body: JSON.stringify({ regenerate }),
    }),
  deleteDelivery: (id: number) =>
    request<{ ok: boolean }>(`/api/deliveries/${id}`, { method: 'DELETE' }),

  // Paramètres
  getSettings: () =>
    request<Settings & { has_smtp_password: boolean }>('/api/settings'),
  updateSettings: (data: Settings & { clear_smtp_password?: boolean }) =>
    request<{ settings: Settings; has_smtp_password: boolean }>('/api/settings', {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  testEmail: (data?: {
    recipient?: string;
    smtp_host?: string;
    smtp_port?: number;
    smtp_user?: string;
    smtp_password?: string;
    smtp_security?: string;
    sender_email?: string;
    sender_name?: string;
    kindle_email?: string;
  }) =>
    request<{ message: string }>('/api/settings/test-email', {
      method: 'POST',
      body: JSON.stringify(data || {}),
    }),

  // Statut
  getStatus: () => request<StatusResponse>('/api/status'),
};
