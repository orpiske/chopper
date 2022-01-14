/*
Copyright 2022.

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
	"github.com/go-logr/logr"
	v1 "k8s.io/api/apps/v1"
	v1core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	chopperv1alpha1 "github.com/orpiske/chopper/api/v1alpha1"
)

// OffsetStoreReconciler reconciles a OffsetStore object
type OffsetStoreReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=chopper.camel.apache.org,resources=offsetstores,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=chopper.camel.apache.org,resources=offsetstores/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=chopper.camel.apache.org,resources=offsetstores/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OffsetStore object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *OffsetStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	offsetStore := &chopperv1alpha1.OffsetStore{}

	if err := r.Get(ctx, req.NamespacedName, offsetStore); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Get failed because the resource was not found")
			return ctrl.Result{}, nil
		} else {
			logger.Error(err, "Get failed for other reasons")
		}
		return ctrl.Result{}, err
	}

	var deploy v1.Deployment
	var roleNamespace = types.NamespacedName{
		Namespace: req.Namespace,
		Name:      "zookeeper-store",
	}

	defaultDeploy := v1.Deployment{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "zookeeper-store",
			Namespace: req.Namespace,
			Labels:    map[string]string{"app": "chopper"},
		},
		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "chopper"},
			},
			Template: v1core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "chopper"},
				},
				Spec: v1core.PodSpec{
					Containers: []v1core.Container{{
						Name:  offsetStore.Spec.Store.Type + "-store",
						Image: offsetStore.Spec.Image,
						Ports: []v1core.ContainerPort{
							{
								ContainerPort: 2181,
								Protocol:      "TCP",
							},
						},
						Resources: v1core.ResourceRequirements{},
					},
					},
				},
			},
		},
	}

	if err := r.Get(ctx, roleNamespace, &deploy); err != nil {

		if errors.IsNotFound(err) {
			logger.Info("Deploying", "store", offsetStore.Spec.Store.Type)
			deploy = defaultDeploy

			if createErr := r.Create(ctx, &deploy); createErr != nil {
				logger.Error(createErr, "Failed to deploy store")
				return ctrl.Result{}, createErr
			}
		}
	} else {
		if updateErr := r.Update(ctx, &defaultDeploy); updateErr != nil {
			logger.Error(updateErr, "Failed to update store deployment")
			return ctrl.Result{}, updateErr
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OffsetStoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chopperv1alpha1.OffsetStore{}).
		Complete(r)
}
