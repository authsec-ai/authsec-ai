import type { ReactNode } from "react";
import { PageHeader } from "@/components/layout/PageHeader";

interface TrustDelegationPageFrameProps {
  title: string;
  description: string;
  actions?: ReactNode;
  children: ReactNode;
}

export function TrustDelegationPageFrame({
  title,
  description,
  actions,
  children,
}: TrustDelegationPageFrameProps) {
  return (
    <div className="min-h-screen">
      <div className="space-y-4 p-6 max-w-10xl mx-auto">
        <PageHeader title={title} description={description} actions={actions} />
        {children}
      </div>
    </div>
  );
}
