alter table artifacts add column run_id text references runs(id);
alter table artifacts add column storage_path text not null default '';

create index if not exists artifacts_run_created_idx
  on artifacts(run_id, created_at desc, id desc)
  where run_id is not null;
