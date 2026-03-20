export interface CodeStep {
  label: string;
  code: string;
}

export interface SDKCodeExample {
  python: CodeStep[];
  typescript: CodeStep[];
  go?: CodeStep[];
  java?: CodeStep[];
}

export interface SDKHelpItem {
  id: string;
  question: string;
  description: string;
  code: SDKCodeExample;
  docsLink?: string;
  hubModule?: string;
}

export interface SDKEntitySnippet {
  // The entity type (permission, resource, role, scope, auth-method)
  entityType: 'permission' | 'resource' | 'role' | 'scope' | 'role-binding' | 'auth-method';
  // Dynamic data to populate the SDK code
  entityData: Record<string, any>;
}

export type SDKLanguage = 'python' | 'typescript' | 'go' | 'java';
