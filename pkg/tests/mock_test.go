/*
 * Copyright (c) 2022 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tests

import (
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/process-deployment/lib/model/deploymentmodel"
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg"
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg/model"
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg/tests/mocks"
	"io/ioutil"
	"reflect"
	"sync"
	"testing"
	"time"
)

const RESOURCE_BASE_DIR = "./resources/test-cases/"

func TestWithMocks(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, depl, camunda, repo, err := prepareMocks(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}

	infos, err := ioutil.ReadDir(RESOURCE_BASE_DIR)
	if err != nil {
		t.Error(err)
		return
	}
	for _, info := range infos {
		name := info.Name()
		if info.IsDir() && isValidaForMockTest(RESOURCE_BASE_DIR+name) {
			t.Run(name, func(t *testing.T) {
				mockTest(t, depl, camunda, repo, name)
			})
		}
	}
}

func prepareMocks(ctx context.Context, wg *sync.WaitGroup) (conf configuration.Config, deployment *mocks.DeploymentMock, camunda *mocks.CamundaMock, smartServiceRepo *mocks.SmartServiceRepoMock, err error) {
	conf, err = configuration.Load("../../config.json")
	if err != nil {
		return
	}
	conf.CamundaWorkerWaitDurationInMs = 200

	deployment = mocks.NewDeploymentMock()
	conf.ProcessDeploymentUrl = deployment.Start(ctx, wg)

	camunda = mocks.NewCamundaMock()
	conf.CamundaUrl = camunda.Start(ctx, wg)

	smartServiceRepo = mocks.NewSmartServiceRepoMock(conf)
	conf.SmartServiceRepositoryUrl = smartServiceRepo.Start(ctx, wg)

	conf.AuthEndpoint = mocks.Keycloak(ctx, wg)

	err = pkg.Start(ctx, wg, conf)

	return
}

func isValidaForMockTest(dir string) bool {
	expectedFiles := []string{
		"camunda_tasks.json",
		"expected_camunda_requests.json",
		"expected_deployment_requests.json",
		"expected_smart_service_repo_requests.json",
		"prepared_deployments.json",
	}
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	files := map[string]bool{}
	for _, info := range infos {
		if !info.IsDir() {
			files[info.Name()] = true
		}
	}
	for _, expected := range expectedFiles {
		if !files[expected] {
			return false
		}
	}
	return true
}

func mockTest(t *testing.T, depl *mocks.DeploymentMock, camunda *mocks.CamundaMock, repo *mocks.SmartServiceRepoMock, name string) {
	preparedDeploymentsFile, err := ioutil.ReadFile(RESOURCE_BASE_DIR + name + "/prepared_deployments.json")
	if err != nil {
		t.Error(err)
		return
	}
	var preparedDepl map[string]deploymentmodel.Deployment
	err = json.Unmarshal(preparedDeploymentsFile, &preparedDepl)
	if err != nil {
		t.Error(err)
		return
	}
	depl.SetPreparedDeployments(preparedDepl)

	expectedCamundaRequestsFile, err := ioutil.ReadFile(RESOURCE_BASE_DIR + name + "/expected_camunda_requests.json")
	if err != nil {
		t.Error(err)
		return
	}
	var expectedCamundaRequests []mocks.Request
	err = json.Unmarshal(expectedCamundaRequestsFile, &expectedCamundaRequests)
	if err != nil {
		t.Error(err)
		return
	}

	expectedDeploymentRequestsFile, err := ioutil.ReadFile(RESOURCE_BASE_DIR + name + "/expected_deployment_requests.json")
	if err != nil {
		t.Error(err)
		return
	}
	var expectedDeploymentRequests []mocks.Request
	err = json.Unmarshal(expectedDeploymentRequestsFile, &expectedDeploymentRequests)
	if err != nil {
		t.Error(err)
		return
	}

	expectedSmartServiceRepoRequestsFile, err := ioutil.ReadFile(RESOURCE_BASE_DIR + name + "/expected_smart_service_repo_requests.json")
	if err != nil {
		t.Error(err)
		return
	}
	var expectedSmartServiceRepoRequests []mocks.Request
	err = json.Unmarshal(expectedSmartServiceRepoRequestsFile, &expectedSmartServiceRepoRequests)
	if err != nil {
		t.Error(err)
		return
	}

	tasksFile, err := ioutil.ReadFile(RESOURCE_BASE_DIR + name + "/camunda_tasks.json")
	if err != nil {
		t.Error(err)
		return
	}
	var tasks []model.CamundaExternalTask
	err = json.Unmarshal(tasksFile, &tasks)
	if err != nil {
		t.Error(err)
		return
	}
	camunda.AddToQueue(tasks)

	time.Sleep(1 * time.Second)

	actualCamundaRequests := camunda.PopRequestLog()
	actualDeplRequests := depl.PopRequestLog()
	actualSmartServiceRepoRequests := repo.PopRequestLog()

	if !reflect.DeepEqual(expectedCamundaRequests, actualCamundaRequests) {
		temp, _ := json.Marshal(actualCamundaRequests)
		t.Error(string(temp))
	}

	if !reflect.DeepEqual(expectedDeploymentRequests, actualDeplRequests) {
		temp, _ := json.Marshal(actualDeplRequests)
		t.Error(string(temp))
	}

	if !reflect.DeepEqual(expectedSmartServiceRepoRequests, actualSmartServiceRepoRequests) {
		temp, _ := json.Marshal(actualSmartServiceRepoRequests)
		t.Error(string(temp))
	}
}