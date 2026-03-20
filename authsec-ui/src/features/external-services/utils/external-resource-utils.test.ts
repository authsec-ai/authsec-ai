import { describe, it, expect } from "vitest";
import { convertExternalServicesToResources } from "./external-resource-utils";
import { isExternalServiceResource, getExternalServiceIdFromResource } from "./external-role-utils";
import type { ExternalService } from "@/types/entities";

describe("External Resource Utils", () => {
  const mockExternalService: ExternalService = {
    id: "google-workspace-1",
    name: "Google Workspace",
    provider: "google",
    category: "productivity",
    clientCount: 5,
    userTokenCount: 25,
    status: "connected",
    lastSync: "2024-01-15T10:00:00Z",
    createdAt: "2024-01-01T00:00:00Z",
  };

  describe("convertExternalServicesToResources", () => {
    it("should convert connected external services to resources", () => {
      const services = [mockExternalService];
      const resources = convertExternalServicesToResources(services);

      expect(resources).toHaveLength(1);
      expect(resources[0]).toMatchObject({
        id: "ext-google-workspace-1",
        name: "Google Workspace",
        description: "External google service",
        clientId: "external",
        clientName: "External services and secrets management",
        type: "external",
        isActive: true,
        isExternal: true,
        externalServiceId: "google-workspace-1",
        externalServiceName: "Google Workspace",
      });
    });

    it("should filter out non-connected services", () => {
      const services = [
        { ...mockExternalService, status: "needs_consent" as const },
        { ...mockExternalService, id: "service-2", status: "connected" as const },
      ];
      const resources = convertExternalServicesToResources(services);

      expect(resources).toHaveLength(1);
      expect(resources[0].id).toBe("ext-service-2");
    });

    it("should generate provider-specific scopes", () => {
      const services = [mockExternalService];
      const resources = convertExternalServicesToResources(services);

      expect(resources[0].scopes.length).toBe(4);

      const scopeNames = resources[0].scopes.map((s) => s.name);
      expect(scopeNames).toContain("google:read");
      expect(scopeNames).toContain("google:drive");
      expect(scopeNames).toContain("google:sheets");
      expect(scopeNames).toContain("google:write");
    });
  });

  describe("isExternalServiceResource", () => {
    it("should identify external service resources", () => {
      expect(isExternalServiceResource("ext-google-workspace-1")).toBe(true);
      expect(isExternalServiceResource("internal-resource-1")).toBe(false);
    });
  });

  describe("getExternalServiceIdFromResource", () => {
    it("should extract service ID from external resource ID", () => {
      expect(getExternalServiceIdFromResource("ext-google-workspace-1")).toBe("google-workspace-1");
      expect(getExternalServiceIdFromResource("internal-resource-1")).toBe(null);
    });
  });
});
