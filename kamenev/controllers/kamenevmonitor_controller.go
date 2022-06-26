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
	"time"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	changepreviewerv1 "com.github.felipemarinho97/kamenev/api/v1"
)

// KamenevMonitorReconciler reconciles a KamenevMonitor object
type KamenevMonitorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=change-previewer.com.github.felipemarinho97,resources=kamenevmonitors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=change-previewer.com.github.felipemarinho97,resources=kamenevmonitors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=change-previewer.com.github.felipemarinho97,resources=kamenevmonitors/finalizers,verbs=update
//+kubebuilder:rbac:groups=deployment.apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KamenevMonitor object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *KamenevMonitorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the KamenevMonitor instance
	instances := &changepreviewerv1.KamenevMonitorList{}
	err := r.Client.List(ctx, instances, client.InNamespace(req.Namespace), client.Limit(1))
	if err != nil || len(instances.Items) == 0 {
		log.Error(err, "unable to fetch KamenevMonitor")
		if errors.IsNotFound(err) {
			// Object not found, return. Created objects are automatically
			// garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{
			RequeueAfter: time.Second * 1,
		}, err
	}
	instance := instances.Items[0]
	maxLifeTime := instance.Spec.MaxLifeTime * time.Second

	if req.NamespacedName.Name == instance.Name {
		return ctrl.Result{}, nil
	}

	var deployment v1.Deployment

	err = r.Get(ctx, req.NamespacedName, &deployment)
	if err != nil {
		log.Error(err, "unable to get deployment")
		return ctrl.Result{}, err
	}

	// destroy the deployment if it is older than the max life time
	elapsedSinceCreation := time.Since(deployment.ObjectMeta.CreationTimestamp.Time)
	if elapsedSinceCreation > maxLifeTime {
		log.Info("deployment is older than max life time, destroying it")
		err := r.Delete(ctx, &deployment)
		if err != nil {
			log.Error(err, "unable to delete deployment")
			return ctrl.Result{}, err
		}
	}

	remainingLifeTime := maxLifeTime - elapsedSinceCreation

	return ctrl.Result{
		RequeueAfter: remainingLifeTime,
	}, nil
}

func (r *KamenevMonitorReconciler) findObjectsForDeployment(deploy client.Object) []reconcile.Request {
	attachedDeployments := &v1.DeploymentList{}
	listOps := &client.ListOptions{
		Namespace: deploy.GetNamespace(),
		LabelSelector: labels.SelectorFromSet(map[string]string{
			"com.github.felipemarinho97.change-previewer/enabled": "true",
		}),
	}
	err := r.List(context.TODO(), attachedDeployments, listOps)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(attachedDeployments.Items))
	for i, item := range attachedDeployments.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}

// SetupWithManager sets up the controller with the Manager.
func (r *KamenevMonitorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&changepreviewerv1.KamenevMonitor{}).
		Owns(&v1.Deployment{}).
		Watches(
			&source.Kind{Type: &v1.Deployment{}},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForDeployment),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}
