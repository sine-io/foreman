create table if not exists projects (
  id text primary key,
  name text not null,
  repo_root text not null
);

create table if not exists modules (
  id text primary key,
  project_id text not null references projects(id),
  name text not null,
  board_state text not null
);

create table if not exists tasks (
  id text primary key,
  module_id text not null references modules(id),
  summary text not null default '',
  acceptance text not null default '',
  priority integer not null default 0,
  task_type text not null,
  state text not null,
  write_scope text not null
);

create table if not exists runs (
  id text primary key,
  task_id text not null references tasks(id),
  runner_kind text not null,
  state text not null
);

create table if not exists approvals (
  id text primary key,
  task_id text not null references tasks(id),
  reason text not null,
  state text not null
);

create unique index if not exists approvals_pending_task_idx on approvals(task_id) where state = 'pending';

create table if not exists artifacts (
  id text primary key,
  task_id text not null references tasks(id),
  kind text not null,
  path text not null,
  summary text not null default ''
);

create table if not exists leases (
  id text primary key,
  task_id text not null references tasks(id),
  scope_key text not null,
  state text not null
);

create unique index if not exists leases_active_scope_idx on leases(scope_key) where state = 'active';
