import React from "react";
import { Activity, Users, UserCog, Clock } from "lucide-react";
import CountUp from "react-countup";
import { useInView } from "react-intersection-observer";

interface KPICardsProps {
  activeSessions: number;
  inactiveSessions: number;
  totalEndUsers: number;
  totalAdminUsers: number;
  isLoading?: boolean;
}

export function KPICards({
  activeSessions,
  inactiveSessions,
  totalEndUsers,
  totalAdminUsers,
  isLoading = false,
}: KPICardsProps) {
  const [ref, inView] = useInView({ threshold: 0.1, triggerOnce: true });

  const cards = [
    {
      title: "Active Sessions",
      value: activeSessions,
      icon: Activity,
      gradient: "from-blue-50 to-sky-50 dark:from-neutral-800/50 dark:to-neutral-700/30",
      iconBg: "bg-blue-500/10 dark:bg-blue-400/20",
      iconColor: "text-blue-600 dark:text-blue-400",
      decorBg: "bg-blue-500/5 dark:bg-neutral-600/20",
    },
    {
      title: "Total Users",
      value: totalEndUsers,
      icon: Users,
      gradient: "from-emerald-50 to-teal-50 dark:from-neutral-800/50 dark:to-neutral-700/30",
      iconBg: "bg-emerald-500/10 dark:bg-emerald-400/20",
      iconColor: "text-emerald-600 dark:text-emerald-400",
      decorBg: "bg-emerald-500/5 dark:bg-neutral-600/20",
    },
    {
      title: "Admins",
      value: totalAdminUsers,
      icon: UserCog,
      gradient: "from-amber-50 to-orange-50 dark:from-neutral-800/50 dark:to-neutral-700/30",
      iconBg: "bg-amber-500/10 dark:bg-amber-400/20",
      iconColor: "text-amber-600 dark:text-amber-400",
      decorBg: "bg-amber-500/5 dark:bg-neutral-600/20",
    },
    {
      title: "Inactive Sessions",
      value: inactiveSessions,
      icon: Clock,
      gradient: "from-rose-50 to-pink-50 dark:from-neutral-800/50 dark:to-neutral-700/30",
      iconBg: "bg-rose-500/10 dark:bg-rose-400/20",
      iconColor: "text-rose-600 dark:text-rose-400",
      decorBg: "bg-rose-500/5 dark:bg-neutral-600/20",
    },
  ];

  return (
    <div
      ref={ref}
      className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4"
    >
      {cards.map((card) => (
        <div
          key={card.title}
        >
          <div className="dash-panel dash-panel-interactive relative overflow-hidden p-6">
            <div className="flex items-center gap-4">
              <div className={`p-3 ${card.iconBg} rounded-[var(--dash-radius-md)] border border-[var(--dash-border-soft)]`}>
                <card.icon className={`h-6 w-6 ${card.iconColor}`} />
              </div>
              <div>
                <div className="text-4xl font-extrabold dash-text-1">
                  {inView && !isLoading && <CountUp end={card.value} duration={2} />}
                  {isLoading && <span className="animate-pulse">...</span>}
                </div>
                <div className="text-sm font-medium dash-text-2">
                  {card.title}
                </div>
              </div>
            </div>
            <div className={`absolute top-0 right-0 w-20 h-20 ${card.decorBg} rounded-full -mr-10 -mt-10 opacity-60`}></div>
          </div>
        </div>
      ))}
    </div>
  );
}
