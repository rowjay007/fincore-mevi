create table if not exists accounts_projection (
  account_id text primary key,
  customer_id text not null,
  status text not null,
  version bigint not null,
  updated_at timestamptz not null default now()
);

create index if not exists accounts_projection_customer_id_idx
  on accounts_projection(customer_id);
