create table if not exists auth_roles (
  id text primary key,
  name text not null unique
);

create table if not exists auth_permissions (
  id text primary key,
  name text not null unique
);

create table if not exists auth_user_roles (
  user_id text not null references auth_users(id) on delete cascade,
  role_id text not null references auth_roles(id) on delete cascade,
  primary key (user_id, role_id)
);

create index if not exists auth_user_roles_user_idx on auth_user_roles(user_id);

create table if not exists auth_role_permissions (
  role_id text not null references auth_roles(id) on delete cascade,
  permission_id text not null references auth_permissions(id) on delete cascade,
  primary key (role_id, permission_id)
);

create index if not exists auth_role_permissions_role_idx on auth_role_permissions(role_id);

insert into auth_permissions (id, name) values
  ('perm_account_read', 'account:read'),
  ('perm_account_write', 'account:write')
on conflict (id) do nothing;

insert into auth_roles (id, name) values
  ('role_customer', 'customer')
on conflict (id) do nothing;

insert into auth_role_permissions (role_id, permission_id) values
  ('role_customer', 'perm_account_read'),
  ('role_customer', 'perm_account_write')
on conflict do nothing;
