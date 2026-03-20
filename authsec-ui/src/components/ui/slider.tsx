import * as React from "react";
import { cn } from "@/lib/utils";

interface SliderProps {
  value?: number[];
  onValueChange?: (value: number[]) => void;
  max?: number;
  min?: number;
  step?: number;
  className?: string;
}

const Slider = React.forwardRef<HTMLDivElement, SliderProps>(
  ({ className, value = [0, 100], onValueChange, max = 100, min = 0, step = 1, ...props }, ref) => {
    const [minValue, setMinValue] = React.useState(value[0] || min);
    const [maxValue, setMaxValue] = React.useState(value[1] || max);

    React.useEffect(() => {
      setMinValue(value[0] || min);
      setMaxValue(value[1] || max);
    }, [value, min, max]);

    const handleMinChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const newMin = parseInt(e.target.value, 10);
      if (newMin <= maxValue) {
        setMinValue(newMin);
        onValueChange?.([newMin, maxValue]);
      }
    };

    const handleMaxChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const newMax = parseInt(e.target.value, 10);
      if (newMax >= minValue) {
        setMaxValue(newMax);
        onValueChange?.([minValue, newMax]);
      }
    };

    return (
      <div ref={ref} className={cn("relative w-full", className)}>
        <input
          type="range"
          min={min}
          max={max}
          step={step}
          value={minValue}
          onChange={handleMinChange}
          className="absolute w-full h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer dark:bg-gray-700 z-10"
        />
        <input
          type="range"
          min={min}
          max={max}
          step={step}
          value={maxValue}
          onChange={handleMaxChange}
          className="absolute w-full h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer dark:bg-gray-700 z-20"
        />
      </div>
    );
  }
);
Slider.displayName = "Slider";

export { Slider };
