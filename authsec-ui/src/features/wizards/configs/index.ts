import type { WizardRegistry } from "../types";
import { m2mWorkloadWizard } from "./m2mWorkloadWizard";
import { userAuthWizard } from "./userAuthWizard";
import { rbacWizard } from "./rbacWizard";
import { scopesWizard } from "./scopesWizard";

/**
 * Wizard Registry
 * Maps wizard IDs to their configurations
 */
export const wizardConfigs: WizardRegistry = {
  "m2m-workload-wizard": m2mWorkloadWizard,
  "user-auth-wizard": userAuthWizard,
  "rbac-wizard": rbacWizard,
  "scopes-wizard": scopesWizard,
  // Add more wizards here in the future:
  // 'certificate-wizard': certificateWizard,
};

/**
 * Get wizard configuration by ID
 */
export function getWizardConfig(wizardId: string) {
  return wizardConfigs[wizardId] || null;
}
