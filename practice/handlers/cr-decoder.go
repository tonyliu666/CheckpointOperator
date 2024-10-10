package handlers

import (
	"fmt"
	// "context"

	"context"
    "encoding/json"

    "k8s.io/apimachinery/pkg/api/meta"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
    "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
    "k8s.io/apimachinery/pkg/types"
    "k8s.io/client-go/discovery"
    "k8s.io/client-go/discovery/cached/memory"
    "k8s.io/client-go/dynamic"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/restmapper"
	corev1 "k8s.io/api/core/v1"
	"time"

)
var decUnstructured = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

func DoSSA(ctx context.Context, cfg *rest.Config, pod *corev1.Pod, migrationNode string) error {
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
    if err != nil {
        return err
    }
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))
	// 2. Prepare the dynamic client
    dyn, err := dynamic.NewForConfig(cfg)
    if err != nil {
        return err
    }
	
	migrationYAML := fmt.Sprintf(`
apiVersion: api.my.domain/v1alpha1
kind: Migration
metadata:
  name: migration-sample
  labels:
    example-webhook-enabled: "true"
    force-update: "%s"
  namespace: practice-system
spec:
  podname: %s
  deployment: ""
  namespace: %s
  destinationNode: %s
  destinationNamespace: migration
  specify: ""
`, time.Now().String(), pod.Name, pod.Namespace, migrationNode)

	// Decode the YAML into an unstructured object
	obj := &unstructured.Unstructured{}
	_, gvk, err := decUnstructured.Decode([]byte(migrationYAML), nil, obj)
    if err != nil {
        return err
    }
	// 4. Find GVR
    mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
    if err != nil {
        return err
    }
	// 5. Obtain REST interface for the GVR
    var dr dynamic.ResourceInterface
    if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
        // namespaced resources should specify the namespace
        dr = dyn.Resource(mapping.Resource).Namespace(obj.GetNamespace())
    } else {
        // for cluster-wide resources
        dr = dyn.Resource(mapping.Resource)
    }
	// 6. Marshal object into JSON
    data, err := json.Marshal(obj)
    if err != nil {
        return err
    }

    // 7. Create or Update the object with SSA
    //     types.ApplyPatchType indicates SSA.
    //     FieldManager specifies the field owner ID.
    _, err = dr.Patch(ctx, obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
        FieldManager: "node-controller",
    })
	if err != nil {
		return err
	}
	
	return nil

}
