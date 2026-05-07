-- 000003_add_audit_sequence.up.sql
alter table audit_logs add column sequence bigserial;
create index if not exists audit_logs_sequence_idx on audit_logs(sequence asc);
drop index if exists audit_logs_created_at_idx;
create index if not exists audit_logs_created_at_idx on audit_logs(created_at desc, sequence desc);
