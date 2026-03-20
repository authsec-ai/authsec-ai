import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "../../../components/ui/dialog";
import { Button } from "../../../components/ui/button";
import { useGetAllClientsQuery } from "../../../app/api/clientApi";
import { SessionManager } from "../../../utils/sessionManager";
import { ArrowRight } from "lucide-react";
import { Badge } from "../../../components/ui/badge";

interface ClientSelectionModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export function ClientSelectionModal({
  isOpen,
  onClose,
}: ClientSelectionModalProps) {
  const navigate = useNavigate();
  const sessionData = SessionManager.getSession();
  const tenantId = sessionData?.tenant_id || "";

  const { data: clientsResponse, isLoading } = useGetAllClientsQuery(
    { tenant_id: tenantId },
    { skip: !tenantId }
  );

  const clients = clientsResponse?.clients || [];
  const [selectedClient, setSelectedClient] = useState<string | null>(null);

  const handleClientSelect = (clientId: string) => {
    setSelectedClient(clientId);
  };

  const handleContinue = () => {
    if (selectedClient) {
      navigate(`/sdk/clients/${selectedClient}`);
      onClose();
    }
  };

  if (isLoading) {
    return (
      <Dialog open={isOpen} onOpenChange={onClose}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Select Client</DialogTitle>
            <DialogDescription>Loading clients...</DialogDescription>
          </DialogHeader>
          <div className="flex items-center justify-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
          </div>
        </DialogContent>
      </Dialog>
    );
  }

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-3xl max-h-[80vh]">
        <DialogHeader>
          <DialogTitle>Select Client for SDK Integration</DialogTitle>
          <DialogDescription>
            Choose a client to view its SDK integration guide
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 overflow-y-auto max-h-[50vh] pr-2">
          {clients.map((client) => (
            <button
              key={client.client_id}
              onClick={() => handleClientSelect(client.client_id)}
              className={`w-full text-left p-4 rounded-lg border-2 transition-all ${
                selectedClient === client.client_id
                  ? "border-blue-500 bg-blue-50 dark:bg-blue-900/20"
                  : "border-slate-200 dark:border-neutral-700 hover:border-slate-300 dark:hover:border-neutral-600 bg-white dark:bg-neutral-800"
              }`}
            >
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2 mb-1">
                    <h4 className="font-semibold text-slate-900 dark:text-neutral-100">
                      {client.name || client.client_name || "Unnamed Client"}
                    </h4>
                    {client.active ? (
                      <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800">
                        Active
                      </Badge>
                    ) : (
                      <Badge variant="outline" className="bg-slate-50 text-slate-700 border-slate-200 dark:bg-neutral-800 dark:text-neutral-300 dark:border-neutral-700">
                        Inactive
                      </Badge>
                    )}
                  </div>
                  <p className="text-sm text-slate-600 dark:text-neutral-400">
                    Client ID: {client.client_id}
                  </p>
                  {client.email && (
                    <p className="text-sm text-slate-500 dark:text-neutral-500 mt-1">
                      {client.email}
                    </p>
                  )}
                </div>
                {selectedClient === client.client_id && (
                  <div className="flex items-center justify-center w-6 h-6 rounded-full bg-blue-500 text-white">
                    <svg
                      className="w-4 h-4"
                      fill="currentColor"
                      viewBox="0 0 20 20"
                    >
                      <path
                        fillRule="evenodd"
                        d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                        clipRule="evenodd"
                      />
                    </svg>
                  </div>
                )}
              </div>
            </button>
          ))}
        </div>

        <div className="flex items-center justify-end gap-3 pt-4 border-t">
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button
            onClick={handleContinue}
            disabled={!selectedClient}
            className="gap-2"
          >
            Continue to SDK
            <ArrowRight className="h-4 w-4" />
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
