import React from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../../../components/ui/card";
import { Shield, ShieldCheck, ShieldAlert, ShieldOff } from "lucide-react";
import type { MFADistribution } from "../../../app/api/dashboardApi";
import { getMFAColor } from "../utils/dashboard-utils";

interface MFADistributionChartProps {
  data: MFADistribution[];
  isLoading?: boolean;
}

const getMFAIcon = (method: string) => {
  if (method.includes("No MFA")) return <ShieldOff className="h-4 w-4" />;
  if (method.includes("Multiple")) return <ShieldCheck className="h-4 w-4" />;
  if (method.includes("WebAuthn")) return <Shield className="h-4 w-4" />;
  return <ShieldAlert className="h-4 w-4" />;
};

export function MFADistributionChart({
  data,
  isLoading = false,
}: MFADistributionChartProps) {
  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shield className="h-5 w-5" />
            MFA Adoption
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {[1, 2, 3].map((i) => (
              <div key={i} className="space-y-2">
                <div className="h-4 w-40 bg-muted animate-pulse rounded" />
                <div className="h-6 bg-muted animate-pulse rounded" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  if (!data || data.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shield className="h-5 w-5" />
            MFA Adoption
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center h-64 text-foreground">
            No MFA data available
          </div>
        </CardContent>
      </Card>
    );
  }

  const totalUsers = data.reduce((sum, item) => sum + item.count, 0);
  const usersWithMFA = data
    .filter((item) => !item.method.includes("No MFA"))
    .reduce((sum, item) => sum + item.count, 0);
  const adoptionRate = totalUsers > 0 ? (usersWithMFA / totalUsers) * 100 : 0;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Shield className="h-5 w-5" />
          MFA Adoption
        </CardTitle>
        <p className="text-sm text-foreground">
          Multi-factor authentication status
        </p>
      </CardHeader>
      <CardContent>
        {/* Overall adoption rate */}
        <div className="mb-6 p-4 rounded-lg bg-primary/5 border border-primary/20">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium">Overall MFA Adoption</span>
            <span className="text-2xl font-bold">{adoptionRate.toFixed(1)}%</span>
          </div>
          <div className="relative h-3 w-full overflow-hidden rounded-full bg-muted">
            <div
              className="h-full rounded-full bg-gradient-to-r from-green-500 to-emerald-500 transition-all duration-500"
              style={{ width: `${adoptionRate}%` }}
            />
          </div>
          <p className="text-xs text-foreground mt-2">
            {usersWithMFA} of {totalUsers} users have MFA enabled
          </p>
        </div>

        {/* MFA methods breakdown */}
        <div className="space-y-3">
          {data.map((item) => (
            <div key={item.method} className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <div className="flex items-center gap-2">
                  <div
                    className="p-1 rounded"
                    style={{
                      backgroundColor: `${getMFAColor(item.method)}20`,
                      color: getMFAColor(item.method),
                    }}
                  >
                    {getMFAIcon(item.method)}
                  </div>
                  <span className="font-medium">{item.method}</span>
                </div>
                <div className="flex items-center gap-3">
                  <span className="text-foreground">{item.count} users</span>
                  <span className="font-semibold">{item.percentage.toFixed(1)}%</span>
                </div>
              </div>
              <div className="relative h-2 w-full overflow-hidden rounded-full bg-muted">
                <div
                  className="h-full rounded-full transition-all duration-500"
                  style={{
                    width: `${item.percentage}%`,
                    backgroundColor: getMFAColor(item.method),
                  }}
                />
              </div>
            </div>
          ))}
        </div>

        
      </CardContent>
    </Card>
  );
}
