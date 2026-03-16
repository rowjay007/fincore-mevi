alter table auth_refresh_sessions add column if not exists session_id text;
alter table auth_refresh_sessions add column if not exists absolute_expires_at timestamptz;
alter table auth_refresh_sessions add column if not exists user_agent text;
alter table auth_refresh_sessions add column if not exists ip text;

update auth_refresh_sessions set session_id = token_hash where session_id is null;
update auth_refresh_sessions set absolute_expires_at = expires_at where absolute_expires_at is null;

create index if not exists auth_refresh_sessions_session_idx on auth_refresh_sessions(session_id);
create index if not exists auth_refresh_sessions_abs_expires_idx on auth_refresh_sessions(absolute_expires_at);
