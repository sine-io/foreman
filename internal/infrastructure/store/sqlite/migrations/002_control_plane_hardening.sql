create table if not exists schema_migrations (
  version text primary key
);

alter table runs add column created_at text not null default '';
alter table approvals add column created_at text not null default '';
alter table artifacts add column created_at text not null default '';
