create table if not exists auth_refresh_sessions (
  token_hash text primary key,
  user_id text not null references auth_users(id) on delete cascade,
  expires_at timestamptz not null,
  created_at timestamptz not null default now(),
  last_used_at timestamptz,
  revoked_at timestamptz,
  replaced_by_hash text
);

create index if not exists auth_refresh_sessions_user_idx on auth_refresh_sessions(user_id);
create index if not exists auth_refresh_sessions_expires_idx on auth_refresh_sessions(expires_at);
create index if not exists auth_refresh_sessions_revoked_idx on auth_refresh_sessions(revoked_at);
