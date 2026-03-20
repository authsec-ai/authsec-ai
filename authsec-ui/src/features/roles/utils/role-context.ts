import type { RoleContext, RoleFormData } from "../types";

const CONTEXT_KEY = "authsec_role_context";

export const setRoleContext = (context: RoleContext) => {
  try {
    sessionStorage.setItem(CONTEXT_KEY, JSON.stringify(context));
  } catch (error) {
    console.warn("Failed to save role context:", error);
  }
};

export const getRoleContext = (): RoleContext | null => {
  try {
    const stored = sessionStorage.getItem(CONTEXT_KEY);
    return stored ? JSON.parse(stored) : null;
  } catch (error) {
    console.warn("Failed to retrieve role context:", error);
    return null;
  }
};

export const clearRoleContext = () => {
  try {
    sessionStorage.removeItem(CONTEXT_KEY);
  } catch (error) {
    console.warn("Failed to clear role context:", error);
  }
};

export const createInitialFormData = (context?: RoleContext): RoleFormData => {
  const baseData: RoleFormData = {
    roleId: "",
    displayName: "",
    description: "",
    grants: [],
    assignedUsers: [],
    assignedGroups: [],
  };

  if (!context) return baseData;

  // Handle prefilled resource
  if (context.prefillResource) {
    baseData.grants = [
      {
        resource: context.prefillResource,
        scopes: [],
      },
    ];
  }

  // Handle assign to group
  if (context.assignToGroup) {
    baseData.assignedGroups = [context.assignToGroup];
  }

  return baseData;
};

export const generateRoleId = (displayName: string): string => {
  if (!displayName.trim()) return "";

  return displayName
    .trim()
    .toUpperCase()
    .replace(/[^A-Z0-9\s]/g, "")
    .replace(/\s+/g, "_")
    .substring(0, 50);
};

export const validateRoleId = (roleId: string): { isValid: boolean; error?: string } => {
  if (!roleId.trim()) {
    return { isValid: false, error: "Role ID is required" };
  }

  if (!/^[A-Z0-9_]+$/.test(roleId)) {
    return {
      isValid: false,
      error: "Role ID must contain only uppercase letters, numbers, and underscores",
    };
  }

  if (roleId.length > 50) {
    return { isValid: false, error: "Role ID must be 50 characters or less" };
  }

  // TODO: Check for duplicates against existing roles
  // This would typically be an API call
  if (roleId === "ADMIN" || roleId === "SYSTEM") {
    return { isValid: false, error: "Role ID already in use" };
  }

  return { isValid: true };
};

export const validateFormData = (
  formData: RoleFormData
): { isValid: boolean; errors: string[] } => {
  const errors: string[] = [];

  // Validate Role ID
  const roleIdValidation = validateRoleId(formData.roleId);
  if (!roleIdValidation.isValid) {
    errors.push(roleIdValidation.error!);
  }

  // Validate Display Name
  if (!formData.displayName.trim()) {
    errors.push("Display name is required");
  } else if (formData.displayName.length > 50) {
    errors.push("Display name must be 50 characters or less");
  }

  // Validate Description
  if (formData.description && formData.description.length > 200) {
    errors.push("Description must be 200 characters or less");
  }

  // Validate Grants
  if (formData.grants.length === 0) {
    errors.push("At least one permission grant is required");
  }

  for (const grant of formData.grants) {
    if (!grant.resource.trim()) {
      errors.push("All grants must have a resource");
    }
    if (grant.scopes.length === 0) {
      errors.push("All grants must have at least one scope");
    }
  }

  return { isValid: errors.length === 0, errors };
};
