interface DiscreteDeleteFailure {
  id: string;
  reason: unknown;
}

export interface DiscreteDeleteResult {
  successCount: number;
  failureCount: number;
  failures: DiscreteDeleteFailure[];
}

/**
 * Runs multiple delete operations (when the API only supports single-entity deletes)
 * and aggregates the result so callers can surface a consistent toast/UX.
 */
export async function performDiscreteDeletes(
  ids: string[],
  deleteFn: (id: string) => Promise<unknown>
): Promise<DiscreteDeleteResult> {
  const results = await Promise.allSettled(ids.map((id) => deleteFn(id)));

  const failures: DiscreteDeleteFailure[] = [];
  results.forEach((result, index) => {
    if (result.status === "rejected") {
      failures.push({ id: ids[index], reason: result.reason });
    }
  });

  return {
    successCount: ids.length - failures.length,
    failureCount: failures.length,
    failures,
  };
}
