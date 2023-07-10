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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	ctrlRuntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// ReceiverReconciler reconciles a Receiver object
type ReceiverReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=spot.release.com,resources=receivers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=spot.release.com,resources=receivers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=spot.release.com,resources=receivers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Receiver object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *ReceiverReconciler) Reconcile(ctx context.Context, req ctrlRuntime.Request) (ctrlRuntime.Result, error) {
	deployment, err := r.reconcileDeployment(ctx)
	if err != nil {
		return ctrlRuntime.Result{}, err
	}

	service, err := r.reconcileService(ctx, deployment)
	if err != nil {
		return ctrlRuntime.Result{}, err
	}

	if err := r.reconcileIngress(ctx, service, deployment); err != nil {
		return ctrlRuntime.Result{}, err
	}

	return ctrlRuntime.Result{}, nil
}

func (r *ReceiverReconciler) reconcileService(ctx context.Context, _ *appsv1.Deployment) (*corev1.Service, error) {
	logger := log.FromContext(ctx)
	objectKey := client.ObjectKey{
		Name:      "receiver",
		Namespace: "spot-system",
	}

	var service corev1.Service

	if err := r.Client.Get(ctx, objectKey, &service); err != nil {
		logger.Info("service not present")

		service = corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "receiver",
				Namespace: "spot-system",
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"spot.release.com/pods": "receiver",
				},
				Ports: []corev1.ServicePort{{
					Protocol:   corev1.ProtocolTCP,
					Port:       3333,
					TargetPort: intstr.FromInt(3333),
				}},
			},
		}

		err = r.Client.Create(ctx, &service)

		if err != nil {
			return nil, err
		}
	}

	return &service, nil
}

func (r *ReceiverReconciler) reconcileIngress(ctx context.Context, _ *corev1.Service, deployment *appsv1.Deployment) error {
	logger := log.FromContext(ctx)
	objectKey := client.ObjectKey{
		Name:      "receiver",
		Namespace: "spot-system",
	}

	if err := r.Client.Get(ctx, objectKey, &networkingv1.Ingress{}); err != nil {
		logger.Info("ingress not present")

		ingressClassName := "nginx"
		pathType := networkingv1.PathTypePrefix

		ingress := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "receiver",
				Namespace: "spot-system",
			},
			Spec: networkingv1.IngressSpec{
				IngressClassName: &ingressClassName,
				Rules: []networkingv1.IngressRule{{
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{{
								Path:     "/",
								PathType: &pathType,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "receiver",
										Port: networkingv1.ServiceBackendPort{Number: 3333},
									},
								},
							}},
						},
					},
				}},
			},
		}

		return r.Client.Create(ctx, ingress)
	}

	return nil
}

func (r *ReceiverReconciler) reconcileDeployment(ctx context.Context) (*appsv1.Deployment, error) {
	logger := log.FromContext(ctx)

	var deployment appsv1.Deployment

	objectKey := client.ObjectKey{
		Name:      "receiver",
		Namespace: "spot-system",
	}

	if err := r.Client.Get(ctx, objectKey, &deployment); err != nil {
		logger.Info("deployment not present")

		var replicaCount int32 = 1

		//"public.ecr.aws/releasehub-com/receiver:latest"

		deployment = appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "receiver",
				Namespace: "spot-system",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicaCount,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"spot.release.com/pods": "receiver",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"spot.release.com/pods": "receiver",
						},
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: "spot-controller-manager", // TODO: Most likely to change spot-system/default to support the RBAC settings we need instead
						Containers: []corev1.Container{{
							Image:           "receiver:latest",
							ImagePullPolicy: corev1.PullNever,
							Name:            "receiver",
							Ports: []corev1.ContainerPort{{
								ContainerPort: 3333,
								Name:          "receiver-port",
							}},
						}},
					},
				},
			},
		}

		err = r.Client.Create(ctx, &deployment)

		if err != nil {
			return nil, err
		}
	}

	return &deployment, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReceiverReconciler) SetupWithManager(mgr ctrlRuntime.Manager) error {
	ctrl, err := controller.New("receiver-controller", mgr, controller.Options{Reconciler: r})

	if err != nil {
		return err
	}

	// Always need to make sure that we have a deployment fully configured to
	// receive webhooks.
	err = ctrl.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForObject{}, predicate.NewPredicateFuncs(func(obj client.Object) bool {
		return obj.GetName() == "receiver" && obj.GetNamespace() == "spot-system"
	}))

	err = ctrl.Watch(&source.Kind{Type: &networkingv1.Ingress{}}, &handler.EnqueueRequestForObject{}, predicate.NewPredicateFuncs(func(obj client.Object) bool {
		return obj.GetName() == "receiver" && obj.GetNamespace() == "spot-system"
	}))

	err = ctrl.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForObject{}, predicate.NewPredicateFuncs(func(obj client.Object) bool {
		return obj.GetName() == "receiver" && obj.GetNamespace() == "spot-system"
	}))

	return err
}
