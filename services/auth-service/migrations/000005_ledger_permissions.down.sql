delete from auth_role_permissions where role_id = 'role_customer' and permission_id in ('perm_ledger_read', 'perm_ledger_write');
delete from auth_permissions where id in ('perm_ledger_read', 'perm_ledger_write');
