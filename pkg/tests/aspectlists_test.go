/*
 * Copyright (c) 2026 InfAI (CC SES)
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
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/SENERGY-Platform/process-deployment/lib/model/deploymentmodel"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/model"
	"github.com/SENERGY-Platform/smart-service-module-worker-process/pkg/tests/mocks"
)

// The worker takes the filter criteria of a deployment from the prepared deployment of process-deployment
// and posts them back unchanged. AspectId is the deprecated alias for an AspectIds list with one element.
// Neither spelling may be folded into the other here: the criteria are evaluated by device-repository and the
// event pipeline, which resolve the alias themselves, and a reader that predates the lists only sees AspectId.
func TestAspectListsArePassedThrough(t *testing.T) {
	aspectA := "urn:infai:ses:aspect:a"
	aspectB := "urn:infai:ses:aspect:b"

	criteriaVariants := map[string]func(criteria *deploymentmodel.FilterCriteria){
		"list": func(criteria *deploymentmodel.FilterCriteria) {
			criteria.AspectId = nil
			criteria.AspectIds = []string{aspectA, aspectB}
		},
		"single-list-element": func(criteria *deploymentmodel.FilterCriteria) {
			criteria.AspectId = nil
			criteria.AspectIds = []string{aspectA}
		},
		"deprecated-single-id": func(criteria *deploymentmodel.FilterCriteria) {
			criteria.AspectId = &aspectA
			criteria.AspectIds = nil
		},
		"both": func(criteria *deploymentmodel.FilterCriteria) {
			criteria.AspectId = &aspectA
			criteria.AspectIds = []string{aspectA, aspectB}
		},
	}

	// one test case per element type that carries a selection
	baseCases := []string{"task-device", "msg-event-device", "conditional-event-device"}

	for _, baseCase := range baseCases {
		for variantName, variant := range criteriaVariants {
			t.Run(baseCase+"/"+variantName, func(t *testing.T) {
				testAspectListPassThrough(t, baseCase, variant)
			})
		}
	}
}

func testAspectListPassThrough(t *testing.T, baseCase string, setAspects func(criteria *deploymentmodel.FilterCriteria)) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, _, depl, _, camunda, _, err := prepareMocks(ctx, wg, []mocks.Response{})
	if err != nil {
		t.Error(err)
		return
	}

	preparedDeploymentsFile, err := os.ReadFile(RESOURCE_BASE_DIR + baseCase + "/prepared_deployments.json")
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

	expectedCriteria := map[string]deploymentmodel.FilterCriteria{}
	for modelId, deployment := range preparedDepl {
		for i, element := range deployment.Elements {
			selection := getSelection(&element)
			if selection == nil {
				continue
			}
			setAspects(&selection.FilterCriteria)
			expectedCriteria[element.BpmnId] = selection.FilterCriteria
			deployment.Elements[i] = element
		}
		preparedDepl[modelId] = deployment
	}
	if len(expectedCriteria) == 0 {
		t.Error("base case has no element with a selection")
		return
	}
	depl.SetPreparedDeployments(preparedDepl)

	tasksFile, err := os.ReadFile(RESOURCE_BASE_DIR + baseCase + "/camunda_tasks.json")
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

	deployRequests := []mocks.Request{}
	for _, request := range depl.PopRequestLog() {
		if request.Method == "POST" && request.Endpoint == "/v3/deployments" {
			deployRequests = append(deployRequests, request)
		}
	}
	if len(deployRequests) != 1 {
		t.Errorf("expected exactly one deployment request, got %v", len(deployRequests))
		return
	}

	deployed := deploymentmodel.Deployment{}
	err = json.Unmarshal([]byte(deployRequests[0].Message), &deployed)
	if err != nil {
		t.Error(err)
		return
	}
	for _, element := range deployed.Elements {
		selection := getSelection(&element)
		if selection == nil {
			continue
		}
		expected, ok := expectedCriteria[element.BpmnId]
		if !ok {
			t.Errorf("unexpected element with selection in deployment: %v", element.BpmnId)
			continue
		}
		delete(expectedCriteria, element.BpmnId)
		if !reflect.DeepEqual(expected, selection.FilterCriteria) {
			e, _ := json.Marshal(expected)
			a, _ := json.Marshal(selection.FilterCriteria)
			t.Error(element.BpmnId, "\n", string(e), "\n", string(a))
		}
	}
	for bpmnId := range expectedCriteria {
		t.Errorf("element %v missing in deployment", bpmnId)
	}
}

func getSelection(element *deploymentmodel.Element) *deploymentmodel.Selection {
	switch {
	case element.Task != nil:
		return &element.Task.Selection
	case element.MessageEvent != nil:
		return &element.MessageEvent.Selection
	case element.ConditionalEvent != nil:
		return &element.ConditionalEvent.Selection
	default:
		return nil
	}
}
