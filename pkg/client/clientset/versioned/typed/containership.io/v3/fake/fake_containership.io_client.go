/*
Copyright 2018 The Kubernetes Authors.

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

package fake

import (
	v3 "github.com/containership/cloud-agent/pkg/client/clientset/versioned/typed/containership.io/v3"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeContainershipV3 struct {
	*testing.Fake
}

func (c *FakeContainershipV3) Registries(namespace string) v3.RegistryInterface {
	return &FakeRegistries{c, namespace}
}

func (c *FakeContainershipV3) Users(namespace string) v3.UserInterface {
	return &FakeUsers{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeContainershipV3) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}