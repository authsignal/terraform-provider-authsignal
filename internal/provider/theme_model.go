package provider

import (
	"github.com/authsignal/authsignal-management-go/v3"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// THEME
type themeModel struct {
	Name           types.String `tfsdk:"name"`
	LogoUrl        types.String `tfsdk:"logo_url"`
	WatermarkUrl   types.String `tfsdk:"watermark_url"`
	FaviconUrl     types.String `tfsdk:"favicon_url"`
	PrimaryColor   types.String `tfsdk:"primary_color"`
	DarkMode       types.Object `tfsdk:"dark_mode"`
	Colors         types.Object `tfsdk:"colors"`
	Container      types.Object `tfsdk:"container"`
	Borders        types.Object `tfsdk:"borders"`
	Typography     types.Object `tfsdk:"typography"`
	PageBackground types.Object `tfsdk:"page_background"`
}

func (m *themeModel) CreateObject(input authsignal.ThemeResponse) types.Object {
	if len(input.Name) > 0 {
		m.Name = types.StringValue(input.Name)
	} else {
		m.Name = types.StringNull()
	}

	if len(input.LogoUrl) > 0 {
		m.LogoUrl = types.StringValue(input.LogoUrl)
	} else {
		m.LogoUrl = types.StringNull()
	}

	if len(input.WatermarkUrl) > 0 {
		m.WatermarkUrl = types.StringValue(input.WatermarkUrl)
	} else {
		m.WatermarkUrl = types.StringNull()
	}

	if len(input.FaviconUrl) > 0 {
		m.FaviconUrl = types.StringValue(input.FaviconUrl)
	} else {
		m.FaviconUrl = types.StringNull()
	}

	if len(input.PrimaryColor) > 0 {
		m.PrimaryColor = types.StringValue(input.PrimaryColor)
	} else {
		m.PrimaryColor = types.StringNull()
	}

	var colors colorsModel
	m.Colors = colors.CreateObject(input.Colors)

	var container containerModel
	m.Container = container.CreateObject(input.Container)

	var borders bordersModel
	m.Borders = borders.CreateObject(input.Borders)

	var typography typographyModel
	m.Typography = typography.CreateObject(input.Typography)

	var pageBackground pageBackgroundModel
	m.PageBackground = pageBackground.CreateObject(input.PageBackground)

	var darkMode darkModeModel
	m.DarkMode = darkMode.CreateObject(input.DarkMode)

	object, _ := types.ObjectValue(m.AttributeTypes(), m.AttributeValues())
	return object
}

func (m themeModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":            types.StringType,
		"logo_url":        types.StringType,
		"watermark_url":   types.StringType,
		"favicon_url":     types.StringType,
		"primary_color":   types.StringType,
		"dark_mode":       types.ObjectType{AttrTypes: darkModeModel{}.AttributeTypes()},
		"colors":          types.ObjectType{AttrTypes: colorsModel{}.AttributeTypes()},
		"container":       types.ObjectType{AttrTypes: containerModel{}.AttributeTypes()},
		"borders":         types.ObjectType{AttrTypes: bordersModel{}.AttributeTypes()},
		"typography":      types.ObjectType{AttrTypes: typographyModel{}.AttributeTypes()},
		"page_background": types.ObjectType{AttrTypes: pageBackgroundModel{}.AttributeTypes()},
	}
}

func (m themeModel) AttributeValues() map[string]attr.Value {
	elements := map[string]attr.Value{}
	elements["name"] = m.Name
	elements["logo_url"] = m.LogoUrl
	elements["watermark_url"] = m.WatermarkUrl
	elements["favicon_url"] = m.FaviconUrl
	elements["primary_color"] = m.PrimaryColor
	elements["colors"] = m.Colors
	elements["container"] = m.Container
	elements["borders"] = m.Borders
	elements["typography"] = m.Typography
	elements["page_background"] = m.PageBackground
	elements["dark_mode"] = m.DarkMode

	return elements
}

// DARK MODE
type darkModeModel struct {
	LogoUrl        types.String `tfsdk:"logo_url"`
	WatermarkUrl   types.String `tfsdk:"watermark_url"`
	FaviconUrl     types.String `tfsdk:"favicon_url"`
	PrimaryColor   types.String `tfsdk:"primary_color"`
	Colors         types.Object `tfsdk:"colors"`
	Container      types.Object `tfsdk:"container"`
	Borders        types.Object `tfsdk:"borders"`
	Typography     types.Object `tfsdk:"typography"`
	PageBackground types.Object `tfsdk:"page_background"`
}

func (m *darkModeModel) CreateObject(input authsignal.DarkModeResponse) types.Object {
	isNull := 1

	if len(input.LogoUrl) > 0 {
		isNull = 0
		m.LogoUrl = types.StringValue(input.LogoUrl)
	} else {
		m.LogoUrl = types.StringNull()
	}

	if len(input.WatermarkUrl) > 0 {
		isNull = 0
		m.WatermarkUrl = types.StringValue(input.WatermarkUrl)
	} else {
		m.WatermarkUrl = types.StringNull()
	}

	if len(input.FaviconUrl) > 0 {
		isNull = 0
		m.FaviconUrl = types.StringValue(input.FaviconUrl)
	} else {
		m.FaviconUrl = types.StringNull()
	}

	if len(input.PrimaryColor) > 0 {
		isNull = 0
		m.PrimaryColor = types.StringValue(input.PrimaryColor)
	} else {
		m.PrimaryColor = types.StringNull()
	}

	var colors colorsModel
	m.Colors = colors.CreateObject(input.Colors)
	if !m.Colors.IsNull() {
		isNull = 0
	}

	var container containerModel
	m.Container = container.CreateObject(input.Container)
	if !m.Container.IsNull() {
		isNull = 0
	}

	var borders bordersModel
	m.Borders = borders.CreateObject(input.Borders)
	if !m.Borders.IsNull() {
		isNull = 0
	}

	var typography typographyModel
	m.Typography = typography.CreateObject(input.Typography)
	if !m.Typography.IsNull() {
		isNull = 0
	}

	var pageBackground pageBackgroundModel
	m.PageBackground = pageBackground.CreateObject(input.PageBackground)
	if !m.PageBackground.IsNull() {
		isNull = 0
	}

	if isNull == 1 {
		return types.ObjectNull(m.AttributeTypes())
	}

	object, _ := types.ObjectValue(m.AttributeTypes(), m.AttributeValues())
	return object
}

func (m darkModeModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"logo_url":        types.StringType,
		"watermark_url":   types.StringType,
		"favicon_url":     types.StringType,
		"primary_color":   types.StringType,
		"colors":          types.ObjectType{AttrTypes: colorsModel{}.AttributeTypes()},
		"container":       types.ObjectType{AttrTypes: containerModel{}.AttributeTypes()},
		"borders":         types.ObjectType{AttrTypes: bordersModel{}.AttributeTypes()},
		"typography":      types.ObjectType{AttrTypes: typographyModel{}.AttributeTypes()},
		"page_background": types.ObjectType{AttrTypes: pageBackgroundModel{}.AttributeTypes()},
	}
}

func (m darkModeModel) AttributeValues() map[string]attr.Value {
	elements := map[string]attr.Value{}
	elements["logo_url"] = m.LogoUrl
	elements["watermark_url"] = m.WatermarkUrl
	elements["favicon_url"] = m.FaviconUrl
	elements["primary_color"] = m.PrimaryColor
	elements["colors"] = m.Colors
	elements["container"] = m.Container
	elements["borders"] = m.Borders
	elements["typography"] = m.Typography
	elements["page_background"] = m.PageBackground

	return elements
}

// COLORS
type colorsModel struct {
	ButtonPrimaryText         types.String `tfsdk:"button_primary_text"`
	ButtonPrimaryBorder       types.String `tfsdk:"button_primary_border"`
	ButtonSecondaryText       types.String `tfsdk:"button_secondary_text"`
	ButtonSecondaryBackground types.String `tfsdk:"button_secondary_background"`
	ButtonSecondaryBorder     types.String `tfsdk:"button_secondary_border"`
	CardBackground            types.String `tfsdk:"card_background"`
	CardBorder                types.String `tfsdk:"card_border"`
	InputBackground           types.String `tfsdk:"input_background"`
	InputBorder               types.String `tfsdk:"input_border"`
	Link                      types.String `tfsdk:"link"`
	HeadingText               types.String `tfsdk:"heading_text"`
	BodyText                  types.String `tfsdk:"body_text"`
	ContainerBackground       types.String `tfsdk:"container_background"`
	ContainerBorder           types.String `tfsdk:"container_border"`
	Divider                   types.String `tfsdk:"divider"`
	Icon                      types.String `tfsdk:"icon"`
	Loader                    types.String `tfsdk:"loader"`
	Positive                  types.String `tfsdk:"positive"`
	Critical                  types.String `tfsdk:"critical"`
	Information               types.String `tfsdk:"information"`
	Hover                     types.String `tfsdk:"hover"`
	Focus                     types.String `tfsdk:"focus"`
}

func (m *colorsModel) CreateObject(input authsignal.ColorsResponse) types.Object {
	isNull := 1
	if len(input.ButtonPrimaryText) > 0 {
		m.ButtonPrimaryText = types.StringValue(input.ButtonPrimaryText)
		isNull = 0
	} else {
		m.ButtonPrimaryText = types.StringNull()
	}

	if len(input.ButtonPrimaryBorder) > 0 {
		m.ButtonPrimaryBorder = types.StringValue(input.ButtonPrimaryBorder)
		isNull = 0
	} else {
		m.ButtonPrimaryBorder = types.StringNull()
	}

	if len(input.ButtonSecondaryText) > 0 {
		m.ButtonSecondaryText = types.StringValue(input.ButtonSecondaryText)
		isNull = 0
	} else {
		m.ButtonSecondaryText = types.StringNull()
	}

	if len(input.ButtonSecondaryBackground) > 0 {
		m.ButtonSecondaryBackground = types.StringValue(input.ButtonSecondaryBackground)
		isNull = 0
	} else {
		m.ButtonSecondaryBackground = types.StringNull()
	}

	if len(input.ButtonSecondaryBorder) > 0 {
		m.ButtonSecondaryBorder = types.StringValue(input.ButtonSecondaryBorder)
		isNull = 0
	} else {
		m.ButtonSecondaryBorder = types.StringNull()
	}

	if len(input.CardBackground) > 0 {
		m.CardBackground = types.StringValue(input.CardBackground)
		isNull = 0
	} else {
		m.CardBackground = types.StringNull()
	}

	if len(input.CardBorder) > 0 {
		m.CardBorder = types.StringValue(input.CardBorder)
		isNull = 0
	} else {
		m.CardBorder = types.StringNull()
	}

	if len(input.InputBackground) > 0 {
		m.InputBackground = types.StringValue(input.InputBackground)
		isNull = 0
	} else {
		m.InputBackground = types.StringNull()
	}

	if len(input.InputBorder) > 0 {
		m.InputBorder = types.StringValue(input.InputBorder)
		isNull = 0
	} else {
		m.InputBorder = types.StringNull()
	}

	if len(input.Link) > 0 {
		m.Link = types.StringValue(input.Link)
		isNull = 0
	} else {
		m.Link = types.StringNull()
	}

	if len(input.HeadingText) > 0 {
		m.HeadingText = types.StringValue(input.HeadingText)
		isNull = 0
	} else {
		m.HeadingText = types.StringNull()
	}

	if len(input.BodyText) > 0 {
		m.BodyText = types.StringValue(input.BodyText)
		isNull = 0
	} else {
		m.BodyText = types.StringNull()
	}

	if len(input.ContainerBackground) > 0 {
		m.ContainerBackground = types.StringValue(input.ContainerBackground)
		isNull = 0
	} else {
		m.ContainerBackground = types.StringNull()
	}

	if len(input.ContainerBorder) > 0 {
		m.ContainerBorder = types.StringValue(input.ContainerBorder)
		isNull = 0
	} else {
		m.ContainerBorder = types.StringNull()
	}

	if len(input.Divider) > 0 {
		m.Divider = types.StringValue(input.Divider)
		isNull = 0
	} else {
		m.Divider = types.StringNull()
	}

	if len(input.Icon) > 0 {
		m.Icon = types.StringValue(input.Icon)
		isNull = 0
	} else {
		m.Icon = types.StringNull()
	}

	if len(input.Loader) > 0 {
		m.Loader = types.StringValue(input.Loader)
		isNull = 0
	} else {
		m.Loader = types.StringNull()
	}

	if len(input.Positive) > 0 {
		m.Positive = types.StringValue(input.Positive)
		isNull = 0
	} else {
		m.Positive = types.StringNull()
	}

	if len(input.Critical) > 0 {
		m.Critical = types.StringValue(input.Critical)
		isNull = 0
	} else {
		m.Critical = types.StringNull()
	}

	if len(input.Information) > 0 {
		m.Information = types.StringValue(input.Information)
		isNull = 0
	} else {
		m.Information = types.StringNull()
	}

	if len(input.Hover) > 0 {
		m.Hover = types.StringValue(input.Hover)
		isNull = 0
	} else {
		m.Hover = types.StringNull()
	}

	if len(input.Focus) > 0 {
		m.Focus = types.StringValue(input.Focus)
		isNull = 0
	} else {
		m.Focus = types.StringNull()
	}

	if isNull == 1 {
		return types.ObjectNull(m.AttributeTypes())
	}

	object, _ := types.ObjectValue(m.AttributeTypes(), m.AttributeValues())
	return object
}

func (m colorsModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"button_primary_text":         types.StringType,
		"button_primary_border":       types.StringType,
		"button_secondary_text":       types.StringType,
		"button_secondary_background": types.StringType,
		"button_secondary_border":     types.StringType,
		"card_background":             types.StringType,
		"card_border":                 types.StringType,
		"input_background":            types.StringType,
		"input_border":                types.StringType,
		"link":                        types.StringType,
		"heading_text":                types.StringType,
		"body_text":                   types.StringType,
		"container_background":        types.StringType,
		"container_border":            types.StringType,
		"divider":                     types.StringType,
		"icon":                        types.StringType,
		"loader":                      types.StringType,
		"positive":                    types.StringType,
		"critical":                    types.StringType,
		"information":                 types.StringType,
		"hover":                       types.StringType,
		"focus":                       types.StringType,
	}
}

func (m colorsModel) AttributeValues() map[string]attr.Value {
	elements := map[string]attr.Value{}
	elements["button_primary_text"] = m.ButtonPrimaryText
	elements["button_primary_border"] = m.ButtonPrimaryBorder
	elements["button_secondary_text"] = m.ButtonSecondaryText
	elements["button_secondary_background"] = m.ButtonSecondaryBackground
	elements["button_secondary_border"] = m.ButtonSecondaryBorder
	elements["card_background"] = m.CardBackground
	elements["card_border"] = m.CardBorder
	elements["input_background"] = m.InputBackground
	elements["input_border"] = m.InputBorder
	elements["link"] = m.Link
	elements["heading_text"] = m.HeadingText
	elements["body_text"] = m.BodyText
	elements["container_background"] = m.ContainerBackground
	elements["container_border"] = m.ContainerBorder
	elements["divider"] = m.Divider
	elements["icon"] = m.Icon
	elements["loader"] = m.Loader
	elements["positive"] = m.Positive
	elements["critical"] = m.Critical
	elements["information"] = m.Information
	elements["hover"] = m.Hover
	elements["focus"] = m.Focus

	return elements
}

// CONTAINER
type containerModel struct {
	ContentAlignment types.String `tfsdk:"content_alignment"`
	Position         types.String `tfsdk:"position"`
	Padding          types.Int64  `tfsdk:"padding"`
	LogoAlignment    types.String `tfsdk:"logo_alignment"`
	LogoPosition     types.String `tfsdk:"logo_position"`
	LogoHeight       types.Int64  `tfsdk:"logo_height"`
}

func (m *containerModel) CreateObject(input authsignal.ContainerResponse) types.Object {
	isNull := 1
	if len(input.ContentAlignment) > 0 {
		isNull = 0
		m.ContentAlignment = types.StringValue(input.ContentAlignment)
	} else {
		m.ContentAlignment = types.StringNull()
	}

	if len(input.Position) > 0 {
		isNull = 0
		m.Position = types.StringValue(input.Position)
	} else {
		m.Position = types.StringNull()
	}

	if input.Padding != 0 {
		isNull = 0
		m.Padding = types.Int64Value(input.Padding)
	} else {
		m.Padding = types.Int64Null()
	}

	if len(input.LogoAlignment) > 0 {
		isNull = 0
		m.LogoAlignment = types.StringValue(input.LogoAlignment)
	} else {
		m.LogoAlignment = types.StringNull()
	}

	if len(input.LogoPosition) > 0 {
		isNull = 0
		m.LogoPosition = types.StringValue(input.LogoPosition)
	} else {
		m.LogoPosition = types.StringNull()
	}

	if input.LogoHeight != 0 {
		isNull = 0
		m.LogoHeight = types.Int64Value(input.LogoHeight)
	} else {
		m.LogoHeight = types.Int64Null()
	}

	if isNull == 1 {
		return types.ObjectNull(m.AttributeTypes())
	}

	object, _ := types.ObjectValue(m.AttributeTypes(), m.AttributeValues())
	return object
}

func (m containerModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"content_alignment": types.StringType,
		"position":          types.StringType,
		"padding":           types.Int64Type,
		"logo_alignment":    types.StringType,
		"logo_position":     types.StringType,
		"logo_height":       types.Int64Type,
	}
}

func (m containerModel) AttributeValues() map[string]attr.Value {
	elements := map[string]attr.Value{}
	elements["content_alignment"] = m.ContentAlignment
	elements["position"] = m.Position
	elements["padding"] = m.Padding
	elements["logo_alignment"] = m.LogoAlignment
	elements["logo_position"] = m.LogoPosition
	elements["logo_height"] = m.LogoHeight
	return elements
}

// BORDERS
type bordersModel struct {
	ButtonBorderRadius    types.Int64 `tfsdk:"button_border_radius"`
	ButtonBorderWidth     types.Int64 `tfsdk:"button_border_width"`
	CardBorderRadius      types.Int64 `tfsdk:"card_border_radius"`
	CardBorderWidth       types.Int64 `tfsdk:"card_border_width"`
	InputBorderRadius     types.Int64 `tfsdk:"input_border_radius"`
	InputBorderWidth      types.Int64 `tfsdk:"input_border_width"`
	ContainerBorderRadius types.Int64 `tfsdk:"container_border_radius"`
}

func (m *bordersModel) CreateObject(input authsignal.BordersResponse) types.Object {
	isNull := 1
	if input.ButtonBorderRadius != 0 {
		isNull = 0
		m.ButtonBorderRadius = types.Int64Value(input.ButtonBorderRadius)
	} else {
		m.ButtonBorderRadius = types.Int64Null()
	}

	if input.ButtonBorderWidth != 0 {
		isNull = 0
		m.ButtonBorderWidth = types.Int64Value(input.ButtonBorderWidth)
	} else {
		m.ButtonBorderWidth = types.Int64Null()
	}

	if input.CardBorderRadius != 0 {
		isNull = 0
		m.CardBorderRadius = types.Int64Value(input.CardBorderRadius)
	} else {
		m.CardBorderRadius = types.Int64Null()
	}

	if input.CardBorderWidth != 0 {
		isNull = 0
		m.CardBorderWidth = types.Int64Value(input.CardBorderWidth)
	} else {
		m.CardBorderWidth = types.Int64Null()
	}

	if input.InputBorderRadius != 0 {
		isNull = 0
		m.InputBorderRadius = types.Int64Value(input.InputBorderRadius)
	} else {
		m.InputBorderRadius = types.Int64Null()
	}

	if input.InputBorderWidth != 0 {
		isNull = 0
		m.InputBorderWidth = types.Int64Value(input.InputBorderWidth)
	} else {
		m.InputBorderWidth = types.Int64Null()
	}

	if input.ContainerBorderRadius != 0 {
		isNull = 0
		m.ContainerBorderRadius = types.Int64Value(input.ContainerBorderRadius)
	} else {
		m.ContainerBorderRadius = types.Int64Null()
	}

	if isNull == 1 {
		return types.ObjectNull(m.AttributeTypes())
	}

	object, _ := types.ObjectValue(m.AttributeTypes(), m.AttributeValues())
	return object
}

func (m bordersModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"button_border_radius":    types.Int64Type,
		"button_border_width":     types.Int64Type,
		"card_border_radius":      types.Int64Type,
		"card_border_width":       types.Int64Type,
		"input_border_radius":     types.Int64Type,
		"input_border_width":      types.Int64Type,
		"container_border_radius": types.Int64Type,
	}
}

func (m bordersModel) AttributeValues() map[string]attr.Value {
	elements := map[string]attr.Value{}
	elements["button_border_radius"] = m.ButtonBorderRadius
	elements["button_border_width"] = m.ButtonBorderWidth
	elements["card_border_radius"] = m.CardBorderRadius
	elements["card_border_width"] = m.CardBorderWidth
	elements["input_border_radius"] = m.InputBorderRadius
	elements["input_border_width"] = m.InputBorderWidth
	elements["container_border_radius"] = m.ContainerBorderRadius
	return elements
}

// DISPLAY
type displayModel struct {
	FontUrl types.String `tfsdk:"font_url"`
}

func (m *displayModel) CreateObject(input authsignal.DisplayResponse) types.Object {
	isNull := 1

	if len(input.FontUrl) > 0 {
		isNull = 0
		m.FontUrl = types.StringValue(input.FontUrl)
	} else {
		m.FontUrl = types.StringNull()
	}

	if isNull == 1 {
		return types.ObjectNull(m.AttributeTypes())
	}

	object, _ := types.ObjectValue(m.AttributeTypes(), m.AttributeValues())
	return object
}

func (m displayModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"font_url": types.StringType,
	}
}

func (m displayModel) AttributeValues() map[string]attr.Value {
	elements := map[string]attr.Value{}
	elements["font_url"] = m.FontUrl
	return elements
}

// TYPOGRAPHY
type typographyModel struct {
	Display types.Object `tfsdk:"display"`
}

func (m *typographyModel) CreateObject(input authsignal.TypographyResponse) types.Object {
	isNull := 1

	var display displayModel
	m.Display = display.CreateObject(input.Display)
	if !m.Display.IsNull() {
		isNull = 0
	}

	if isNull == 1 {
		return types.ObjectNull(m.AttributeTypes())
	}

	object, _ := types.ObjectValue(m.AttributeTypes(), m.AttributeValues())
	return object
}

func (m typographyModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"display": types.ObjectType{AttrTypes: displayModel{}.AttributeTypes()},
	}
}

func (m typographyModel) AttributeValues() map[string]attr.Value {
	elements := map[string]attr.Value{}
	elements["display"] = m.Display
	return elements
}

// PAGE BACKGROUND
type pageBackgroundModel struct {
	BackgroundColor    types.String `tfsdk:"background_color"`
	BackgroundImageUrl types.String `tfsdk:"background_image_url"`
}

func (m *pageBackgroundModel) CreateObject(input authsignal.PageBackgroundResponse) types.Object {
	isNull := 1

	if len(input.BackgroundColor) > 0 {
		isNull = 0
		m.BackgroundColor = types.StringValue(input.BackgroundColor)
	} else {
		m.BackgroundColor = types.StringNull()
	}

	if len(input.BackgroundImageUrl) > 0 {
		isNull = 0
		m.BackgroundImageUrl = types.StringValue(input.BackgroundImageUrl)
	} else {
		m.BackgroundImageUrl = types.StringNull()
	}

	if isNull == 1 {
		return types.ObjectNull(m.AttributeTypes())
	}

	object, _ := types.ObjectValue(m.AttributeTypes(), m.AttributeValues())
	return object
}

func (m pageBackgroundModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"background_color":     types.StringType,
		"background_image_url": types.StringType,
	}
}

func (m pageBackgroundModel) AttributeValues() map[string]attr.Value {
	elements := map[string]attr.Value{}
	elements["background_color"] = m.BackgroundColor
	elements["background_image_url"] = m.BackgroundImageUrl
	return elements
}
