//go:build emulator

/*
Copyright 2026 The KubeVela Authors.

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

package emulator_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// The KubeVela side of the suite. Nothing here knows about DynamoDB: copy this
// file as it is into the emulator suite of another definition module.

var applicationResource = schema.GroupVersionResource{Group: "core.oam.dev", Version: "v1beta1", Resource: "applications"}

// velaResync is how long KubeVela can take to look at a component again when
// nothing about the Application changed. A cloud controller goes out of sync
// and back within seconds of an update, but that does not wake the
// Application: it re-checks health on its re-sync period, 5 minutes by
// default. Wait this long for an Application to report healthy after an
// update; the first apply reports sooner, as part of its workflow.
const velaResync = 7 * time.Minute

// applyModule installs the definition module in dir the way a platform team
// does, with `vela def apply-module`, replacing definitions of the same name.
// The vela CLI is taken from VELA_BIN, or from PATH.
func applyModule(ctx context.Context, dir string) error {
	vela := os.Getenv("VELA_BIN")
	if vela == "" {
		vela = "vela"
	}
	out, err := exec.CommandContext(ctx, vela, "def", "apply-module", dir, "--conflict=overwrite").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s def apply-module %s: %w (set VELA_BIN to the vela CLI)\n%s", vela, dir, err, out)
	}
	return nil
}

// applyComponent applies an Application with a single component. The
// Application and the component share the name.
func applyComponent(ctx context.Context, namespace, name, componentType string, properties map[string]any) error {
	return cluster.Apply(ctx, &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "core.oam.dev/v1beta1",
		"kind":       "Application",
		"metadata":   map[string]any{"name": name, "namespace": namespace},
		"spec": map[string]any{"components": []any{map[string]any{
			"name":       name,
			"type":       componentType,
			"properties": properties,
		}}},
	}})
}

// componentsHealthy reports whether KubeVela's health policies call every
// component of an Application healthy. An Application that has not reported
// any component yet is not healthy.
func componentsHealthy(ctx context.Context, namespace, name string) (bool, error) {
	app, err := cluster.Dynamic.Resource(applicationResource).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	services, _, err := unstructured.NestedSlice(app.Object, "status", "services")
	if err != nil {
		return false, fmt.Errorf("status.services of Application %s/%s: %w", namespace, name, err)
	}
	if len(services) == 0 {
		return false, nil
	}
	for _, s := range services {
		service, _ := s.(map[string]any)
		if healthy, _ := service["healthy"].(bool); !healthy {
			return false, nil
		}
	}
	return true, nil
}
