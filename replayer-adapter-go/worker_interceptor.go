package replayer_adapter_go

import (
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/workflow"
)

type runnerWorkerInterceptor struct {
	interceptor.InterceptorBase
}

func (r *runnerWorkerInterceptor) InterceptWorkflow(
	ctx workflow.Context,
	next interceptor.WorkflowInboundInterceptor,
) interceptor.WorkflowInboundInterceptor {
	ib := &runnerWorkflowInboundInterceptor{}
	ib.Next = next
	return ib
}
