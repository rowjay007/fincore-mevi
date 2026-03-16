insert into auth_permissions (id, name) values
  ('perm_auth_admin', 'auth:admin')
on conflict (id) do nothing;

insert into auth_roles (id, name) values
  ('role_admin', 'admin')
on conflict (id) do nothing;

insert into auth_role_permissions (role_id, permission_id) values
  ('role_admin', 'perm_auth_admin'),
  ('role_admin', 'perm_account_read'),
  ('role_admin', 'perm_account_write'),
  ('role_admin', 'perm_ledger_read'),
  ('role_admin', 'perm_ledger_write')
on conflict do nothing;
