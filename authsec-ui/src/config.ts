// src/config.ts
declare global {
  interface Window {
    ENV?: {
      VITE_API_URL?: string;
    };
  }
}

const config = {
  VITE_API_URL:
    window.ENV?.VITE_API_URL ||
    import.meta.env.VITE_API_URL ||
    "http://localhost:3000",
};

export default config;
