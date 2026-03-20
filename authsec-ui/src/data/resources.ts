// Mock data for resources - bulk actions
export const bulkResourceActions = [
  {
    id: 'delete',
    label: 'Delete Selected',
    icon: 'trash',
    variant: 'destructive' as const,
    requiresConfirmation: true
  },
  {
    id: 'enable',
    label: 'Enable Selected',
    icon: 'check',
    variant: 'default' as const
  },
  {
    id: 'disable',
    label: 'Disable Selected', 
    icon: 'x',
    variant: 'secondary' as const
  },
  {
    id: 'export',
    label: 'Export Selected',
    icon: 'download',
    variant: 'outline' as const
  }
];