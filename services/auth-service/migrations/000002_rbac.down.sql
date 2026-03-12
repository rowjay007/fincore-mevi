delete from auth_role_permissions where role_id = 'role_customer';
delete from auth_roles where id = 'role_customer';
delete from auth_permissions where id in ('perm_account_read', 'perm_account_write');

drop table if exists auth_role_permissions;
drop table if exists auth_user_roles;
drop table if exists auth_permissions;
drop table if exists auth_roles;
