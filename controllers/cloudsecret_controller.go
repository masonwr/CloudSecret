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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		Name:      cloudSecret.GetName(),
		Namespace: req.Namespace,
	}

	var childSecret corev1.Secret
	if err := r.Get(ctx, childSecretKey, &childSecret); err != nil {
		log.Info("creating new child secret")

		childSecret.Name = childSecretKey.Name
		childSecret.Namespace = childSecretKey.Namespace
<<<<<<< HEAD
		childSecret.SetOwnerReferences([]metav1.OwnerReference{
			metav1.OwnerReference{
				APIVersion: cloudSecret.APIVersion,
				Kind:       cloudSecret.Kind,
				Name:       cloudSecret.Name,
				UID:        cloudSecret.UID,
			},
		})

		if err := r.Create(ctx, &childSecret); err != nil {
			log.Error(err, "unable to create child secret")
=======

		ownerRef := metav1.OwnerReference{
			APIVersion: cloudSecret.TypeMeta.APIVersion,
			Kind:       cloudSecret.TypeMeta.Kind,
			Name:       cloudSecret.Name,
			UID:        cloudSecret.UID,
		}

		childSecret.OwnerReferences = []metav1.OwnerReference{ownerRef}

		err := r.Create(ctx, &childSecret)
		if err != nil {
			log.Error(err, "unable to create secret")
>>>>>>> 7728966d15a9d8258ee506bf4eec17b48602a037
			return ctrl.Result{}, err
		}
	}

	// init and copy data to child k8s secret
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

	return ctrl.Result{}, nil
}

func (r *CloudSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&secretsv1.CloudSecret{}).
		Complete(r)
}
