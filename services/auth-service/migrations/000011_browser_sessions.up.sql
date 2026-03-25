create table if not exists browser_sessions (
  id text primary key,
  user_id text not null references auth_users(id) on delete cascade,
  access_token text not null,
  expires_at timestamptz not null,
  created_at timestamptz not null default now()
);

create index if not exists browser_sessions_user_idx on browser_sessions(user_id);
create index if not exists browser_sessions_expires_idx on browser_sessions(expires_at);
