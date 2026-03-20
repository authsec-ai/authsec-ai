import type { ReactNode } from "react";
import { useEffect, useState } from "react";
import { AppSidebar } from "./AppSidebar";
import { AppHeader } from "./AppHeader";
import { AppRightSidebar } from "./AppRightSidebar";
import { SidebarProvider } from "@/components/ui/sidebar";
import { ResponsiveSidebarController } from "./ResponsiveSidebarController";
import { VoiceAgentWatcher } from "@/features/voice-auth/VoiceAgentWatcher";
import { useWizard } from "@/contexts/WizardContext";
import "../../theme/admin-shell.css";

interface AppLayoutProps {
  children: ReactNode;
}

export function AppLayout({ children }: AppLayoutProps) {
  const [isRightSidebarOpen, setIsRightSidebarOpen] = useState(false);
  const [rightSidebarWidth, setRightSidebarWidth] = useState(520);
  const { isActive: isWizardActive, isCompleted: isWizardCompleted, resetCompletion, isAwaitingPlatformAction } = useWizard();

  const handleRightSidebarToggle = () => {
    setIsRightSidebarOpen(!isRightSidebarOpen);
  };

  const handleRightSidebarClose = () => {
    setIsRightSidebarOpen(false);
    // Reset completion state when closing
    if (isWizardCompleted) {
      resetCompletion();
    }
  };

  // Auto-open sidebar when wizard starts
  // Keep it open when completed, only close when dismissed (isActive=false && isCompleted=false)
  useEffect(() => {
    if (isWizardActive && !isRightSidebarOpen) {
      setIsRightSidebarOpen(true);
    } else if (!isWizardActive && !isWizardCompleted && isRightSidebarOpen) {
      // Close sidebar only when wizard is dismissed (not completed)
      setIsRightSidebarOpen(false);
    }
  }, [isWizardActive, isWizardCompleted, isRightSidebarOpen]);

  // LEGACY/OBSOLETE: SDK Manager route no longer exists
  // useEffect(() => {
  //   if (isRightSidebarOpen && location.pathname === "/sdk/manager") {
  //     setIsRightSidebarOpen(false);
  //   }
  // }, [isRightSidebarOpen, location.pathname]);

  return (
    <>
      <SidebarProvider
        defaultOpen={true}
        style={
          {
            "--sidebar-width": "calc(var(--spacing) * 72)",
            "--header-height": "calc(var(--spacing) * 16)",
            "--app-shell-surface": "var(--background)",
            "--app-shell-border": "var(--border)",
            "--sidebar-surface": "var(--app-shell-surface)",
            "--sidebar-border": "var(--app-shell-border)",
          } as React.CSSProperties
        }
      >
        <div
          data-ui-scope="admin-shell"
          className="h-screen w-screen flex overflow-hidden bg-background"
        >
          {/* Left Sidebar */}
          <div>
            <ResponsiveSidebarController
              isRightSidebarOpen={isRightSidebarOpen}
              rightSidebarWidth={rightSidebarWidth}
            />
            <AppSidebar />
          </div>

          {/* Main Content Area */}
          <div data-slot="admin-main-surface" className="flex-1 flex flex-col min-w-0">
            <AppHeader
              onRightSidebarToggle={handleRightSidebarToggle}
              isRightSidebarOpen={isRightSidebarOpen}
            />
            <div className="flex-1 overflow-hidden">
              <div
                className="h-full w-full overflow-y-auto scrollbar-hide"
                data-main-content-area="true"
              >
                {children}
              </div>
            </div>
          </div>

          {/* Right Sidebar */}
          {isRightSidebarOpen && (
            <AppRightSidebar
              mode={isWizardActive || isWizardCompleted ? "wizard" : "chat"}
              onClose={handleRightSidebarClose}
              onWidthChange={setRightSidebarWidth}
              initialWidth={rightSidebarWidth}
            />
          )}
        </div>
      </SidebarProvider>
      <VoiceAgentWatcher />
    </>
  );
}
