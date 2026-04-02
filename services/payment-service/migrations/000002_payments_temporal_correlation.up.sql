create index if not exists payments_projection_status_idx
  on payments_projection(status);

alter table payments_projection
  add column if not exists temporal_workflow_id text,
  add column if not exists temporal_run_id text;
