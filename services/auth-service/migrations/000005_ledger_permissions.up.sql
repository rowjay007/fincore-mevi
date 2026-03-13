insert into auth_permissions (id, name) values
  ('perm_ledger_read', 'ledger:read'),
  ('perm_ledger_write', 'ledger:write')
on conflict (id) do nothing;

insert into auth_role_permissions (role_id, permission_id) values
  ('role_customer', 'perm_ledger_read'),
  ('role_customer', 'perm_ledger_write')
on conflict do nothing;
