/*
Copyright The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v3 "github.com/containership/cluster-manager/pkg/apis/provision.containership.io/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeClusterUpgrades implements ClusterUpgradeInterface
type FakeClusterUpgrades struct {
	Fake *FakeContainershipProvisionV3
	ns   string
}

var clusterupgradesResource = schema.GroupVersionResource{Group: "provision.containership.io", Version: "v3", Resource: "clusterupgrades"}

var clusterupgradesKind = schema.GroupVersionKind{Group: "provision.containership.io", Version: "v3", Kind: "ClusterUpgrade"}

// Get takes name of the clusterUpgrade, and returns the corresponding clusterUpgrade object, and an error if there is any.
func (c *FakeClusterUpgrades) Get(name string, options v1.GetOptions) (result *v3.ClusterUpgrade, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(clusterupgradesResource, c.ns, name), &v3.ClusterUpgrade{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v3.ClusterUpgrade), err
}

// List takes label and field selectors, and returns the list of ClusterUpgrades that match those selectors.
func (c *FakeClusterUpgrades) List(opts v1.ListOptions) (result *v3.ClusterUpgradeList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(clusterupgradesResource, clusterupgradesKind, c.ns, opts), &v3.ClusterUpgradeList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v3.ClusterUpgradeList{ListMeta: obj.(*v3.ClusterUpgradeList).ListMeta}
	for _, item := range obj.(*v3.ClusterUpgradeList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested clusterUpgrades.
func (c *FakeClusterUpgrades) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(clusterupgradesResource, c.ns, opts))

}

// Create takes the representation of a clusterUpgrade and creates it.  Returns the server's representation of the clusterUpgrade, and an error, if there is any.
func (c *FakeClusterUpgrades) Create(clusterUpgrade *v3.ClusterUpgrade) (result *v3.ClusterUpgrade, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(clusterupgradesResource, c.ns, clusterUpgrade), &v3.ClusterUpgrade{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v3.ClusterUpgrade), err
}

// Update takes the representation of a clusterUpgrade and updates it. Returns the server's representation of the clusterUpgrade, and an error, if there is any.
func (c *FakeClusterUpgrades) Update(clusterUpgrade *v3.ClusterUpgrade) (result *v3.ClusterUpgrade, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(clusterupgradesResource, c.ns, clusterUpgrade), &v3.ClusterUpgrade{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v3.ClusterUpgrade), err
}

// Delete takes name of the clusterUpgrade and deletes it. Returns an error if one occurs.
func (c *FakeClusterUpgrades) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(clusterupgradesResource, c.ns, name), &v3.ClusterUpgrade{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeClusterUpgrades) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(clusterupgradesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v3.ClusterUpgradeList{})
	return err
}

// Patch applies the patch and returns the patched clusterUpgrade.
func (c *FakeClusterUpgrades) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.ClusterUpgrade, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(clusterupgradesResource, c.ns, name, pt, data, subresources...), &v3.ClusterUpgrade{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v3.ClusterUpgrade), err
}
