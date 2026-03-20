import React, { useState, useEffect } from "react";
import { Link, useNavigate, useLocation } from "react-router-dom";
import { useRegisterVerifyMutation, useResendOtpMutation } from "../../app/api/authApi";
import { Button } from "../../components/ui/button";
import { OTPInput } from "../../components/ui/otp-input";
import { Label } from "../../components/ui/label";
import { IconShield, IconClock } from "@tabler/icons-react";
import { toast } from "react-hot-toast";
import { AuthSplitFrame } from "../components/AuthSplitFrame";
import { AuthValuePanel } from "../components/AuthValuePanel";
import { AuthActionPanel } from "../components/AuthActionPanel";
import { AuthStepHeader } from "../components/AuthStepHeader";

interface LocationState {
  email?: string;
  tenantDomain?: string;
}

export const OTPVerificationPage: React.FC = () => {
  const [otp, setOtp] = useState("");
  const [isVerifying, setIsVerifying] = useState(false);
  const [isResending, setIsResending] = useState(false);
  const [timeLeft, setTimeLeft] = useState(60);
  const [canResend, setCanResend] = useState(false);

  const navigate = useNavigate();
  const location = useLocation();

  const state = location.state as LocationState;
  const email = state?.email;
  
  const [verifyOtp] = useRegisterVerifyMutation();
  const [resendOtpMutation] = useResendOtpMutation();

  // Redirect if no email in state
  useEffect(() => {
    if (!email) {
      navigate("/admin/login", { replace: true });
    }
  }, [email, navigate]);

  // Countdown timer
  useEffect(() => {
    if (timeLeft > 0 && !canResend) {
      const timer = setTimeout(() => setTimeLeft(timeLeft - 1), 1000);
      return () => clearTimeout(timer);
    } else if (timeLeft === 0) {
      setCanResend(true);
    }
  }, [timeLeft, canResend]);

  const handleVerifyOtp = async () => {
    if (!email || otp.length !== 6) return;
    
    setIsVerifying(true);
    try {
      const result = await verifyOtp({ email, otp });
      
      if ('data' in result) {
        toast.success("Account verified! Please login to continue.");
        const tenantDomain = location.state?.tenantDomain;
        if (tenantDomain) {
          window.location.href = `https://${tenantDomain}.app.authsec.dev/admin/login`;
        } else {
          navigate("/admin/login", { state: { email, verificationComplete: true } });
        }
      } else if ('error' in result) {
        const error = result.error as any;
        toast.error(error.data?.message || "Invalid OTP. Please try again.");
        setOtp(""); // Clear OTP on error
      }
    } catch (error) {
      console.error("OTP verification error:", error);
      toast.error("Failed to verify OTP");
      setOtp("");
    } finally {
      setIsVerifying(false);
    }
  };

  const handleResendOtp = async () => {
    if (!email || isResending) return;
    
    setIsResending(true);
    try {
      const result = await resendOtpMutation({ email });
      
      if ('data' in result) {
        toast.success("OTP sent successfully!");
        setTimeLeft(60);
        setCanResend(false);
        setOtp("");
      } else if ('error' in result) {
        const error = result.error as any;
        toast.error(error.data?.message || "Failed to resend OTP");
      }
    } catch (error) {
      console.error("Resend OTP error:", error);
      toast.error("Failed to resend OTP");
    } finally {
      setIsResending(false);
    }
  };

  const handleOtpComplete = (value: string) => {
    setOtp(value);
    // Auto-verify when OTP is complete
    if (value.length === 6) {
      setTimeout(() => handleVerifyOtp(), 100);
    }
  };

  if (!email) {
    return null; // Will redirect in useEffect
  }

  return (
    <AuthSplitFrame
      valuePanel={
        <AuthValuePanel
          eyebrow="Email Verification"
          title="Confirm your email before secure admin access."
          subtitle="A one-time verification code has been sent to your inbox."
          points={[
            `Verification email sent to ${email}.`,
            "Code expires quickly for stronger account protection.",
            "You can resend if delivery is delayed.",
          ]}
          trustLabel="Troubleshooting"
          trustItems={
            <div className="space-y-2 text-sm text-slate-600">
              <p>Check spam and promotions folders.</p>
              <p>Make sure the email address is correct.</p>
              <p>Wait a minute before requesting a new code.</p>
            </div>
          }
        />
      }
    >
      <AuthActionPanel className="space-y-6">
        <div className="flex items-center gap-3">
          <IconShield className="h-8 w-8 text-slate-900" />
          <span className="text-sm font-medium text-slate-600">
            AuthSec Verification
          </span>
        </div>

        <AuthStepHeader
          title="Verify Your Email"
          subtitle={
            <>
              We&apos;ve sent a 6-digit code to <strong>{email}</strong>
            </>
          }
        />

        <div className="space-y-5">
          <div className="space-y-4">
            <Label htmlFor="otp" className="text-center block">
              Verification Code
            </Label>
            <OTPInput
              value={otp}
              onChange={setOtp}
              onComplete={handleOtpComplete}
              disabled={isVerifying}
              length={6}
            />
          </div>

          <div className="text-center space-y-2">
            {!canResend ? (
              <div className="flex items-center justify-center gap-2 text-sm text-slate-600">
                <IconClock className="h-4 w-4" />
                <span>Resend code in {timeLeft}s</span>
              </div>
            ) : (
              <button
                type="button"
                onClick={handleResendOtp}
                disabled={isResending}
                className="text-sm font-medium text-slate-700 hover:text-slate-950"
              >
                {isResending ? "Sending..." : "Resend code"}
              </button>
            )}
          </div>

          <Button
            onClick={handleVerifyOtp}
            className="w-full h-11 rounded-xl"
            disabled={isVerifying || otp.length !== 6}
          >
            {isVerifying ? (
              <div className="flex items-center">
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                Verifying...
              </div>
            ) : (
              "Verify Account"
            )}
          </Button>

          <div className="text-center text-sm">
            <span className="text-slate-600">Wrong email? </span>
            <Link
              to="/admin/login"
              className="font-medium text-slate-900 underline underline-offset-2"
            >
              Back to login
            </Link>
          </div>
        </div>
      </AuthActionPanel>
    </AuthSplitFrame>
  );
};
