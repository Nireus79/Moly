package agents

import (
	"context"
	"errors"

	"moly/models"
	"moly/tools"
)

// SafetyJudge is the constitutional evaluator, as the safety stage needs it.
type SafetyJudge interface {
	EvaluateWithAnalysisContextAndMaturity(ctx context.Context, analysisCtx *models.AnalysisContext, maturity float64, severityGate string) (*tools.ConstitutionalVerdict, error)
}

// ErrSafetyUnavailable means no safety verdict could be had. The caller must not deliver a reply: safety fails closed.
var ErrSafetyUnavailable = errors.New("the safety check did not complete")

// Severity gate: the less is known about the conversation, the fewer violations block it.
// Maturity below severityGateImmature blocks only critical violations; each higher band blocks one more level.
const (
	severityGateImmature = 0.3
	severityGateGrowing  = 0.5
	severityGateMature   = 0.7
)

// SeverityGate returns the lowest violation severity that blocks at this maturity.
func SeverityGate(maturity float64) string {
	switch {
	case maturity < severityGateImmature:
		return "critical"
	case maturity < severityGateGrowing:
		return "high"
	case maturity < severityGateMature:
		return "medium"
	default:
		return "low"
	}
}

// SafetyResult is the one safety verdict of a message.
type SafetyResult struct {
	Verdict *tools.ConstitutionalVerdict
	Alert   *models.SafetyAlert // set when a principle is violated
	Blocked bool                // obvious harm: nothing is saved from this message and the reply is a refusal
}

// EvaluateSafety is the single safety call of a message (ORCHESTRATOR_DESIGN.md step 4). It runs before anything is
// saved. Any failure returns ErrSafetyUnavailable: the message is never answered without its verdict.
func EvaluateSafety(ctx context.Context, judge SafetyJudge, analysis *models.AnalysisContext, maturity float64) (SafetyResult, error) {
	if judge == nil || analysis == nil {
		return SafetyResult{}, ErrSafetyUnavailable
	}
	verdict, err := judge.EvaluateWithAnalysisContextAndMaturity(ctx, analysis, maturity, SeverityGate(maturity))
	if err != nil || verdict == nil {
		return SafetyResult{}, ErrSafetyUnavailable
	}
	alert := verdict.ToSafetyAlert()
	return SafetyResult{Verdict: verdict, Alert: alert, Blocked: alert != nil && alert.IsObviousHarm}, nil
}
