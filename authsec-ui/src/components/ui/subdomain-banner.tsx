/**
 * SubdomainBanner Component
 *
 * Displays a minimalist, professional banner to guide users to the correct auth page
 */

import React from "react";
import { IconArrowRight, IconX } from "@tabler/icons-react";

interface SubdomainBannerProps {
  type: 'info' | 'warning';
  message: string;
  actionLabel: string;
  onAction: () => void;
  onDismiss?: () => void;
}

export const SubdomainBanner: React.FC<SubdomainBannerProps> = ({
  type,
  message,
  actionLabel,
  onAction,
  onDismiss
}) => {
  const styles = type === 'info'
    ? {
        bg: 'bg-gradient-to-r from-blue-50 to-sky-50 dark:from-slate-900 dark:to-slate-800',
        text: 'text-slate-700 dark:text-slate-300',
        button: 'text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300',
        buttonBg: 'hover:bg-blue-100/50 dark:hover:bg-blue-950/50'
      }
    : {
        bg: 'bg-gradient-to-r from-amber-50 to-orange-50 dark:from-slate-900 dark:to-slate-800',
        text: 'text-slate-700 dark:text-slate-300',
        button: 'text-amber-600 hover:text-amber-700 dark:text-amber-400 dark:hover:text-amber-300',
        buttonBg: 'hover:bg-amber-100/50 dark:hover:bg-amber-950/50'
      };

  return (
    <div className={`${styles.bg} border-b border-border/50 backdrop-blur-sm`}>
      <div className="max-w-7xl mx-auto px-6 py-3">
        <div className="flex items-center justify-between gap-6">
          {/* Message */}
          <p className={`text-sm ${styles.text} flex-1`}>
            {message}
          </p>

          {/* Actions */}
          <div className="flex items-center gap-2">
            <button
              onClick={onAction}
              className={`
                inline-flex items-center gap-1.5 px-3.5 py-1.5
                text-sm font-medium rounded-md
                ${styles.button} ${styles.buttonBg}
                transition-all duration-200 ease-in-out
                whitespace-nowrap
              `}
            >
              {actionLabel}
              <IconArrowRight className="h-3.5 w-3.5" strokeWidth={2.5} />
            </button>

            {onDismiss && (
              <button
                onClick={onDismiss}
                className="
                  text-slate-400 hover:text-slate-600 dark:hover:text-slate-300
                  hover:bg-slate-200/50 dark:hover:bg-slate-700/50
                  rounded-md p-1 transition-all duration-200
                "
                aria-label="Dismiss"
              >
                <IconX className="h-4 w-4" strokeWidth={2} />
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
