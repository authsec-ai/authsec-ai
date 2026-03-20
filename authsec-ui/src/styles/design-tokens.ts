// Design tokens extracted from ClientsPage for consistent styling across components

export const GRADIENTS = {
  // Main background gradient used in ClientsPage
  background: "bg-gradient-to-br from-slate-50 via-white to-slate-100/50 dark:from-neutral-950 dark:via-neutral-900 dark:to-stone-950",
} as const;

export const GLASSMORPHISM = {
  // Exact card style from ClientsPage
  card: "border-0 bg-white/80 dark:bg-neutral-900/80 backdrop-blur-sm ring-1 ring-slate-200/50 dark:ring-neutral-800/50",
  // Same card with transition for interactive elements
  cardWithTransition: "border-0 bg-white/80 dark:bg-neutral-900/80 backdrop-blur-sm ring-1 ring-slate-200/50 dark:ring-neutral-800/50 transition-all duration-500",
} as const;

export const TYPOGRAPHY = {
  // Header styles matching ClientsPage
  h1: "text-3xl font-bold tracking-tight text-foreground",
  h2: "text-xl font-semibold text-foreground",
  h3: "text-lg font-semibold text-foreground",
  // Body text styles
  body: "text-base text-foreground",
  bodyLarge: "text-lg text-foreground",
  muted: "text-muted-foreground text-base leading-relaxed",
  mutedSmall: "text-muted-foreground text-sm",
  // Special text styles
  description: "text-slate-600 dark:text-neutral-400 text-lg",
} as const;

export const SPACING = {
  // Container and layout spacing
  container: "max-w-7xl mx-auto",
  containerOnboard: "max-w-8xl w-full", // Wider for onboard page
  sectionGap: "space-y-8",
  cardGap: "space-y-6",
  elementGap: "space-y-4",
  // Padding patterns
  cardPadding: "p-8",
  cardHeaderPadding: "p-6",
  pagePadding: "p-6",
} as const;

export const ANIMATIONS = {
  // Framer Motion variants for consistent animations
  fadeInUp: {
    initial: { opacity: 0, y: -20 },
    animate: { opacity: 1, y: 0 },
    transition: { duration: 0.6, ease: "easeOut" }
  },
  fadeIn: {
    initial: { opacity: 0 },
    animate: { opacity: 1 },
    transition: { duration: 0.4 }
  },
  slideIn: {
    initial: { opacity: 0, x: -20 },
    animate: { opacity: 1, x: 0 },
    transition: { duration: 0.5 }
  },
  staggerChildren: {
    animate: {
      transition: {
        staggerChildren: 0.1
      }
    }
  }
} as const;

export const COLORS = {
  // Status colors used throughout ClientsPage
  status: {
    active: "text-green-600 bg-green-50 dark:text-green-400 dark:bg-green-950/20",
    restricted: "text-yellow-600 bg-yellow-50 dark:text-yellow-400 dark:bg-yellow-950/20",
    disabled: "text-red-600 bg-red-50 dark:text-red-400 dark:bg-red-950/20",
  },
  // Type colors for different client types
  type: {
    mcpServer: "text-blue-600 bg-blue-50 dark:text-blue-400 dark:bg-blue-950/20",
    aiAgent: "text-green-600 bg-green-50 dark:text-green-400 dark:bg-green-950/20",
  },
  // Accent colors
  primary: "text-primary bg-primary/10",
  secondary: "text-muted-foreground bg-muted",
} as const;

export const BORDERS = {
  // Border styles matching ClientsPage
  subtle: "border border-border/60",
  prominent: "border-2 border-primary/20",
  none: "border-0",
  ring: "ring-1 ring-slate-200/50 dark:ring-neutral-800/50",
} as const;

export const SHADOWS = {
  // Shadow styles for elevated elements
  soft: "shadow-lg",
  medium: "shadow-xl",
  strong: "shadow-2xl",
} as const;