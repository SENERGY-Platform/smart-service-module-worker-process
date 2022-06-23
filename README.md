
## Camunda-Input-Variables

### Process-Model-Id

- Desc: sets the name of the process deployment
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

### Task-IoT-Selection

- Desc: sets the iot selection of a task
- Variable-Name-Template: `{{config.WorkerParamPrefix}}.{{element.BpmnId}}.selection`
- Variable-Name-Example: `process_deployment.Task_1uopw0b.selection`
- Value: json.Marshal(model.IotOption{})
- Value-Example: `{"device_selection":{"device_id":"device_7","service_id":"s12","path":"root.value_s12.v2"}}`

### Task-Parameter

- Desc: sets a input parameter of a task
- Variable-Name-Template: `{{config.WorkerParamPrefix}}.{{element.BpmnId}}.parameter.{{key}}`
- Variable-Name-Example: `process_deployment.Task_1uopw0b.parameter.inputs.r`
- Value: any; strings will be used as is; other types will be marshalled to as json

