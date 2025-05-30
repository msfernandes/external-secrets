/*
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

package gitlab

import (
	"context"
	"os"
	"strings"

	// nolint
	. "github.com/onsi/ginkgo/v2"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	// nolint
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type gitlabProvider struct {
	credentials string
	projectID   string
	environment string
	framework   *framework.Framework
}

func newGitlabProvider(f *framework.Framework, credentials, projectID string, environment string) *gitlabProvider {
	prov := &gitlabProvider{
		credentials: credentials,
		projectID:   projectID,
		environment: environment,
		framework:   f,
	}
	BeforeEach(prov.BeforeEach)
	return prov
}

func newFromEnv(f *framework.Framework) *gitlabProvider {
	credentials := os.Getenv("GITLAB_TOKEN")
	projectID := os.Getenv("GITLAB_PROJECT_ID")
	environment := os.Getenv("GITLAB_ENVIRONMENT")
	return newGitlabProvider(f, credentials, projectID, environment)
}

func (s *gitlabProvider) CreateSecret(key string, val framework.SecretEntry) {
	// **Open the client
	client, err := gitlab.NewClient(s.credentials)
	Expect(err).ToNot(HaveOccurred())
	// Open the client**

	// Set variable options
	variableKey := strings.ReplaceAll(key, "-", "_")
	variableValue := val

	opt := gitlab.CreateProjectVariableOptions{
		Key:              &variableKey,
		Value:            &variableValue.Value,
		VariableType:     nil,
		Protected:        nil,
		Masked:           nil,
		EnvironmentScope: nil,
	}

	// Create a variable
	_, _, err = client.ProjectVariables.CreateVariable(s.projectID, &opt)

	Expect(err).ToNot(HaveOccurred())
	// Versions aren't supported by Gitlab, but we could add
	// more parameters to test
}

func (s *gitlabProvider) DeleteSecret(key string) {
	// **Open a client
	client, err := gitlab.NewClient(s.credentials)
	Expect(err).ToNot(HaveOccurred())
	// Open a client**

	// Delete the secret
	_, err = client.ProjectVariables.RemoveVariable(s.projectID, strings.ReplaceAll(key, "-", "_"), &gitlab.RemoveProjectVariableOptions{})
	Expect(err).ToNot(HaveOccurred())
}

func (s *gitlabProvider) BeforeEach() {
	By("creating a gitlab variable")
	gitlabCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "provider-secret",
			Namespace: s.framework.Namespace.Name,
		},
		// Puts access token into StringData

		StringData: map[string]string{
			"token":       s.credentials,
			"projectID":   s.projectID,
			"environment": s.environment,
		},
	}
	err := s.framework.CRClient.Create(context.Background(), gitlabCreds)
	Expect(err).ToNot(HaveOccurred())

	// Create a secret store - change these values to match YAML
	By("creating a secret store for credentials")
	secretStore := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.framework.Namespace.Name,
			Namespace: s.framework.Namespace.Name,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Gitlab: &esv1.GitlabProvider{
					ProjectID: s.projectID,
					Auth: esv1.GitlabAuth{
						SecretRef: esv1.GitlabSecretRef{
							AccessToken: esmeta.SecretKeySelector{
								Name: "provider-secret",
								Key:  "token",
							},
						},
					},
				},
			},
		},
	}

	err = s.framework.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}
