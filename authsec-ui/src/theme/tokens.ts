export type SpacingToken = keyof typeof spacingValues;
export type RadiusToken = keyof typeof radiusValues;
export type ShadowToken = keyof typeof shadowValues;

const spacingValues = {
  none: "var(--space-0)",
  xs: "var(--space-1)",
  sm: "var(--space-2)",
  md: "var(--space-3)",
  lg: "var(--space-4)",
  xl: "var(--space-6)",
  "2xl": "var(--space-8)",
  "3xl": "var(--space-9)",
} as const;

const radiusValues = {
  xs: "var(--radius-xs)",
  sm: "var(--radius-sm)",
  md: "var(--radius-md)",
  lg: "var(--radius-lg)",
  xl: "var(--radius-xl)",
  "2xl": "var(--radius-2xl)",
  pill: "var(--radius-pill)",
} as const;

const shadowValues = {
  none: "var(--shadow-none)",
  xs: "var(--shadow-xs)",
  sm: "var(--shadow-sm)",
  md: "var(--shadow-md)",
} as const;

const colorValues = {
  background: "var(--color-surface-base)",
  surfaceRaised: "var(--color-surface-raised)",
  surfaceSubtle: "var(--color-surface-subtle)",
  surfaceInverse: "var(--color-surface-inverse)",
  borderSubtle: "var(--color-border-subtle)",
  borderStrong: "var(--color-border-strong)",
  primary: "var(--color-primary)",
  primaryStrong: "var(--color-primary-strong)",
  primarySoft: "var(--color-primary-soft)",
  success: "var(--color-success)",
  warning: "var(--color-warning)",
  danger: "var(--color-danger)",
  muted: "var(--color-muted)",
  accent: "var(--color-accent)",
  textPrimary: "var(--color-text-primary)",
  textSecondary: "var(--color-text-secondary)",
  textOnDark: "var(--color-text-on-dark)",
} as const;

const typographyTokens = {
  family: {
    sans: "var(--font-family-sans)",
    mono: "var(--font-family-mono)",
  },
  size: {
    display: "var(--font-size-display)",
    displayLg: "var(--font-size-display-lg)",
    headingXl: "var(--font-size-heading-xl)",
    headingLg: "var(--font-size-heading-lg)",
    headingMd: "var(--font-size-heading-md)",
    headingSm: "var(--font-size-heading-sm)",
    headingXs: "var(--font-size-heading-xs)",
    bodyLg: "var(--font-size-body-lg)",
    bodyMd: "var(--font-size-body-md)",
    bodySm: "var(--font-size-body-sm)",
    bodyXs: "var(--font-size-body-xs)",
  },
  weight: {
    regular: "var(--font-weight-regular)",
    medium: "var(--font-weight-medium)",
    semibold: "var(--font-weight-semibold)",
    bold: "var(--font-weight-bold)",
  },
  lineHeight: {
    tight: "var(--line-height-tight)",
    normal: "var(--line-height-normal)",
    relaxed: "var(--line-height-relaxed)",
    heading: "var(--line-height-heading)",
    body: "var(--line-height-body)",
  },
  letterSpacing: {
    tight: "var(--letter-spacing-tight)",
    normal: "var(--letter-spacing-normal)",
    wide: "var(--letter-spacing-wide)",
  },
} as const;

const componentTokens = {
  card: {
    background: "var(--component-card-background)",
    foreground: "var(--component-card-foreground)",
    border: "var(--component-card-border)",
    shadow: "var(--component-card-shadow)",
    radius: "var(--component-card-radius)",
    gap: "var(--component-card-gap)",
    paddingBlock: "var(--component-card-padding-block)",
    paddingInline: "var(--component-card-padding-inline)",
    headerPaddingBlock: "var(--component-card-header-padding-block)",
    headerPaddingInline: "var(--component-card-header-padding-inline)",
    footerPaddingBlock: "var(--component-card-footer-padding-block)",
    footerPaddingInline: "var(--component-card-footer-padding-inline)",
    variants: {
      header: {
        background: "var(--component-card-variant-header-background)",
        border: "var(--component-card-variant-header-border)",
        shadow: "var(--component-card-variant-header-shadow)",
      },
      filter: {
        background: "var(--component-card-variant-filter-background)",
        border: "var(--component-card-variant-filter-border)",
        shadow: "var(--component-card-variant-filter-shadow)",
      },
      table: {
        background: "var(--component-card-variant-table-background)",
        border: "var(--component-card-variant-table-border)",
        shadow: "var(--component-card-variant-table-shadow)",
      },
      pagination: {
        background: "var(--component-card-variant-pagination-background)",
        border: "var(--component-card-variant-pagination-border)",
        shadow: "var(--component-card-variant-pagination-shadow)",
      },
    },
  },
  button: {
    radius: "var(--component-button-radius)",
    gap: "var(--component-button-gap)",
    fontWeight: "var(--component-button-font-weight)",
    height: {
      sm: "var(--component-button-height-sm)",
      md: "var(--component-button-height-md)",
      lg: "var(--component-button-height-lg)",
    },
    paddingInline: {
      sm: "var(--component-button-padding-inline-sm)",
      md: "var(--component-button-padding-inline-md)",
      lg: "var(--component-button-padding-inline-lg)",
    },
    paddingBlock: "var(--component-button-padding-block)",
    variants: {
      primary: {
        bg: "var(--component-button-primary-bg)",
        fg: "var(--component-button-primary-fg)",
        hoverBg: "var(--component-button-primary-hover-bg)",
      },
      secondary: {
        bg: "var(--component-button-secondary-bg)",
        fg: "var(--component-button-secondary-fg)",
        hoverBg: "var(--component-button-secondary-hover-bg)",
      },
      outline: {
        border: "var(--component-button-outline-border)",
      },
      destructive: {
        bg: "var(--component-button-destructive-bg)",
        hoverBg: "var(--component-button-destructive-hover-bg)",
      },
      cta: {
        bg: "var(--component-button-cta-bg)",
        fg: "var(--component-button-cta-fg)",
        hoverBg: "var(--component-button-cta-hover-bg)",
        shadow: "var(--component-button-cta-shadow)",
      },
      ctaSecondary: {
        bg: "var(--component-button-cta-secondary-bg)",
        fg: "var(--component-button-cta-secondary-fg)",
        hoverBg: "var(--component-button-cta-secondary-hover-bg)",
      },
      bulk: {
        bg: "var(--component-button-bulk-bg)",
        fg: "var(--component-button-bulk-fg)",
        hoverBg: "var(--component-button-bulk-hover-bg)",
      },
    },
  },
  badge: {
    radius: "var(--component-badge-radius)",
    paddingInline: "var(--component-badge-padding-inline)",
    paddingBlock: "var(--component-badge-padding-block)",
    fontSize: "var(--component-badge-font-size)",
  },
  table: {
    surface: "var(--component-table-surface)",
    border: "var(--component-table-border)",
    headerBackground: "var(--component-table-header-bg)",
    headerForeground: "var(--component-table-header-fg)",
    rowHover: "var(--component-table-row-hover)",
    rowSelected: "var(--component-table-row-selected)",
    cellPaddingInline: "var(--component-table-cell-padding-inline)",
    cellPaddingBlock: "var(--component-table-cell-padding-block)",
    radius: "var(--component-table-radius)",
    headerHeight: "var(--component-table-header-height)",
    rowHeight: "var(--component-table-row-height)",
    shadow: "var(--component-table-shadow)",
  },
  input: {
    height: "var(--component-input-height)",
    radius: "var(--component-input-radius)",
    border: "var(--component-input-border)",
    background: "var(--component-input-background)",
    foreground: "var(--component-input-foreground)",
    placeholder: "var(--component-input-placeholder)",
    shadow: "var(--component-input-shadow)",
    focusRing: "var(--component-input-focus-ring)",
    disabledBackground: "var(--component-input-disabled-background)",
    disabledForeground: "var(--component-input-disabled-foreground)",
  },
  form: {
    maxWidth: "var(--component-form-max-width)",
    shellGap: "var(--component-form-shell-gap)",
    sectionGap: "var(--component-form-section-gap)",
    sectionPaddingInline: "var(--component-form-section-padding-inline)",
    sectionPaddingBlock: "var(--component-form-section-padding-block)",
    sectionBackground: "var(--component-form-section-background)",
    sectionBorder: "var(--component-form-section-border)",
    sectionShadow: "var(--component-form-section-shadow)",
    labelColor: "var(--component-form-label-color)",
    labelWeight: "var(--component-form-label-weight)",
    helperColor: "var(--component-form-helper-color)",
    fieldGap: "var(--component-form-field-gap)",
    fieldBorder: "var(--component-form-field-border)",
    actionsBackground: "var(--component-form-actions-background)",
    callout: {
      background: "var(--component-form-callout-background)",
      border: "var(--component-form-callout-border)",
      iconBackground: "var(--component-form-callout-icon-bg)",
      iconColor: "var(--component-form-callout-icon-color)",
    },
  },
} as const;

export const themeTokens = {
  spacing: spacingValues,
  radius: radiusValues,
  shadow: shadowValues,
  color: colorValues,
  typography: typographyTokens,
  component: componentTokens,
};

export type ThemeTokens = typeof themeTokens;

export const spacing = (token: SpacingToken): string => spacingValues[token];
export const radius = (token: RadiusToken): string => radiusValues[token];
export const shadow = (token: ShadowToken): string => shadowValues[token];
