export function TrustDelegationFieldMeta({
  helper,
  error,
}: {
  helper?: string;
  error?: string;
}) {
  if (error) {
    return <p className="text-xs text-destructive">{error}</p>;
  }

  if (helper) {
    return <p className="text-[11px] text-muted-foreground">{helper}</p>;
  }

  return null;
}
