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

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	networkingv1beta1 "github.com/sp-yduck/ingress-resource-controller/api/v1beta1"
)

// IngressResourceReconciler reconciles a IngressResource object
type IngressResourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=networking.sp-yduck.com,resources=ingressresources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.sp-yduck.com,resources=ingressresources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=networking.sp-yduck.com,resources=ingressresources/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the IngressResource object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *IngressResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var ingressResource networkingv1beta1.IngressResource
	if err := r.Get(ctx, req.NamespacedName, &ingressResource); err != nil {
		logger.Error(err, "unable to fetch ingressResource")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var ingressResourceList networkingv1beta1.IngressResourceList
	if err := r.List(ctx, &ingressResourceList, client.InNamespace(req.Namespace), &client.ListOptions{}); err != nil {
		logger.Error(err, "unable to list ingressResources")
		return ctrl.Result{}, err
	}

	if err := r.reconcileIngress(ctx, ingressResource); err != nil {
		logger.Error(err, "unable to reconcile ingresss")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1beta1.IngressResource{}).
		Complete(r)
}

func (r *IngressResourceReconciler) reconcileIngress(ctx context.Context, ingressResource networkingv1beta1.IngressResource) error {
	logger := log.FromContext(ctx)

	ingress := &networkingv1.Ingress{}
	ingress.SetNamespace(ingressResource.Namespace)
	ingress.SetName(ingressResource.Name)
	op, err := ctrl.CreateOrUpdate(ctx, r.Client, ingress, func() error {
		ingress.Spec = ingressResource.Spec.Template.Spec
		return ctrl.SetControllerReference(&ingressResource, ingress, r.Scheme)
	})
	if err != nil {
		logger.Error(err, "unable to create or update ingress")
		return err
	}
	if op != controllerutil.OperationResultNone {
		logger.Info("reconcile ingress successfully", "op", op)
	}
	return nil
}
