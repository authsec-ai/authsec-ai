import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Label } from "../../components/ui/label";
import { Textarea } from "../../components/ui/textarea";
import { IconShield, IconUsers, IconBuilding, IconRocket } from "@tabler/icons-react";
import { AuthSplitFrame } from "../components/AuthSplitFrame";
import { AuthValuePanel } from "../components/AuthValuePanel";
import { AuthActionPanel } from "../components/AuthActionPanel";
import { AuthStepHeader } from "../components/AuthStepHeader";

export const CreateWorkspacePage: React.FC = () => {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const { createProject, user } = useAuth();
  const navigate = useNavigate();

  const handleCreateWorkspace = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);

    const success = await createProject(name, description);
    if (success) {
      navigate("/dashboard");
    }

    setIsLoading(false);
  };

  const steps = [
    {
      icon: IconUsers,
      title: "Create Workspace",
      description: "Set up your team workspace",
      active: true,
    },
    {
      icon: IconBuilding,
      title: "Configure Settings",
      description: "Customize security policies",
      active: false,
    },
    {
      icon: IconRocket,
      title: "Start Securing",
      description: "Begin managing identities",
      active: false,
    },
  ];

  const features = [
    "Unlimited team members",
    "Advanced security policies",
    "Real-time monitoring",
    "Audit logs & compliance",
    "SSO integrations",
    "24/7 support",
  ];

  return (
    <AuthSplitFrame
      valuePanel={
        <AuthValuePanel
          eyebrow="Workspace Setup"
          title="Create your security workspace."
          subtitle="Initialize your admin tenant with core controls, monitoring, and policy-ready defaults."
          points={features}
          trustLabel="Activation Journey"
          trustItems={
            <div className="space-y-3">
              {steps.map((step) => (
                <div key={step.title} className="auth-callout flex items-center gap-3">
                  <step.icon className="h-4 w-4 text-slate-700" />
                  <div>
                    <p className="text-sm font-medium text-slate-900">{step.title}</p>
                    <p className="text-xs text-slate-600">{step.description}</p>
                  </div>
                </div>
              ))}
            </div>
          }
        />
      }
    >
      <AuthActionPanel className="space-y-6">
        <div className="flex items-center gap-3">
          <IconShield className="h-8 w-8 text-slate-900" />
          <span className="text-sm font-medium text-slate-600">
            AuthSec Workspace Provisioning
          </span>
        </div>

        <AuthStepHeader
          title="Create your workspace"
          subtitle="Set up your team's security workspace"
        />

        <form onSubmit={handleCreateWorkspace} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Workspace Name *</Label>
            <Input
              id="name"
              type="text"
              placeholder="e.g., Development Team"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              disabled={isLoading}
              className="h-11 rounded-xl"
            />
            <p className="text-xs text-slate-600">
              This will be the name of your team workspace.
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="description">Description (Optional)</Label>
            <Textarea
              id="description"
              placeholder="Brief description of your workspace..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              disabled={isLoading}
              rows={3}
              className="rounded-xl"
            />
          </div>

          <div className="auth-inline-note space-y-2">
            <div className="flex items-center gap-2">
              <IconUsers className="h-4 w-4 text-slate-700" />
              <p className="text-sm font-medium text-slate-900">Workspace Admin</p>
            </div>
            <p className="text-sm text-slate-700">{user?.email} (You)</p>
          </div>

          <Button
            type="submit"
            className="w-full h-11 rounded-xl"
            disabled={isLoading || !name.trim()}
          >
            {isLoading ? (
              <div className="flex items-center">
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                Creating workspace...
              </div>
            ) : (
              "Create Workspace"
            )}
          </Button>
        </form>
      </AuthActionPanel>
    </AuthSplitFrame>
  );
};
