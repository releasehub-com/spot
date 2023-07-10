package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	buildkit "github.com/releasehub-com/spot/builder/internal/buildkit"
	"github.com/releasehub-com/spot/builder/internal/buildkit/sources"
	spot "github.com/releasehub-com/spot/operator/api/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

func main() {
	var wg sync.WaitGroup

	ctx := context.Background()

	wg.Add(1)
	go func() {
		defer wg.Done()

		daemon := buildkit.NewDaemon(buildkit.WithStderr(os.Stdout))
		if err := daemon.Start(ctx); err != nil {
			fmt.Println(err)
		}
	}()

	time.Sleep(2 * time.Second)
	buildImage, err := sources.HardCodedBuildFromGithub(ctx)
	if err != nil {
		panic(err)
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()
	config.ContentConfig.GroupVersion = &spot.GroupVersion
	config.APIPath = "/apis"

	client, err := rest.RESTClientFor(config)
	if err != nil {
		panic(err.Error())
	}

	spot.AddToScheme(scheme.Scheme)

	references := strings.Split(os.Getenv("BUILD_REFERENCE"), "/")
	if len(references) != 2 {
		panic(fmt.Sprintf("BUILD_REFERENCE is expected to have 2 components, had %d: %s", len(references), os.Getenv("BUILD_REFERENCE")))
	}

	var build spot.Build
	req := client.Get().Resource("builds").Namespace(references[0]).Name(references[1])
	result := req.Do(context.TODO())

	if err := result.Error(); err != nil {
		panic(fmt.Sprintf("Error trying to get the build CRD: %v", err))
	}

	err = result.Into(&build)
	if err != nil {
		panic(fmt.Sprintf("Error trying format the build: %v", err))
	}

	build.Status.Stage = spot.BuildStageDone
	build.Status.Image = buildImage
	result = client.Put().Resource("builds").SubResource("status").Namespace(build.Namespace).Name(build.Name).Body(&build).Do(ctx)
	if err = result.Error(); err != nil {
		panic(fmt.Sprintf("Error updating build: %v", err))
	}

	fmt.Print("See ya!")
	// Just for now to get thing moving.
	os.Exit(0)
}
