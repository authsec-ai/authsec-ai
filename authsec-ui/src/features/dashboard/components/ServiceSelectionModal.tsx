import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { X, ArrowRight } from "lucide-react";
import { Button } from "../../../components/ui/button";
import { useGetExternalServicesQuery } from "../../../app/api/externalServiceApi";
import { SessionManager } from "../../../utils/sessionManager";

interface ServiceSelectionModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export function ServiceSelectionModal({
  isOpen,
  onClose,
}: ServiceSelectionModalProps) {
  const navigate = useNavigate();
  const sessionData = SessionManager.getSession();
  const tenantId = sessionData?.tenant_id || "";
  const { data: servicesData, isLoading } = useGetExternalServicesQuery(
    undefined,
    { skip: !tenantId }
  );

  const [selectedServiceId, setSelectedServiceId] = useState<string>("");

  const services = servicesData || [];

  const handleContinue = () => {
    if (selectedServiceId) {
      navigate(`/sdk/external-services/${selectedServiceId}`);
      onClose();
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <div className="bg-background border border-border rounded-lg shadow-2xl w-full max-w-md mx-4 overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-border bg-muted/30">
          <h2 className="text-lg font-semibold">Select External Service</h2>
          <Button
            variant="ghost"
            size="sm"
            onClick={onClose}
            className="h-8 w-8 p-0"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>

        {/* Content */}
        <div className="p-6">
          {isLoading ? (
            <div className="text-center py-8 text-foreground">
              Loading services...
            </div>
          ) : (
            <div className="space-y-4">
              <p className="text-sm text-foreground mb-4">
                Select a service to view SDK integration instructions
              </p>

              <div className="space-y-2 max-h-64 overflow-y-auto">
                {services.map((service) => (
                  <div
                    key={service.id}
                    onClick={() => setSelectedServiceId(service.id)}
                    className={`p-3 border rounded-lg cursor-pointer transition-all ${
                      selectedServiceId === service.id
                        ? "border-primary bg-primary/5 shadow-sm"
                        : "border-border hover:border-primary/50 hover:bg-muted/30"
                    }`}
                  >
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <h3 className="font-medium text-sm">
                          {service.name}
                        </h3>
                        {service.description && (
                          <p className="text-xs text-foreground mt-1">
                            {service.description}
                          </p>
                        )}
                        <p className="text-xs text-foreground/70 mt-1">
                          {service.url}
                        </p>
                      </div>
                      {selectedServiceId === service.id && (
                        <div className="ml-2 mt-1">
                          <div className="h-5 w-5 rounded-full bg-primary flex items-center justify-center">
                            <div className="h-2 w-2 rounded-full bg-white" />
                          </div>
                        </div>
                      )}
                    </div>
                  </div>
                ))}
              </div>

              <Button
                onClick={handleContinue}
                disabled={!selectedServiceId}
                className="w-full mt-4"
              >
                View SDK Integration
                <ArrowRight className="ml-2 h-4 w-4" />
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
