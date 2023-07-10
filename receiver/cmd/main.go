package main

import (
	"fmt"
	"net/http"

	"github.com/releasehub-com/spot/receiver/internal/controllers"

	spot "github.com/releasehub-com/spot/operator/api/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()
	config.ContentConfig.GroupVersion = &spot.GroupVersion
	config.APIPath = "/apis"

	fmt.Println("Config: ", config)
	client, err := rest.RESTClientFor(config)
	if err != nil {
		panic(err.Error())
	}

	spot.AddToScheme(scheme.Scheme)

	fmt.Printf("Starting the server\n")
	http.Handle("/", &controllers.Workspace{Client: client})
	http.ListenAndServe(":3333", nil)
}
