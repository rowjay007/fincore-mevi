create table if not exists event_store_events (
  id text primary key,
  aggregate_id text not null,
  aggregate_type text not null,
  version bigint not null,
  type text not null,
  occurred_at timestamptz not null,
  data bytea not null,
  metadata bytea
);

create unique index if not exists event_store_events_agg_ver_uidx
  on event_store_events(aggregate_id, version);

create index if not exists event_store_events_agg_idx
  on event_store_events(aggregate_id);

create table if not exists event_store_snapshots (
  aggregate_id text not null,
  aggregate_type text not null,
  version bigint not null,
  created_at timestamptz not null default now(),
  data bytea not null,
  primary key (aggregate_id, version)
);

create table if not exists outbox_messages (
  id text primary key,
  topic text not null,
  key bytea,
  value bytea not null,
  headers jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  published_at timestamptz
);

create index if not exists outbox_messages_unpublished_idx
  on outbox_messages(created_at)
  where published_at is null;

create table if not exists payments_projection (
  payment_id text primary key,
  from_account_id text not null,
  to_account_id text not null,
  currency text not null,
  amount_kobo bigint not null,
  narration text not null,
  status text not null,
  temporal_workflow_id text,
  temporal_run_id text,
  version bigint not null,
  updated_at timestamptz not null default now()
);

create index if not exists payments_projection_from_account_id_idx
  on payments_projection(from_account_id);

create index if not exists payments_projection_to_account_id_idx
  on payments_projection(to_account_id);
