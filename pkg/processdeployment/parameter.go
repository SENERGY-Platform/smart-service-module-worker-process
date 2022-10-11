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

package processdeployment

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/process-deployment/lib/model/deploymentmodel"
	"github.com/SENERGY-Platform/process-deployment/lib/model/deviceselectionmodel"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/model"
	"strconv"
)

func (this *ProcessDeployment) getModuleData(task model.CamundaExternalTask) (result map[string]interface{}) {
	result = map[string]interface{}{}
	variable, ok := task.Variables[this.config.WorkerParamPrefix+"module_data"]
	if !ok {
		return result
	}
	str, ok := variable.Value.(string)
	if !ok {
		return result
	}
	err := json.Unmarshal([]byte(str), &result)
	if err != nil {
		return map[string]interface{}{}
	}
	return result
}

func (this *ProcessDeployment) getProcessModelId(task model.CamundaExternalTask) string {
	variable, ok := task.Variables[this.config.WorkerParamPrefix+"process_model_id"]
	if !ok {
		return ""
	}
	result, ok := variable.Value.(string)
	if !ok {
		return ""
	}
	return result
}

func (this *ProcessDeployment) getProcessName(task model.CamundaExternalTask) string {
	variable, ok := task.Variables[this.config.WorkerParamPrefix+"name"]
	if !ok {
		return ""
	}
	result, ok := variable.Value.(string)
	if !ok {
		return ""
	}
	return result
}

func (this *ProcessDeployment) setSelection(task model.CamundaExternalTask, element *deploymentmodel.Element) error {
	var elementSelection *deploymentmodel.Selection
	switch {
	case element.Task != nil:
		elementSelection = &element.Task.Selection
	case element.MessageEvent != nil:
		elementSelection = &element.MessageEvent.Selection
	default:
		return nil
	}

	selectionVariable, ok := task.Variables[this.config.WorkerParamPrefix+element.BpmnId+".selection"]
	if !ok {
		return errors.New("missing iot selection for " + element.BpmnId)
	}
	seelctionString, ok := selectionVariable.Value.(string)
	if !ok {
		return errors.New("invalid iot selection for " + element.BpmnId)
	}
	selection := model.IotOption{}
	err := json.Unmarshal([]byte(seelctionString), &selection)
	if err != nil {
		return fmt.Errorf("invalid iot selection for %v: %w", element.BpmnId, err)
	}

	if selection.DeviceSelection != nil {
		elementSelection.SelectedDeviceId = &selection.DeviceSelection.DeviceId
		elementSelection.SelectedServiceId = selection.DeviceSelection.ServiceId
		if selection.DeviceSelection.Path != nil {
			elementSelection.SelectedPath = &deviceselectionmodel.PathOption{
				Path: *selection.DeviceSelection.Path,
			}
			if selection.DeviceSelection.CharacteristicId != nil {
				elementSelection.SelectedPath.CharacteristicId = *selection.DeviceSelection.CharacteristicId
			}
		}
	}
	if selection.ImportSelection != nil {
		elementSelection.SelectedImportId = &selection.ImportSelection.Id
		if selection.ImportSelection.Path != nil {
			elementSelection.SelectedPath = &deviceselectionmodel.PathOption{
				Path: *selection.ImportSelection.Path,
			}
			if selection.ImportSelection.CharacteristicId != nil {
				elementSelection.SelectedPath.CharacteristicId = *selection.ImportSelection.CharacteristicId
			}
		}
	}
	if selection.DeviceGroupSelection != nil {
		elementSelection.SelectedDeviceGroupId = &selection.DeviceGroupSelection.Id
	}
	if selection.GenericEventSource != nil {
		elementSelection.SelectedGenericEventSource = &deploymentmodel.GenericEventSource{
			FilterType: selection.GenericEventSource.FilterType,
			FilterIds:  selection.GenericEventSource.FilterIds,
			Topic:      selection.GenericEventSource.Topic,
		}
		elementSelection.SelectedPath = &deviceselectionmodel.PathOption{
			Path: selection.GenericEventSource.Path,
		}
		if selection.GenericEventSource.CharacteristicId != nil {
			elementSelection.SelectedPath.CharacteristicId = *selection.GenericEventSource.CharacteristicId
		}
	}
	return nil
}

func (this *ProcessDeployment) setParameter(task model.CamundaExternalTask, element *deploymentmodel.Element) error {
	if element.Task == nil {
		return nil
	}
	for key, _ := range element.Task.Parameter {
		parameterName := this.config.WorkerParamPrefix + element.BpmnId + ".parameter." + key
		parameterVariable, ok := task.Variables[parameterName]
		if !ok {
			continue
		}
		parameterString, ok := parameterVariable.Value.(string)
		if ok {
			element.Task.Parameter[key] = parameterString
		} else {
			jsonParameter, err := json.Marshal(parameterVariable.Value)
			if err != nil {
				return fmt.Errorf("unable to interpret %v parameter: %w", parameterName, err)
			}
			element.Task.Parameter[key] = string(jsonParameter)
		}
	}

	return nil
}

func (this *ProcessDeployment) setMsgEventConfig(task model.CamundaExternalTask, element *deploymentmodel.Element) error {
	if element.MessageEvent == nil {
		return nil
	}
	parameterName := this.config.WorkerParamPrefix + element.BpmnId + ".event.value"
	parameterVariable, ok := task.Variables[parameterName]
	if ok && parameterVariable.Value != nil {
		var err error
		element.MessageEvent.Value, err = ensureJsonString(parameterVariable.Value)
		if err != nil {
			return err
		}
		/*
			element.MessageEvent.Value, ok = parameterVariable.Value.(string)
			if !ok {
				return fmt.Errorf("unable to interpret %v = %T %v parameter as string", parameterName, parameterVariable.Value, parameterVariable.Value)
			}
		*/
	}
	parameterName = this.config.WorkerParamPrefix + element.BpmnId + ".event.flow_id"
	parameterVariable, ok = task.Variables[parameterName]
	if !ok {
		return fmt.Errorf("missing %v parameter", parameterName)
	}
	element.MessageEvent.FlowId, ok = parameterVariable.Value.(string)
	if !ok {
		return fmt.Errorf("unable to interpret %v parameter as string", parameterName)
	}

	parameterName = this.config.WorkerParamPrefix + element.BpmnId + ".event.use_marshaller"
	parameterVariable, ok = task.Variables[parameterName]
	if ok {
		switch v := parameterVariable.Value.(type) {
		case string:
			var err error
			element.MessageEvent.UseMarshaller, err = strconv.ParseBool(v)
			if err != nil {
				return fmt.Errorf("unable to handle %v parameter: %w", parameterName, err)
			}
		case bool:
			element.MessageEvent.UseMarshaller = v
		default:
			return fmt.Errorf("unable to handle %v parameter with given type", parameterName)
		}
	}

	return nil
}

func ensureJsonString(value interface{}) (result string, err error) {
	if str, ok := value.(string); ok {
		var temp interface{}
		err = json.Unmarshal([]byte(str), &temp)
		if err == nil {
			return str, nil
		}
		buf, err := json.Marshal(str)
		return string(buf), err
	}
	buf, err := json.Marshal(value)
	return string(buf), err
}

func (this *ProcessDeployment) setTime(task model.CamundaExternalTask, element *deploymentmodel.Element) error {
	if element.TimeEvent == nil {
		return nil
	}
	variableName := this.config.WorkerParamPrefix + element.BpmnId + ".time"
	timeVariable, ok := task.Variables[variableName]
	if !ok {
		return nil
	}
	timeString, ok := timeVariable.Value.(string)
	if !ok {
		return errors.New("invalid time value in " + variableName)
	}
	element.TimeEvent.Time = timeString
	return nil
}
