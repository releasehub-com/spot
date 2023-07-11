package v1alpha1

type RegistrySpec struct {
	// URL is where the container registry is located (ecr, docker, google, etc.)
	URL string `json:"url"`
}
