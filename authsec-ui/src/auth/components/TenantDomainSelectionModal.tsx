import React, { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../../components/ui/dialog';
import { Button } from '../../components/ui/button';
import { Input } from '../../components/ui/input';
import { Label } from '../../components/ui/label';
import { IconCheck, IconX, IconAlertTriangle, IconLoader } from '@tabler/icons-react';
import { useLazyCheckTenantDomainQuery, useCompleteUFlowOIDCRegistrationMutation } from '../../app/api/oidcApi';
import { AuthStepHeader } from './AuthStepHeader';

interface TenantDomainSelectionModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  userData: {
    email: string;
    name: string;
    picture: string;
    provider: string;
    provider_user_id: string;
  };
  onSuccess: (data: {
    tenant_id: string;
    client_id: string;
    tenant_domain: string;
  }) => void;
}

export const TenantDomainSelectionModal: React.FC<TenantDomainSelectionModalProps> = ({
  open,
  onOpenChange,
  userData,
  onSuccess,
}) => {
  const [domain, setDomain] = useState('');
  const [checking, setChecking] = useState(false);
  const [domainStatus, setDomainStatus] = useState<'idle' | 'available' | 'taken' | 'error'>('idle');
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [checkDebounceTimer, setCheckDebounceTimer] = useState<NodeJS.Timeout | null>(null);

  const [checkTenantDomain] = useLazyCheckTenantDomainQuery();
  const [completeRegistration, { isLoading: isSubmitting }] = useCompleteUFlowOIDCRegistrationMutation();

  // Debounced domain check
  useEffect(() => {
    if (checkDebounceTimer) {
      clearTimeout(checkDebounceTimer);
    }

    if (!domain || domain.length < 2) {
      setDomainStatus('idle');
      setErrorMessage(null);
      return;
    }

    // Basic validation
    const isValid = /^[a-z0-9-]+$/i.test(domain);
    if (!isValid) {
      setDomainStatus('error');
      setErrorMessage('Domain can only contain letters, numbers, and hyphens');
      return;
    }

    setChecking(true);
    const timer = setTimeout(async () => {
      try {
        const result = await checkTenantDomain(domain).unwrap();
        if (result.exists) {
          setDomainStatus('taken');
          setErrorMessage('This domain is already taken');
        } else {
          setDomainStatus('available');
          setErrorMessage(null);
        }
      } catch (error) {
        console.error('Domain check error:', error);
        setDomainStatus('error');
        setErrorMessage('Failed to check domain availability');
      } finally {
        setChecking(false);
      }
    }, 500); // 500ms debounce

    setCheckDebounceTimer(timer);

    return () => {
      if (timer) clearTimeout(timer);
    };
  }, [domain, checkTenantDomain]);

  const handleSubmit = async () => {
    if (domainStatus !== 'available' || !domain) {
      return;
    }

    try {
      const result = await completeRegistration({
        tenant_domain: domain,
        provider: userData.provider,
        email: userData.email,
        name: userData.name,
        picture: userData.picture,
        provider_user_id: userData.provider_user_id,
      }).unwrap();

      if (result.success) {
        onSuccess({
          tenant_id: result.tenant_id,
          client_id: result.client_id,
          tenant_domain: result.tenant_domain,
        });
        onOpenChange(false);
      }
    } catch (error: any) {
      console.error('Registration completion error:', error);
      setErrorMessage(error?.data?.message || 'Failed to complete registration. Please try again.');
      setDomainStatus('error');
    }
  };

  const getDomainStatusIcon = () => {
    if (checking) {
      return (
        <motion.div
          animate={{ rotate: 360 }}
          transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
        >
          <IconLoader className="w-4 h-4 text-slate-500" />
        </motion.div>
      );
    }

    switch (domainStatus) {
      case 'available':
        return <IconCheck className="w-4 h-4 text-green-600" />;
      case 'taken':
        return <IconX className="w-4 h-4 text-red-600" />;
      case 'error':
        return <IconAlertTriangle className="w-4 h-4 text-red-600" />;
      default:
        return null;
    }
  };

  const getDomainStatusColor = () => {
    switch (domainStatus) {
      case 'available':
        return 'border-green-500';
      case 'taken':
      case 'error':
        return 'border-red-500';
      default:
        return '';
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg border-slate-200 bg-white p-0" showCloseButton={false}>
        <DialogHeader className="sr-only">
          <DialogTitle>Choose Your Workspace Domain</DialogTitle>
          <DialogDescription>
            This will be your unique workspace identifier.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-5 p-6">
          <AuthStepHeader
            title="Choose your workspace domain"
            subtitle="Pick a unique tenant identifier. This value is used for workspace routing."
          />

          <div className="auth-inline-note flex items-center gap-3">
            {userData.picture ? (
              <img
                src={userData.picture}
                alt={userData.name}
                className="h-10 w-10 rounded-full"
              />
            ) : (
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-slate-200 text-xs font-semibold text-slate-700">
                {userData.name.slice(0, 1).toUpperCase()}
              </div>
            )}
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-medium text-slate-900">{userData.name}</p>
              <p className="truncate text-xs text-slate-500">{userData.email}</p>
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="domain" className="text-sm font-medium text-slate-700">
              Workspace Domain
            </Label>
            <div className="relative">
              <Input
                id="domain"
                type="text"
                value={domain}
                onChange={(e) => setDomain(e.target.value.toLowerCase())}
                placeholder="my-workspace"
                className={`pr-10 ${getDomainStatusColor()}`}
                disabled={isSubmitting}
                autoFocus
              />
              <div className="absolute right-3 top-1/2 -translate-y-1/2">
                {getDomainStatusIcon()}
              </div>
            </div>
            <p className="text-xs text-slate-500">
              Workspace URL: <span className="font-mono">{domain || 'your-domain'}.app.authsec.dev</span>
            </p>
          </div>

          {errorMessage && (
            <motion.div
              initial={{ opacity: 0, y: -8 }}
              animate={{ opacity: 1, y: 0 }}
              className="auth-inline-note border-red-200 bg-red-50"
            >
              <div className="flex items-center gap-2">
                <IconAlertTriangle className="h-4 w-4 text-red-600" />
                <p className="text-sm text-red-700">{errorMessage}</p>
              </div>
            </motion.div>
          )}

          {domainStatus === 'available' && !errorMessage && (
            <motion.div
              initial={{ opacity: 0, y: -8 }}
              animate={{ opacity: 1, y: 0 }}
              className="auth-inline-note border-green-200 bg-green-50"
            >
              <div className="flex items-center gap-2">
                <IconCheck className="h-4 w-4 text-green-600" />
                <p className="text-sm text-green-700">Great, this domain is available.</p>
              </div>
            </motion.div>
          )}
        </div>

        <DialogFooter className="border-t border-slate-200 px-6 py-4">
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isSubmitting}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={domainStatus !== 'available' || !domain || isSubmitting}
          >
            {isSubmitting ? (
              <>
                <motion.div
                  animate={{ rotate: 360 }}
                  transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
                  className="mr-2 h-4 w-4 rounded-full border-2 border-white border-t-transparent"
                />
                Creating Workspace...
              </>
            ) : (
              'Create Workspace'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
