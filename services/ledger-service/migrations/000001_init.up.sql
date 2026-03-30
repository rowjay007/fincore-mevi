create table if not exists event_store_events (
  sequence bigint generated always as identity,
  id text primary key,
  aggregate_id text not null,
  aggregate_type text not null,
  version bigint not null,
  type text not null,
  occurred_at timestamptz not null,
  data bytea not null,
  metadata bytea
);

alter table if exists event_store_events
  add column if not exists sequence bigint generated always as identity;

create unique index if not exists event_store_events_sequence_uidx
  on event_store_events(sequence);

create unique index if not exists event_store_events_agg_ver_uidx
  on event_store_events(aggregate_id, version);

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

create table if not exists ledger_account_balances (
  account_id text primary key,
  balance_kobo bigint not null
);

create table if not exists ledger_idempotency (
  key text primary key,
  entry_id text not null,
  created_at timestamptz not null default now()
);
