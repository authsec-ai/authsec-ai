import { useState } from "react";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../../components/ui/card";
import { Badge } from "../../components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../../components/ui/tabs";
import {
  ArrowLeft,
  Search,
  Shield,
  Users,
  Database,
  Key,
  Settings,
  Star,
  Copy,
} from "lucide-react";
import { useContextualNavigate } from "@/hooks/useContextualNavigate";

interface RoleTemplate {
  id: string;
  name: string;
  description: string;
  category: "enterprise" | "development" | "security" | "operations";
  popular: boolean;
  permissions: Array<{
    resource: string;
    actions: string[];
    description: string;
  }>;
  usageCount: number;
  tags: string[];
}

/**
 * Role Templates Page - Pre-built role templates for common use cases
 */
export function RoleTemplatesPage() {
  const navigate = useContextualNavigate();
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedCategory, setSelectedCategory] = useState<string>("all");

  const roleTemplates: RoleTemplate[] = [
    {
      id: "admin",
      name: "System Administrator",
      description: "Complete system access with full administrative privileges",
      category: "enterprise",
      popular: true,
      permissions: [
        {
          resource: "agents",
          actions: ["read", "write", "delete", "admin"],
          description: "Full agent management",
        },
        {
          resource: "users",
          actions: ["read", "write", "delete", "admin"],
          description: "Complete user administration",
        },
        {
          resource: "services",
          actions: ["read", "write", "delete", "admin"],
          description: "Service management and configuration",
        },
        {
          resource: "vault",
          actions: ["read", "write", "delete", "admin"],
          description: "Full secrets management",
        },
        {
          resource: "policies",
          actions: ["read", "write", "delete", "admin"],
          description: "Policy creation and management",
        },
        { resource: "logs", actions: ["read", "admin"], description: "System monitoring and logs" },
        {
          resource: "roles",
          actions: ["read", "write", "delete", "admin"],
          description: "Role and permission management",
        },
      ],
      usageCount: 145,
      tags: ["admin", "full-access", "enterprise"],
    },
    {
      id: "security-manager",
      name: "Security Manager",
      description: "Manage security policies, access controls, and compliance",
      category: "security",
      popular: true,
      permissions: [
        {
          resource: "policies",
          actions: ["read", "write", "delete"],
          description: "Conditional access policies",
        },
        {
          resource: "vault",
          actions: ["read", "write"],
          description: "Secret management and rotation",
        },
        {
          resource: "authentication",
          actions: ["read", "write", "configure"],
          description: "Authentication methods",
        },
        { resource: "logs", actions: ["read"], description: "Security event monitoring" },
        { resource: "users", actions: ["read", "write"], description: "User security settings" },
        { resource: "roles", actions: ["read", "write"], description: "Role management" },
      ],
      usageCount: 89,
      tags: ["security", "compliance", "policies"],
    },
    {
      id: "developer",
      name: "Developer",
      description: "Development team access for building and deploying applications",
      category: "development",
      popular: true,
      permissions: [
        {
          resource: "agents",
          actions: ["read", "write"],
          description: "Agent development and testing",
        },
        {
          resource: "services",
          actions: ["read", "write"],
          description: "Service integration and management",
        },
        { resource: "vault", actions: ["read"], description: "Access to development secrets" },
        { resource: "logs", actions: ["read"], description: "Application logs and debugging" },
        {
          resource: "authentication",
          actions: ["read"],
          description: "Auth method configuration viewing",
        },
      ],
      usageCount: 234,
      tags: ["developer", "build", "deploy"],
    },
    {
      id: "devops-engineer",
      name: "DevOps Engineer",
      description: "Infrastructure and deployment pipeline management",
      category: "operations",
      popular: false,
      permissions: [
        {
          resource: "services",
          actions: ["read", "write", "configure"],
          description: "Service deployment and scaling",
        },
        {
          resource: "agents",
          actions: ["read", "write", "configure"],
          description: "Agent deployment and monitoring",
        },
        {
          resource: "vault",
          actions: ["read", "write"],
          description: "Infrastructure secrets management",
        },
        { resource: "logs", actions: ["read"], description: "System and application monitoring" },
        {
          resource: "authentication",
          actions: ["read", "configure"],
          description: "Infrastructure authentication",
        },
      ],
      usageCount: 67,
      tags: ["devops", "infrastructure", "deployment"],
    },
    {
      id: "viewer",
      name: "Read-Only Viewer",
      description: "Read-only access across all system resources",
      category: "enterprise",
      popular: false,
      permissions: [
        { resource: "agents", actions: ["read"], description: "View agent information" },
        { resource: "services", actions: ["read"], description: "View service status" },
        { resource: "users", actions: ["read"], description: "View user information" },
        { resource: "vault", actions: ["read"], description: "View secret metadata (not values)" },
        { resource: "policies", actions: ["read"], description: "View access policies" },
        { resource: "logs", actions: ["read"], description: "View system logs" },
        { resource: "roles", actions: ["read"], description: "View role definitions" },
      ],
      usageCount: 156,
      tags: ["viewer", "read-only", "audit"],
    },
    {
      id: "support-analyst",
      name: "Support Analyst",
      description: "Customer support and troubleshooting access",
      category: "operations",
      popular: false,
      permissions: [
        {
          resource: "users",
          actions: ["read", "write"],
          description: "User support and account management",
        },
        { resource: "agents", actions: ["read"], description: "Agent status for troubleshooting" },
        { resource: "services", actions: ["read"], description: "Service health monitoring" },
        { resource: "logs", actions: ["read"], description: "Event logs for issue resolution" },
        { resource: "authentication", actions: ["read"], description: "Auth troubleshooting" },
      ],
      usageCount: 43,
      tags: ["support", "troubleshooting", "customer"],
    },
    {
      id: "compliance-officer",
      name: "Compliance Officer",
      description: "Audit and compliance monitoring with read-only access",
      category: "security",
      popular: false,
      permissions: [
        { resource: "logs", actions: ["read"], description: "Audit trail and compliance logs" },
        { resource: "policies", actions: ["read"], description: "Policy compliance review" },
        { resource: "users", actions: ["read"], description: "User access audit" },
        { resource: "roles", actions: ["read"], description: "Role and permission audit" },
        { resource: "vault", actions: ["read"], description: "Secret access audit" },
        { resource: "authentication", actions: ["read"], description: "Authentication audit" },
      ],
      usageCount: 29,
      tags: ["compliance", "audit", "governance"],
    },
    {
      id: "api-developer",
      name: "API Developer",
      description: "Focused access for API development and integration",
      category: "development",
      popular: false,
      permissions: [
        { resource: "agents", actions: ["read", "write"], description: "API agent management" },
        {
          resource: "services",
          actions: ["read", "write"],
          description: "API service integration",
        },
        { resource: "vault", actions: ["read"], description: "API keys and credentials" },
        {
          resource: "authentication",
          actions: ["read", "write"],
          description: "API authentication setup",
        },
        { resource: "logs", actions: ["read"], description: "API usage and error logs" },
      ],
      usageCount: 78,
      tags: ["api", "integration", "development"],
    },
  ];

  const categories = [
    { value: "all", label: "All Categories", count: roleTemplates.length },
    {
      value: "enterprise",
      label: "Enterprise",
      count: roleTemplates.filter((t) => t.category === "enterprise").length,
    },
    {
      value: "development",
      label: "Development",
      count: roleTemplates.filter((t) => t.category === "development").length,
    },
    {
      value: "security",
      label: "Security",
      count: roleTemplates.filter((t) => t.category === "security").length,
    },
    {
      value: "operations",
      label: "Operations",
      count: roleTemplates.filter((t) => t.category === "operations").length,
    },
  ];

  const filteredTemplates = roleTemplates.filter((template) => {
    const matchesSearch =
      searchQuery === "" ||
      template.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      template.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
      template.tags.some((tag) => tag.toLowerCase().includes(searchQuery.toLowerCase()));

    const matchesCategory = selectedCategory === "all" || template.category === selectedCategory;

    return matchesSearch && matchesCategory;
  });

  const popularTemplates = roleTemplates.filter((t) => t.popular);

  const handleUseTemplate = (template: RoleTemplate) => {
    // Navigate to create role page with template data
    console.warn("Using template:", template);
    navigate("/roles/create", { state: { template } });
  };

  const handleCloneTemplate = (template: RoleTemplate) => {
    console.warn("Cloning template:", template);
    // Here you would create a copy of the template
  };

  const getCategoryIcon = (category: string) => {
    switch (category) {
      case "enterprise":
        return <Settings className="h-4 w-4" />;
      case "development":
        return <Database className="h-4 w-4" />;
      case "security":
        return <Shield className="h-4 w-4" />;
      case "operations":
        return <Users className="h-4 w-4" />;
      default:
        return <Key className="h-4 w-4" />;
    }
  };

  const getCategoryColor = (category: string) => {
    switch (category) {
      case "enterprise":
        return "bg-badge-blue text-badge-blue-foreground";
      case "development":
        return "bg-badge-green text-badge-green-foreground";
      case "security":
        return "bg-badge-red text-badge-red-foreground";
      case "operations":
        return "bg-badge-purple text-badge-purple-foreground";
      default:
        return "bg-badge-gray text-badge-gray-foreground";
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => navigate("/roles")}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Role Templates</h1>
          <p className="text-foreground">Pre-built role templates for common use cases</p>
        </div>
      </div>

      {/* Search and Filters */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex flex-col gap-4 md:flex-row md:items-center">
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-foreground" />
                <Input
                  placeholder="Search templates by name, description, or tags..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-10"
                />
              </div>
            </div>
            <div className="flex gap-2">
              {categories.map((category) => (
                <Button
                  key={category.value}
                  variant={selectedCategory === category.value ? "default" : "outline"}
                  size="sm"
                  onClick={() => setSelectedCategory(category.value)}
                >
                  {category.label} ({category.count})
                </Button>
              ))}
            </div>
          </div>
        </CardContent>
      </Card>

      <Tabs defaultValue="all" className="space-y-4">
        <TabsList>
          <TabsTrigger value="all">All Templates</TabsTrigger>
          <TabsTrigger value="popular">Popular</TabsTrigger>
        </TabsList>

        <TabsContent value="all">
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {filteredTemplates.map((template) => (
              <Card key={template.id} className="transition-colors hover:bg-muted/50">
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between">
                    <div className="space-y-1 flex-1">
                      <div className="flex items-center gap-2">
                        <CardTitle className="text-base">{template.name}</CardTitle>
                        {template.popular && (
                          <Star className="h-4 w-4 text-yellow-500 fill-current" />
                        )}
                      </div>
                      <CardDescription className="text-sm">{template.description}</CardDescription>
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="flex items-center gap-2">
                    {getCategoryIcon(template.category)}
                    <Badge className={getCategoryColor(template.category)} variant="outline">
                      {template.category}
                    </Badge>
                    <Badge variant="secondary" className="text-xs">
                      {template.permissions.length} permissions
                    </Badge>
                  </div>

                  <div>
                    <div className="text-sm font-medium mb-2">Key Permissions:</div>
                    <div className="space-y-1">
                      {template.permissions.slice(0, 3).map((perm, index) => (
                        <div key={index} className="text-xs text-foreground">
                          • {perm.description}
                        </div>
                      ))}
                      {template.permissions.length > 3 && (
                        <div className="text-xs text-foreground">
                          • +{template.permissions.length - 3} more permissions
                        </div>
                      )}
                    </div>
                  </div>

                  <div className="flex flex-wrap gap-1">
                    {template.tags.slice(0, 3).map((tag) => (
                      <Badge key={tag} variant="outline" className="text-xs">
                        {tag}
                      </Badge>
                    ))}
                  </div>

                  <div className="text-xs text-foreground">
                    Used {template.usageCount} times
                  </div>

                  <div className="flex gap-2 pt-2">
                    <Button
                      size="sm"
                      className="flex-1"
                      onClick={() => handleUseTemplate(template)}
                    >
                      Use Template
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => handleCloneTemplate(template)}
                    >
                      <Copy className="h-3 w-3" />
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        <TabsContent value="popular">
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {popularTemplates.map((template) => (
              <Card
                key={template.id}
                className="transition-colors hover:bg-muted/50 border-yellow-200 dark:border-yellow-800"
              >
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between">
                    <div className="space-y-1 flex-1">
                      <div className="flex items-center gap-2">
                        <CardTitle className="text-base">{template.name}</CardTitle>
                        <Star className="h-4 w-4 text-yellow-500 fill-current" />
                      </div>
                      <CardDescription className="text-sm">{template.description}</CardDescription>
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="flex items-center gap-2">
                    {getCategoryIcon(template.category)}
                    <Badge className={getCategoryColor(template.category)} variant="outline">
                      {template.category}
                    </Badge>
                    <Badge variant="secondary" className="text-xs">
                      {template.permissions.length} permissions
                    </Badge>
                  </div>

                  <div>
                    <div className="text-sm font-medium mb-2">Key Permissions:</div>
                    <div className="space-y-1">
                      {template.permissions.slice(0, 3).map((perm, index) => (
                        <div key={index} className="text-xs text-foreground">
                          • {perm.description}
                        </div>
                      ))}
                      {template.permissions.length > 3 && (
                        <div className="text-xs text-foreground">
                          • +{template.permissions.length - 3} more permissions
                        </div>
                      )}
                    </div>
                  </div>

                  <div className="text-xs text-foreground font-medium text-yellow-600 dark:text-yellow-400">
                    🔥 Popular • Used {template.usageCount} times
                  </div>

                  <div className="flex gap-2 pt-2">
                    <Button
                      size="sm"
                      className="flex-1"
                      onClick={() => handleUseTemplate(template)}
                    >
                      Use Template
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => handleCloneTemplate(template)}
                    >
                      <Copy className="h-3 w-3" />
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}
