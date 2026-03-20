import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from "react";

export type RbacAudience = "admin" | "endUser";

interface RbacAudienceContextValue {
  audience: RbacAudience;
  isAdmin: boolean;
  setAudience: (next: RbacAudience) => void;
  toggleAudience: () => void;
}

const RbacAudienceContext = createContext<RbacAudienceContextValue | undefined>(
  undefined
);

export function RbacAudienceProvider({ children }: { children: ReactNode }) {
  const [audience, setAudience] = useState<RbacAudience>("admin");

  const toggleAudience = useCallback(() => {
    setAudience((prev) => (prev === "admin" ? "endUser" : "admin"));
  }, []);

  const setAudienceValue = useCallback((next: RbacAudience) => {
    setAudience(next);
  }, []);

  const value = useMemo<RbacAudienceContextValue>(
    () => ({
      audience,
      isAdmin: audience === "admin",
      setAudience: setAudienceValue,
      toggleAudience,
    }),
    [audience, setAudienceValue, toggleAudience]
  );

  return (
    <RbacAudienceContext.Provider value={value}>
      {children}
    </RbacAudienceContext.Provider>
  );
}

export function useRbacAudience() {
  const context = useContext(RbacAudienceContext);
  if (!context) {
    throw new Error(
      "useRbacAudience must be used within a RbacAudienceProvider"
    );
  }
  return context;
}
