# Example custom data point with text data type for user model
resource "authsignal_custom_data_point" "my_custom_string_data_point" {
  name        = "MyCustomStringDataPoint"
  data_type   = "text"
  model_type  = "user"
  description = "My custom string data point"
}

# Example custom data point with number data type for action model
resource "authsignal_custom_data_point" "my_custom_number_data_point" {
  name        = "MyCustomNumberDataPoint"
  data_type   = "number"
  model_type  = "action"
  description = "My custom number data point"
}

# Example custom data point with boolean data type for user model
resource "authsignal_custom_data_point" "my_custom_boolean_data_point" {
  name        = "MyCustomBooleanDataPoint"
  data_type   = "boolean"
  model_type  = "user"
  description = "My custom boolean data point"
}

# Example custom data point with multiselect data type for action model
resource "authsignal_custom_data_point" "my_custom_multiselect_data_point" {
  name        = "MyCustomMultiselectDataPoint"
  data_type   = "multiselect"
  model_type  = "action"
  description = "My custom multiselect data point"
}

