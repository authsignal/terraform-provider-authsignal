---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "authsignal_custom_data_point Data Source - terraform-provider-authsignal"
subcategory: ""
description: |-
  
---

# authsignal_custom_data_point (Data Source)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `id` (String) The id of the custom data point.

### Read-Only

- `data_type` (String) The data type of the custom data point. Allowed values: `text`, `number`, `boolean`, 'multiselect'.
- `description` (String) The description of the custom data point.
- `model_type` (String) The model type of the custom data point. Allowed values: `action`, `user`.
- `name` (String) The name of the custom data point.