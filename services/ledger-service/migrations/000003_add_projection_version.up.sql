-- 000003_add_projection_version.up.sql
alter table ledger_account_balances add column projection_version integer not null default 1;
alter table ledger_account_balances drop constraint ledger_account_balances_pkey;
alter table ledger_account_balances add primary key (account_id, projection_version);
