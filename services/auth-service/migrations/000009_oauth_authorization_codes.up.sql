create table if not exists oauth_authorization_codes (
  code_hash text primary key,
  client_id text not null references oauth_clients(id) on delete cascade,
  user_id text not null references auth_users(id) on delete cascade,
  redirect_uri text not null,
  scopes text[] not null default '{}',
  code_challenge text not null,
  code_challenge_method text not null,
  expires_at timestamptz not null,
  consumed_at timestamptz,
  created_at timestamptz not null default now()
);

create index if not exists oauth_authorization_codes_client_idx on oauth_authorization_codes(client_id);
create index if not exists oauth_authorization_codes_user_idx on oauth_authorization_codes(user_id);
create index if not exists oauth_authorization_codes_expires_idx on oauth_authorization_codes(expires_at);
create index if not exists oauth_authorization_codes_consumed_idx on oauth_authorization_codes(consumed_at);
