import React from "react";
import { AlertTriangle, Shield, Info, Home } from "lucide-react";
import { Card, CardContent } from "./card";
import { Button } from "./button";
import { useNavigate } from "react-router-dom";

interface ErrorDisplayProps {
  title?: string;
  message: string;
  type?: 'error' | 'warning' | 'auth';
  showHomeButton?: boolean;
  showAuthButton?: boolean;
  helpText?: string;
  className?: string;
}

/**
 * Reusable error display component with beautiful UI
 * Supports different error types and actions
 */
export function ErrorDisplay({
  title,
  message,
  type = 'error',
  showHomeButton = false,
  showAuthButton = false,
  helpText,
  className = ""
}: ErrorDisplayProps) {
  const navigate = useNavigate();
  
  const getTypeConfig = () => {
    switch (type) {
      case 'auth':
        return {
          title: title || 'Authentication Required',
          icon: Shield,
          colors: {
            bg: 'bg-amber-50/50 dark:bg-amber-950/20',
            border: 'border-amber-200 dark:border-amber-800',
            iconBg: 'bg-amber-100 dark:bg-amber-900/50',
            iconColor: 'text-amber-600 dark:text-amber-400',
            titleColor: 'text-amber-900 dark:text-amber-100',
            messageColor: 'text-amber-800 dark:text-amber-200',
            buttonColor: 'border-amber-300 dark:border-amber-700 text-amber-700 dark:text-amber-300 hover:bg-amber-50 dark:hover:bg-amber-950/50'
          }
        };
      case 'warning':
        return {
          title: title || 'Warning',
          icon: AlertTriangle,
          colors: {
            bg: 'bg-yellow-50/50 dark:bg-yellow-950/20',
            border: 'border-yellow-200 dark:border-yellow-800',
            iconBg: 'bg-yellow-100 dark:bg-yellow-900/50',
            iconColor: 'text-yellow-600 dark:text-yellow-400',
            titleColor: 'text-yellow-900 dark:text-yellow-100',
            messageColor: 'text-yellow-800 dark:text-yellow-200',
            buttonColor: 'border-yellow-300 dark:border-yellow-700 text-yellow-700 dark:text-yellow-300 hover:bg-yellow-50 dark:hover:bg-yellow-950/50'
          }
        };
      default:
        return {
          title: title || 'Error',
          icon: AlertTriangle,
          colors: {
            bg: 'bg-red-50/50 dark:bg-red-950/20',
            border: 'border-red-200 dark:border-red-800',
            iconBg: 'bg-red-100 dark:bg-red-900/50',
            iconColor: 'text-red-600 dark:text-red-400',
            titleColor: 'text-red-900 dark:text-red-100',
            messageColor: 'text-red-800 dark:text-red-200',
            buttonColor: 'border-red-300 dark:border-red-700 text-red-700 dark:text-red-300 hover:bg-red-50 dark:hover:bg-red-950/50'
          }
        };
    }
  };

  const config = getTypeConfig();
  const IconComponent = config.icon;

  const isAuthError = message.includes('user not found') || message.includes('authentication') || type === 'auth';

  return (
    <Card className={`${config.colors.bg} ${config.colors.border} backdrop-blur-sm ${className}`}>
      <CardContent className="p-6">
        <div className="space-y-4">
          {/* Error Message */}
          <div className="flex items-start gap-4 text-left">
            <div className={`p-2 ${config.colors.iconBg} rounded-lg flex-shrink-0`}>
              <IconComponent className={`h-5 w-5 ${config.colors.iconColor}`} />
            </div>
            <div className="space-y-2 flex-1">
              <h3 className={`font-semibold ${config.colors.titleColor}`}>
                {config.title}
              </h3>
              <p className={`${config.colors.messageColor} leading-relaxed text-sm`}>
                {message}
              </p>
              {helpText && (
                <div className={`flex items-center gap-2 text-xs ${config.colors.messageColor} mt-3`}>
                  <div className={`w-1 h-1 ${config.colors.iconColor.replace('text-', 'bg-')} rounded-full`}></div>
                  {helpText}
                </div>
              )}
            </div>
          </div>

          {/* Action Buttons */}
          <div className={`flex flex-col sm:flex-row gap-3 pt-4 border-t ${config.colors.border}`}>
          
            {(showAuthButton || isAuthError) && (
              <Button 
                variant="outline"
                onClick={() => navigate('/admin/login')}
                className={`flex-1 ${config.colors.buttonColor}`}
              >
                <Shield className="h-4 w-4 mr-2" />
                Complete Login
              </Button>
            )}

            {showHomeButton && (
              <Button 
                variant="outline"
                onClick={() => navigate('/')}
                className={`flex-1 ${config.colors.buttonColor}`}
              >
                <Home className="h-4 w-4 mr-2" />
                Go Home
              </Button>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

/**
 * Full-page error display for critical errors
 */
export function FullPageErrorDisplay(props: ErrorDisplayProps & { 
  pageTitle?: string;
  pageSubtitle?: string;
  showHelpSection?: boolean;
}) {
  const {
    pageTitle = "Something went wrong",
    pageSubtitle = "We encountered an issue while loading the page",
    showHelpSection = false,
    ...errorProps
  } = props;

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-slate-100/50 dark:from-neutral-950 dark:via-neutral-900 dark:to-stone-950">
      <div className="flex flex-col items-center justify-center min-h-screen p-6">
        <div className="w-full max-w-2xl mx-auto text-center space-y-8">
          {/* Icon and Title */}
          <div className="space-y-4">
            <div className="relative mx-auto w-24 h-24">
              <div className={`absolute inset-0 ${errorProps.type === 'auth' ? 'bg-amber-100' : 'bg-red-100'} dark:${errorProps.type === 'auth' ? 'bg-amber-950/30' : 'bg-red-950/30'} rounded-full blur-lg`}></div>
              <div className="relative p-6 bg-white dark:bg-neutral-800 rounded-full shadow-lg ring-1 ring-slate-200/50 dark:ring-neutral-700/50">
                <AlertTriangle className={`h-12 w-12 ${errorProps.type === 'auth' ? 'text-amber-600 dark:text-amber-400' : 'text-red-600 dark:text-red-400'}`} />
              </div>
            </div>
            <div>
              <h1 className="text-4xl font-bold text-slate-900 dark:text-neutral-100 mb-2">
                {pageTitle}
              </h1>
              <p className="text-lg text-slate-600 dark:text-neutral-400">
                {pageSubtitle}
              </p>
            </div>
          </div>

          {/* Error Details */}
          <ErrorDisplay {...errorProps} className="text-left" />

          {/* Help Section for Auth Errors */}
          {showHelpSection && errorProps.message.includes('user not found') && (
            <Card className="border-blue-200 dark:border-blue-800 bg-blue-50/50 dark:bg-blue-950/20 backdrop-blur-sm">
              <CardContent className="p-6">
                <div className="flex items-start gap-3">
                  <div className="p-2 bg-blue-100 dark:bg-blue-900/50 rounded-lg flex-shrink-0">
                    <Info className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                  </div>
                  <div className="text-left">
                    <h4 className="font-medium text-blue-900 dark:text-blue-100 mb-2">
                      What's happening?
                    </h4>
                    <p className="text-sm text-blue-800 dark:text-blue-200 leading-relaxed mb-3">
                      Your user account needs to be created in the AuthSec system first. This happens automatically when you complete the OIDC authentication flow.
                    </p>
                    <div className="flex items-center gap-2 text-xs text-blue-700 dark:text-blue-300">
                      <div className="w-1 h-1 bg-blue-600 dark:bg-blue-400 rounded-full"></div>
                      Complete the login process to access your account
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
}
