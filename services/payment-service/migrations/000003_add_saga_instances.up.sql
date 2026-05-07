-- 000003_add_saga_instances.up.sql
-- Define the possible states for a multi-service transaction saga.
create type saga_status as enum (
    'STARTED',      -- Saga has been initiated
    'DEBITED',      -- Funds have been successfully debited from the ledger
    'PROVISIONED',  -- External resources have been reserved
    'COMPLETING',   -- Finalizing the transaction
    'COMPENSATING', -- Downstream failure detected, rolling back
    'FAILED',       -- Terminal state: rollback complete or fatal error
    'COMPLETED'     -- Terminal state: transaction successful
);

-- Persistent storage for saga state to allow external anti-entropy recovery.
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

-- Indices optimized for the Sentinel worker's polling logic.
create index if not exists saga_instances_status_idx on saga_instances(status) 
where status not in ('COMPLETED', 'FAILED');

create index if not exists saga_instances_heartbeat_idx on saga_instances(last_heartbeat_at);
