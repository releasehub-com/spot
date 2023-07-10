package stages

import (
	"context"
	"errors"

	meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	spot "github.com/releasehub-com/spot/operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Builder struct {
	client.Client
}

func (b *Builder) Start(ctx context.Context, workspace *spot.Workspace) error {
	logger := log.FromContext(ctx)

	if len(workspace.Status.Builds) != 0 {
		return b.markWorkspaceHasErrored(ctx, workspace, errors.New("unexpected builds present for this workspace"))
	}

	builds := []*spot.Build{{
		ObjectMeta: meta.ObjectMeta{
			Namespace:    workspace.Namespace,
			GenerateName: "my-build-",
			OwnerReferences: []meta.OwnerReference{
				{
					Kind:       workspace.Kind,
					Name:       workspace.Name,
					APIVersion: workspace.APIVersion,
					UID:        workspace.UID,
				},
			},
		},
		Spec: spot.BuildSpec{
			RepositoryURL: "https://github.com/releasehub-com/click-mania-test.git",
			Image: spot.ImageSpec{
				Name: "click-mania",
				Tag:  "my-branch",
				Registry: spot.RegistrySpec{
					URL: "docker.io/pierolivierrh/click-mania",
				},
			},
		},
	}}

	var references []spot.BuildReference
	for _, build := range builds {
		if err := b.Client.Create(ctx, build); err != nil {
			logger.Error(err, "unexpected error creating a build")
			return b.markWorkspaceHasErrored(ctx, workspace, err)
		}

		references = append(references, build.GetReference())
	}

	workspace.Status.Builds = references
	workspace.Status.Stage = spot.WorkspaceStageBuilding

	return b.Client.Status().Update(ctx, workspace)
}

// Reconcile attempts to finish the work toward the next stage for the Workspace.
// If it succeeds, it will return the Stage that should follow so the reconciler can act on that new information
// It's normal that the Work function can't finish up the work in 1 go as there might be work that needs to happen
// asynchronously.
func (b *Builder) Update(ctx context.Context, workspace *spot.Workspace) error {
	// logger := log.FromContext(ctx)
	if err := b.trackProgressAndUpdate(ctx, workspace); err != nil {
		b.markWorkspaceHasErrored(ctx, workspace, err)
	}

	if b.completed(workspace) {
		workspace.Status.Stage = spot.WorkspaceStageDeploying
		return b.Client.Status().Update(ctx, workspace)
	}

	// Build stage is not completed but it's progressing, return with no error
	// and wait for an update from one of the builds to re-evaluate.
	return nil
}

func (b *Builder) trackProgressAndUpdate(ctx context.Context, workspace *spot.Workspace) error {
	return nil
}

func (b *Builder) completed(workspace *spot.Workspace) bool {
	return len(workspace.Status.Images) == 1
}

func (b *Builder) markWorkspaceHasErrored(ctx context.Context, workspace *spot.Workspace, err error) error {
	workspace.Status.Stage = spot.WorkspaceStageError
	return b.Client.Status().Update(ctx, workspace)
}
