type ErrorData = {
  message?: unknown;
};

type ErrorLike = {
  data?: ErrorData;
  message?: unknown;
  error?: unknown;
  name?: unknown;
};

function asErrorLike(error: unknown): ErrorLike | null {
  if (!error || typeof error !== "object") {
    return null;
  }

  return error as ErrorLike;
}

export function getErrorMessage(error: unknown, fallback: string) {
  const errorLike = asErrorLike(error);
  const dataMessage = errorLike?.data?.message;
  if (typeof dataMessage === "string" && dataMessage.trim()) {
    return dataMessage;
  }

  if (typeof errorLike?.message === "string" && errorLike.message.trim()) {
    return errorLike.message;
  }

  if (typeof errorLike?.error === "string" && errorLike.error.trim()) {
    return errorLike.error;
  }

  return fallback;
}

export function getErrorName(error: unknown) {
  const errorLike = asErrorLike(error);
  return typeof errorLike?.name === "string" ? errorLike.name : undefined;
}
