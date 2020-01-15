/*
Copyright 2018 BlackRock, Inc.

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

package resource

import (
	"testing"

	"github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/util/argo"
	"github.com/ghodss/yaml"
	"github.com/mitchellh/mapstructure"
	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/fake"
)

const newFakeApp = `
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: fake
  namespace: fake1
spec:
  source:
    path: "some/path"
    repoURL: "https://github.com/argoproj/argocd-example-apps.git"
    targetRevision: "HEAD"
  destination:
    namespace: fake
    server: "https://cluster-api.com"
status:
  observedAt: "2020-01-13T22:26:06Z"
  summary:
    externalURLs:
    - http://sonar.te-engg-dev-us.thousandeyes.com:9000
    images:
    - gcr.io/te-engg-dev/ci/sonarqube:7.3-developer
`

const oldFakeApp = `
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: fake
  namespace: fake2
spec:
  source:
    path: "some/path"
    repoURL: "https://github.com/argoproj/argocd-example-apps.git"
    targetRevision: "HEAD"
  destination:
    namespace: fake
    server: "https://cluster-api.com"
status:
  observedAt: "2020-01-13T14:26:06Z"
  summary:
    externalURLs:
    - http://sonar.te-engg-dev-us.thousandeyes.com:9000
    images:
    - gcr.io/te-engg-dev/ci/sonarqube:7.3-developer
`

func TestBasicIgnoreDifferences(t *testing.T) {
	convey.Convey("Given a resource object with only unimportant updates, ensure no update executes", t, func() {
		normalizer, err := argo.NewDiffNormalizer(
			[]v1alpha1.ResourceIgnoreDifferences{{
				Group: "argoproj.io",
				Kind:  "Application",
				JSONPointers: []string{
					"/status/observedAt",
					"/metadata/namespace",
				},
			}}, make(map[string]v1alpha1.ResourceOverride))

		var newUn, oldUn unstructured.Unstructured
		err = yaml.Unmarshal([]byte(newFakeApp), &newUn)
		convey.So(err, convey.ShouldBeNil)
		err = yaml.Unmarshal([]byte(oldFakeApp), &oldUn)
		convey.So(err, convey.ShouldBeNil)

		event := &InformerEvent{
			&newUn,
			&oldUn,
			"UPDATE",
		}

		unchanged := hasNotChanged(normalizer, event)
		convey.So(unchanged, convey.ShouldBeTrue)
	})
}

var esIgnore = `
namespace: "argo"
group: "argoproj.io"
version: "v1alpha1"
resource: "applications"
filter:
  ignoreDifferences:
  - "/status/observedAt"
  - "/metadata/namespace"
`

func TestFilterIgnoreDifferences(t *testing.T) {
	convey.Convey("Given an ignoreDifferences filter in the event source, filter an application", t, func() {
		ps, err := parseEventSource(esIgnore)
		convey.So(err, convey.ShouldBeNil)

		normalizer, err := normalizerIgnoreDifferences(ps.(*resource).Filter)
		convey.So(err, convey.ShouldBeNil)

		var newUn, oldUn unstructured.Unstructured
		err = yaml.Unmarshal([]byte(newFakeApp), &newUn)
		convey.So(err, convey.ShouldBeNil)
		err = yaml.Unmarshal([]byte(oldFakeApp), &oldUn)
		convey.So(err, convey.ShouldBeNil)

		event := &InformerEvent{
			&newUn,
			&oldUn,
			"UPDATE",
		}

		unchanged := hasNotChanged(normalizer, event)
		convey.So(unchanged, convey.ShouldBeTrue)
	})
}

func TestFilter(t *testing.T) {
	convey.Convey("Given a resource object, apply filter on it", t, func() {
		ps, err := parseEventSource(es)
		convey.So(err, convey.ShouldBeNil)
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake",
				Namespace: "fake",
				Labels: map[string]string{
					"workflows.argoproj.io/phase": "Succeeded",
					"name":                        "my-workflow",
				},
			},
		}
		pod, err = fake.NewSimpleClientset().CoreV1().Pods("fake").Create(pod)
		convey.So(err, convey.ShouldBeNil)

		outmap := make(map[string]interface{})
		err = mapstructure.Decode(pod, &outmap)
		convey.So(err, convey.ShouldBeNil)

		err = passFilters(&unstructured.Unstructured{
			Object: outmap,
		}, ps.(*resource).Filter)
		convey.So(err, convey.ShouldBeNil)
	})
}
