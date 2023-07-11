package v1alpha1

type ImageSpec struct {
	// Name of the component inside the workspace that
	// needs to build this. It needs to be a unique name inside a workspace.
	// This is used to update the workspace with the image information
	// when the image is ready.
	Name string `json:"name,omitempty"`

	// Tag is what will be used to tag the image once it's
	// pushed to the container's registry (ecr, etc.)
	// If no tag is set, it will use the workspace tag
	// This can be useful if a workspace builds multiple images
	// and each of the images will be tagged the same value.
	//+optional
	Tag *string `json:"tag,omitempty"`

	// RepositoryContext information is passed down to buildkit
	// as instruction on how to proceed with the repository
	// If the RepositoryContext is not set, it will use the default:
	// 		Dockerfile: "Dockerfile"
	//  	Path: "."
	//+optional
	RepositoryContext *RepositoryContextSpec `json:"repository_context"`

	// Registry is where all the information for the container
	// Registry is located.
	Registry RegistrySpec `json:"registry,omitempty"`
}

type RepositoryContextSpec struct {
	// Location of your Dockerfile within the repository.
	Dockerfile string `json:"dockerfile"`

	// Path is what docker calls `context`. It's the location
	// for the content of your build within the repository.
	Path string `json:"path"`
}
