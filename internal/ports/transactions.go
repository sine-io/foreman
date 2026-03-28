package ports

import "context"

type TransactionRepositories struct {
	Projects  ProjectRepository
	Modules   ModuleRepository
	Tasks     TaskRepository
	Runs      RunRepository
	Approvals ApprovalRepository
	Artifacts ArtifactRepository
	Leases    LeaseRepository
}

type Transactor interface {
	WithinTransaction(context.Context, func(context.Context, TransactionRepositories) error) error
}
