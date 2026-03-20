type XPixelCommand = "config" | "event";
type XPixelFn = {
  (...args: [XPixelCommand, string, Record<string, unknown>?]): void;
  exe?: (...args: [XPixelCommand, string, Record<string, unknown>?]) => void;
  queue?: Array<[XPixelCommand, string, Record<string, unknown>?]>;
  version?: string;
};

type XPixelWindow = Window & typeof globalThis & { twq?: XPixelFn };

const X_PIXEL_SCRIPT_ID = "x-pixel-base";
const X_PIXEL_URL = "https://static.ads-twitter.com/uwt.js";
const X_ADVERTISER_ID = "r1mw9";
const X_SIGNUP_COMPLETED_EVENT_ID = "tw-r1mw9-r1mwb";

function getTwq(): XPixelFn | undefined {
  return (window as XPixelWindow).twq;
}

export function initializeXPixel(): void {
  if (typeof window === "undefined" || typeof document === "undefined") {
    return;
  }

  const xWindow = window as XPixelWindow;

  if (!xWindow.twq) {
    const twq: XPixelFn = function (...args) {
      if (twq.exe) {
        twq.exe(...args);
        return;
      }
      twq.queue?.push(args);
    };

    twq.version = "1.1";
    twq.queue = [];
    xWindow.twq = twq;
  }

  if (!document.getElementById(X_PIXEL_SCRIPT_ID)) {
    const script = document.createElement("script");
    script.id = X_PIXEL_SCRIPT_ID;
    script.async = true;
    script.src = X_PIXEL_URL;
    document.head.appendChild(script);
  }

  xWindow.twq("config", X_ADVERTISER_ID);
}

export function trackXSignupCompleted(emailAddress?: string | null): void {
  if (typeof window === "undefined") {
    return;
  }

  if (!getTwq()) {
    initializeXPixel();
  }

  getTwq()?.("event", X_SIGNUP_COMPLETED_EVENT_ID, {
    email_address: emailAddress?.trim().toLowerCase() || null,
  });
}
