create table if not exists oauth_consent_history (
  id bigserial primary key,
  user_id text not null references auth_users(id) on delete cascade,
  client_id text not null references oauth_clients(id) on delete cascade,
  scopes text[] not null default '{}',
  created_at timestamptz not null default now()
);

create index if not exists oauth_consent_history_user_client_idx on oauth_consent_history(user_id, client_id);
create index if not exists oauth_consent_history_created_at_idx on oauth_consent_history(created_at);
