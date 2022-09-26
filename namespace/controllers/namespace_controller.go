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
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1v1 "change-previewer.github.com/namespace/api/v1"
)

// NamespaceReconciler reconciles a Namespace object
type NamespaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=v1.change-previewer.github.io,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v1.change-previewer.github.io,resources=namespaces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v1.change-previewer.github.io,resources=namespaces/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=namespaces/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Namespace object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *NamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the v1v1.Namespace instance
	var namespace v1v1.Namespace
	if err := r.Get(ctx, req.NamespacedName, &namespace); err != nil {
		// log.Error(err, "unable to fetch Namespace")
		return ctrl.Result{Requeue: true}, client.IgnoreNotFound(err)
	}
	maxLifeTime := namespace.Spec.MaxLifeTime * time.Second

	// destroy the deployment if it is older than the max life time
	elapsedSinceCreation := time.Since(namespace.ObjectMeta.CreationTimestamp.Time)
	if elapsedSinceCreation > maxLifeTime {
		log.Info("object is older than max life time, destroying it")
		log.Info(fmt.Sprintf("object name: %s, kind: %s", namespace.ObjectMeta.Name, namespace.Kind))
		err := r.Delete(ctx, &namespace)
		if err != nil {
			log.Error(err, "unable to delete deployment")
			return ctrl.Result{Requeue: true}, err
		}
	}

	var childNamespace corev1.NamespaceList
	if err := r.List(ctx, &childNamespace, client.MatchingFields{ownerKey: req.Name}); err != nil {
		log.Error(err, "unable to list child namespaces")
		return ctrl.Result{}, err
	}

	if len(childNamespace.Items) == 0 {
		// Create a new child Namespace
		log.Info("Creating a new child Namespace", "Namespace", namespace.Name)
		knamespace, err := r.createNamespace(&namespace)
		if err != nil {
			log.Error(err, "unable to create Namespace")
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, knamespace); err != nil {
			log.Error(err, "unable to create KNamespace for Namespace", "ns", knamespace)
			if errors.IsAlreadyExists(err) {
				// set the existing namespace as the owner
				log.Info("Namespace already exists, owning it", "ns", knamespace)
				if err := ctrl.SetControllerReference(&namespace, knamespace, r.Scheme); err != nil {
					log.Error(err, "unable to set owner for KNamespace", "ns", knamespace)
					return ctrl.Result{}, err
				}
				return ctrl.Result{
					RequeueAfter: maxLifeTime - time.Since(namespace.ObjectMeta.CreationTimestamp.Time),
				}, nil
			}

			return ctrl.Result{
				Requeue: true,
			}, err
		}

		if err := ctrl.SetControllerReference(&namespace, knamespace, r.Scheme); err != nil {
			log.Error(err, "unable to set owner for KNamespace", "ns", knamespace)
			return ctrl.Result{
				RequeueAfter: maxLifeTime - time.Since(namespace.ObjectMeta.CreationTimestamp.Time),
			}, nil
		}

		log.V(1).Info("created KNamespace for Namespace", "job", knamespace)

		return ctrl.Result{}, nil
	}

	// namespaceFinalizer := "namespace.finalizers.change-previewer.github.io"
	// // if namespace is deleted, delete the child namespace
	// if !namespace.ObjectMeta.DeletionTimestamp.IsZero() {
	// 	if controllerutil.ContainsFinalizer(&namespace, namespaceFinalizer) {

	// 		log.Info("Deleting child Namespace", "Namespace", namespace.Name)
	// 		if err := r.Delete(ctx, &childNamespace.Items[0]); err != nil {
	// 			log.Error(err, "unable to delete child Namespace")
	// 			return ctrl.Result{}, err
	// 		}

	// 		controllerutil.RemoveFinalizer(&namespace, namespaceFinalizer)
	// 		if err := r.Update(ctx, &namespace); err != nil {
	// 			log.Error(err, "unable to remove finalizer from Namespace")
	// 			return ctrl.Result{}, err
	// 		}
	// 	}
	// } else {
	// 	if !controllerutil.ContainsFinalizer(&namespace, namespaceFinalizer) {
	// 		controllerutil.AddFinalizer(&namespace, namespaceFinalizer)
	// 		if err := r.Update(ctx, &namespace); err != nil {
	// 			return ctrl.Result{}, err
	// 		}
	// 	}
	// }

	remainingLifeTime := maxLifeTime - elapsedSinceCreation

	return ctrl.Result{
		RequeueAfter: remainingLifeTime,
	}, nil
}

func (r *NamespaceReconciler) createNamespace(namespace *v1v1.Namespace) (*corev1.Namespace, error) {
	var knamespace corev1.Namespace
	knamespace.ObjectMeta = metav1.ObjectMeta{
		Name:      namespace.Name,
		Namespace: namespace.Namespace,
		Labels:    namespace.Labels,
	}
	knamespace.Spec = *namespace.Spec.Template.DeepCopy()

	return &knamespace, nil
}

var (
	ownerKey = ".metadata.controller"
	apiGVStr = corev1.SchemeGroupVersion.String()
)

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Namespace{}, ownerKey, func(rawObj client.Object) []string {
		// grab the ns object, extract the owner...
		ns := rawObj.(*corev1.Namespace)
		owner := metav1.GetControllerOf(ns)
		if owner == nil {
			return nil
		}
		// ...make sure it's a NS...
		if (owner.APIVersion != apiGVStr && owner.APIVersion != "v1.change-previewer.github.io/v1") || owner.Kind != "Namespace" {
			fmt.Printf("owner is not a Namespace '%s' '%s'\n", owner.APIVersion, owner.Kind)
			return nil
		}

		// ...and if so, return it
		fmt.Println(owner.Name)
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1v1.Namespace{}).
		Owns(&corev1.Namespace{}).
		Complete(r)
}
