package replayer_adapter_go

import (
	"context"
	"time"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/workflow"
)

type runnerWorkflowOutboundInterceptor struct {
	interceptor.WorkflowOutboundInterceptorBase
}

func (oi *runnerWorkflowOutboundInterceptor) ExecuteLocalActivity(
	ctx workflow.Context,
	activityType string,
	args ...interface{},
) workflow.Future {
	raiseSentinelBreakpoint("ExecuteLocalActivity", workflow.GetInfo(ctx))
	return oi.Next.ExecuteLocalActivity(ctx, activityType, args...)
}

func (oi *runnerWorkflowOutboundInterceptor) ExecuteActivity(ctx workflow.Context, activityType string,
	args ...interface{}) workflow.Future {
	raiseSentinelBreakpoint("ExecuteActivity", workflow.GetInfo(ctx))
	return oi.Next.ExecuteActivity(ctx, activityType, args...)
}

func (oi *runnerWorkflowOutboundInterceptor) NewTimer(ctx workflow.Context, d time.Duration) workflow.Future {
	raiseSentinelBreakpoint("NewTimer", workflow.GetInfo(ctx))
	return oi.Next.NewTimer(ctx, d)
}

func (oi *runnerWorkflowOutboundInterceptor) SignalExternalWorkflow(ctx workflow.Context, workflowID, runID, signalName string,
	arg interface{}) workflow.Future {
	raiseSentinelBreakpoint("SignalExternalWorkflow", workflow.GetInfo(ctx))
	return oi.Next.SignalExternalWorkflow(ctx, workflowID, runID, signalName, arg)
}

func (oi *runnerWorkflowOutboundInterceptor) ExecuteChildWorkflow(ctx workflow.Context, childWorkflowType string,
	args ...interface{}) workflow.ChildWorkflowFuture {
	raiseSentinelBreakpoint("ExecuteChildWorkflow", workflow.GetInfo(ctx))
	return oi.Next.ExecuteChildWorkflow(ctx, childWorkflowType, args...)
}

func (oi *runnerWorkflowOutboundInterceptor) SideEffect(ctx workflow.Context, f func(ctx workflow.Context) interface{}) converter.EncodedValue {
	raiseSentinelBreakpoint("SideEffect", workflow.GetInfo(ctx))
	return oi.Next.SideEffect(ctx, f)
}

func (oi *runnerWorkflowOutboundInterceptor) Sleep(ctx workflow.Context, d time.Duration) (err error) {
	raiseSentinelBreakpoint("Sleep", workflow.GetInfo(ctx))
	return oi.Next.Sleep(ctx, d)
}

func (oi *runnerWorkflowOutboundInterceptor) Go(ctx workflow.Context, name string, f func(ctx workflow.Context)) workflow.Context {
	raiseSentinelBreakpoint("Go", workflow.GetInfo(ctx))
	return oi.Next.Go(ctx, name, f)
}

func (oi *runnerWorkflowOutboundInterceptor) GetLogger(ctx workflow.Context) log.Logger {
	raiseSentinelBreakpoint("GetLogger", workflow.GetInfo(ctx))
	return oi.Next.GetLogger(ctx)
}

func (oi *runnerWorkflowOutboundInterceptor) Now(ctx workflow.Context) time.Time {
	raiseSentinelBreakpoint("Now", workflow.GetInfo(ctx))
	return oi.Next.Now(ctx)
}

func (oi *runnerWorkflowOutboundInterceptor) RequestCancelExternalWorkflow(ctx workflow.Context, workflowID, runID string) workflow.Future {
	raiseSentinelBreakpoint("RequestCancelExternalWorkflow", workflow.GetInfo(ctx))
	return oi.Next.RequestCancelExternalWorkflow(ctx, workflowID, runID)
}

func (oi *runnerWorkflowOutboundInterceptor) SignalChildWorkflow(ctx workflow.Context, workflowID, signalName string, arg interface{}) workflow.Future {
	raiseSentinelBreakpoint("SignalChildWorkflow", workflow.GetInfo(ctx))
	return oi.Next.SignalChildWorkflow(ctx, workflowID, signalName, arg)
}

func (oi *runnerWorkflowOutboundInterceptor) UpsertSearchAttributes(ctx workflow.Context, attributes map[string]interface{}) error {
	raiseSentinelBreakpoint("UpsertSearchAttributes", workflow.GetInfo(ctx))
	return oi.Next.UpsertSearchAttributes(ctx, attributes)
}

func (oi *runnerWorkflowOutboundInterceptor) UpsertMemo(ctx workflow.Context, memo map[string]interface{}) error {
	raiseSentinelBreakpoint("UpsertMemo", workflow.GetInfo(ctx))
	return oi.Next.UpsertMemo(ctx, memo)
}

func (oi *runnerWorkflowOutboundInterceptor) GetSignalChannel(ctx workflow.Context, signalName string) workflow.ReceiveChannel {
	raiseSentinelBreakpoint("GetSignalChannel", workflow.GetInfo(ctx))
	return oi.Next.GetSignalChannel(ctx, signalName)
}

func (oi *runnerWorkflowOutboundInterceptor) MutableSideEffect(ctx workflow.Context, id string, f func(ctx workflow.Context) interface{}, equals func(a, b interface{}) bool) converter.EncodedValue {
	raiseSentinelBreakpoint("MutableSideEffect", workflow.GetInfo(ctx))
	return oi.Next.MutableSideEffect(ctx, id, f, equals)
}

func (oi *runnerWorkflowOutboundInterceptor) GetVersion(ctx workflow.Context, changeID string, minSupported, maxSupported workflow.Version) workflow.Version {
	raiseSentinelBreakpoint("GetVersion", workflow.GetInfo(ctx))
	return oi.Next.GetVersion(ctx, changeID, minSupported, maxSupported)
}

func (oi *runnerWorkflowOutboundInterceptor) SetQueryHandler(ctx workflow.Context, queryType string, handler interface{}) error {
	raiseSentinelBreakpoint("SetQueryHandler", workflow.GetInfo(ctx))
	return oi.Next.SetQueryHandler(ctx, queryType, handler)
}

func (oi *runnerWorkflowOutboundInterceptor) SetUpdateHandler(ctx workflow.Context, updateName string, handler interface{}, opts workflow.UpdateHandlerOptions) error {
	raiseSentinelBreakpoint("SetUpdateHandler", workflow.GetInfo(ctx))
	return oi.Next.SetUpdateHandler(ctx, updateName, handler, opts)
}

func (oi *runnerWorkflowOutboundInterceptor) IsReplaying(ctx workflow.Context) bool {
	raiseSentinelBreakpoint("IsReplaying", workflow.GetInfo(ctx))
	return oi.Next.IsReplaying(ctx)
}

func (oi *runnerWorkflowOutboundInterceptor) HasLastCompletionResult(ctx workflow.Context) bool {
	raiseSentinelBreakpoint("HasLastCompletionResult", workflow.GetInfo(ctx))
	return oi.Next.HasLastCompletionResult(ctx)
}

func (oi *runnerWorkflowOutboundInterceptor) GetLastCompletionResult(ctx workflow.Context, d ...interface{}) error {
	raiseSentinelBreakpoint("GetLastCompletionResult", workflow.GetInfo(ctx))
	return oi.Next.GetLastCompletionResult(ctx, d...)
}

func (oi *runnerWorkflowOutboundInterceptor) GetLastError(ctx workflow.Context) error {
	raiseSentinelBreakpoint("GetLastError", workflow.GetInfo(ctx))
	return oi.Next.GetLastError(ctx)
}

func (oi *runnerWorkflowOutboundInterceptor) NewContinueAsNewError(ctx workflow.Context, wfn interface{}, args ...interface{}) error {
	raiseSentinelBreakpoint("NewContinueAsNewError", workflow.GetInfo(ctx))
	return oi.Next.NewContinueAsNewError(ctx, wfn, args...)
}

type runnerActivityOutboundInterceptor struct {
	interceptor.ActivityOutboundInterceptorBase
}

func (r *runnerActivityOutboundInterceptor) RecordHeartbeat(ctx context.Context, details ...interface{}) {
	raiseSentinelBreakpoint("RecordHeartbeat", nil)
	r.Next.RecordHeartbeat(ctx, details...)
}
