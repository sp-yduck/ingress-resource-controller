/*
Copyright 2023 Teppei Sudo.

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

package controllers

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	networkingv1beta1 "github.com/sp-yduck/ingress-resource-controller/api/v1beta1"
)

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",resources=services/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Service object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var svc v1.Service
	if err := r.Get(ctx, req.NamespacedName, &svc); err != nil {
		logger.Error(err, "unable to fetch service")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.reconcileService(ctx, svc); err != nil {
		logger.Error(err, "unable to reconcile service")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return ctrl.Result{}, nil
}

func (r *ServiceReconciler) reconcileService(ctx context.Context, service v1.Service) error {
	logger := log.FromContext(ctx)
	logger.Info("start reconcile service")

	ingressResource := &networkingv1beta1.IngressResource{}
	ingressResource.ObjectMeta.SetName(service.Name)
	ingressResource.ObjectMeta.SetNamespace(service.Namespace)
	// ingressResource.SetAnnotations()
	logger.Info(fmt.Sprintf("Name: %s, Namespace: %s", ingressResource.Name, ingressResource.Namespace))

	hostNameSuffix := "localhost"
	hostName := service.Name + "." + service.Namespace + "." + hostNameSuffix
	defaultPathType := networkingv1.PathType("Prefix")
	defaultIngressClass := "nginx"
	labels := make(map[string]string)
	labels["ingressresource.networking.sp-yduck.com/owner"] = service.Name
	op, err := ctrl.CreateOrUpdate(ctx, r.Client, ingressResource, func() error {
		ingressResource.Spec.Template.Spec.IngressClassName = &defaultIngressClass
		ingressResource.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
		ingressResource.Spec.Template.Spec.Rules = []networkingv1.IngressRule{
			{
				Host: hostName,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								Path:     "/",
								PathType: &defaultPathType,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: service.Name,
										Port: networkingv1.ServiceBackendPort{
											// Number: service.Spec.Ports[0].Port,
											Number: 80,
										},
									},
								},
							},
						},
					},
				},
			},
		}
		logger.Info(fmt.Sprintf("creating ingress resource: %v", ingressResource))
		return ctrl.SetControllerReference(&service, ingressResource, r.Scheme)
	})
	if err != nil {
		logger.Error(err, "unable to create or update IngressResource")
		return err
	}
	if op != controllerutil.OperationResultNone {
		logger.Info("reconcile service successfully", "op", op)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		For(&v1.Service{}).
		Complete(r)
}
