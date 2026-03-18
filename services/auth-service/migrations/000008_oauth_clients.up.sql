create table if not exists oauth_clients (
  id text primary key,
  name text not null,
  type text not null,
  secret_hash text,
  redirect_uris text[] not null,
  allowed_scopes text[] not null default '{}',
  created_at timestamptz not null default now()
);

create index if not exists oauth_clients_type_idx on oauth_clients(type);
