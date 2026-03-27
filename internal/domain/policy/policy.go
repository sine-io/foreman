package policy

import "strings"

type Decision struct {
	RequiresApproval bool
	Reason           string
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
		}
	case strings.Contains(normalized, "git tag"):
		return Decision{
			RequiresApproval: true,
			Reason:           "git tag requires approval",
		}
	case strings.Contains(normalized, "rm -rf"):
		return Decision{
			RequiresApproval: true,
			Reason:           "destructive filesystem operations require approval",
		}
	default:
		return Decision{}
	}
}
