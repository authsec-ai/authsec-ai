// src/types/window.d.ts
export {};

declare global {
  interface Window {
    ENV?: {
      VITE_API_URL: string;
      VITE_AMPLITUDE_API_KEY: string;
      VITE_HUBSPOT_ACCESS_TOKEN: string;
    };
  }
}