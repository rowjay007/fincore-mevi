alter table payments_projection
  drop column if exists temporal_run_id,
  drop column if exists temporal_workflow_id;

drop index if exists payments_projection_status_idx;
