package servicecatalog

import (
	"encoding/json"
	"strings"

	"github.com/carolynvs/cloudkinds/pkg/providers"
	"github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	svcatclient "github.com/kubernetes-incubator/service-catalog/pkg/client/clientset_generated/clientset"
	"github.com/kubernetes-incubator/service-catalog/pkg/util/kube"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type CatalogProvider struct {
	crdClient   dynamic.Interface
	svcatClient svcatclient.Interface
}

func NewProvider() (*CatalogProvider, error) {
	// load up the current cluster config
	config := kube.GetConfig("", "")
	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get Kubernetes config")
	}
	crdClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create dynamic kubernetes client")
	}
	svcatClient, err := svcatclient.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create service catalog client")
	}

	p := &CatalogProvider{
		crdClient:   crdClient,
		svcatClient: svcatClient,
	}
	return p, nil
}

func (p CatalogProvider) DealWithIt(payload []byte) ([]byte, error) {
	// parse the payload
	evt := &providers.ResourceEvent{}
	err := json.Unmarshal(payload, evt)
	if err != nil {
		return nil, err
	}

	// Retrieve the CRD
	// TODO: put this method on ResourceEvent, and make sure using the name ApiVersion is correct (since it's really the group version)
	gv, err := schema.ParseGroupVersion(evt.Resource.APIVersion)
	if err != nil {
		return nil, err
	}

	// TODO: damn dirty hack, I think I need to read the crd to get this?
	pluralResource := strings.ToLower(evt.Resource.Kind) + "s"
	resourceType := schema.GroupVersionResource{Group: gv.Group, Version: gv.Version, Resource: pluralResource}
	resourceClient := p.crdClient.Resource(resourceType)
	r, err := resourceClient.Namespace(evt.Resource.Namespace).Get(evt.Resource.Name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "could not retrieve the resource %#v", evt.Resource)
	}

	// find the corresponding service instance
	_, err = p.svcatClient.ServicecatalogV1beta1().ServiceInstances(evt.Resource.Namespace).Get(evt.Resource.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			_, err = p.createService(r)
			return []byte(`{"msg": "created service instance"}`), err
		} else {
			return nil, err
		}
	}

	return []byte(`{"msg": "instance already exists, update not implemented yet"}`), nil
}

func (p CatalogProvider) createService(r *unstructured.Unstructured) (*v1beta1.ServiceInstance, error) {
	ref, err := p.resolveService(r)
	if err != nil {
		return nil, err
	}

	params := r.Object["spec"]
	paramsJson, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	// Create the instance
	inst := &v1beta1.ServiceInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.GetName(),
			Namespace: r.GetNamespace(),
		},
		Spec: v1beta1.ServiceInstanceSpec{
			PlanReference: *ref,
			Parameters:    &runtime.RawExtension{Raw: paramsJson},
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

	return p.svcatClient.ServicecatalogV1beta1().ServiceInstances(inst.Namespace).Create(inst)
}

func (p CatalogProvider) resolveService(r *unstructured.Unstructured) (*v1beta1.PlanReference, error) {
	// this is a boring search, we can do better later
	ref := &v1beta1.PlanReference{}

	classes, err := p.svcatClient.ServicecatalogV1beta1().ClusterServiceClasses().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, class := range classes.Items {
		if strings.ToLower(class.Name) == strings.ToLower(r.GetKind()) {
			ref.ClusterServiceClassName = class.Name
			break
		}
	}
	if ref.ClusterServiceClassName == "" {
		return nil, errors.Errorf("could not find a class for %v", r)
	}

	plans, err := p.svcatClient.ServicecatalogV1beta1().ClusterServicePlans().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, plan := range plans.Items {
		if plan.Spec.ClusterServiceClassRef.Name == ref.ClusterServiceClassName {
			ref.ClusterServicePlanName = plan.Name
			return ref, nil
		}
	}
	return nil, errors.Errorf("could not find a plan for %v", r)
}
