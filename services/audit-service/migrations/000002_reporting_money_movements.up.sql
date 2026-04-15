create table if not exists reporting_money_movements (
  id bigserial primary key,
  event_type text not null,
  amount_kobo bigint not null,
  currency text not null,
  account_id text,
  user_id text,
  correlation_id text,
  trace_id text,
  occurred_at timestamptz not null,
  created_at timestamptz not null default now()
);

create index if not exists reporting_money_movements_occurred_at_idx on reporting_money_movements(occurred_at desc);
create index if not exists reporting_money_movements_event_type_idx on reporting_money_movements(event_type);
create index if not exists reporting_money_movements_account_idx on reporting_money_movements(account_id);
create index if not exists reporting_money_movements_user_idx on reporting_money_movements(user_id);
