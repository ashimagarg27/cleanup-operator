package controllers

import (
	"context"
	inerror "errors"
	"fmt"
	"strings"
	"time"

	localv1 "github.com/openshift/local-storage-operator/pkg/apis/local/v1"
	corev1 "k8s.io/api/core/v1"
	apiextenstionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func (cr *CleanUpOperatorReconciler) localVolumeCleanUp(ctx context.Context, namespace string) error {
	defer logFunctionDuration(cr.Log, "localVolumeCleanUp", time.Now())

	// Find PVs
	pvList := &corev1.PersistentVolumeList{}
	err := cr.List(ctx, pvList)
	if err != nil {
		fmt.Println(err, "Error in getting PVs")
		return err
	}

	// Check PV
	for _, pv := range pvList.Items {
		if strings.HasPrefix(pv.Name, "local-pv-") {
			fmt.Println("PV status- ", pv.Status.Phase)
			if pv.Status.Phase == "Bound" {
				err = inerror.New("PV is in Bounded state " + pv.Name)
				fmt.Println("PV is in Bounded state ", pv.Name)
				fmt.Println("Please remove bounded PVC/PoD")
				return err
			}
		}
	}

	localVolumeStatus := true
	localDisk := &localv1.LocalVolume{}
	err = cr.Get(ctx, types.NamespacedName{Name: "local-disk", Namespace: namespace}, localDisk)
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Println("LocalVolume 'local-disk' not found")
			localVolumeStatus = false
		} else {
			fmt.Println(err, "Error in getting LocalVolume 'local-disk'")
			return err
		}
	}

	if localVolumeStatus {
		localDisk.SetFinalizers([]string{})
		if err := cr.Update(ctx, localDisk); err != nil {
			fmt.Println(err, "Error is removing finalizers from Local-Volume 'local-disk'")
			return err
		}
	}

	// PV Deletion
	for _, pv := range pvList.Items {
		if strings.HasPrefix(pv.Name, "local-pv-") {
			err = cr.Delete(ctx, &pv)
			if err != nil {
				fmt.Print("Error in Deleting PV ", pv.Name)
				return err
			}
		}
	}
	fmt.Println("PV(s) Deleted.....")

	// Remove Mounted Path
	nodesList := &corev1.NodeList{}
	err = cr.List(ctx, nodesList)
	if err != nil {
		fmt.Println(err, "Error in getting Nodes List")
		return err
	}

	for _, node := range nodesList.Items {
		command := "oc debug node/" + node.Name + " -- chroot /host rm -rf /mnt"
		_, out, err := ExecuteCommand(command)
		if err != nil {
			fmt.Println("Error in removing mounted path from node: ", node.Name)
			return err
		}
		fmt.Println(out)
	}
	fmt.Println("Mounted Paths Removed....")

	return nil
}

// removeLocalVolmeCRDs patches and deletes localVolume crds
func (cr *CleanUpOperatorReconciler) removeLocalVolmeCRDs(ctx context.Context) error {
	defer logFunctionDuration(cr.Log, "removeLocalVolmeCRDs", time.Now())
	crdNames := []string{"localvolumediscoveries.local.storage.openshift.io", "localvolumediscoveryresults.local.storage.openshift.io",
		"localvolumes.local.storage.openshift.io", "localvolumesets.local.storage.openshift.io"}
	for _, crd := range crdNames {
		CRD := &apiextenstionsv1.CustomResourceDefinition{}
		err := cr.Get(ctx, types.NamespacedName{Name: crd}, CRD)
		if err != nil {
			if errors.IsNotFound(err) {
				fmt.Println("CRD not found: ", crd)
				continue
			}
			fmt.Println(err, "error in getting crd: ", crd)
			return err
		}

		CRD.SetFinalizers([]string{})
		if err := cr.Update(ctx, CRD); err != nil {
			fmt.Println(err, "Error is removing finalizers from CustomResoure ", CRD.Name)
			return err
		}

		err = cr.Delete(ctx, CRD)
		if err != nil {
			fmt.Println(err, "Error is deleting CustomResoure ", CRD.Name)
			return err
		}

		fmt.Println(CRD.Name)
	}
	return nil
}
