alter table auth_refresh_sessions drop column if exists ip;
alter table auth_refresh_sessions drop column if exists user_agent;
alter table auth_refresh_sessions drop column if exists absolute_expires_at;
alter table auth_refresh_sessions drop column if exists session_id;
