import type { ClientOption, User, Group } from "../types";

export const mockClients: ClientOption[] = [
  {
    id: "order-api",
    name: "Order API",
    resources: [
      {
        path: "orders/*",
        label: "All Orders",
        scopes: [
          { name: "read", description: "View and list orders" },
          { name: "write", description: "Create and update orders" },
          { name: "refund", description: "Process refunds", isDeprecated: true },
          { name: "cancel", description: "Cancel orders" },
        ],
      },
      {
        path: "orders/drafts/*",
        label: "Draft Orders",
        scopes: [
          { name: "read", description: "View draft orders" },
          { name: "write", description: "Create and update drafts" },
          { name: "publish", description: "Publish draft orders" },
        ],
      },
    ],
  },
  {
    id: "invoice-api",
    name: "Invoice API",
    resources: [
      {
        path: "invoices/*",
        label: "All Invoices",
        scopes: [
          { name: "read", description: "View and list invoices" },
          { name: "write", description: "Create and update invoices" },
          { name: "send", description: "Send invoices to customers" },
          { name: "void", description: "Void invoices" },
        ],
      },
      {
        path: "invoices/templates/*",
        label: "Invoice Templates",
        scopes: [
          { name: "read", description: "View templates" },
          { name: "write", description: "Create and update templates" },
          { name: "delete", description: "Delete templates" },
        ],
      },
    ],
  },
  {
    id: "user-api",
    name: "User API",
    resources: [
      {
        path: "users/*",
        label: "All Users",
        scopes: [
          { name: "read", description: "View and list users" },
          { name: "write", description: "Create and update users" },
          { name: "delete", description: "Delete users" },
          { name: "admin", description: "Full user administration" },
        ],
      },
      {
        path: "users/profiles/*",
        label: "User Profiles",
        scopes: [
          { name: "read", description: "View user profiles" },
          { name: "write", description: "Update user profiles" },
        ],
      },
    ],
  },
  {
    id: "external-drive",
    name: "Drive API (External)",
    resources: [
      {
        path: "drive/files/*",
        label: "Drive Files",
        isExternal: true,
        scopes: [
          { name: "read", description: "View files", isExternal: true },
          { name: "write", description: "Create and update files", isExternal: true },
          { name: "share", description: "Share files with others", isExternal: true },
        ],
      },
      {
        path: "drive/folders/*",
        label: "Drive Folders",
        isExternal: true,
        scopes: [
          { name: "read", description: "View folders", isExternal: true },
          { name: "write", description: "Create and update folders", isExternal: true },
          { name: "admin", description: "Full folder administration", isExternal: true },
        ],
      },
    ],
  },
];

export const mockUsers: User[] = [
  {
    id: "user_1",
    name: "Alice Johnson",
    email: "alice@company.com",
    avatar: "https://ui-avatars.com/api/?name=Alice+Johnson&background=0ea5e9&color=fff",
  },
  {
    id: "user_2",
    name: "Bob Smith",
    email: "bob@company.com",
    avatar: "https://ui-avatars.com/api/?name=Bob+Smith&background=10b981&color=fff",
  },
  {
    id: "user_3",
    name: "Charlie Brown",
    email: "charlie@company.com",
    avatar: "https://ui-avatars.com/api/?name=Charlie+Brown&background=f59e0b&color=fff",
  },
  {
    id: "user_4",
    name: "Diana Prince",
    email: "diana@company.com",
    avatar: "https://ui-avatars.com/api/?name=Diana+Prince&background=8b5cf6&color=fff",
  },
  {
    id: "user_5",
    name: "Eve Wilson",
    email: "eve@company.com",
    avatar: "https://ui-avatars.com/api/?name=Eve+Wilson&background=ef4444&color=fff",
  },
];

export const mockGroups: Group[] = [
  {
    id: "group_engineering",
    name: "Engineering",
    memberCount: 12,
    description: "Software engineering team",
  },
  {
    id: "group_marketing",
    name: "Marketing",
    memberCount: 8,
    description: "Marketing and growth team",
  },
  {
    id: "group_ops",
    name: "Operations",
    memberCount: 5,
    description: "Operations and support team",
  },
  {
    id: "group_sales",
    name: "Sales",
    memberCount: 15,
    description: "Sales and customer success team",
  },
  {
    id: "group_finance",
    name: "Finance",
    memberCount: 4,
    description: "Finance and accounting team",
  },
];
