import { useState, useCallback } from "react";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { PasswordInput } from "../../components/ui/password-input";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../../components/ui/card";
import { Badge } from "../../components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../../components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../../components/ui/tabs";
import { Progress } from "../../components/ui/progress";
import { Upload, FileText, Database, Key, AlertTriangle, Check, X } from "lucide-react";

interface ImportSecret {
  name: string;
  type: string;
  source: string;
  status: "pending" | "imported" | "error" | "duplicate";
  error?: string;
}

/**
 * Import Secrets Page - Bulk import secrets from external sources
 */
export function ImportSecretsPage() {
  const [importSource, setImportSource] = useState("");
  const [isImporting, setIsImporting] = useState(false);
  const [importProgress, setImportProgress] = useState(0);
  const [secrets, setSecrets] = useState<ImportSecret[]>([]);
  const [dragActive, setDragActive] = useState(false);

  const importSources = [
    { value: "aws", label: "AWS Secrets Manager", icon: Database },
    { value: "azure", label: "Azure Key Vault", icon: Key },
    { value: "gcp", label: "Google Secret Manager", icon: Database },
    { value: "hashicorp", label: "HashiCorp Vault", icon: Key },
    { value: "kubernetes", label: "Kubernetes Secrets", icon: Database },
    { value: "csv", label: "CSV File", icon: FileText },
    { value: "json", label: "JSON File", icon: FileText },
  ];

  const handleDrag = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === "dragenter" || e.type === "dragover") {
      setDragActive(true);
    } else if (e.type === "dragleave") {
      setDragActive(false);
    }
  }, []);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);

    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      handleFile(e.dataTransfer.files[0]);
    }
  }, []);

  const handleFile = (file: File) => {
    // Simulate file processing
    const mockSecrets: ImportSecret[] = [
      { name: "database-password", type: "password", source: file.name, status: "pending" },
      { name: "api-key-stripe", type: "api_key", source: file.name, status: "pending" },
      { name: "jwt-secret", type: "token", source: file.name, status: "duplicate" },
      { name: "oauth-client-secret", type: "oauth", source: file.name, status: "pending" },
      { name: "encryption-key", type: "certificate", source: file.name, status: "pending" },
    ];
    setSecrets(mockSecrets);
  };

  const handleStartImport = async () => {
    setIsImporting(true);
    setImportProgress(0);

    // Simulate import process
    for (let i = 0; i < secrets.length; i++) {
      if (secrets[i].status === "pending") {
        await new Promise((resolve) => setTimeout(resolve, 800));
        setSecrets((prev) =>
          prev.map((secret, idx) =>
            idx === i && secret.status === "pending"
              ? {
                  ...secret,
                  status: Math.random() > 0.1 ? "imported" : "error",
                  error: Math.random() > 0.1 ? undefined : "Validation failed",
                }
              : secret
          )
        );
        setImportProgress(((i + 1) / secrets.length) * 100);
      }
    }

    setIsImporting(false);
  };

  const getStatusIcon = (status: ImportSecret["status"]) => {
    switch (status) {
      case "imported":
        return <Check className="h-4 w-4 text-green-500" />;
      case "error":
        return <X className="h-4 w-4 text-red-500" />;
      case "duplicate":
        return <AlertTriangle className="h-4 w-4 text-yellow-500" />;
      default:
        return <Key className="h-4 w-4 text-foreground" />;
    }
  };

  const getStatusVariant = (status: ImportSecret["status"]) => {
    switch (status) {
      case "imported":
        return "default";
      case "error":
        return "destructive";
      case "duplicate":
        return "secondary";
      default:
        return "outline";
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Import Secrets</h1>
          <p className="text-foreground">Bulk import secrets from external sources</p>
        </div>
        <Button onClick={handleStartImport} disabled={secrets.length === 0 || isImporting}>
          {isImporting ? (
            <>Importing... ({Math.round(importProgress)}%)</>
          ) : (
            <>
              <Upload className="mr-2 h-4 w-4" />
              Import Secrets ({secrets.filter((s) => s.status === "pending").length})
            </>
          )}
        </Button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Import Configuration */}
        <div className="lg:col-span-2 space-y-4">
          <Tabs defaultValue="file" className="space-y-4">
            <TabsList>
              <TabsTrigger value="file">File Upload</TabsTrigger>
              <TabsTrigger value="cloud">Cloud Providers</TabsTrigger>
              <TabsTrigger value="database">Database Import</TabsTrigger>
            </TabsList>

            <TabsContent value="file">
              <Card>
                <CardHeader>
                  <CardTitle>File Import</CardTitle>
                  <CardDescription>
                    Upload CSV or JSON files containing secret definitions
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div
                    className={`border-2 border-dashed rounded-lg p-8 text-center transition-colors ${
                      dragActive ? "border-primary bg-primary/5" : "border-muted-foreground/25"
                    }`}
                    onDragEnter={handleDrag}
                    onDragLeave={handleDrag}
                    onDragOver={handleDrag}
                    onDrop={handleDrop}
                  >
                    <Upload className="mx-auto h-8 w-8 text-foreground mb-2" />
                    <p className="text-sm text-foreground mb-2">
                      Drag and drop your CSV or JSON file here, or click to browse
                    </p>
                    <Button variant="outline">Choose File</Button>
                  </div>

                  <div className="grid grid-cols-2 gap-4 text-sm">
                    <div>
                      <h4 className="font-medium mb-2">CSV Format:</h4>
                      <code className="text-xs bg-muted p-2 rounded block">
                        name,type,value,service
                        <br />
                        api-key,api_key,sk_test_...,payment
                        <br />
                        db-pass,password,secret123,database
                      </code>
                    </div>
                    <div>
                      <h4 className="font-medium mb-2">JSON Format:</h4>
                      <code className="text-xs bg-muted p-2 rounded block">
                        {`{
  "secrets": [
    {
      "name": "api-key",
      "type": "api_key",
      "value": "sk_test_...",
      "service": "payment"
    }
  ]
}`}
                      </code>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="cloud">
              <Card>
                <CardHeader>
                  <CardTitle>Cloud Provider Import</CardTitle>
                  <CardDescription>
                    Import secrets from cloud secret management services
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-2">
                    <label className="text-sm font-medium">Provider</label>
                    <Select value={importSource} onValueChange={setImportSource}>
                      <SelectTrigger>
                        <SelectValue placeholder="Select cloud provider" />
                      </SelectTrigger>
                      <SelectContent>
                        {importSources
                          .filter((s) => s.value !== "csv" && s.value !== "json")
                          .map((source) => (
                            <SelectItem key={source.value} value={source.value}>
                              {source.label}
                            </SelectItem>
                          ))}
                      </SelectContent>
                    </Select>
                  </div>

                  {importSource && (
                    <div className="space-y-4 border rounded-lg p-4">
                      <h4 className="font-medium">Connection Configuration</h4>
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div className="space-y-2">
                          <label className="text-sm font-medium">Access Key / Client ID</label>
                          <PasswordInput placeholder="Enter access credentials" />
                        </div>
                        <div className="space-y-2">
                          <label className="text-sm font-medium">Secret Key / Client Secret</label>
                          <PasswordInput placeholder="Enter secret credentials" />
                        </div>
                      </div>
                      <div className="space-y-2">
                        <label className="text-sm font-medium">Region / Endpoint</label>
                        <Input placeholder="us-east-1" />
                      </div>
                      <Button variant="outline">Test Connection</Button>
                    </div>
                  )}
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="database">
              <Card>
                <CardHeader>
                  <CardTitle>Database Import</CardTitle>
                  <CardDescription>
                    Import secrets from existing database or configuration files
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <label className="text-sm font-medium">Database Type</label>
                      <Select>
                        <SelectTrigger>
                          <SelectValue placeholder="Select database" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="postgresql">PostgreSQL</SelectItem>
                          <SelectItem value="mysql">MySQL</SelectItem>
                          <SelectItem value="mongodb">MongoDB</SelectItem>
                          <SelectItem value="redis">Redis</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-2">
                      <label className="text-sm font-medium">Connection String</label>
                      <PasswordInput placeholder="postgresql://user:pass@host:port/db" />
                    </div>
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium">Query / Collection</label>
                    <Input placeholder="SELECT name, value, type FROM secrets" />
                  </div>
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>
        </div>

        {/* Import Preview */}
        <div className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Import Preview</CardTitle>
              <CardDescription>{secrets.length} secrets ready for import</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              {isImporting && (
                <div className="space-y-2 mb-4">
                  <div className="flex items-center justify-between text-sm">
                    <span>Importing...</span>
                    <span>{Math.round(importProgress)}%</span>
                  </div>
                  <Progress value={importProgress} />
                </div>
              )}

              {secrets.length === 0 ? (
                <p className="text-sm text-foreground text-center py-4">
                  Upload a file to preview secrets
                </p>
              ) : (
                secrets.map((secret, index) => (
                  <div key={index} className="flex items-center justify-between p-2 border rounded">
                    <div className="flex items-center space-x-2">
                      {getStatusIcon(secret.status)}
                      <div className="flex-1 min-w-0">
                        <div className="text-sm font-medium truncate">{secret.name}</div>
                        <div className="text-xs text-foreground">
                          {secret.type} • {secret.source}
                        </div>
                        {secret.error && <div className="text-xs text-red-500">{secret.error}</div>}
                      </div>
                    </div>
                    <Badge variant={getStatusVariant(secret.status)}>{secret.status}</Badge>
                  </div>
                ))
              )}
            </CardContent>
          </Card>

          {secrets.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Import Statistics</CardTitle>
              </CardHeader>
              <CardContent className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span>Ready:</span>
                  <span>{secrets.filter((s) => s.status === "pending").length}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span>Imported:</span>
                  <span className="text-green-600">
                    {secrets.filter((s) => s.status === "imported").length}
                  </span>
                </div>
                <div className="flex justify-between text-sm">
                  <span>Duplicates:</span>
                  <span className="text-yellow-600">
                    {secrets.filter((s) => s.status === "duplicate").length}
                  </span>
                </div>
                <div className="flex justify-between text-sm">
                  <span>Errors:</span>
                  <span className="text-red-600">
                    {secrets.filter((s) => s.status === "error").length}
                  </span>
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
}
