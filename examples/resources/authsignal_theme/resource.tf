resource "authsignal_theme" "theme" {
  name          = "My Tenant"
  logo_url      = "<url to an image>"
  favicon_url   = "<url to an image>"
  watermark_url = "<url to an image>"
  primary_color = "#ABCD12"
  borders = {
    button_border_radius    = 1
    button_border_width     = 2
    card_border_radius      = 3
    card_border_width       = 4
    input_border_radius     = 5
    input_border_width      = 6
    container_border_radius = 7
  }
  colors = {
    button_primary_text         = "#ABCD12"
    button_primary_border       = "#ABCD12"
    button_secondary_text       = "#ABCD12"
    button_secondary_background = "#ABCD12"
    button_secondary_border     = "#ABCD12"
    card_background             = "#ABCD12"
    card_border                 = "#ABCD12"
    input_background            = "#ABCD12"
    input_border                = "#ABCD12"
    link                        = "#ABCD12"
    heading_text                = "#ABCD12"
    body_text                   = "#ABCD12"
    container_background        = "#ABCD12"
    container_border            = "#ABCD12"
    divider                     = "#ABCD12"
    icon                        = "#ABCD12"
    loader                      = "#ABCD12"
    positive                    = "#ABCD12"
    critical                    = "#ABCD12"
    information                 = "#ABCD12"
    hover                       = "#ABCD12"
    focus                       = "#ABCD12"
  }
  page_background = {
    background_color     = "#ABCD12"
    background_image_url = "<url to an image>"
  }
  container = {
    content_alignment = "left"
    padding           = 1
    logo_alignment    = "center"
    logo_position     = "inside"
    logo_height       = 2
    position          = "outside"
  }
  dark_mode = {
    logo_url      = "<url to an image>"
    favicon_url   = "<url to an image>"
    watermark_url = "<url to an image>"
    primary_color = "#ABCD12"
    borders = {
      button_border_radius    = 1
      button_border_width     = 2
      card_border_radius      = 3
      card_border_width       = 4
      input_border_radius     = 5
      input_border_width      = 6
      container_border_radius = 7
    }
    colors = {
      button_primary_text         = "#ABCD12"
      button_primary_border       = "#ABCD12"
      button_secondary_text       = "#ABCD12"
      button_secondary_background = "#ABCD12"
      button_secondary_border     = "#ABCD12"
      card_background             = "#ABCD12"
      card_border                 = "#ABCD12"
      input_background            = "#ABCD12"
      input_border                = "#ABCD12"
      link                        = "#ABCD12"
      heading_text                = "#ABCD12"
      body_text                   = "#ABCD12"
      container_background        = "#ABCD12"
      container_border            = "#ABCD12"
      divider                     = "#ABCD12"
      icon                        = "#ABCD12"
      loader                      = "#ABCD12"
      positive                    = "#ABCD12"
      critical                    = "#ABCD12"
      information                 = "#ABCD12"
      hover                       = "#ABCD12"
      focus                       = "#ABCD12"
    }
    page_background = {
      background_color     = "#ABCD12"
      background_image_url = "<url to an image>"
    }
    container = {
      content_alignment = "center"
      padding           = 1
      logo_alignment    = "center"
      logo_position     = "inside"
      logo_height       = 2
    }
  }
}
