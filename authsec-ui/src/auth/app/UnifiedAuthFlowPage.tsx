import { Navigate, useLocation } from "react-router-dom";
import { resolveAuthRouteDescriptor } from "./auth-route-registry";
import { AuthSceneRenderer } from "./AuthSceneRenderer";

export function UnifiedAuthFlowPage() {
  const location = useLocation();
  const descriptor = resolveAuthRouteDescriptor(location.pathname);

  if (!descriptor) {
    return <Navigate to="/admin/login" replace />;
  }

  return (
    <main
      data-auth-entrypoint={descriptor.entrypoint}
      data-auth-scene-id={descriptor.sceneId}
    >
      <AuthSceneRenderer descriptor={descriptor} />
    </main>
  );
}

export default UnifiedAuthFlowPage;
