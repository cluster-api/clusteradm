// Copyright Contributors to the Open Cluster Management project
package join

import (
	"context"
	"encoding/json"

	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

func CreateOrPatchConfigMap(ctx context.Context, c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*core.ConfigMap) *core.ConfigMap, opts metav1.PatchOptions) (*core.ConfigMap, error) {
	cur, err := c.CoreV1().ConfigMaps(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		klog.V(3).Infof("Creating ConfigMap %s/%s.", meta.Namespace, meta.Name)
		out, err := c.CoreV1().ConfigMaps(meta.Namespace).Create(ctx, transform(&core.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: core.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}), metav1.CreateOptions{
			DryRun:       opts.DryRun,
			FieldManager: opts.FieldManager,
		})
		return out, err
	} else if err != nil {
		return nil, err
	}
	return PatchConfigMap(ctx, c, cur, transform, opts)
}

func PatchConfigMap(ctx context.Context, c kubernetes.Interface, cur *core.ConfigMap, transform func(*core.ConfigMap) *core.ConfigMap, opts metav1.PatchOptions) (*core.ConfigMap, error) {
	return PatchConfigMapObject(ctx, c, cur, transform(cur.DeepCopy()), opts)
}

func PatchConfigMapObject(ctx context.Context, c kubernetes.Interface, cur, mod *core.ConfigMap, opts metav1.PatchOptions) (*core.ConfigMap, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := json.Marshal(mod)
	if err != nil {
		return nil, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, core.ConfigMap{})
	if err != nil {
		return nil, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, nil
	}
	klog.V(3).Infof("Patching ConfigMap %s/%s with %s", cur.Namespace, cur.Name, string(patch))
	out, err := c.CoreV1().ConfigMaps(cur.Namespace).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
	return out, err
}
