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
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	cleanupv1 "github.com/operator/cleanup-operator/api/v1"
)

// CleanUpOperatorReconciler reconciles a CleanUpOperator object
type CleanUpOperatorReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

var finalizer_name = "custom/finalizer"

const (
	tridentOperatorName     = "trident-operator"
	localVolumeOperatorName = "local-storage-operator"
)

//+kubebuilder:rbac:groups=cleanup.ibm.com,resources=cleanupoperators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cleanup.ibm.com,resources=cleanupoperators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cleanup.ibm.com,resources=cleanupoperators/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CleanUpOperator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *CleanUpOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("cleanupoperator", req.NamespacedName)

	instance := &cleanupv1.CleanUpOperator{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("CleanUpOperator resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object.
		log.Error(err, "Failed to get CleanUpOperator")
		return ctrl.Result{}, err
	}

	template := instance.Spec.ResourceName
	version := instance.Spec.Version
	namespace := instance.Spec.Namespace

	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !containsString(instance.GetFinalizers(), finalizer_name) {
			controllerutil.AddFinalizer(instance, finalizer_name)
			if err := r.Update(ctx, instance); err != nil {
				log.Error(err, "Error is adding custom finalizer in CleanupOperator CR")
				return ctrl.Result{}, err
			}
			log.Info("custom finalizer added to CleanupOperator CR")
		} else {
			log.Info("custom finalizer already present in CleanupOperator CR")
		}

		switch {
		case template == "trident" && version == "20.07":
			tridentOperator := &appsv1.Deployment{}
			err = r.Get(ctx, types.NamespacedName{Name: tridentOperatorName, Namespace: namespace}, tridentOperator)
			if err != nil {
				if errors.IsNotFound(err) {
					log.Info("Trident Operator not found. Check again...", "name", tridentOperatorName)
					return ctrl.Result{Requeue: true}, nil
				}
				log.Error(err, "Failed to get Trident Operator", "name", tridentOperatorName)
				return ctrl.Result{}, err
			}
			if tridentOperator.ObjectMeta.DeletionTimestamp.IsZero() {
				if !containsString(tridentOperator.GetFinalizers(), finalizer_name) {
					controllerutil.AddFinalizer(tridentOperator, finalizer_name)
					if err := r.Update(ctx, tridentOperator); err != nil {
						log.Error(err, "Error is adding custom finalizer in Trident Operator ", "name", tridentOperatorName)
						return ctrl.Result{}, err
					}
					log.Info("custom finalizer added to Trident Operator", "name", tridentOperatorName)
				}
			}

		case template == "local-volume":
			localVolumeOperator := &appsv1.Deployment{}
			err = r.Get(ctx, types.NamespacedName{Name: localVolumeOperatorName, Namespace: namespace}, localVolumeOperator)
			if err != nil {
				if errors.IsNotFound(err) {
					log.Info("Local Volume Operator not found. Check again...", "name", localVolumeOperatorName)
					return ctrl.Result{Requeue: true}, nil
				}
				log.Error(err, "Failed to get Local Volume Operator", "name", localVolumeOperatorName)
				return ctrl.Result{}, err
			}
			if localVolumeOperator.ObjectMeta.DeletionTimestamp.IsZero() {
				if !containsString(localVolumeOperator.GetFinalizers(), finalizer_name) {
					controllerutil.AddFinalizer(localVolumeOperator, finalizer_name)
					if err := r.Update(ctx, localVolumeOperator); err != nil {
						log.Error(err, "Error is adding custom finalizer in Local Volume Operator ", "name", localVolumeOperatorName)
						return ctrl.Result{}, err
					}
					log.Info("custom finalizer added to Local Volume Operator", "name", localVolumeOperatorName)
				}
			}

		case template == "ocs-remote":
			log.Info("OCS cleanup not yet supported.")
			return ctrl.Result{}, nil
		}
	} else {
		// The object is being deleted
		if containsString(instance.GetFinalizers(), finalizer_name) {
			// Custom finalizer is present, so perform cleanup

			switch {
			case template == "trident" && version == "20.07":
				log.Info("Storage Template Cleanup started...", "name", template)

				err = r.patchCRs(ctx, namespace)
				if err != nil {
					// Failed to remove finalizer from CRs
					return ctrl.Result{}, err
				}
				err = r.removeCRDs(ctx)
				if err != nil {
					// Failed to perform CleanUp
					return ctrl.Result{}, err
				}

				tridentOpFound := true
				tridentOperator := &appsv1.Deployment{}
				err = r.Get(ctx, types.NamespacedName{Name: tridentOperatorName, Namespace: namespace}, tridentOperator)
				if err != nil {
					if errors.IsNotFound(err) {
						log.Info("Trident Operator not found. Ignoring...", "name", tridentOperatorName)
						tridentOpFound = false
					} else {
						log.Error(err, "Failed to get Trident Operator", "name", tridentOperatorName)
						return ctrl.Result{}, err
					}
				}
				if tridentOpFound {
					// remove custom finalizer from the trident Operator and update it.
					controllerutil.RemoveFinalizer(tridentOperator, finalizer_name)
					if err := r.Update(ctx, tridentOperator); err != nil {
						log.Error(err, "Error is removing custom finalizer from Trident Operator ", "name", tridentOperatorName)
						return ctrl.Result{}, err
					}
					log.Info("custom finalizer removed from Trident Operator", "name", tridentOperatorName)
				}
				log.Info("NetApp Trident Template Cleaned Successfully!!!")

			case template == "local-volume":
				log.Info("Storage Template Cleanup started...", "name", template)

				err = r.localVolumeCleanUp(ctx, namespace)
				if err != nil {
					// Failed to perform CleanUp
					return ctrl.Result{}, err
				}
				err = r.removeLocalVolmeCRDs(ctx)
				if err != nil {
					// Failed to remove CRDs
					return ctrl.Result{}, err
				}

				localVolumeOpFound := true
				localVolumeOperator := &appsv1.Deployment{}
				err = r.Get(ctx, types.NamespacedName{Name: localVolumeOperatorName, Namespace: namespace}, localVolumeOperator)
				if err != nil {
					if errors.IsNotFound(err) {
						log.Info("Local Volume Operator not found. Ignoring...", "name", localVolumeOperatorName)
						localVolumeOpFound = false
					} else {
						log.Error(err, "Failed to get Local Volume Operator", "name", localVolumeOperatorName)
						return ctrl.Result{}, err
					}
				}
				if localVolumeOpFound {
					// remove custom finalizer from the local volume Operator and update it.
					controllerutil.RemoveFinalizer(localVolumeOperator, finalizer_name)
					if err := r.Update(ctx, localVolumeOperator); err != nil {
						log.Error(err, "Error is removing custom finalizer from local volume Operator ", "name", localVolumeOperatorName)
						return ctrl.Result{}, err
					}
					log.Info("custom finalizer removed from local volume Operator", "name", localVolumeOperatorName)
				}
				log.Info("Local Volume Template Cleaned Successfully!!!")

			case template == "ocs-remote":
				log.Info("OCS cleanup not yet supported.")
				return ctrl.Result{}, nil
			}

			// remove custom finalizer from the resource and update it.
			controllerutil.RemoveFinalizer(instance, finalizer_name)
			if err := r.Update(ctx, instance); err != nil {
				log.Error(err, "Error is removing custom finalizer from CleanupOperator")
				return ctrl.Result{}, err
			}
			log.Info("custom finalizer removed from CleanupOperator")
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CleanUpOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cleanupv1.CleanUpOperator{}).
		WithEventFilter(ignoreDeletionPredicate()).
		Complete(r)
}

// Helper function to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func logFunctionDuration(logger logr.Logger, label string, start time.Time) {
	duration := time.Since(start)
	logger.Info("Time to complete", "method", label, "time", duration.Seconds())
}

// ExecuteCommand to execute shell commands
func ExecuteCommand(command string) (int, string, error) {
	fmt.Println("in ExecuteCommand running: ", command)
	var cmd *exec.Cmd
	var cmdErr bytes.Buffer
	var cmdOut bytes.Buffer
	cmdErr.Reset()
	cmdOut.Reset()

	cmd = exec.Command("bash", "-c", command)
	cmd.Stderr = &cmdErr
	cmd.Stdout = &cmdOut
	err := cmd.Run()

	var waitStatus syscall.WaitStatus

	errStr := strings.TrimSpace(cmdErr.String())
	outStr := strings.TrimSpace(cmdOut.String())

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus = exitError.Sys().(syscall.WaitStatus)
		}
		if errStr != "" {
			fmt.Println(command)
			fmt.Println(errStr)
		}
	} else {
		waitStatus = cmd.ProcessState.Sys().(syscall.WaitStatus)
	}
	if waitStatus.ExitStatus() == -1 {
		fmt.Print(time.Now().String() + " Timed out " + command)
	}
	return waitStatus.ExitStatus(), outStr, err
}

func ignoreDeletionPredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
	}
}
