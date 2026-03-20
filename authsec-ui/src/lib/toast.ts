// Simple toast utility for notifications
// In a production app, you'd use a proper toast library like react-hot-toast or sonner

interface ToastOptions {
  duration?: number;
  position?: "top-right" | "top-left" | "bottom-right" | "bottom-left";
}

class ToastManager {
  private container: HTMLElement | null = null;
  private toastCounter = 0;

  private ensureContainer() {
    if (!this.container) {
      this.container = document.createElement("div");
      this.container.id = "toast-container";
      this.container.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        z-index: 9999;
        display: flex;
        flex-direction: column;
        gap: 8px;
        pointer-events: none;
      `;
      document.body.appendChild(this.container);
    }
    return this.container;
  }

  private createToast(
    message: string,
    type: "success" | "error" | "info",
    options: ToastOptions = {}
  ) {
    const container = this.ensureContainer();
    const toast = document.createElement("div");
    const id = `toast-${++this.toastCounter}`;

    toast.id = id;
    toast.style.cssText = `
      background-color: ${
        type === "success" ? "#16a34a" : type === "error" ? "#dc2626" : "#2563eb"
      };
      color: white;
      padding: 12px 16px;
      border-radius: 8px;
      box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1);
      font-size: 14px;
      font-weight: 500;
      max-width: 350px;
      pointer-events: auto;
      animation: slideIn 0.3s ease-out;
      opacity: 1;
      transform: translateX(0);
      transition: all 0.3s ease;
    `;

    toast.textContent = message;

    // Add CSS keyframes if not already added
    if (!document.getElementById("toast-styles")) {
      const style = document.createElement("style");
      style.id = "toast-styles";
      style.textContent = `
        @keyframes slideIn {
          from {
            opacity: 0;
            transform: translateX(100%);
          }
          to {
            opacity: 1;
            transform: translateX(0);
          }
        }
        @keyframes slideOut {
          from {
            opacity: 1;
            transform: translateX(0);
          }
          to {
            opacity: 0;
            transform: translateX(100%);
          }
        }
      `;
      document.head.appendChild(style);
    }

    container.appendChild(toast);

    // Auto remove after duration
    const duration = options.duration || 3000;
    setTimeout(() => {
      toast.style.animation = "slideOut 0.3s ease-in";
      setTimeout(() => {
        if (toast.parentNode) {
          toast.parentNode.removeChild(toast);
        }
      }, 300);
    }, duration);

    return id;
  }

  success(message: string, options?: ToastOptions) {
    return this.createToast(message, "success", options);
  }

  error(message: string, options?: ToastOptions) {
    return this.createToast(message, "error", options);
  }

  info(message: string, options?: ToastOptions) {
    return this.createToast(message, "info", options);
  }
}

export const toast = new ToastManager();
