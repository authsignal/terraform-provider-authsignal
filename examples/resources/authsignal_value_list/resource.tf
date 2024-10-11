resource "authsignal_value_list" "example_value_list_strings" {
  name      = "Example Value List Strings"
  is_active = true
  value_list_items_strings = [
    "hello",
    "world",
    "I am a string",
  ]
}

resource "authsignal_value_list" "example_value_list_numbers" {
  name      = "Example Value List Numbers"
  is_active = true
  value_list_items_strings = [
    1,
    2,
    3,
  ]
}
