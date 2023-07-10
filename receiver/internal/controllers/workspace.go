package controllers

import (
	"context"
	"fmt"
	"net/http"

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	spot "github.com/releasehub-com/spot/operator/api/v1alpha1"
)

type Workspace struct {
	Client rest.Interface
}

func (w *Workspace) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	var project spot.Project

	r := w.Client.Get().Resource("projects").Namespace("spot-system").Name("spacedust")
	fmt.Println("Path: ", r.URL())

	result := r.Do(context.TODO())

	if err := result.Error(); err != nil {
		fmt.Println("Error trying to get the project CRD: ", err)
		return
	}

	err := result.Into(&project)
	if err != nil {
		fmt.Println("Error trying format the receiver: ", err)
		return
	}

	fmt.Printf("Project: %+v\n", project)

	workspace, err := w.workspace(&project, "rel-1234-my-testing")

	if err != nil {
		fmt.Println("Error trying to get the list of workspaces: ", err)
	}

	fmt.Printf("Workspaces: %+v\n", workspace)
}

func (w *Workspace) workspace(project *spot.Project, name string) (*spot.Workspace, error) {
	var workspaces spot.Workspace

	request := w.Client.Get().Resource("workspaces").Namespace("spot-system").Name(name)
	fmt.Println("Path: ", request.URL())

	result := request.Do(context.TODO())
	if err := result.Error(); err != nil {
		if errors.IsNotFound(err) {
			fmt.Println("Not Found, creating a new workspace: ")
			return w.createWorkspace(project, name)
		}
		return nil, err
	}

	err := result.Into(&workspaces)
	return &workspaces, err
}

func (w *Workspace) createWorkspace(project *spot.Project, name string) (*spot.Workspace, error) {
	var workspace spot.Workspace

	err := w.Client.
		Post().
		Resource("workspaces").
		Namespace("spot-system").
		Body(&spot.Workspace{
			ObjectMeta: v1.ObjectMeta{
				Name:      name,
				Namespace: project.Namespace,
			},
			Spec: spot.WorkspaceSpec{},
		}).
		Do(context.TODO()).
		Into(&workspace)

	return &workspace, err
}
