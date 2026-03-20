import { useEffect, useRef, useState } from "react";
import confetti from "canvas-confetti";
import { DashboardHero } from "./DashboardHero";

interface ActivationSectionProps {
  step1Done: boolean;
  step2Done: boolean;
  onStart: () => void;
  isWizardActive: boolean;
}

export function ActivationSection({
  step1Done,
  step2Done,
  onStart,
  isWizardActive,
}: ActivationSectionProps) {
  const DISMISS_KEY = "activationCard_dismissed";
  const [isDismissed, setIsDismissed] = useState<boolean>(() => {
    try {
      return localStorage.getItem(DISMISS_KEY) === "true";
    } catch {
      return false;
    }
  });

  const isComplete = step1Done && step2Done;

  const handleDismiss = () => {
    setIsDismissed(true);
    try {
      localStorage.setItem(DISMISS_KEY, "true");
    } catch {}
  };
  const CONFETTI_KEY = "activationCard_confettiFired";
  const hasFiredConfetti = useRef<boolean>(
    (() => {
      try {
        return sessionStorage.getItem(CONFETTI_KEY) === "true";
      } catch {
        return false;
      }
    })()
  );

  useEffect(() => {
    if (isComplete && !isDismissed && !hasFiredConfetti.current) {
      hasFiredConfetti.current = true;
      try {
        sessionStorage.setItem(CONFETTI_KEY, "true");
      } catch {}

      const defaults = {
        spread: 55,
        ticks: 100,
        gravity: 1,
        decay: 0.94,
        startVelocity: 30,
        particleCount: 80,
        scalar: 1,
      };

      requestAnimationFrame(() => {
        confetti({ ...defaults, angle: 60, origin: { x: 0, y: 0.6 } });
        confetti({ ...defaults, angle: 120, origin: { x: 1, y: 0.6 } });
      });
    } else if (!isComplete) {
      hasFiredConfetti.current = false;
      try {
        sessionStorage.removeItem(CONFETTI_KEY);
      } catch {}
    }
  }, [isComplete, isDismissed]);

  if (isDismissed) return null;

  const stepsCompleted = (step1Done ? 1 : 0) + (step2Done ? 1 : 0);

  const steps = [
    {
      number: 1,
      title: "Create a client & configure auth",
      description:
        "Set up your application client and choose an authentication provider",
      done: step1Done,
    },
    {
      number: 2,
      title: "Integrate the SDK",
      description:
        "Add the AuthSec SDK to your app — users will see a login box",
      done: step2Done,
    },
  ];

  return (
    <DashboardHero
      title="Get Started"
      subtitle="Two essential steps to a production-ready authentication flow."
      steps={steps.map((step) => ({
        id: step.number,
        title: step.title,
        description: step.description,
        done: step.done,
      }))}
      completed={stepsCompleted}
      total={2}
      complete={isComplete}
      completeTitle="Authentication is live"
      completeDescription="Your app is connected and users can sign in."
      primaryActionLabel={
        isWizardActive
          ? "Setup in progress..."
          : stepsCompleted > 0
            ? "Continue Setup"
            : "Start Setup"
      }
      onPrimaryAction={onStart}
      primaryActionDisabled={isWizardActive}
      secondaryActionLabel={isComplete ? "Dismiss" : undefined}
      onSecondaryAction={isComplete ? handleDismiss : undefined}
      onDismiss={handleDismiss}
    />
  );
}
