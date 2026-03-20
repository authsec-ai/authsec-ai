import React from "react";
import { Loader2, Server } from "lucide-react";
import { Card, CardContent } from "./card";

interface LoadingDisplayProps {
  message?: string;
  subMessage?: string;
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

/**
 * Reusable loading display component
 */
export function LoadingDisplay({
  message = "Loading...",
  subMessage,
  size = 'md',
  className = ""
}: LoadingDisplayProps) {
  const getSizeConfig = () => {
    switch (size) {
      case 'sm':
        return {
          spinner: 'h-6 w-6',
          text: 'text-base',
          subText: 'text-sm',
          padding: 'p-4'
        };
      case 'lg':
        return {
          spinner: 'h-12 w-12',
          text: 'text-2xl',
          subText: 'text-lg',
          padding: 'p-8'
        };
      default:
        return {
          spinner: 'h-8 w-8',
          text: 'text-xl',
          subText: 'text-base',
          padding: 'p-6'
        };
    }
  };

  const config = getSizeConfig();

  return (
    <Card className={`border-slate-200 dark:border-neutral-700 bg-white/80 dark:bg-neutral-800/80 backdrop-blur-sm ${className}`}>
      <CardContent className={config.padding}>
        <div className="flex flex-col items-center justify-center text-center space-y-4">
          <div className="relative">
            <Loader2 className={`${config.spinner} animate-spin text-slate-600 dark:text-neutral-400`} />
          </div>
          <div className="space-y-2">
            <p className={`font-medium text-slate-900 dark:text-neutral-100 ${config.text}`}>
              {message}
            </p>
            {subMessage && (
              <p className={`text-slate-600 dark:text-neutral-400 ${config.subText}`}>
                {subMessage}
              </p>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

/**
 * Full-page loading display for entire pages
 */
export function FullPageLoadingDisplay({
  title = "Loading",
  subtitle = "Please wait while we fetch your data",
  className = ""
}: {
  title?: string;
  subtitle?: string;
  className?: string;
}) {
  return (
    <div className={`min-h-screen bg-gradient-to-br from-slate-50 via-white to-slate-100/50 dark:from-neutral-950 dark:via-neutral-900 dark:to-stone-950 ${className}`}>
      <div className="flex flex-col items-center justify-center min-h-screen p-6">
        <div className="w-full max-w-md mx-auto text-center space-y-8">
          {/* Icon and Spinner */}
          <div className="space-y-4">
            <div className="relative mx-auto w-20 h-20">
              <div className="absolute inset-0 bg-slate-200/50 dark:bg-neutral-700/30 rounded-xl blur-sm"></div>
              <div className="relative p-4 bg-white dark:bg-neutral-800/80 rounded-xl shadow-sm ring-1 ring-slate-200/50 dark:ring-neutral-700/50">
                <Server className="h-6 w-6 text-slate-700 dark:text-neutral-300 mx-auto mb-2" />
                <Loader2 className="h-6 w-6 animate-spin text-slate-600 dark:text-neutral-400 mx-auto" />
              </div>
            </div>
            <div>
              <h1 className="text-2xl font-semibold text-slate-900 dark:text-neutral-100 mb-2">
                {title}
              </h1>
              <p className="text-slate-600 dark:text-neutral-400">
                {subtitle}
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

/**
 * Inline loading spinner for buttons and small areas
 */
export function InlineLoading({
  size = 16,
  className = ""
}: {
  size?: number;
  className?: string;
}) {
  return (
    <Loader2 
      className={`animate-spin ${className}`} 
      style={{ width: size, height: size }}
    />
  );
}