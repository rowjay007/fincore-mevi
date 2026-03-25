create table if not exists oauth_consents (
  user_id text not null references auth_users(id) on delete cascade,
  client_id text not null references oauth_clients(id) on delete cascade,
  scopes text[] not null default '{}',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  primary key (user_id, client_id)
);

create index if not exists oauth_consents_user_client_idx on oauth_consents(user_id, client_id);
