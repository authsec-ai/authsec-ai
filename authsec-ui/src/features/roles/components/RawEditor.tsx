import { useState, useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { AlertCircle, Check, Copy, FileText } from "lucide-react";
import type { RoleFormData } from "../types";
import { toast } from "@/lib/toast";

interface RawEditorProps {
  formData: RoleFormData;
  onUpdate: (data: RoleFormData) => void;
  isOpen: boolean;
  onToggle: (open: boolean) => void;
}

export function RawEditor({ formData, onUpdate, isOpen, onToggle }: RawEditorProps) {
  const [format, setFormat] = useState<"json" | "yaml">("json");
  const [rawValue, setRawValue] = useState("");
  const [parseError, setParseError] = useState<string | null>(null);
  const [isValidJson, setIsValidJson] = useState(true);

  // Convert form data to JSON/YAML
  useEffect(() => {
    const jsonData = {
      role_id: formData.roleId,
      display_name: formData.displayName,
      description: formData.description,
      grants: formData.grants,
      assigned_users: formData.assignedUsers,
      assigned_groups: formData.assignedGroups,
    };

    try {
      if (format === "json") {
        setRawValue(JSON.stringify(jsonData, null, 2));
      } else {
        // For YAML, we'll simulate it with a structured format
        const yamlLines = [
          `role_id: "${jsonData.role_id}"`,
          `display_name: "${jsonData.display_name}"`,
          `description: "${jsonData.description}"`,
          `grants:`,
          ...jsonData.grants.map(
            (grant) =>
              `  - resource: "${grant.resource}"\n    scopes: [${grant.scopes
                .map((s) => `"${s}"`)
                .join(", ")}]`
          ),
          `assigned_users: [${jsonData.assigned_users.map((u) => `"${u}"`).join(", ")}]`,
          `assigned_groups: [${jsonData.assigned_groups.map((g) => `"${g}"`).join(", ")}]`,
        ];
        setRawValue(yamlLines.join("\n"));
      }
    } catch (error) {
      console.error("Error converting form data:", error);
    }
  }, [formData, format]);

  const handleRawValueChange = (value: string) => {
    setRawValue(value);

    if (format === "json") {
      try {
        const parsed = JSON.parse(value);
        setParseError(null);
        setIsValidJson(true);

        // Validate structure
        if (typeof parsed !== "object" || parsed === null) {
          setParseError("Root must be an object");
          setIsValidJson(false);
          return;
        }

        // Update form data
        const newFormData: RoleFormData = {
          roleId: parsed.role_id || "",
          displayName: parsed.display_name || "",
          description: parsed.description || "",
          grants: Array.isArray(parsed.grants) ? parsed.grants : [],
          assignedUsers: Array.isArray(parsed.assigned_users) ? parsed.assigned_users : [],
          assignedGroups: Array.isArray(parsed.assigned_groups) ? parsed.assigned_groups : [],
        };

        onUpdate(newFormData);
      } catch (error) {
        setParseError(error instanceof Error ? error.message : "Invalid JSON");
        setIsValidJson(false);
      }
    } else {
      // For YAML, we'll just show a message that it's not fully implemented
      setParseError("YAML parsing not fully implemented in this demo");
      setIsValidJson(false);
    }
  };

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(rawValue);
      toast.success("Raw data copied to clipboard");
    } catch (error) {
      toast.error("Failed to copy data");
    }
  };

  const handleFormatChange = (newFormat: "json" | "yaml") => {
    setFormat(newFormat);
    setParseError(null);
    setIsValidJson(true);
  };

  if (!isOpen) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Badge variant="outline" className="text-xs">
              D
            </Badge>
            Raw Editor
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center space-x-2">
            <Switch id="raw-editor" checked={isOpen} onCheckedChange={onToggle} />
            <Label htmlFor="raw-editor" className="text-sm">
              Edit as JSON/YAML
            </Label>
          </div>
          <p className="text-xs text-foreground mt-2">
            Switch to raw editor for advanced editing
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Badge variant="outline" className="text-xs">
            D
          </Badge>
          Raw Editor
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <Switch id="raw-editor" checked={isOpen} onCheckedChange={onToggle} />
            <Label htmlFor="raw-editor" className="text-sm">
              Edit as JSON/YAML
            </Label>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant={format === "json" ? "default" : "outline"}
              size="sm"
              onClick={() => handleFormatChange("json")}
            >
              JSON
            </Button>
            <Button
              variant={format === "yaml" ? "default" : "outline"}
              size="sm"
              onClick={() => handleFormatChange("yaml")}
            >
              YAML
            </Button>
          </div>
        </div>

        <div className="relative">
          <Textarea
            value={rawValue}
            onChange={(e) => handleRawValueChange(e.target.value)}
            className={`font-mono text-sm min-h-[300px] ${parseError ? "border-destructive" : ""}`}
            placeholder={`Enter ${format.toUpperCase()} data...`}
          />

          {/* Format indicator */}
          <div className="absolute top-2 right-2">
            <Badge variant="secondary" className="text-xs">
              {format.toUpperCase()}
            </Badge>
          </div>
        </div>

        {/* Error display */}
        {parseError && (
          <div className="flex items-center gap-2 p-3 bg-destructive/10 border border-destructive/20 rounded-md">
            <AlertCircle className="h-4 w-4 text-destructive" />
            <div className="flex-1">
              <div className="text-sm font-medium text-destructive">Parse Error</div>
              <div className="text-xs text-destructive">{parseError}</div>
            </div>
          </div>
        )}

        {/* Success indicator */}
        {!parseError && format === "json" && (
          <div className="flex items-center gap-2 p-3 bg-green-50 border border-green-200 rounded-md">
            <Check className="h-4 w-4 text-green-600" />
            <div className="text-sm text-green-800">Valid JSON format</div>
          </div>
        )}

        <div className="flex justify-between items-center">
          <Button variant="outline" size="sm" onClick={handleCopy}>
            <Copy className="mr-2 h-4 w-4" />
            Copy Raw Data
          </Button>

          <div className="text-xs text-foreground">
            {rawValue.split("\n").length} lines • {rawValue.length} characters
          </div>
        </div>

        {/* Format help */}
        <div className="p-3 bg-muted/50 rounded-md">
          <div className="text-sm font-medium mb-1">Format Guide</div>
          <div className="text-xs text-foreground">
            {format === "json" ? (
              <div>
                JSON format allows direct editing of the role structure. Parse errors are
                highlighted and prevent saving.
              </div>
            ) : (
              <div>
                YAML format provides a more readable structure. Full YAML parsing is not implemented
                in this demo.
              </div>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
