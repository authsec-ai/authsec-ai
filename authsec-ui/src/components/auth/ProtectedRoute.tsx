import React from "react";
import { Navigate, useLocation } from "react-router-dom";
import { useAuth } from "../../auth/context/AuthContext";

interface ProtectedRouteProps {
  children: React.ReactNode;
  requireProject?: boolean;
}

export const ProtectedRoute: React.FC<ProtectedRouteProps> = ({
  children,
}) => {
  const { isAuthenticated, isLoading } = useAuth();
  const location = useLocation();

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <div className="flex flex-col items-center space-y-4">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
          <p className="text-foreground">Loading...</p>
        </div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/admin/login" state={{ from: location }} replace />;
  }


  // Note: We don't check for missing projects here because:
  // 1. Users always get a default project when created
  // 2. completeWebAuthnAuthentication() creates a default project
  // 3. If there's ever a race condition, we let the component render and handle it gracefully

  return <>{children}</>;
};
