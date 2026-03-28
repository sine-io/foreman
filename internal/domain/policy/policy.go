package policy

import (
	"strings"

	"github.com/sine-io/foreman/internal/domain/approval"
)

type Decision struct {
	RequiresApproval bool
	Reason           string
	RiskLevel        approval.RiskLevel
	PolicyRule       string
}

func EvaluateStrictAction(action string) Decision {
	normalized := strings.ToLower(strings.TrimSpace(action))

	switch {
	case normalized == "":
		return Decision{}
	case strings.Contains(normalized, "git push"):
		return Decision{
			RequiresApproval: true,
			Reason:           "git push requires approval",
			RiskLevel:        approval.RiskHigh,
			PolicyRule:       "strict.git_push",
		}
	case strings.Contains(normalized, "git tag"):
		return Decision{
			RequiresApproval: true,
			Reason:           "git tag requires approval",
			RiskLevel:        approval.RiskMedium,
			PolicyRule:       "strict.git_tag",
		}
	case strings.Contains(normalized, "rm -rf"):
		return Decision{
			RequiresApproval: true,
			Reason:           "destructive filesystem operations require approval",
			RiskLevel:        approval.RiskCritical,
			PolicyRule:       "strict.rm_rf",
		}
	default:
		return Decision{}
	}
}
