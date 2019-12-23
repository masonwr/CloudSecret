/*

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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	secretsv1 "github.com/masonwr/CloudSecret/api/v1"
)

// CloudSecretReconciler reconciles a CloudSecret object
type CloudSecretReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const secretPrefix = "cloudsecret-child"
const finalizer = "cloudsecret.finalizers"

// +kubebuilder:rbac:groups=secrets.masonwr.dev,resources=cloudsecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=secrets.masonwr.dev,resources=cloudsecrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=secret,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secret/status,verbs=get
func (r *CloudSecretReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("cloudsecret", req.NamespacedName)

	// fetch cloud secret object
	var cloudSecret secretsv1.CloudSecret
	if err := r.Get(ctx, req.NamespacedName, &cloudSecret); err != nil {
		log.Error(err, "unable to fetch cloud secret")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	childSecretKey := types.NamespacedName{
		Name:      secretPrefix + req.Name,
		Namespace: req.Namespace,
	}

	var childSecret corev1.Secret
	if err := r.Get(ctx, childSecretKey, &childSecret); err != nil {
		log.Info("creating new child secret")
		childSecret.Name = childSecretKey.Name
		childSecret.Namespace = childSecretKey.Namespace

		err := r.Create(ctx, &childSecret)
		if err != nil {
			log.Error(err, "unable to create secret")
			return ctrl.Result{}, err
		}
	}

	// init and copy data to child (or native) k8s secret
	if childSecret.Data == nil {
		childSecret.Data = make(map[string][]byte)
	}

	for k, v := range cloudSecret.Spec.Data {
		childSecret.Data[k] = []byte(v)
	}

	log.Info("updating child secret")
	err := r.Update(ctx, &childSecret)
	if err != nil {
		log.Error(err, "unable to update child secret")
		return ctrl.Result{}, err
	}

	// register/deregister finalizer
	if cloudSecret.ObjectMeta.DeletionTimestamp.IsZero() {
		// is the cloud secret does not have the finalizer, add it
		if !containsString(cloudSecret.ObjectMeta.Finalizers, finalizer) {
			log.Info("setting finalizer")

			cloudSecret.ObjectMeta.Finalizers = append(cloudSecret.ObjectMeta.Finalizers, finalizer)
			if err := r.Update(ctx, &cloudSecret); err != nil {
				log.Error(err, "unable to add finalizer to cloud secret")
			}
		}
	} else { // cloud secrete is being destroyed
		if containsString(cloudSecret.ObjectMeta.Finalizers, finalizer) {
			if err := r.Delete(ctx, &childSecret); err != nil {
				log.Error(err, "unable to delete child secret")
			}

			cloudSecret.ObjectMeta.Finalizers = removeString(cloudSecret.ObjectMeta.Finalizers, finalizer)
			if err := r.Update(ctx, &cloudSecret); err != nil {
				log.Error(err, "unable to remove finalizer to clud secret")
			}
		}

		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *CloudSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&secretsv1.CloudSecret{}).
		Complete(r)
}

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
