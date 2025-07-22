package replayer_adapter

import (
	"context"

	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/workflow"
)

// Pass the call to the next interceptor and notify runner when it's done
type runnerWorkflowInboundInterceptor struct {
	interceptor.WorkflowInboundInterceptorBase
}

func (ii *runnerWorkflowInboundInterceptor) Init(outbound interceptor.WorkflowOutboundInterceptor) error {
	runnerOutbount := &runnerWorkflowOutboundInterceptor{}
	runnerOutbount.Next = outbound
	return ii.Next.Init(runnerOutbount)
}

func (ii *runnerWorkflowInboundInterceptor) ExecuteWorkflow(ctx workflow.Context,
	in *interceptor.ExecuteWorkflowInput) (interface{}, error) {
	notifyRunner("ExecuteWorkflow", workflow.GetInfo(ctx))
	return ii.Next.ExecuteWorkflow(ctx, in)
}

func (ii *runnerWorkflowInboundInterceptor) HandleSignal(ctx workflow.Context, in *interceptor.HandleSignalInput) error {
	notifyRunner("HandleSignal", workflow.GetInfo(ctx))
	return ii.Next.HandleSignal(ctx, in)
}

func (ii *runnerWorkflowInboundInterceptor) HandleQuery(ctx workflow.Context, in *interceptor.HandleQueryInput) (interface{}, error) {
	notifyRunner("HandleQuery", workflow.GetInfo(ctx))
	return ii.Next.HandleQuery(ctx, in)
}

func (ii *runnerWorkflowInboundInterceptor) ValidateUpdate(ctx workflow.Context, in *interceptor.UpdateInput) error {
	notifyRunner("ValidateUpdate", workflow.GetInfo(ctx))
	return ii.Next.ValidateUpdate(ctx, in)
}

func (ii *runnerWorkflowInboundInterceptor) ExecuteUpdate(ctx workflow.Context, in *interceptor.UpdateInput) (interface{}, error) {
	notifyRunner("ExecuteUpdate", workflow.GetInfo(ctx))
	return ii.Next.ExecuteUpdate(ctx, in)
}

type runnerInboundActivityIntercetor struct {
	interceptor.ActivityInboundInterceptorBase
}

func (i *runnerInboundActivityIntercetor) ExecuteActivity(ctx context.Context,
	in *interceptor.ExecuteActivityInput) (interface{}, error) {
	notifyRunner("ExecuteActivity", nil)
	return i.ActivityInboundInterceptorBase.ExecuteActivity(ctx, in)
}

func (i *runnerInboundActivityIntercetor) Init(outbound interceptor.ActivityOutboundInterceptor) error {
	runnerOutbount := &runnerActivityOutboundInterceptor{}
	runnerOutbount.Next = outbound
	return i.Next.Init(runnerOutbount)
}
