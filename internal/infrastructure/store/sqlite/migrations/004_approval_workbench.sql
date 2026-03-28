alter table approvals add column risk_level text not null default '';
alter table approvals add column policy_rule text not null default '';
alter table approvals add column rejection_reason text not null default '';
