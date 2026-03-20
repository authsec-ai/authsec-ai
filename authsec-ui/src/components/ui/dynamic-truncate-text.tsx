import React, { useRef, useEffect, useState } from "react";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "./tooltip";

interface DynamicTruncateTextProps {
  text: string;
  className?: string;
  showTooltip?: boolean;
}

/**
 * A component that dynamically truncates text based on available width.
 * Uses ResizeObserver to detect container size changes and adjusts truncation accordingly.
 */
export function DynamicTruncateText({
  text,
  className = "",
  showTooltip = true,
}: DynamicTruncateTextProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [displayText, setDisplayText] = useState(text);
  const [isTruncated, setIsTruncated] = useState(false);

  useEffect(() => {
    const updateTruncation = () => {
      if (!containerRef.current) return;

      const container = containerRef.current;
      const containerWidth = container.offsetWidth;

      // Create a temporary element to measure text width
      const tempElement = document.createElement("span");
      tempElement.style.visibility = "hidden";
      tempElement.style.position = "absolute";
      tempElement.style.whiteSpace = "nowrap";
      tempElement.className = className;
      document.body.appendChild(tempElement);

      // Check if full text fits
      tempElement.textContent = text;
      const fullTextWidth = tempElement.offsetWidth;

      if (fullTextWidth <= containerWidth) {
        setDisplayText(text);
        setIsTruncated(false);
        document.body.removeChild(tempElement);
        return;
      }

      // Binary search for the right truncation point
      let left = 0;
      let right = text.length;
      let bestFit = 0;

      while (left <= right) {
        const mid = Math.floor((left + right) / 2);
        const truncated = text.slice(0, mid) + "...";
        tempElement.textContent = truncated;

        if (tempElement.offsetWidth <= containerWidth) {
          bestFit = mid;
          left = mid + 1;
        } else {
          right = mid - 1;
        }
      }

      const finalText = bestFit > 0 ? text.slice(0, bestFit) + "..." : "...";
      setDisplayText(finalText);
      setIsTruncated(finalText !== text);
      document.body.removeChild(tempElement);
    };

    // Initial truncation
    updateTruncation();

    // Watch for size changes
    const resizeObserver = new ResizeObserver(updateTruncation);
    if (containerRef.current) {
      resizeObserver.observe(containerRef.current);
    }

    return () => {
      resizeObserver.disconnect();
    };
  }, [text, className]);

  const content = (
    <div
      ref={containerRef}
      className={`overflow-hidden ${className}`}
      style={{ width: "100%" }}
    >
      {displayText}
    </div>
  );

  if (showTooltip && isTruncated) {
    return (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>{content}</TooltipTrigger>
          <TooltipContent>
            <p className="max-w-xs break-words">{text}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
  }

  return content;
}
