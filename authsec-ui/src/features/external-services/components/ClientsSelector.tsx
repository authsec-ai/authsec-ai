import { useState, useEffect } from "react";
import { Checkbox } from "../../../components/ui/checkbox";
import { RadioGroup, RadioGroupItem } from "../../../components/ui/radio-group";
import { Label } from "../../../components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../../../components/ui/select";
import { Badge } from "../../../components/ui/badge";
import type { ExternalServiceFormData, ClientOption } from "../types";

// Mock client data
const mockClients: ClientOption[] = [
  { id: "order-api", name: "Order API", environment: "prod", type: "MCP-Server" },
  { id: "chat-bot", name: "Chat Bot", environment: "dev", type: "AI-Agent" },
  { id: "analytics", name: "Analytics Dashboard", environment: "prod", type: "MCP-Server" },
  { id: "customer-portal", name: "Customer Portal", environment: "staging", type: "MCP-Server" },
  { id: "assistant", name: "Virtual Assistant", environment: "prod", type: "AI-Agent" },
  { id: "admin-api", name: "Admin API", environment: "dev", type: "MCP-Server" },
];

interface ClientsSelectorProps {
  formData: ExternalServiceFormData;
  onUpdate: (updates: Partial<ExternalServiceFormData>) => void;
}

export function ClientsSelector({ formData, onUpdate }: ClientsSelectorProps) {
  const [envFilter, setEnvFilter] = useState<string>("all");
  const [filteredClients, setFilteredClients] = useState<ClientOption[]>(mockClients);

  // Filter clients by environment
  useEffect(() => {
    if (envFilter === "all") {
      setFilteredClients(mockClients);
    } else {
      setFilteredClients(mockClients.filter((client) => client.environment === envFilter));
    }
  }, [envFilter]);

  const handleClientToggle = (clientId: string, checked: boolean) => {
    const updatedClients = checked
      ? [...formData.linkedClients, clientId]
      : formData.linkedClients.filter((id) => id !== clientId);

    onUpdate({ linkedClients: updatedClients });

    // If this was the only client and it was just selected, make it the default
    if (updatedClients.length === 1 && checked) {
      onUpdate({ defaultClientId: clientId });
    }

    // If the default client was removed, clear the default
    if (!checked && clientId === formData.defaultClientId) {
      onUpdate({ defaultClientId: "" });
    }
  };

  const handleDefaultChange = (clientId: string) => {
    onUpdate({ defaultClientId: clientId });
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="text-sm text-foreground">
          Select clients that will use this external service
        </div>
        <Select value={envFilter} onValueChange={setEnvFilter}>
          <SelectTrigger className="w-32">
            <SelectValue placeholder="Environment" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Envs</SelectItem>
            <SelectItem value="prod">Production</SelectItem>
            <SelectItem value="staging">Staging</SelectItem>
            <SelectItem value="dev">Development</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="border rounded-md">
        <table className="w-full">
          <thead>
            <tr className="border-b bg-muted/50">
              <th className="text-left p-3 font-medium text-sm">Client</th>
              <th className="text-left p-3 font-medium text-sm">Environment</th>
              <th className="text-center p-3 font-medium text-sm">Default</th>
              <th className="text-center p-3 font-medium text-sm">Attached</th>
            </tr>
          </thead>
          <tbody>
            {filteredClients.length === 0 ? (
              <tr>
                <td colSpan={4} className="p-4 text-center text-foreground">
                  No clients found for the selected environment.
                </td>
              </tr>
            ) : (
              filteredClients.map((client) => (
                <tr key={client.id} className="border-b last:border-b-0 hover:bg-muted/30">
                  <td className="p-3">
                    <div className="flex flex-col">
                      <span className="font-medium">{client.name}</span>
                      <span className="text-xs text-foreground">{client.id}</span>
                    </div>
                  </td>
                  <td className="p-3">
                    <Badge
                      variant={
                        client.environment === "prod"
                          ? "default"
                          : client.environment === "staging"
                          ? "secondary"
                          : "outline"
                      }
                      className="capitalize"
                    >
                      {client.environment}
                    </Badge>
                    <span className="ml-2 text-xs text-foreground">{client.type}</span>
                  </td>
                  <td className="p-3 text-center">
                    <RadioGroup
                      value={formData.defaultClientId}
                      onValueChange={handleDefaultChange}
                      className="flex justify-center"
                    >
                      <RadioGroupItem
                        value={client.id}
                        id={`default-${client.id}`}
                        disabled={!formData.linkedClients.includes(client.id)}
                      />
                    </RadioGroup>
                  </td>
                  <td className="p-3 text-center">
                    <div className="flex justify-center">
                      <Checkbox
                        id={`client-${client.id}`}
                        checked={formData.linkedClients.includes(client.id)}
                        onCheckedChange={(checked) =>
                          handleClientToggle(client.id, checked as boolean)
                        }
                      />
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <div className="text-sm text-foreground">
        {formData.linkedClients.length > 0 ? (
          <>
            <span className="font-medium">{formData.linkedClients.length}</span> client
            {formData.linkedClients.length !== 1 && "s"} selected
            {formData.defaultClientId && (
              <>
                , with{" "}
                <span className="font-medium">
                  {mockClients.find((c) => c.id === formData.defaultClientId)?.name ||
                    formData.defaultClientId}
                </span>{" "}
                as default
              </>
            )}
          </>
        ) : (
          "No clients selected. This service won't be available to any clients until linked."
        )}
      </div>
    </div>
  );
}
