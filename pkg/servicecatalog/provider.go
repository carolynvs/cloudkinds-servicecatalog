package servicecatalog

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/carolynvs/cloudkinds/pkg/providers"
	"github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	svcatclient "github.com/kubernetes-incubator/service-catalog/pkg/client/clientset_generated/clientset"
	"github.com/kubernetes-incubator/service-catalog/pkg/util/kube"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

)

func DealWithIt(payload []byte) ([]byte, error) {
	// parse the payload
	evt := &providers.ResourceEvent{}
	err := json.Unmarshal(payload, evt)
	if err != nil {
		return nil, err
	}

	// load up the current cluster config
	config := kube.GetConfig("", "")
	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, errors.Wrapf(err,"could not get Kubernetes config")
	}
	crdClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create dynamic kubernetes client")
	}
	svcatClient, err := svcatclient.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create service catalog client")
	}

	// Retrieve the CRD
	// TODO: put this on ResourceEvent, and make sure using the name ApiVersion is correct (since it's really the group version)
	gv, err := schema.ParseGroupVersion(evt.Resource.APIVersion)
	if err != nil {
		return nil, err
	}

	// TODO: damn dirty hack, I think I need to read the crd to get this?
	pluralResource := strings.ToLower(evt.Resource.Kind) + "s"
	resourceType := schema.GroupVersionResource{Group: gv.Group, Version: gv.Version, Resource: pluralResource}
	resourceClient := crdClient.Resource(resourceType)
	r, err :=resourceClient.Namespace(evt.Resource.Namespace).Get(evt.Resource.Name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "could not retrieve the resource %#v", evt.Resource)
	}
	fmt.Println(r.Object)

	// find the corresponding service instance
	inst, err := svcatClient.ServicecatalogV1beta1().ServiceInstances(evt.Resource.Namespace).Get(evt.Resource.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create the instance
			inst = &v1beta1.ServiceInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: evt.Resource.Name,
					Namespace: evt.Resource.Namespace,
				},
			}

			// Mark the instance as owned by the CRD
			var t = true
			var f = false
			ownerRef := metav1.OwnerReference{
				APIVersion:         r.GetAPIVersion(),
				Kind:               r.GetKind(),
				Name:               r.GetName(),
				UID:                r.GetUID(),
				BlockOwnerDeletion: &t,
				Controller:         &f,
			}

			inst.SetOwnerReferences(append(inst.GetOwnerReferences(), ownerRef))

			inst, err = svcatClient.ServicecatalogV1beta1().ServiceInstances(inst.Namespace).Create(inst)
			if err != nil {
				return nil, err
			}
			fmt.Printf("created instance %v\n", inst)
		} else {
			return nil, err
		}
	}

	fmt.Println("instance already exists, skipping for now")

	return []byte(`{"msg": "farts are funny"}`), nil
}
