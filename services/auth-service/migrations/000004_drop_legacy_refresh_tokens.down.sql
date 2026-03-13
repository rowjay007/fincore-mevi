create table if not exists auth_refresh_tokens (
  token text primary key,
  user_id text not null references auth_users(id) on delete cascade,
  expires_at timestamptz not null,
  created_at timestamptz not null default now()
);

create index if not exists auth_refresh_tokens_user_idx on auth_refresh_tokens(user_id);
create index if not exists auth_refresh_tokens_expires_idx on auth_refresh_tokens(expires_at);
