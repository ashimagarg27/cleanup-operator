package controllers

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	tridentv1 "github.com/netapp/trident/persistent_store/crd/apis/netapp/v1"
	corev1 "k8s.io/api/core/v1"
	apiextenstionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// removeCRDs patches and deletes all trident crds
func (cr *CleanUpOperatorReconciler) removeCRDs(ctx context.Context) error {
	defer logFunctionDuration(cr.Log, "removeCRDs", time.Now())
	crdNames := []string{"tridentbackends.trident.netapp.io", "tridentsnapshots.trident.netapp.io", "tridentstorageclasses.trident.netapp.io",
		"tridenttransactions.trident.netapp.io", "tridentvolumes.trident.netapp.io", "tridentversions.trident.netapp.io", "tridentnodes.trident.netapp.io"}
	for _, crd := range crdNames {
		CRD := &apiextenstionsv1.CustomResourceDefinition{}
		err := cr.Get(ctx, types.NamespacedName{Name: crd}, CRD)
		if err != nil {
			if errors.IsNotFound(err) {
				cr.Log.Info("CRD not found. Ignoring...", "name", crd)
				continue
			}
			cr.Log.Error(err, "error in getting CRD", "name", crd)
			return err
		}

		CRD.SetFinalizers([]string{})
		if err := cr.Update(ctx, CRD); err != nil {
			if errors.IsNotFound(err) {
				cr.Log.Info("Removing finalizers: CRD not found. Ignoring...", "name", CRD.Name)
				continue
			}
			cr.Log.Error(err, "Error in removing finalizers from CRD.", "name", CRD.Name)
			return err
		}
		cr.Log.Info("removed finalizers from CRD", "name", CRD.Name)

		err = cr.Delete(ctx, CRD)
		if err != nil {
			if errors.IsNotFound(err) {
				cr.Log.Info("Deleting CRD: CRD not found. Ignoring...", "name", CRD.Name)
				continue
			}
			cr.Log.Error(err, "Error in deleting CRD", "name", CRD.Name)
			return err
		}
		cr.Log.Info("CRD deleted", "name", CRD.Name)
	}
	return nil
}

// patchCRs patches all tridentNodes and tridentVersions CRs
func (cr *CleanUpOperatorReconciler) patchCRs(ctx context.Context, namespace string) error {

	nodesList := &corev1.NodeList{}
	err := cr.List(ctx, nodesList)
	if err != nil {
		if errors.IsNotFound(err) {
			cr.Log.Error(err, "Nodes List not found")
			return err
		}
		cr.Log.Error(err, "Error in getting Nodes List")
		return err
	}

	for _, node := range nodesList.Items {
		CRName := node.Name
		CRTridentNode := &tridentv1.TridentNode{}
		err = cr.Get(ctx, types.NamespacedName{Name: CRName, Namespace: namespace}, CRTridentNode)
		if err != nil {
			if errors.IsNotFound(err) {
				cr.Log.Info("CR not found", "name", CRName)
				continue
			}
			cr.Log.Error(err, "error in getting CR", "name", CRName)
			return err
		}

		CRTridentNode.SetFinalizers([]string{})
		if err := cr.Update(ctx, CRTridentNode); err != nil {
			cr.Log.Error(err, "Error in removing finalizers from CR", "name", CRTridentNode.Name)
			return err
		}
		cr.Log.Info("removed finalizers from CR", "name", CRTridentNode.Name)
	}

	ns := &corev1.Namespace{}
	err = cr.Get(ctx, types.NamespacedName{Name: namespace}, ns)
	if err != nil {
		cr.Log.Info("Namespace Not Found. Ignoring...", "name", namespace)
	} else {
		CRName := "trident"
		CRTridentVersion := &tridentv1.TridentVersion{}
		err := cr.Get(ctx, types.NamespacedName{Name: CRName, Namespace: namespace}, CRTridentVersion)
		if err != nil {
			if errors.IsNotFound(err) {
				cr.Log.Info("CR not found", "name", CRName)
				return nil
			}
			cr.Log.Error(err, "error in getting CR", "name", CRName)
			return err
		}

		CRTridentVersion.SetFinalizers([]string{})
		if err := cr.Update(ctx, CRTridentVersion); err != nil {
			fmt.Println(err, "Error is removing finalizers from CR ", CRTridentVersion.Name)
			return err
		}
		cr.Log.Info("removed finalizers from CR", "name", CRTridentVersion.Name)
	}
	return nil
}
