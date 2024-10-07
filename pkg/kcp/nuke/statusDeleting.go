package nuke

import (
	"context"
	cloudcontrolv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-control/v1beta1"
	"github.com/kyma-project/cloud-manager/pkg/composed"
)

func statusDeleting(ctx context.Context, st composed.State) (error, context.Context) {
	state := st.(*State)

	changed := false

	for _, rks := range state.Resources {
		kindStatus, created := state.ObjAsNuke().Status.GetKind(rks.Kind)
		if created {
			changed = true
		}

		for _, obj := range rks.Objects {
			if kindStatus.Objects[obj.GetName()] != cloudcontrolv1beta1.NukeResourceStatusDeleting {
				kindStatus.Objects[obj.GetName()] = cloudcontrolv1beta1.NukeResourceStatusDeleting
				changed = true
			}
		}
	}

	if !changed {
		return nil, ctx
	}

	state.ObjAsNuke().Status.State = "Deleting"

	return composed.PatchStatus(state.ObjAsNuke()).
		ErrorLogMessage("Error patching KCP Nuke status with deleting resources").
		SuccessErrorNil().
		Run(ctx, state)
}