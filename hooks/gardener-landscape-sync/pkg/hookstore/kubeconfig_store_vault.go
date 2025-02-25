// Copyright 2021 The Kubeswitch authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hookstore

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gardener/gardener/pkg/utils/secrets"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	"github.com/danielfoehrkn/kubeswitch/types"
)

func (s *VaultStore) GetKind() types.StoreKind {
	return types.StoreKindVault
}

// NOOP
func (s *VaultStore) CreateLandscapeDirectory(landscapeDirectory string) error {
	return nil
}

func (s *VaultStore) WriteKubeconfigFile(vaultPath, kubeconfigName string, kubeconfigSecret corev1.Secret) error {
	kubeconfigData := kubeconfigSecret.Data[secrets.DataKeyKubeconfig]

	_, err := s.Client.Logical().Write(vaultPath, map[string]interface{}{
		kubeconfigName: kubeconfigData,
	})
	if err != nil {
		return err
	}
	return nil
}

// CleanExistingKubeconfigs recursively deletes secrets under the specified path
func (s *VaultStore) CleanExistingKubeconfigs(log *logrus.Entry, vaultPath string) error {
	log.Infof("deleting secrets from vault under path %q", vaultPath)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go s.recursivePathDeletion(log, &wg, vaultPath)
	wg.Wait()
	return nil
}

func (s *VaultStore) recursivePathDeletion(log *logrus.Entry, wg *sync.WaitGroup, searchPath string) {
	defer wg.Done()

	secret, err := s.Client.Logical().List(searchPath)
	if err != nil {
		log.Infof("failed to list secrets under path %q", searchPath)
		return
	}

	if secret == nil {
		log.Infof("No secrets found for path %s", searchPath)
		return
	}

	items := secret.Data["keys"].([]interface{})
	for _, item := range items {
		itemPath := fmt.Sprintf("%s/%s", strings.TrimSuffix(searchPath, "/"), item)
		if strings.HasSuffix(item.(string), "/") {
			// this is another folder
			wg.Add(1)
			go s.recursivePathDeletion(log, wg, itemPath)
		} else if item != "" {
			// found an actual secret
			_, err := s.Client.Logical().Delete(itemPath)
			if err != nil {
				log.Warnf("failed to dleete secret with path %q", itemPath)
			}
		}
	}
}
