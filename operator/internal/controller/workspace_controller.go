/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	spot "github.com/releasehub-com/spot/operator/api/v1alpha1"
	"github.com/releasehub-com/spot/operator/internal/stages"
)

// WorkspaceReconciler reconciles a Workspace object
type WorkspaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	record.EventRecorder
}

//+kubebuilder:rbac:groups=spot.release.com,resources=workspaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=spot.release.com,resources=workspaces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=spot.release.com,resources=workspaces/finalizers,verbs=update
//+kubebuilder:rbac:groups=spot.release.com,resources=events,verbs=create;patch

func (r *WorkspaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	var workspace spot.Workspace
	if err := r.Client.Get(ctx, req.NamespacedName, &workspace); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		logger.Error(err, "Couldn't retrieve the workspace", "NamespacedName", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	switch workspace.Status.Stage {

	// The Workspace was just created and nothing has happened to it
	// yet. The first step is to start the building process.
	case spot.WorkspaceStageInitialized:
		r.EventRecorder.Event(&workspace, "Normal", "Initialized", "Workspace initialized")
		builder := stages.Builder{Client: r.Client}
		err := builder.Start(ctx, &workspace)
		if err != nil {
			return ctrl.Result{}, r.markWorkspaceHasErrored(ctx, &workspace, err)
		}

	// The Workspace launched the builders but those have not
	// completed yet. Need to monitor each of the builder object
	// to see if they are completed and we can move forward to the next
	// stage
	case spot.WorkspaceStageBuilding:
		r.EventRecorder.Event(&workspace, "Normal", "Building", "Waiting for builds to complete")
		builder := stages.Builder{Client: r.Client}
		err := builder.Update(ctx, &workspace)
		if err != nil {
			return ctrl.Result{}, err
		}

	case spot.WorkspaceStageDeploying:
		r.EventRecorder.Event(&workspace, "Normal", "Deploying", "Deploying services and updating routes")
		deployment := stages.Deployment{Client: r.Client}
		if err := deployment.Start(ctx, &workspace); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkspaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&spot.Workspace{}).
		Complete(r)
}

func (r *WorkspaceReconciler) markWorkspaceHasErrored(ctx context.Context, workspace *spot.Workspace, err error) error {
	r.EventRecorder.Event(workspace, "Warning", string(spot.WorkspaceStageError), err.Error())
	workspace.Status.Stage = spot.WorkspaceStageError
	return r.Client.Status().Update(ctx, workspace)
}
