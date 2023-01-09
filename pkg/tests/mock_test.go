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
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/model"
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg"
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg/processdeployment"
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg/tests/mocks"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"
)

const RESOURCE_BASE_DIR = "./resources/test-cases/"

func TestWithMocks(t *testing.T) {
	infos, err := os.ReadDir(RESOURCE_BASE_DIR)
	if err != nil {
		t.Error(err)
		return
	}
	for _, info := range infos {
		name := info.Name()
		if info.IsDir() && isValidaForMockTest(RESOURCE_BASE_DIR+name) {
			t.Run(name, func(t *testing.T) {
				mockTest(t, name)
			})
		}
	}
}

func prepareMocks(ctx context.Context, wg *sync.WaitGroup) (libConf configuration.Config, conf processdeployment.Config, deployment *mocks.DeploymentMock, camunda *mocks.CamundaMock, smartServiceRepo *mocks.SmartServiceRepoMock, err error) {
	libConf, err = configuration.LoadLibConfig("../../config.json")
	if err != nil {
		return
	}
	conf, err = configuration.Load[processdeployment.Config]("../../config.json")
	if err != nil {
		return
	}
	libConf.CamundaWorkerWaitDurationInMs = 200

	deployment = mocks.NewDeploymentMock()
	conf.ProcessDeploymentUrl = deployment.Start(ctx, wg)

	camunda = mocks.NewCamundaMock()
	libConf.CamundaUrl = camunda.Start(ctx, wg)

	smartServiceRepo = mocks.NewSmartServiceRepoMock(libConf, conf)
	libConf.SmartServiceRepositoryUrl = smartServiceRepo.Start(ctx, wg)

	libConf.AuthEndpoint = mocks.Keycloak(ctx, wg)

	err = pkg.Start(ctx, wg, conf, libConf)

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
	infos, err := os.ReadDir(dir)
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

func mockTest(t *testing.T, name string) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, _, depl, camunda, repo, err := prepareMocks(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}

	preparedDeploymentsFile, err := os.ReadFile(RESOURCE_BASE_DIR + name + "/prepared_deployments.json")
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

	expectedCamundaRequestsFile, err := os.ReadFile(RESOURCE_BASE_DIR + name + "/expected_camunda_requests.json")
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

	expectedDeploymentRequestsFile, err := os.ReadFile(RESOURCE_BASE_DIR + name + "/expected_deployment_requests.json")
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

	expectedSmartServiceRepoRequestsFile, err := os.ReadFile(RESOURCE_BASE_DIR + name + "/expected_smart_service_repo_requests.json")
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

	tasksFile, err := os.ReadFile(RESOURCE_BASE_DIR + name + "/camunda_tasks.json")
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
		e, _ := json.Marshal(expectedDeploymentRequests)
		a, _ := json.Marshal(actualDeplRequests)
		t.Error("\n", string(e), "\n", string(a))
	}

	if !reflect.DeepEqual(expectedSmartServiceRepoRequests, actualSmartServiceRepoRequests) {
		e, _ := json.Marshal(expectedSmartServiceRepoRequests)
		a, _ := json.Marshal(actualSmartServiceRepoRequests)
		t.Error("\n", string(e), "\n", string(a))
	}
}
