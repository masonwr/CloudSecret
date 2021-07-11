/*
Copyright 2021.

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

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretsv1 "github.com/masonwr/CloudSecret/api/v1"
	secrets "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const defaultRetryTime = time.Duration(5) * time.Second

// CloudSecretReconciler reconciles a CloudSecret object
type CloudSecretReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	GcpSecrets *secretmanager.Client
}

//+kubebuilder:rbac:groups=secrets.masonwr.dev,resources=cloudsecrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=secrets.masonwr.dev,resources=cloudsecrets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=secrets.masonwr.dev,resources=cloudsecrets/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CloudSecret object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *CloudSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// fetch cloud secret object
	var cloudSecret secretsv1.CloudSecret
	if err := r.Get(ctx, req.NamespacedName, &cloudSecret); err != nil {
		log.Error(err, "unable to fetch cloud secret")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	recDelay := time.Duration(cloudSecret.Spec.SyncPeriod) * time.Second
	result := ctrl.Result{RequeueAfter: recDelay}

	// fetch associated k8s (child) secret, creating it if not found
	var childSecret corev1.Secret
	if err := r.Get(ctx, cloudSecret.GetChildSecretKey(), &childSecret); err != nil {
		log.Info("creating child secret")
		childSecret = cloudSecret.InitChildSecret()
		if err := r.Create(ctx, &childSecret); err != nil {
			log.Error(err, "unable to create child secret")
			return ctrl.Result{}, err
		}
	}

	if cloudSecret.Status.SecretResolution == nil {
		cloudSecret.Status.SecretResolution = make(map[string]string)
	}

	// nothing left todo
	if len(cloudSecret.Spec.Data) == 0 {
		log.Info("empty cloud secret")
		return result, nil
	}

	// init and copy data to child k8s secret
	getSecretFail := false
	childSecret.Data = make(map[string][]byte)
	for k, v := range cloudSecret.Spec.Data {
		access, err := r.GcpSecrets.AccessSecretVersion(ctx, &secrets.AccessSecretVersionRequest{Name: v})
		if err != nil {
			log.Info("could not access secret")
			log.Error(err, "unable to access secret", "secret_path", v)
			cloudSecret.Status.SecretResolution[k] = fmt.Sprintf("%s", err)
			getSecretFail = true
			continue
		}

		cloudSecret.Status.SecretResolution[k] = "RESOLVED"
		childSecret.Data[k] = access.Payload.GetData()
	}

	if err := r.Status().Update(ctx, &cloudSecret); err != nil {
		log.Error(err, "error updating cloudsecret")
	}

	// if we failed to fetch a secret we retry by re-queuing.
	// NB: the secret manager client by deault will retry as well.
	if getSecretFail && recDelay > defaultRetryTime {
		result.RequeueAfter = defaultRetryTime
	}

	// if no secrets were resolved, we delete the child secret
	// in an attempt to fail with more noise.
	if len(childSecret.Data) == 0 {
		if err := r.Delete(ctx, &childSecret); err != nil {
			log.Error(err, "unable to delete child secret")
			return ctrl.Result{}, err
		}

		return result, nil
	}

	log.Info("updating child secret")
	if err := r.Update(ctx, &childSecret); err != nil {
		log.Error(err, "unable to update child secret")
		return ctrl.Result{}, err
	}

	return result, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloudSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&secretsv1.CloudSecret{}).
		Complete(r)
}
