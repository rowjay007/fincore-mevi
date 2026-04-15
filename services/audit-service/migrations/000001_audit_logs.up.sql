create table if not exists audit_logs (
  id uuid primary key default gen_random_uuid(),
  user_id text,
  action text not null,
  resource_type text,
  resource_id text,
  payload jsonb,
  correlation_id text,
  trace_id text,
  service_name text,
  created_at timestamptz not null default now()
);

create index if not exists audit_logs_user_idx on audit_logs(user_id);
create index if not exists audit_logs_resource_idx on audit_logs(resource_type, resource_id);
create index if not exists audit_logs_created_at_idx on audit_logs(created_at desc);
create index if not exists audit_logs_correlation_id_idx on audit_logs(correlation_id);
