package provider

import (
	"context"

	"github.com/authsignal/authsignal-management-go/v2"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// CREATE
func buildAuthsignalColorsCreateObject(colors colorsModel) authsignal.Colors {
	var authsignalColors authsignal.Colors

	if len(colors.ButtonPrimaryText.ValueString()) > 0 {
		authsignalColors.ButtonPrimaryText = authsignal.SetValue(colors.ButtonPrimaryText.ValueString())
	}

	if len(colors.ButtonPrimaryBorder.ValueString()) > 0 {
		authsignalColors.ButtonPrimaryBorder = authsignal.SetValue(colors.ButtonPrimaryBorder.ValueString())
	}

	if len(colors.ButtonSecondaryText.ValueString()) > 0 {
		authsignalColors.ButtonSecondaryText = authsignal.SetValue(colors.ButtonSecondaryText.ValueString())
	}

	if len(colors.ButtonSecondaryBackground.ValueString()) > 0 {
		authsignalColors.ButtonSecondaryBackground = authsignal.SetValue(colors.ButtonSecondaryBackground.ValueString())
	}

	if len(colors.ButtonSecondaryBorder.ValueString()) > 0 {
		authsignalColors.ButtonSecondaryBorder = authsignal.SetValue(colors.ButtonSecondaryBorder.ValueString())
	}

	if len(colors.CardBackground.ValueString()) > 0 {
		authsignalColors.CardBackground = authsignal.SetValue(colors.CardBackground.ValueString())
	}

	if len(colors.CardBorder.ValueString()) > 0 {
		authsignalColors.CardBorder = authsignal.SetValue(colors.CardBorder.ValueString())
	}

	if len(colors.InputBackground.ValueString()) > 0 {
		authsignalColors.InputBackground = authsignal.SetValue(colors.InputBackground.ValueString())
	}

	if len(colors.InputBorder.ValueString()) > 0 {
		authsignalColors.InputBorder = authsignal.SetValue(colors.InputBorder.ValueString())
	}

	if len(colors.Link.ValueString()) > 0 {
		authsignalColors.Link = authsignal.SetValue(colors.Link.ValueString())
	}

	if len(colors.HeadingText.ValueString()) > 0 {
		authsignalColors.HeadingText = authsignal.SetValue(colors.HeadingText.ValueString())
	}

	if len(colors.BodyText.ValueString()) > 0 {
		authsignalColors.BodyText = authsignal.SetValue(colors.BodyText.ValueString())
	}

	if len(colors.ContainerBackground.ValueString()) > 0 {
		authsignalColors.ContainerBackground = authsignal.SetValue(colors.ContainerBackground.ValueString())
	}

	if len(colors.ContainerBorder.ValueString()) > 0 {
		authsignalColors.ContainerBorder = authsignal.SetValue(colors.ContainerBorder.ValueString())
	}

	if len(colors.Divider.ValueString()) > 0 {
		authsignalColors.Divider = authsignal.SetValue(colors.Divider.ValueString())
	}

	if len(colors.Icon.ValueString()) > 0 {
		authsignalColors.Icon = authsignal.SetValue(colors.Icon.ValueString())
	}

	if len(colors.Loader.ValueString()) > 0 {
		authsignalColors.Loader = authsignal.SetValue(colors.Loader.ValueString())
	}

	if len(colors.Positive.ValueString()) > 0 {
		authsignalColors.Positive = authsignal.SetValue(colors.Positive.ValueString())
	}

	if len(colors.Critical.ValueString()) > 0 {
		authsignalColors.Critical = authsignal.SetValue(colors.Critical.ValueString())
	}

	if len(colors.Information.ValueString()) > 0 {
		authsignalColors.Information = authsignal.SetValue(colors.Information.ValueString())
	}

	if len(colors.Hover.ValueString()) > 0 {
		authsignalColors.Hover = authsignal.SetValue(colors.Hover.ValueString())
	}

	if len(colors.Focus.ValueString()) > 0 {
		authsignalColors.Focus = authsignal.SetValue(colors.Focus.ValueString())
	}

	return authsignalColors
}

func buildAuthsignalDisplayCreateObject(display displayModel) authsignal.Display {
	var authsignalDisplay authsignal.Display

	if len(display.FontUrl.ValueString()) > 0 {
		authsignalDisplay.FontUrl = authsignal.SetValue(display.FontUrl.ValueString())
	}

	return authsignalDisplay
}

func buildAuthsignalTypographyCreateObject(ctx context.Context, resp *resource.CreateResponse, typography typographyModel) authsignal.Typography {
	var authsignalTypography authsignal.Typography
	var displayValues displayModel

	if !typography.Display.IsNull() {
		diags := typography.Display.As(ctx, &displayValues, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
	}

	authsignalTypography.Display = authsignal.SetValue(buildAuthsignalDisplayCreateObject(displayValues))

	return authsignalTypography
}

func buildAuthsignalContainerCreateObject(container containerModel) authsignal.Container {
	var authsignalContainer authsignal.Container

	if len(container.ContentAlignment.ValueString()) > 0 {
		authsignalContainer.ContentAlignment = authsignal.SetValue(container.ContentAlignment.ValueString())
	}

	if len(container.Position.ValueString()) > 0 {
		authsignalContainer.Position = authsignal.SetValue(container.Position.ValueString())
	}

	if container.Padding.ValueInt64() != 0 {
		authsignalContainer.Padding = authsignal.SetValue(container.Padding.ValueInt64())
	}

	if len(container.LogoAlignment.ValueString()) > 0 {
		authsignalContainer.LogoAlignment = authsignal.SetValue(container.LogoAlignment.ValueString())
	}

	if len(container.LogoPosition.ValueString()) > 0 {
		authsignalContainer.LogoPosition = authsignal.SetValue(container.LogoPosition.ValueString())
	}

	if container.LogoHeight.ValueInt64() != 0 {
		authsignalContainer.LogoHeight = authsignal.SetValue(container.LogoHeight.ValueInt64())
	}

	return authsignalContainer
}

func buildAuthsignalBordersCreateObject(borders bordersModel) authsignal.Borders {
	var authsignalBorders authsignal.Borders

	if borders.ButtonBorderRadius.ValueInt64() != 0 {
		authsignalBorders.ButtonBorderRadius = authsignal.SetValue(borders.ButtonBorderRadius.ValueInt64())
	}

	if borders.ButtonBorderWidth.ValueInt64() != 0 {
		authsignalBorders.ButtonBorderWidth = authsignal.SetValue(borders.ButtonBorderWidth.ValueInt64())
	}

	if borders.CardBorderRadius.ValueInt64() != 0 {
		authsignalBorders.CardBorderRadius = authsignal.SetValue(borders.CardBorderRadius.ValueInt64())
	}

	if borders.CardBorderWidth.ValueInt64() != 0 {
		authsignalBorders.CardBorderWidth = authsignal.SetValue(borders.CardBorderWidth.ValueInt64())
	}

	if borders.InputBorderRadius.ValueInt64() != 0 {
		authsignalBorders.InputBorderRadius = authsignal.SetValue(borders.InputBorderRadius.ValueInt64())
	}

	if borders.InputBorderWidth.ValueInt64() != 0 {
		authsignalBorders.InputBorderWidth = authsignal.SetValue(borders.InputBorderWidth.ValueInt64())
	}

	if borders.ContainerBorderRadius.ValueInt64() != 0 {
		authsignalBorders.ContainerBorderRadius = authsignal.SetValue(borders.ContainerBorderRadius.ValueInt64())
	}

	return authsignalBorders
}

func buildAuthsignalPageBackgroundCreateObject(pageBackground pageBackgroundModel) authsignal.PageBackground {
	var authsignalPageBackground authsignal.PageBackground

	if len(pageBackground.BackgroundColor.ValueString()) > 0 {
		authsignalPageBackground.BackgroundColor = authsignal.SetValue(pageBackground.BackgroundColor.ValueString())
	}

	if len(pageBackground.BackgroundImageUrl.ValueString()) > 0 {
		authsignalPageBackground.BackgroundImageUrl = authsignal.SetValue(pageBackground.BackgroundImageUrl.ValueString())
	}

	return authsignalPageBackground
}

func buildAuthsignalThemeCreateObject(ctx context.Context, resp *resource.CreateResponse, input themeModel) authsignal.Theme {
	var darkModeValues darkModeModel
	var darkModeColorsValues colorsModel
	var darkModeBordersValues bordersModel
	var darkModeContainerValues containerModel
	var darkModeTypographyValues typographyModel
	var darkModePageBackgroundValues pageBackgroundModel

	if !input.DarkMode.IsNull() {
		diags := input.DarkMode.As(ctx, &darkModeValues, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)

		if !darkModeValues.Colors.IsNull() {
			diags = darkModeValues.Colors.As(ctx, &darkModeColorsValues, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(diags...)
		}

		if !darkModeValues.Borders.IsNull() {
			diags = darkModeValues.Borders.As(ctx, &darkModeBordersValues, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(diags...)
		}

		if !darkModeValues.Container.IsNull() {
			diags = darkModeValues.Container.As(ctx, &darkModeContainerValues, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(diags...)
		}

		if !darkModeValues.PageBackground.IsNull() {
			diags = darkModeValues.PageBackground.As(ctx, &darkModePageBackgroundValues, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(diags...)
		}

		if !darkModeValues.Typography.IsNull() {
			diags = darkModeValues.Typography.As(ctx, &darkModeTypographyValues, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(diags...)
		}
	}

	var colorsValues colorsModel
	var bordersValues bordersModel
	var containerValues containerModel
	var typographyValues typographyModel
	var pageBackgroundValues pageBackgroundModel

	if !input.Colors.IsNull() {
		diags := input.Colors.As(ctx, &colorsValues, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
	}

	if !input.Borders.IsNull() {
		diags := input.Borders.As(ctx, &bordersValues, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
	}

	if !input.Container.IsNull() {
		diags := input.Container.As(ctx, &containerValues, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
	}

	if !input.Typography.IsNull() {
		diags := input.Typography.As(ctx, &typographyValues, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
	}

	if !input.PageBackground.IsNull() {
		diags := input.PageBackground.As(ctx, &pageBackgroundValues, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
	}

	var authsignalDarkMode authsignal.DarkMode
	authsignalDarkMode.Colors = authsignal.SetValue(buildAuthsignalColorsCreateObject(darkModeColorsValues))
	authsignalDarkMode.Borders = authsignal.SetValue(buildAuthsignalBordersCreateObject(darkModeBordersValues))
	authsignalDarkMode.Container = authsignal.SetValue(buildAuthsignalContainerCreateObject(darkModeContainerValues))
	authsignalDarkMode.PageBackground = authsignal.SetValue(buildAuthsignalPageBackgroundCreateObject(darkModePageBackgroundValues))
	authsignalDarkMode.Typography = authsignal.SetValue(buildAuthsignalTypographyCreateObject(ctx, resp, darkModeTypographyValues))

	if len(darkModeValues.LogoUrl.ValueString()) > 0 {
		authsignalDarkMode.LogoUrl = authsignal.SetValue(darkModeValues.LogoUrl.ValueString())
	}

	if len(darkModeValues.WatermarkUrl.ValueString()) > 0 {
		authsignalDarkMode.WatermarkUrl = authsignal.SetValue(darkModeValues.WatermarkUrl.ValueString())
	}

	if len(darkModeValues.FaviconUrl.ValueString()) > 0 {
		authsignalDarkMode.FaviconUrl = authsignal.SetValue(darkModeValues.FaviconUrl.ValueString())
	}

	if len(darkModeValues.PrimaryColor.ValueString()) > 0 {
		authsignalDarkMode.PrimaryColor = authsignal.SetValue(darkModeValues.PrimaryColor.ValueString())
	}

	var authsignalTheme authsignal.Theme

	authsignalTheme.Colors = authsignal.SetValue(buildAuthsignalColorsCreateObject(colorsValues))
	authsignalTheme.Borders = authsignal.SetValue(buildAuthsignalBordersCreateObject(bordersValues))
	authsignalTheme.Container = authsignal.SetValue(buildAuthsignalContainerCreateObject(containerValues))
	authsignalTheme.PageBackground = authsignal.SetValue(buildAuthsignalPageBackgroundCreateObject(pageBackgroundValues))
	authsignalTheme.Typography = authsignal.SetValue(buildAuthsignalTypographyCreateObject(ctx, resp, typographyValues))
	authsignalTheme.DarkMode = authsignal.SetValue(authsignalDarkMode)

	if len(input.Name.ValueString()) > 0 {
		authsignalTheme.Name = authsignal.SetValue(input.Name.ValueString())
	}

	if len(input.LogoUrl.ValueString()) > 0 {
		authsignalTheme.LogoUrl = authsignal.SetValue(input.LogoUrl.ValueString())
	}

	if len(input.WatermarkUrl.ValueString()) > 0 {
		authsignalTheme.WatermarkUrl = authsignal.SetValue(input.WatermarkUrl.ValueString())
	}

	if len(input.FaviconUrl.ValueString()) > 0 {
		authsignalTheme.FaviconUrl = authsignal.SetValue(input.FaviconUrl.ValueString())
	}

	if len(input.PrimaryColor.ValueString()) > 0 {
		authsignalTheme.PrimaryColor = authsignal.SetValue(input.PrimaryColor.ValueString())
	}

	return authsignalTheme
}

// UPDATE
func buildAuthsignalColorsUpdateObject(colors colorsModel) authsignal.Colors {
	var authsignalColors authsignal.Colors

	if len(colors.ButtonPrimaryText.ValueString()) > 0 {
		authsignalColors.ButtonPrimaryText = authsignal.SetValue(colors.ButtonPrimaryText.ValueString())
	} else {
		authsignalColors.ButtonPrimaryText = authsignal.SetNull(colors.ButtonPrimaryText.ValueString())
	}

	if len(colors.ButtonPrimaryBorder.ValueString()) > 0 {
		authsignalColors.ButtonPrimaryBorder = authsignal.SetValue(colors.ButtonPrimaryBorder.ValueString())
	} else {
		authsignalColors.ButtonPrimaryBorder = authsignal.SetNull(colors.ButtonPrimaryBorder.ValueString())
	}

	if len(colors.ButtonSecondaryText.ValueString()) > 0 {
		authsignalColors.ButtonSecondaryText = authsignal.SetValue(colors.ButtonSecondaryText.ValueString())
	} else {
		authsignalColors.ButtonSecondaryText = authsignal.SetNull(colors.ButtonSecondaryText.ValueString())
	}

	if len(colors.ButtonSecondaryBackground.ValueString()) > 0 {
		authsignalColors.ButtonSecondaryBackground = authsignal.SetValue(colors.ButtonSecondaryBackground.ValueString())
	} else {
		authsignalColors.ButtonSecondaryBackground = authsignal.SetNull(colors.ButtonSecondaryBackground.ValueString())
	}

	if len(colors.ButtonSecondaryBorder.ValueString()) > 0 {
		authsignalColors.ButtonSecondaryBorder = authsignal.SetValue(colors.ButtonSecondaryBorder.ValueString())
	} else {
		authsignalColors.ButtonSecondaryBorder = authsignal.SetNull(colors.ButtonSecondaryBorder.ValueString())
	}

	if len(colors.CardBackground.ValueString()) > 0 {
		authsignalColors.CardBackground = authsignal.SetValue(colors.CardBackground.ValueString())
	} else {
		authsignalColors.CardBackground = authsignal.SetNull(colors.CardBackground.ValueString())
	}

	if len(colors.CardBorder.ValueString()) > 0 {
		authsignalColors.CardBorder = authsignal.SetValue(colors.CardBorder.ValueString())
	} else {
		authsignalColors.CardBorder = authsignal.SetNull(colors.CardBorder.ValueString())
	}

	if len(colors.InputBackground.ValueString()) > 0 {
		authsignalColors.InputBackground = authsignal.SetValue(colors.InputBackground.ValueString())
	} else {
		authsignalColors.InputBackground = authsignal.SetNull(colors.InputBackground.ValueString())
	}

	if len(colors.InputBorder.ValueString()) > 0 {
		authsignalColors.InputBorder = authsignal.SetValue(colors.InputBorder.ValueString())
	} else {
		authsignalColors.InputBorder = authsignal.SetNull(colors.InputBorder.ValueString())
	}

	if len(colors.Link.ValueString()) > 0 {
		authsignalColors.Link = authsignal.SetValue(colors.Link.ValueString())
	} else {
		authsignalColors.Link = authsignal.SetNull(colors.Link.ValueString())
	}

	if len(colors.HeadingText.ValueString()) > 0 {
		authsignalColors.HeadingText = authsignal.SetValue(colors.HeadingText.ValueString())
	} else {
		authsignalColors.HeadingText = authsignal.SetNull(colors.HeadingText.ValueString())
	}

	if len(colors.BodyText.ValueString()) > 0 {
		authsignalColors.BodyText = authsignal.SetValue(colors.BodyText.ValueString())
	} else {
		authsignalColors.BodyText = authsignal.SetNull(colors.BodyText.ValueString())
	}

	if len(colors.ContainerBackground.ValueString()) > 0 {
		authsignalColors.ContainerBackground = authsignal.SetValue(colors.ContainerBackground.ValueString())
	} else {
		authsignalColors.ContainerBackground = authsignal.SetNull(colors.ContainerBackground.ValueString())
	}

	if len(colors.ContainerBorder.ValueString()) > 0 {
		authsignalColors.ContainerBorder = authsignal.SetValue(colors.ContainerBorder.ValueString())
	} else {
		authsignalColors.ContainerBorder = authsignal.SetNull(colors.ContainerBorder.ValueString())
	}

	if len(colors.Divider.ValueString()) > 0 {
		authsignalColors.Divider = authsignal.SetValue(colors.Divider.ValueString())
	} else {
		authsignalColors.Divider = authsignal.SetNull(colors.Divider.ValueString())
	}

	if len(colors.Icon.ValueString()) > 0 {
		authsignalColors.Icon = authsignal.SetValue(colors.Icon.ValueString())
	} else {
		authsignalColors.Icon = authsignal.SetNull(colors.Icon.ValueString())
	}

	if len(colors.Loader.ValueString()) > 0 {
		authsignalColors.Loader = authsignal.SetValue(colors.Loader.ValueString())
	} else {
		authsignalColors.Loader = authsignal.SetNull(colors.Loader.ValueString())
	}

	if len(colors.Positive.ValueString()) > 0 {
		authsignalColors.Positive = authsignal.SetValue(colors.Positive.ValueString())
	} else {
		authsignalColors.Positive = authsignal.SetNull(colors.Positive.ValueString())
	}

	if len(colors.Critical.ValueString()) > 0 {
		authsignalColors.Critical = authsignal.SetValue(colors.Critical.ValueString())
	} else {
		authsignalColors.Critical = authsignal.SetNull(colors.Critical.ValueString())
	}

	if len(colors.Information.ValueString()) > 0 {
		authsignalColors.Information = authsignal.SetValue(colors.Information.ValueString())
	} else {
		authsignalColors.Information = authsignal.SetNull(colors.Information.ValueString())
	}

	if len(colors.Hover.ValueString()) > 0 {
		authsignalColors.Hover = authsignal.SetValue(colors.Hover.ValueString())
	} else {
		authsignalColors.Hover = authsignal.SetNull(colors.Hover.ValueString())
	}

	if len(colors.Focus.ValueString()) > 0 {
		authsignalColors.Focus = authsignal.SetValue(colors.Focus.ValueString())
	} else {
		authsignalColors.Focus = authsignal.SetNull(colors.Focus.ValueString())
	}

	return authsignalColors
}

func buildAuthsignalDisplayUpdateObject(display displayModel) authsignal.Display {
	var authsignalDisplay authsignal.Display

	if len(display.FontUrl.ValueString()) > 0 {
		authsignalDisplay.FontUrl = authsignal.SetValue(display.FontUrl.ValueString())
	} else {
		authsignalDisplay.FontUrl = authsignal.SetNull(display.FontUrl.ValueString())
	}

	return authsignalDisplay
}

func buildAuthsignalTypographyUpdateObject(ctx context.Context, resp *resource.UpdateResponse, typography typographyModel) authsignal.Typography {
	var authsignalTypography authsignal.Typography
	var displayValues displayModel

	if !typography.Display.IsNull() {
		diags := typography.Display.As(ctx, &displayValues, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
	}

	authsignalTypography.Display = authsignal.SetValue(buildAuthsignalDisplayUpdateObject(displayValues))

	return authsignalTypography
}

func buildAuthsignalContainerUpdateObject(container containerModel) authsignal.Container {
	var authsignalContainer authsignal.Container

	if len(container.ContentAlignment.ValueString()) > 0 {
		authsignalContainer.ContentAlignment = authsignal.SetValue(container.ContentAlignment.ValueString())
	} else {
		authsignalContainer.ContentAlignment = authsignal.SetNull(container.ContentAlignment.ValueString())
	}

	if len(container.Position.ValueString()) > 0 {
		authsignalContainer.Position = authsignal.SetValue(container.Position.ValueString())
	} else {
		authsignalContainer.Position = authsignal.SetNull(container.Position.ValueString())
	}

	if container.Padding.ValueInt64() != 0 {
		authsignalContainer.Padding = authsignal.SetValue(container.Padding.ValueInt64())
	} else {
		authsignalContainer.Padding = authsignal.SetNull(container.Padding.ValueInt64())
	}

	if len(container.LogoAlignment.ValueString()) > 0 {
		authsignalContainer.LogoAlignment = authsignal.SetValue(container.LogoAlignment.ValueString())
	} else {
		authsignalContainer.LogoAlignment = authsignal.SetNull(container.LogoAlignment.ValueString())
	}

	if len(container.LogoPosition.ValueString()) > 0 {
		authsignalContainer.LogoPosition = authsignal.SetValue(container.LogoPosition.ValueString())
	} else {
		authsignalContainer.LogoPosition = authsignal.SetNull(container.LogoPosition.ValueString())
	}

	if container.LogoHeight.ValueInt64() != 0 {
		authsignalContainer.LogoHeight = authsignal.SetValue(container.LogoHeight.ValueInt64())
	} else {
		authsignalContainer.LogoHeight = authsignal.SetNull(container.LogoHeight.ValueInt64())
	}

	return authsignalContainer
}

func buildAuthsignalBordersUpdateObject(borders bordersModel) authsignal.Borders {
	var authsignalBorders authsignal.Borders

	if borders.ButtonBorderRadius.ValueInt64() != 0 {
		authsignalBorders.ButtonBorderRadius = authsignal.SetValue(borders.ButtonBorderRadius.ValueInt64())
	} else {
		authsignalBorders.ButtonBorderRadius = authsignal.SetNull(borders.ButtonBorderRadius.ValueInt64())
	}

	if borders.ButtonBorderWidth.ValueInt64() != 0 {
		authsignalBorders.ButtonBorderWidth = authsignal.SetValue(borders.ButtonBorderWidth.ValueInt64())
	} else {
		authsignalBorders.ButtonBorderWidth = authsignal.SetNull(borders.ButtonBorderWidth.ValueInt64())
	}

	if borders.CardBorderRadius.ValueInt64() != 0 {
		authsignalBorders.CardBorderRadius = authsignal.SetValue(borders.CardBorderRadius.ValueInt64())
	} else {
		authsignalBorders.CardBorderRadius = authsignal.SetNull(borders.CardBorderRadius.ValueInt64())
	}

	if borders.CardBorderWidth.ValueInt64() != 0 {
		authsignalBorders.CardBorderWidth = authsignal.SetValue(borders.CardBorderWidth.ValueInt64())
	} else {
		authsignalBorders.CardBorderWidth = authsignal.SetNull(borders.CardBorderWidth.ValueInt64())
	}

	if borders.InputBorderRadius.ValueInt64() != 0 {
		authsignalBorders.InputBorderRadius = authsignal.SetValue(borders.InputBorderRadius.ValueInt64())
	} else {
		authsignalBorders.InputBorderRadius = authsignal.SetNull(borders.InputBorderRadius.ValueInt64())
	}

	if borders.InputBorderWidth.ValueInt64() != 0 {
		authsignalBorders.InputBorderWidth = authsignal.SetValue(borders.InputBorderWidth.ValueInt64())
	} else {
		authsignalBorders.InputBorderWidth = authsignal.SetNull(borders.InputBorderWidth.ValueInt64())
	}

	if borders.ContainerBorderRadius.ValueInt64() != 0 {
		authsignalBorders.ContainerBorderRadius = authsignal.SetValue(borders.ContainerBorderRadius.ValueInt64())
	} else {
		authsignalBorders.ContainerBorderRadius = authsignal.SetNull(borders.ContainerBorderRadius.ValueInt64())
	}

	return authsignalBorders
}

func buildAuthsignalPageBackgroundUpdateObject(pageBackground pageBackgroundModel) authsignal.PageBackground {
	var authsignalPageBackground authsignal.PageBackground

	if len(pageBackground.BackgroundColor.ValueString()) > 0 {
		authsignalPageBackground.BackgroundColor = authsignal.SetValue(pageBackground.BackgroundColor.ValueString())
	} else {
		authsignalPageBackground.BackgroundColor = authsignal.SetNull(pageBackground.BackgroundColor.ValueString())
	}

	if len(pageBackground.BackgroundImageUrl.ValueString()) > 0 {
		authsignalPageBackground.BackgroundImageUrl = authsignal.SetValue(pageBackground.BackgroundImageUrl.ValueString())
	} else {
		authsignalPageBackground.BackgroundImageUrl = authsignal.SetNull(pageBackground.BackgroundImageUrl.ValueString())
	}

	return authsignalPageBackground
}

func buildAuthsignalThemeUpdateObject(ctx context.Context, resp *resource.UpdateResponse, input themeModel) authsignal.Theme {
	var darkModeValues darkModeModel
	var darkModeColorsValues colorsModel
	var darkModeBordersValues bordersModel
	var darkModeContainerValues containerModel
	var darkModeTypographyValues typographyModel
	var darkModePageBackgroundValues pageBackgroundModel

	if !input.DarkMode.IsNull() {
		diags := input.DarkMode.As(ctx, &darkModeValues, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)

		if !darkModeValues.Colors.IsNull() {
			diags = darkModeValues.Colors.As(ctx, &darkModeColorsValues, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(diags...)
		}

		if !darkModeValues.Borders.IsNull() {
			diags = darkModeValues.Borders.As(ctx, &darkModeBordersValues, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(diags...)
		}

		if !darkModeValues.Container.IsNull() {
			diags = darkModeValues.Container.As(ctx, &darkModeContainerValues, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(diags...)
		}

		if !darkModeValues.PageBackground.IsNull() {
			diags = darkModeValues.PageBackground.As(ctx, &darkModePageBackgroundValues, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(diags...)
		}

		if !darkModeValues.Typography.IsNull() {
			diags = darkModeValues.Typography.As(ctx, &darkModeTypographyValues, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(diags...)
		}
	}

	var colorsValues colorsModel
	var bordersValues bordersModel
	var containerValues containerModel
	var typographyValues typographyModel
	var pageBackgroundValues pageBackgroundModel

	if !input.Colors.IsNull() {
		diags := input.Colors.As(ctx, &colorsValues, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
	}

	if !input.Borders.IsNull() {
		diags := input.Borders.As(ctx, &bordersValues, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
	}

	if !input.Container.IsNull() {
		diags := input.Container.As(ctx, &containerValues, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
	}

	if !input.Typography.IsNull() {
		diags := input.Typography.As(ctx, &typographyValues, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
	}

	if !input.PageBackground.IsNull() {
		diags := input.PageBackground.As(ctx, &pageBackgroundValues, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
	}

	var authsignalDarkMode authsignal.DarkMode
	authsignalDarkMode.Colors = authsignal.SetValue(buildAuthsignalColorsUpdateObject(darkModeColorsValues))
	authsignalDarkMode.Borders = authsignal.SetValue(buildAuthsignalBordersUpdateObject(darkModeBordersValues))
	authsignalDarkMode.Container = authsignal.SetValue(buildAuthsignalContainerUpdateObject(darkModeContainerValues))
	authsignalDarkMode.PageBackground = authsignal.SetValue(buildAuthsignalPageBackgroundUpdateObject(darkModePageBackgroundValues))
	authsignalDarkMode.Typography = authsignal.SetValue(buildAuthsignalTypographyUpdateObject(ctx, resp, darkModeTypographyValues))

	if len(darkModeValues.LogoUrl.ValueString()) > 0 {
		authsignalDarkMode.LogoUrl = authsignal.SetValue(darkModeValues.LogoUrl.ValueString())
	} else {
		authsignalDarkMode.LogoUrl = authsignal.SetNull(darkModeValues.LogoUrl.ValueString())
	}

	if len(darkModeValues.WatermarkUrl.ValueString()) > 0 {
		authsignalDarkMode.WatermarkUrl = authsignal.SetValue(darkModeValues.WatermarkUrl.ValueString())
	} else {
		authsignalDarkMode.WatermarkUrl = authsignal.SetNull(darkModeValues.WatermarkUrl.ValueString())
	}

	if len(darkModeValues.FaviconUrl.ValueString()) > 0 {
		authsignalDarkMode.FaviconUrl = authsignal.SetValue(darkModeValues.FaviconUrl.ValueString())
	} else {
		authsignalDarkMode.FaviconUrl = authsignal.SetNull(darkModeValues.FaviconUrl.ValueString())
	}

	if len(darkModeValues.PrimaryColor.ValueString()) > 0 {
		authsignalDarkMode.PrimaryColor = authsignal.SetValue(darkModeValues.PrimaryColor.ValueString())
	} else {
		authsignalDarkMode.PrimaryColor = authsignal.SetNull(darkModeValues.PrimaryColor.ValueString())
	}

	var authsignalTheme authsignal.Theme

	authsignalTheme.Colors = authsignal.SetValue(buildAuthsignalColorsUpdateObject(colorsValues))
	authsignalTheme.Borders = authsignal.SetValue(buildAuthsignalBordersUpdateObject(bordersValues))
	authsignalTheme.Container = authsignal.SetValue(buildAuthsignalContainerUpdateObject(containerValues))
	authsignalTheme.PageBackground = authsignal.SetValue(buildAuthsignalPageBackgroundUpdateObject(pageBackgroundValues))
	authsignalTheme.Typography = authsignal.SetValue(buildAuthsignalTypographyUpdateObject(ctx, resp, typographyValues))
	authsignalTheme.DarkMode = authsignal.SetValue(authsignalDarkMode)

	if len(input.Name.ValueString()) > 0 {
		authsignalTheme.Name = authsignal.SetValue(input.Name.ValueString())
	} else {
		authsignalTheme.Name = authsignal.SetNull(input.Name.ValueString())
	}

	if len(input.LogoUrl.ValueString()) > 0 {
		authsignalTheme.LogoUrl = authsignal.SetValue(input.LogoUrl.ValueString())
	} else {
		authsignalTheme.LogoUrl = authsignal.SetNull(input.LogoUrl.ValueString())
	}

	if len(input.WatermarkUrl.ValueString()) > 0 {
		authsignalTheme.WatermarkUrl = authsignal.SetValue(input.WatermarkUrl.ValueString())
	} else {
		authsignalTheme.WatermarkUrl = authsignal.SetNull(input.WatermarkUrl.ValueString())
	}

	if len(input.FaviconUrl.ValueString()) > 0 {
		authsignalTheme.FaviconUrl = authsignal.SetValue(input.FaviconUrl.ValueString())
	} else {
		authsignalTheme.FaviconUrl = authsignal.SetNull(input.FaviconUrl.ValueString())
	}

	if len(input.PrimaryColor.ValueString()) > 0 {
		authsignalTheme.PrimaryColor = authsignal.SetValue(input.PrimaryColor.ValueString())
	} else {
		authsignalTheme.PrimaryColor = authsignal.SetNull(input.PrimaryColor.ValueString())
	}

	return authsignalTheme
}

// DELETE
func buildAuthsignalColorsDeleteObject(colors colorsModel) authsignal.Colors {
	var authsignalColors authsignal.Colors

	authsignalColors.ButtonPrimaryText = authsignal.SetNull(colors.ButtonPrimaryText.ValueString())
	authsignalColors.ButtonPrimaryBorder = authsignal.SetNull(colors.ButtonPrimaryBorder.ValueString())
	authsignalColors.ButtonSecondaryText = authsignal.SetNull(colors.ButtonSecondaryText.ValueString())
	authsignalColors.ButtonSecondaryBackground = authsignal.SetNull(colors.ButtonSecondaryBackground.ValueString())
	authsignalColors.ButtonSecondaryBorder = authsignal.SetNull(colors.ButtonSecondaryBorder.ValueString())
	authsignalColors.CardBackground = authsignal.SetNull(colors.CardBackground.ValueString())
	authsignalColors.CardBorder = authsignal.SetNull(colors.CardBorder.ValueString())
	authsignalColors.InputBackground = authsignal.SetNull(colors.InputBackground.ValueString())
	authsignalColors.InputBorder = authsignal.SetNull(colors.InputBorder.ValueString())
	authsignalColors.Link = authsignal.SetNull(colors.Link.ValueString())
	authsignalColors.HeadingText = authsignal.SetNull(colors.HeadingText.ValueString())
	authsignalColors.BodyText = authsignal.SetNull(colors.BodyText.ValueString())
	authsignalColors.ContainerBackground = authsignal.SetNull(colors.ContainerBackground.ValueString())
	authsignalColors.ContainerBorder = authsignal.SetNull(colors.ContainerBorder.ValueString())
	authsignalColors.Divider = authsignal.SetNull(colors.Divider.ValueString())
	authsignalColors.Icon = authsignal.SetNull(colors.Icon.ValueString())
	authsignalColors.Loader = authsignal.SetNull(colors.Loader.ValueString())
	authsignalColors.Positive = authsignal.SetNull(colors.Positive.ValueString())
	authsignalColors.Critical = authsignal.SetNull(colors.Critical.ValueString())
	authsignalColors.Information = authsignal.SetNull(colors.Information.ValueString())
	authsignalColors.Hover = authsignal.SetNull(colors.Hover.ValueString())
	authsignalColors.Focus = authsignal.SetNull(colors.Focus.ValueString())

	return authsignalColors
}

func buildAuthsignalDisplayDeleteObject(display displayModel) authsignal.Display {
	var authsignalDisplay authsignal.Display

	authsignalDisplay.FontUrl = authsignal.SetNull(display.FontUrl.ValueString())

	return authsignalDisplay
}

func buildAuthsignalTypographyDeleteObject() authsignal.Typography {
	var authsignalTypography authsignal.Typography
	var displayValues displayModel

	authsignalTypography.Display = authsignal.SetValue(buildAuthsignalDisplayDeleteObject(displayValues))

	return authsignalTypography
}

func buildAuthsignalContainerDeleteObject(container containerModel) authsignal.Container {
	var authsignalContainer authsignal.Container

	authsignalContainer.ContentAlignment = authsignal.SetNull(container.ContentAlignment.ValueString())
	authsignalContainer.Position = authsignal.SetNull(container.Position.ValueString())
	authsignalContainer.Padding = authsignal.SetNull(container.Padding.ValueInt64())
	authsignalContainer.LogoAlignment = authsignal.SetNull(container.LogoAlignment.ValueString())
	authsignalContainer.LogoPosition = authsignal.SetNull(container.LogoPosition.ValueString())
	authsignalContainer.LogoHeight = authsignal.SetNull(container.LogoHeight.ValueInt64())

	return authsignalContainer
}

func buildAuthsignalBordersDeleteObject(borders bordersModel) authsignal.Borders {
	var authsignalBorders authsignal.Borders

	authsignalBorders.ButtonBorderRadius = authsignal.SetNull(borders.ButtonBorderRadius.ValueInt64())
	authsignalBorders.ButtonBorderWidth = authsignal.SetNull(borders.ButtonBorderWidth.ValueInt64())
	authsignalBorders.CardBorderRadius = authsignal.SetNull(borders.CardBorderRadius.ValueInt64())
	authsignalBorders.CardBorderWidth = authsignal.SetNull(borders.CardBorderWidth.ValueInt64())
	authsignalBorders.InputBorderRadius = authsignal.SetNull(borders.InputBorderRadius.ValueInt64())
	authsignalBorders.InputBorderWidth = authsignal.SetNull(borders.InputBorderWidth.ValueInt64())
	authsignalBorders.ContainerBorderRadius = authsignal.SetNull(borders.ContainerBorderRadius.ValueInt64())

	return authsignalBorders
}

func buildAuthsignalPageBackgroundDeleteObject(pageBackground pageBackgroundModel) authsignal.PageBackground {
	var authsignalPageBackground authsignal.PageBackground

	authsignalPageBackground.BackgroundColor = authsignal.SetNull(pageBackground.BackgroundColor.ValueString())
	authsignalPageBackground.BackgroundImageUrl = authsignal.SetNull(pageBackground.BackgroundImageUrl.ValueString())

	return authsignalPageBackground
}

func buildAuthsignalThemeDeleteObject(input themeModel) authsignal.Theme {
	var darkModeValues darkModeModel
	var darkModeColorsValues colorsModel
	var darkModeBordersValues bordersModel
	var darkModeContainerValues containerModel
	var darkModePageBackgroundValues pageBackgroundModel

	var colorsValues colorsModel
	var bordersValues bordersModel
	var containerValues containerModel
	var pageBackgroundValues pageBackgroundModel

	var authsignalDarkMode authsignal.DarkMode
	authsignalDarkMode.Colors = authsignal.SetValue(buildAuthsignalColorsDeleteObject(darkModeColorsValues))
	authsignalDarkMode.Borders = authsignal.SetValue(buildAuthsignalBordersDeleteObject(darkModeBordersValues))
	authsignalDarkMode.Container = authsignal.SetValue(buildAuthsignalContainerDeleteObject(darkModeContainerValues))
	authsignalDarkMode.PageBackground = authsignal.SetValue(buildAuthsignalPageBackgroundDeleteObject(darkModePageBackgroundValues))
	authsignalDarkMode.Typography = authsignal.SetValue(buildAuthsignalTypographyDeleteObject())

	authsignalDarkMode.LogoUrl = authsignal.SetNull(darkModeValues.LogoUrl.ValueString())
	authsignalDarkMode.WatermarkUrl = authsignal.SetNull(darkModeValues.WatermarkUrl.ValueString())
	authsignalDarkMode.FaviconUrl = authsignal.SetNull(darkModeValues.FaviconUrl.ValueString())
	authsignalDarkMode.PrimaryColor = authsignal.SetNull(darkModeValues.PrimaryColor.ValueString())

	var authsignalTheme authsignal.Theme

	authsignalTheme.Colors = authsignal.SetValue(buildAuthsignalColorsDeleteObject(colorsValues))
	authsignalTheme.Borders = authsignal.SetValue(buildAuthsignalBordersDeleteObject(bordersValues))
	authsignalTheme.Container = authsignal.SetValue(buildAuthsignalContainerDeleteObject(containerValues))
	authsignalTheme.PageBackground = authsignal.SetValue(buildAuthsignalPageBackgroundDeleteObject(pageBackgroundValues))
	authsignalTheme.Typography = authsignal.SetValue(buildAuthsignalTypographyDeleteObject())
	authsignalTheme.DarkMode = authsignal.SetValue(authsignalDarkMode)

	authsignalTheme.LogoUrl = authsignal.SetNull(input.LogoUrl.ValueString())
	authsignalTheme.WatermarkUrl = authsignal.SetNull(input.WatermarkUrl.ValueString())
	authsignalTheme.FaviconUrl = authsignal.SetNull(input.FaviconUrl.ValueString())
	authsignalTheme.PrimaryColor = authsignal.SetNull(input.PrimaryColor.ValueString())

	return authsignalTheme
}
