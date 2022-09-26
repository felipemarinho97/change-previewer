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
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1v1 "change-previewer.github.com/namespace/api/v1"
)

// EphemeralNamespaceReconciler reconciles a EphemeralNamespace object
type EphemeralNamespaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var namespaceFinalizer = "namespace.finalizers.change-previewer.github.io"

//+kubebuilder:rbac:groups=v1.change-previewer.github.io,resources=ephemeralnamespaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v1.change-previewer.github.io,resources=ephemeralnamespaces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v1.change-previewer.github.io,resources=ephemeralnamespaces/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=namespaces/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EphemeralNamespace object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *EphemeralNamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	obj := &v1v1.EphemeralNamespace{}
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		return ctrl.Result{Requeue: true}, client.IgnoreNotFound(err)
	}
	log = log.WithValues("resourceVersion", obj.ResourceVersion)

	defer func(t time.Time) {
		log.Info(fmt.Sprintf("Cluster has been reconciled in %v", time.Since(t)))
	}(time.Now())

	d, err := r.reconcile(ctx, obj)
	if err := r.Status().Update(ctx, obj); client.IgnoreNotFound(err) != nil {
		log.Error(err, "failed to update Cluster status")
	}
	return ctrl.Result{Requeue: true, RequeueAfter: d}, err
}

func (r *EphemeralNamespaceReconciler) reconcile(ctx context.Context, namespace *v1v1.EphemeralNamespace) (time.Duration, error) {
	log := log.FromContext(ctx)
	log.Info("reconcile")

	r.addFinalizer(ctx, namespace)
	r.finalize(ctx, namespace)

	maxLifeTime := namespace.Spec.MaxLifeTime * time.Second

	// destroy the namespace if it is older than the max life time
	elapsedSinceCreation := time.Since(namespace.ObjectMeta.CreationTimestamp.Time)
	if elapsedSinceCreation > maxLifeTime {
		log.Info("object is older than max life time, destroying it")
		log.Info(fmt.Sprintf("object name: %s, kind: %s", namespace.ObjectMeta.Name, namespace.Kind))
		err := r.Delete(ctx, namespace)
		if err != nil {
			log.Error(err, "unable to delete namespace")
			return 0, err
		}

		err = r.finalize(ctx, namespace)
		if err != nil {
			log.Error(err, "unable to finalize Namespace")
			return 0, err
		}
		return 0, nil
	}

	remainingLifeTime := maxLifeTime - elapsedSinceCreation

	var childNamespace corev1.NamespaceList
	// if err := r.List(ctx, &childNamespace, client.MatchingFields{ownerKey: namespace.Name}); err != nil {
	// 	log.Error(err, "unable to list child namespaces")
	// 	return remainingLifeTime, err
	// }
	if err := r.List(ctx, &childNamespace, client.MatchingLabels{"change-previewer.github.io/ephemeral-namespace": namespace.Name}); err != nil {
		log.Error(err, "unable to list child namespaces")
		return remainingLifeTime, err
	}

	if len(childNamespace.Items) == 0 {
		// Create a new child Namespace
		log.Info("Creating a new child Namespace", "Namespace", namespace.Name)
		knamespace, err := r.createNamespace(namespace)
		if err != nil {
			log.Error(err, "unable to create Namespace")
			return 0, err
		}

		err = r.Create(ctx, knamespace)
		if err != nil {
			log.Error(err, "unable to create KNamespace for Namespace", "ns", knamespace)
			if errors.IsAlreadyExists(err) {
				// set the existing namespace as the owner
				log.Info("Namespace already exists, owning it", "ns", knamespace)
				if err := ctrl.SetControllerReference(namespace, knamespace, r.Scheme); err != nil {
					log.Error(err, "unable to set owner for KNamespace", "ns", knamespace)
					return 0, err
				}
				return remainingLifeTime, nil
			}

			return 0, err
		}

		if err := ctrl.SetControllerReference(namespace, knamespace, r.Scheme); err != nil {
			log.Error(err, "unable to set owner for KNamespace", "ns", knamespace)
			return remainingLifeTime, nil
		}

		log.V(1).Info("created KNamespace for Namespace", "job", knamespace)

		return remainingLifeTime, nil
	} else {
		log.Info("Namespace already exists!!!!!!!!!!!!!!!!!!!!!!!!!")
	}

	// if namespace is deleted, finalize it
	r.finalize(ctx, namespace)

	return remainingLifeTime, nil
}

func (r *EphemeralNamespaceReconciler) removeFinalizer(ctx context.Context, namespace *v1v1.EphemeralNamespace) error {
	log := log.FromContext(ctx)
	controllerutil.RemoveFinalizer(namespace, namespaceFinalizer)
	if err := r.Update(ctx, namespace); err != nil {
		log.Error(err, "unable to remove finalizer from Namespace")
		return err
	}
	return nil
}

func (r *EphemeralNamespaceReconciler) addFinalizer(ctx context.Context, namespace *v1v1.EphemeralNamespace) error {
	log := log.FromContext(ctx)
	if !controllerutil.ContainsFinalizer(namespace, namespaceFinalizer) {
		controllerutil.AddFinalizer(namespace, namespaceFinalizer)
		if err := r.Update(ctx, namespace); err != nil {
			log.Error(err, "unable to add finalizer to Namespace")
			return err
		}
	}
	return nil
}

func (r *EphemeralNamespaceReconciler) finalize(ctx context.Context, namespace *v1v1.EphemeralNamespace) error {
	log := log.FromContext(ctx)
	if !namespace.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(namespace, namespaceFinalizer) {
			var childNamespace corev1.Namespace
			if err := r.Get(ctx, types.NamespacedName{Name: namespace.Name}, &childNamespace); err != nil {
				log.Error(err, "unable to get child Namespace")
				return err
			}

			log.Info("Deleting child Namespace", "Namespace", childNamespace.Name)
			if err := r.Delete(ctx, &childNamespace); err != nil {
				log.Error(err, "unable to delete child Namespace")
				return err
			}

			err := r.removeFinalizer(ctx, namespace)
			if err != nil {
				log.Error(err, "unable to remove finalizer")
				return err
			}
		} else {
			log.Info("Namespace is already finalized")
		}
	} else {
		log.Info("Namespace is not being deleted")
		r.addFinalizer(ctx, namespace)
	}
	return nil
}

func (r *EphemeralNamespaceReconciler) createNamespace(namespace *v1v1.EphemeralNamespace) (*corev1.Namespace, error) {
	var knamespace corev1.Namespace
	knamespace.ObjectMeta = metav1.ObjectMeta{
		Name:      namespace.Name,
		Namespace: namespace.Namespace,
		Labels: map[string]string{
			"change-previewer.github.io/ephemeral-namespace": namespace.Name,
		},
	}
	knamespace.Spec = *namespace.Spec.Template.DeepCopy()

	return &knamespace, nil
}

var fieldKey = ".metadata.controllerr"

// SetupWithManager sets up the controller with the Manager.
func (r *EphemeralNamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Namespace{}, ownerKey, func(rawObj client.Object) []string {
	// 	// grab the ns object, extract the owner...
	// 	ns := rawObj.(*corev1.Namespace)
	// 	owner := metav1.GetControllerOf(ns)
	// 	if owner == nil {
	// 		return nil
	// 	}
	// 	// ...make sure it's a NS...
	// 	if owner.APIVersion != apiGVStr || owner.Kind != "Namespace" {
	// 		fmt.Printf("owner is not a Namespace '%s' '%s'\n", owner.APIVersion, owner.Kind)
	// 		return nil
	// 	}

	// 	// ...and if so, return it
	// 	return []string{owner.Name}
	// }); err != nil {
	// 	return err
	// }

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1v1.EphemeralNamespace{}).
		Owns(&corev1.Namespace{}).
		Complete(r)
}
