import React, { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../../../components/ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../../../components/ui/tabs";
import { Badge } from "../../../components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../../../components/ui/table";
import { Activity, Clock, User, Wrench } from "lucide-react";
import type { SessionData } from "../../../app/api/dashboardApi";
import {
  formatRelativeTime,
  calculateSessionDuration,
  formatDuration,
  parseAccessibleTools,
} from "../utils/dashboard-utils";

interface SessionsTableProps {
  activeSessions: SessionData[];
  inactiveSessions: SessionData[];
  isLoading?: boolean;
}

export function SessionsTable({
  activeSessions,
  inactiveSessions,
  isLoading = false,
}: SessionsTableProps) {
  const [searchTerm, setSearchTerm] = useState("");

  const filterSessions = (sessions: SessionData[]) => {
    if (!searchTerm) return sessions;
    const term = searchTerm.toLowerCase();
    return sessions.filter(
      (session) =>
        session.user_email.toLowerCase().includes(term) ||
        session.client_id.toLowerCase().includes(term) ||
        session.provider?.toLowerCase().includes(term)
    );
  };

  const renderSessionRow = (session: SessionData) => {
    const duration = calculateSessionDuration(session);
    const tools = parseAccessibleTools(session);

    return (
      <TableRow key={session.session_id}>
        <TableCell>
          <div className="flex items-center gap-2">
            <User className="h-4 w-4 text-foreground" />
            <div>
              <p className="font-medium">{session.user_email}</p>
              <p className="text-xs text-foreground">{session.session_id.slice(0, 8)}</p>
            </div>
          </div>
        </TableCell>
        <TableCell>
          <Badge
            variant={session.is_active ? "default" : "secondary"}
            className={session.is_active ? "bg-green-500" : ""}
          >
            {session.is_active ? "Active" : "Inactive"}
          </Badge>
        </TableCell>
        <TableCell>
          <div className="flex items-center gap-1 text-sm">
            <Clock className="h-3 w-3 text-foreground" />
            {formatDuration(duration)}
          </div>
        </TableCell>
        <TableCell>
          <p className="text-sm">{formatRelativeTime(session.last_activity)}</p>
        </TableCell>
        <TableCell>
          <Badge variant="outline">{session.provider || "Unknown"}</Badge>
        </TableCell>
        <TableCell>
          <div className="flex items-center gap-1">
            <Wrench className="h-3 w-3 text-foreground" />
            <span className="text-sm">{tools.length}</span>
          </div>
        </TableCell>
      </TableRow>
    );
  };

  const LoadingSkeleton = () => (
    <TableRow>
      {[1, 2, 3, 4, 5, 6].map((i) => (
        <TableCell key={i}>
          <div className="h-6 bg-muted animate-pulse rounded" />
        </TableCell>
      ))}
    </TableRow>
  );

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2">
            <Activity className="h-5 w-5" />
            User Sessions
          </CardTitle>
          <input
            type="text"
            placeholder="Search sessions..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="px-3 py-1.5 text-sm border border-border rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-primary/20"
          />
        </div>
      </CardHeader>
      <CardContent>
        <Tabs defaultValue="active">
          <TabsList className="mb-4">
            <TabsTrigger value="active" className="flex items-center gap-2">
              Active
              <Badge variant="secondary" className="ml-1">
                {activeSessions.length}
              </Badge>
            </TabsTrigger>
            <TabsTrigger value="inactive" className="flex items-center gap-2">
              Inactive
              <Badge variant="secondary" className="ml-1">
                {inactiveSessions.length}
              </Badge>
            </TabsTrigger>
          </TabsList>

          <TabsContent value="active">
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>User</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Duration</TableHead>
                    <TableHead>Last Activity</TableHead>
                    <TableHead>Provider</TableHead>
                    <TableHead>Tools</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {isLoading ? (
                    <>
                      <LoadingSkeleton />
                      <LoadingSkeleton />
                      <LoadingSkeleton />
                    </>
                  ) : filterSessions(activeSessions).length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={6} className="text-center py-8 text-foreground">
                        {searchTerm ? "No matching active sessions" : "No active sessions"}
                      </TableCell>
                    </TableRow>
                  ) : (
                    filterSessions(activeSessions).map(renderSessionRow)
                  )}
                </TableBody>
              </Table>
            </div>
          </TabsContent>

          <TabsContent value="inactive">
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>User</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Duration</TableHead>
                    <TableHead>Last Activity</TableHead>
                    <TableHead>Provider</TableHead>
                    <TableHead>Tools</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {isLoading ? (
                    <>
                      <LoadingSkeleton />
                      <LoadingSkeleton />
                      <LoadingSkeleton />
                    </>
                  ) : filterSessions(inactiveSessions).length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={6} className="text-center py-8 text-foreground">
                        {searchTerm ? "No matching inactive sessions" : "No inactive sessions"}
                      </TableCell>
                    </TableRow>
                  ) : (
                    filterSessions(inactiveSessions)
                      .slice(0, 10)
                      .map(renderSessionRow)
                  )}
                </TableBody>
              </Table>
            </div>
            {!isLoading && inactiveSessions.length > 10 && (
              <p className="text-sm text-foreground mt-3 text-center">
                Showing 10 of {inactiveSessions.length} inactive sessions
              </p>
            )}
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  );
}
