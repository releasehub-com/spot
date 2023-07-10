package stages

import (
	"context"

	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	spot "github.com/releasehub-com/spot/operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Deployment struct {
	client.Client
}

func (d *Deployment) Start(ctx context.Context, workspace *spot.Workspace) error {
	mysql := core.Pod{
		ObjectMeta: meta.ObjectMeta{
			GenerateName: "mysql-",
			Namespace:    workspace.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name": "mysql",
			},
			OwnerReferences: []meta.OwnerReference{
				{
					APIVersion: workspace.APIVersion,
					Kind:       workspace.Kind,
					Name:       workspace.Name,
					UID:        workspace.UID,
				},
			},
		},
		Spec: core.PodSpec{
			RestartPolicy: core.RestartPolicyNever,
			Containers: []core.Container{
				{
					Name:  "mysql",
					Image: "mysql",
					Ports: []core.ContainerPort{
						{
							Name:          "mysql",
							HostPort:      3306,
							ContainerPort: 3306,
						},
					},
					Env: []core.EnvVar{
						{
							Name:  "MYSQL_DATABASE",
							Value: "click-me",
						},
						{
							Name:  "MYSQL_USER",
							Value: "big",
						},
						{
							Name:  "MYSQL_PASSWORD",
							Value: "lebowski",
						},
						{
							Name:  "MYSQL_ROOT_PASSWORD",
							Value: "Yeah, well, that is just, like, your opinion, man.",
						},
					},
				},
			},
		},
	}

	if err := d.Client.Create(ctx, &mysql); err != nil {
		return err
	}

	mysqlService := core.Service{
		ObjectMeta: meta.ObjectMeta{
			GenerateName: "mysql-service-",
			Namespace:    workspace.Namespace,
			OwnerReferences: []meta.OwnerReference{
				{
					APIVersion: workspace.APIVersion,
					Kind:       workspace.Kind,
					Name:       workspace.Name,
					UID:        workspace.UID,
				},
			},
		},
		Spec: core.ServiceSpec{
			Selector: map[string]string{
				"app.kubernetes.io/name": "mysql",
			},
			Ports: []core.ServicePort{
				{
					Name:       "mysql",
					Port:       3306,
					TargetPort: intstr.FromInt(3306),
				},
			},
		},
	}

	if err := d.Client.Create(ctx, &mysqlService); err != nil {
		return err
	}

	click := core.Pod{
		ObjectMeta: meta.ObjectMeta{
			GenerateName: "click-mania-",
			Namespace:    workspace.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name": "click-mania",
			},
			OwnerReferences: []meta.OwnerReference{
				{
					APIVersion: workspace.APIVersion,
					Kind:       workspace.Kind,
					Name:       workspace.Name,
					UID:        workspace.UID,
				},
			},
		},
		Spec: core.PodSpec{
			RestartPolicy: core.RestartPolicyAlways,
			Containers: []core.Container{
				{
					Name:  "click-mania",
					Image: workspace.Status.Images["click-mania"].URL,
					Ports: []core.ContainerPort{
						{
							Name:          "server",
							HostPort:      3000,
							ContainerPort: 3000,
						},
					},
					Env: []core.EnvVar{
						{
							Name:  "DB_NAME",
							Value: "click-me",
						},
						{
							Name:  "DB_USER",
							Value: "big",
						},
						{
							Name:  "DB_PASSWORD",
							Value: "lebowski",
						},
						{
							Name:  "DB_HOST",
							Value: mysqlService.Name,
						},
					},
					Command: []string{"wait-for-it", "mysql:3306", "--", "/srv/aurora-test", "start"},
				},
			},
		},
	}

	if err := d.Client.Create(ctx, &click); err != nil {
		return err
	}

	clickService := core.Service{
		ObjectMeta: meta.ObjectMeta{
			Name:      "click-mania",
			Namespace: workspace.Namespace,
			OwnerReferences: []meta.OwnerReference{
				{
					APIVersion: workspace.APIVersion,
					Kind:       workspace.Kind,
					Name:       workspace.Name,
					UID:        workspace.UID,
				},
			},
		},
		Spec: core.ServiceSpec{
			Selector: map[string]string{
				"app.kubernetes.io/name": "click-mania",
			},
			Ports: []core.ServicePort{
				{
					Name:       "click-mania",
					Port:       3000,
					TargetPort: intstr.FromInt(3000),
				},
			},
		},
	}

	if err := d.Client.Create(ctx, &clickService); err != nil {
		return err
	}

	ingressClassName := "nginx"
	pathType := networking.PathTypePrefix

	ingress := &networking.Ingress{
		ObjectMeta: meta.ObjectMeta{
			Name:      "click-mania",
			Namespace: workspace.Namespace,
			OwnerReferences: []meta.OwnerReference{
				{
					APIVersion: workspace.APIVersion,
					Kind:       workspace.Kind,
					Name:       workspace.Name,
					UID:        workspace.UID,
				},
			},
		},
		Spec: networking.IngressSpec{
			IngressClassName: &ingressClassName,
			Rules: []networking.IngressRule{{
				Host: "click-mania.po.ngrok.app",
				IngressRuleValue: networking.IngressRuleValue{
					HTTP: &networking.HTTPIngressRuleValue{
						Paths: []networking.HTTPIngressPath{{
							Path:     "/",
							PathType: &pathType,
							Backend: networking.IngressBackend{
								Service: &networking.IngressServiceBackend{
									Name: "click-mania",
									Port: networking.ServiceBackendPort{Number: 3000},
								},
							},
						}},
					},
				},
			}},
		},
	}

	if err := d.Client.Create(ctx, ingress); err != nil {
		return err
	}

	workspace.Status.Stage = spot.WorkspaceStageRunning

	return d.Client.SubResource("status").Update(ctx, workspace)
}
