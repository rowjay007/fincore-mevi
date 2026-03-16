delete from auth_role_permissions where role_id = 'role_admin';

delete from auth_roles where id = 'role_admin';

delete from auth_permissions where id = 'perm_auth_admin';
