export interface Category {
  id: number;
  name: string;
  position: number;
}

export interface Feed {
  id: number;
  category_id: number | null;
  url: string;
  title: string;
  site_url: string;
  enabled: boolean;
  full_text: boolean;
  include_in_digest: boolean;
  max_articles: number;
  last_fetched_at: string | null;
  last_error: string;
  created_at: string;
  unread_count: number;
  article_count: number;
}

export interface Article {
  id: number;
  feed_id: number;
  feed_title: string;
  url: string;
  title: string;
  author: string;
  summary: string;
  content_html?: string;
  full_content_html?: string;
  image_url: string;
  published_at: string;
  fetched_at: string;
  is_read: boolean;
  is_starred: boolean;
  sent_at: string | null;
  has_full_content: boolean;
}

export interface ArticlePage {
  items: Article[];
  next_cursor: string;
}

export interface Collection {
  id: number;
  name: string;
  created_at: string;
  article_count: number;
}

export interface Delivery {
  id: number;
  kind: 'digest' | 'selection' | 'collection' | 'starred' | 'resend';
  title: string;
  status: 'queued' | 'building' | 'sending' | 'sent' | 'ready' | 'empty' | 'failed';
  send: boolean;
  article_count: number;
  size_bytes: number;
  has_epub: boolean;
  error: string;
  recipient: string;
  parent_id: number | null;
  created_at: string;
  finished_at: string | null;
  article_ids?: number[];
}

export interface Settings {
  kindle_email: string;
  sender_email: string;
  sender_name: string;
  smtp_host: string;
  smtp_port: number;
  smtp_user: string;
  smtp_password?: string;
  has_smtp_password?: boolean;
  smtp_security: 'starttls' | 'tls' | 'none';

  digest_enabled: boolean;
  digest_time: string;
  digest_days: number[];
  timezone: string;
  digest_window_hours: number;
  digest_exclude_read: boolean;
  digest_skip_empty: boolean;
  digest_title: string;
  digest_max_articles: number;

  refresh_minutes: number;

  images_enabled: boolean;
  images_grayscale: boolean;
  image_max_width: number;
  language: string;

  retention_days: number;
}

export interface Counts {
  unread: number;
  starred: number;
  unsent: number;
  today: number;
  total: number;
  by_feed: Record<number, number>;
  by_category: Record<number, number>;
}

export interface DailyStat {
  day: string;
  fetched: number;
  sent: number;
}

export interface StatusResponse {
  next_digest_run: string | null;
  last_digest: Delivery | null;
  refresher: {
    running: boolean;
    last_run: string | null;
    next_run: string | null;
  };
  counts: Counts | null;
  stats: DailyStat[];
}
