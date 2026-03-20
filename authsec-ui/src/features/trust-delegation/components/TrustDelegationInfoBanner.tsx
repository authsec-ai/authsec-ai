import { Bot, Clock3, ShieldCheck } from "lucide-react";
import { PageInfoBanner } from "@/components/shared/PageInfoBanner";

interface TrustDelegationInfoBannerProps {
  variant?: "trust-delegation" | "active" | "policies";
}

const COPY = {
  title: "Define the trust delegation guardrails",
  description:
    "Configure the maximum role, target type, client scope, allowed actions, and duration that operators can issue through trust delegation.",
} as const;

export function TrustDelegationInfoBanner({
  variant: _variant = "trust-delegation",
}: TrustDelegationInfoBannerProps) {
  return (
    <PageInfoBanner
      title={COPY.title}
      description={COPY.description}
      features={[
        { text: "Keep delegated access constrained by configuration", icon: ShieldCheck },
        { text: "Use short validity windows for higher-risk actions", icon: Clock3 },
        { text: "Model agents and workloads as explicit trust targets", icon: Bot },
      ]}
      featuresTitle="Operational guardrails"
      storageKey="trust-delegation-banner"
      dismissible
    />
  );
}
