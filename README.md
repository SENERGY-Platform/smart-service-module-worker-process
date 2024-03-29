<a href="https://github.com/SENERGY-Platform/smart-service-module-worker-process/actions/workflows/tests.yml" rel="nofollow">
    <img src="https://github.com/SENERGY-Platform/smart-service-module-worker-process/actions/workflows/tests.yml/badge.svg?branch=main" alt="Tests" />
</a>


## Outputs

### Deployment-Id

- Desc: id of created deployment
- Variable-Name: process_deployment_id

## Camunda-Input-Variables

### Process-Model-Id

- Desc: defines which process-model should be de deployed
- Variable-Name-Template: `{{config.WorkerParamPrefix}}.process_model_id`
- Variable-Name-Example: `process_deployment.process_model_id`
- Value: string

### Module-Data

- Desc: sets fields for Module.ModuleData
- Variable-Name-Template: `{{config.WorkerParamPrefix}}.module_data`
- Variable-Name-Example: `process_deployment.module_data`
- Value: `json.Marshal(map[string]interface{})`


### Process-Name

- Desc: sets the name of the process deployment
- Variable-Name-Template: `{{config.WorkerParamPrefix}}.name`
- Variable-Name-Example: `process_deployment.name`
- Value: string

### IoT-Selection

- Desc: sets the iot selection of a process element
- Variable-Name-Template: `{{config.WorkerParamPrefix}}.{{element.BpmnId}}.selection`
- Variable-Name-Example: `process_deployment.Task_1uopw0b.selection`
- Value: json.Marshal(model.IotOption{})
- Value-Example: `{"device_selection":{"device_id":"device_7","service_id":"s12","path":"root.value_s12.v2"}}`

### Task-Parameter

- Desc: sets a input parameter of a task
- Variable-Name-Template: `{{config.WorkerParamPrefix}}.{{element.BpmnId}}.parameter.{{key}}`
- Variable-Name-Example: `process_deployment.Task_1uopw0b.parameter.inputs.r`
- Value: any; strings will be used as is; other types will be marshalled to as json

### Event-Value

- Desc: sets comparison value for message event
- Variable-Name-Template: `{{config.WorkerParamPrefix}}.{{element.BpmnId}}.event.value`
- Variable-Name-Example: `process_deployment.StartEvent_1.event.value`
- Value: string

### Event-UseMarshaller

- Desc: optional; decides if the event-filter should use the marshaller
- Variable-Name-Template: `{{config.WorkerParamPrefix}}.{{element.BpmnId}}.event.use_marshaller`
- Variable-Name-Example: `process_deployment.StartEvent_1.event.use_marshaller`
- Value: boolean || json.Marshal(boolean)

### Event-Flow-ID

- Desc: sets id of flow to be deployed as event-filter
- Variable-Name-Template: `{{config.WorkerParamPrefix}}.{{element.BpmnId}}.event.flow_id`
- Variable-Name-Example: `process_deployment.StartEvent_1.event.flow_id`
- Value: string

### Time

- Desc: sets time of time-event
- Variable-Name-Template: `{{config.WorkerParamPrefix}}.{{element.BpmnId}}.time`
- Variable-Name-Example: `process_deployment.StartEvent_1.time`
- Value: string