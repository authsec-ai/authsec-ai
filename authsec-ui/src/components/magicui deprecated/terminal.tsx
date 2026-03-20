import React, { useState, useEffect } from "react";
import { motion } from "framer-motion";
import { cn } from "@/lib/utils";

interface TerminalProps {
  children: React.ReactNode;
  className?: string;
}

export function Terminal({ children, className }: TerminalProps) {
  return (
    <div
      className={cn(
        "relative overflow-hidden rounded-lg border bg-black text-white shadow-2xl",
        className
      )}
    >
      {/* Terminal Header */}
      <div className="flex h-8 items-center gap-2 border-b border-gray-800 bg-gray-900 px-4">
        <div className="flex gap-1.5">
          <div className="h-2.5 w-2.5 rounded-full bg-red-500" />
          <div className="h-2.5 w-2.5 rounded-full bg-yellow-500" />
          <div className="h-2.5 w-2.5 rounded-full bg-green-500" />
        </div>
        <div className="flex-1 text-center text-xs text-gray-400">
          Terminal
        </div>
      </div>
      
      {/* Terminal Content */}
      <div className="min-h-[300px] p-4 font-mono text-sm">
        {children}
      </div>
    </div>
  );
}

interface TypingAnimationProps {
  children: React.ReactNode;
  speed?: number;
  delay?: number;
  className?: string;
}

export function TypingAnimation({ 
  children, 
  speed = 50, 
  delay = 0,
  className 
}: TypingAnimationProps) {
  const [displayedText, setDisplayedText] = useState("");
  const text = typeof children === "string" ? children : "";

  useEffect(() => {
    const timer = setTimeout(() => {
      let index = 0;
      const interval = setInterval(() => {
        if (index < text.length) {
          setDisplayedText(text.slice(0, index + 1));
          index++;
        } else {
          clearInterval(interval);
        }
      }, speed);

      return () => clearInterval(interval);
    }, delay);

    return () => clearTimeout(timer);
  }, [text, speed, delay]);

  return (
    <span className={cn("inline-block", className)}>
      {displayedText}
      <motion.span
        animate={{ opacity: [1, 0] }}
        transition={{ duration: 0.8, repeat: Infinity, repeatType: "reverse" }}
        className="inline-block"
      >
        |
      </motion.span>
    </span>
  );
}

interface AnimatedSpanProps {
  children: React.ReactNode;
  delay?: number;
  className?: string;
}

export function AnimatedSpan({ children, delay = 0, className }: AnimatedSpanProps) {
  return (
    <motion.span
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ delay, duration: 0.3 }}
      className={className}
    >
      {children}
    </motion.span>
  );
}

interface CodeLineProps {
  children: React.ReactNode;
  prefix?: string;
  delay?: number;
  className?: string;
}

export function CodeLine({ children, prefix = "$", delay = 0, className }: CodeLineProps) {
  return (
    <motion.div
      initial={{ opacity: 0, x: -10 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ delay, duration: 0.5 }}
      className={cn("flex items-start gap-2", className)}
    >
      <span className="text-green-400">{prefix}</span>
      <span className="flex-1">{children}</span>
    </motion.div>
  );
}

export function TerminalOutput({ children, delay = 0, className }: AnimatedSpanProps) {
  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ delay, duration: 0.3 }}
      className={cn("text-gray-300", className)}
    >
      {children}
    </motion.div>
  );
}