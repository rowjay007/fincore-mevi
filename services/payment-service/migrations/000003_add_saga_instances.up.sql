-- 000003_add_saga_instances.up.sql
create type saga_status as enum ('STARTED', 'DEBITED', 'PROVISIONED', 'COMPLETING', 'COMPENSATING', 'FAILED', 'COMPLETED');

create table if not exists saga_instances (
    id uuid primary key,
    correlation_id text not null,
    saga_type text not null,
    status saga_status not null,
    payload jsonb not null,
    last_heartbeat_at timestamptz not null default now(),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create index if not exists saga_instances_status_idx on saga_instances(status) where status not in ('COMPLETED', 'FAILED');
create index if not exists saga_instances_heartbeat_idx on saga_instances(last_heartbeat_at);
