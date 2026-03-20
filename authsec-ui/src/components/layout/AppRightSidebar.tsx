"use client";

import { useState, useRef, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  X,
  Send,
  Bot,
  User,
  Sparkles,
  Copy,
  ThumbsUp,
  ThumbsDown,
  GripVertical,
} from "lucide-react";
import { AppRightSidebarWizard } from "@/features/wizards";

interface Message {
  id: string;
  type: "user" | "assistant" | "system";
  content: string;
  timestamp: Date;
  status?: "sending" | "sent" | "error";
}

interface AppRightSidebarProps {
  onClose: () => void;
  onWidthChange?: (width: number) => void;
  initialWidth?: number;
  mode?: "chat" | "wizard";
}

export function AppRightSidebar({
  onClose,
  onWidthChange,
  initialWidth = 400,
  mode = "chat",
}: AppRightSidebarProps) {
  // If wizard mode, render wizard component
  if (mode === "wizard") {
    return (
      <AppRightSidebarWizard
        onClose={onClose}
        onWidthChange={onWidthChange}
        initialWidth={initialWidth}
      />
    );
  }

  // Otherwise render chat mode
  return (
    <AppRightSidebarChat
      onClose={onClose}
      onWidthChange={onWidthChange}
      initialWidth={initialWidth}
    />
  );
}

// Extract chat functionality into separate component
function AppRightSidebarChat({
  onClose,
  onWidthChange,
  initialWidth = 400,
}: AppRightSidebarProps) {
  const [messages, setMessages] = useState<Message[]>([
    {
      id: "1",
      type: "system",
      content:
        "👋 Hi! I'm your AI assistant. I can help you with:\n\n• Creating and managing agents\n• Setting up authentication methods\n• Managing roles and permissions\n• Analyzing logs and monitoring\n• General IAM questions\n\nWhat would you like to work on today?",
      timestamp: new Date(),
    },
  ]);
  const [inputValue, setInputValue] = useState("");
  const [isTyping, setIsTyping] = useState(false);
  const [width, setWidth] = useState(initialWidth);
  const [isResizing, setIsResizing] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      setIsResizing(true);

      const startX = e.clientX;
      const startWidth = width;

      const handleMouseMove = (e: MouseEvent) => {
        e.preventDefault();
        const deltaX = startX - e.clientX;
        const newWidth = Math.max(320, Math.min(800, startWidth + deltaX));
        setWidth(newWidth);
        onWidthChange?.(newWidth);
      };

      const handleMouseUp = () => {
        setIsResizing(false);
        document.body.style.cursor = "";
        document.body.style.userSelect = "";
        document.removeEventListener("mousemove", handleMouseMove);
        document.removeEventListener("mouseup", handleMouseUp);
      };

      document.body.style.cursor = "ew-resize";
      document.body.style.userSelect = "none";
      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp);
    },
    [width, onWidthChange]
  );

  const handleSendMessage = async () => {
    if (!inputValue.trim()) return;

    const userMessage: Message = {
      id: Date.now().toString(),
      type: "user",
      content: inputValue,
      timestamp: new Date(),
      status: "sent",
    };

    setMessages((prev) => [...prev, userMessage]);
    setInputValue("");
    setIsTyping(true);

    // Simulate AI response
    setTimeout(() => {
      const aiResponse: Message = {
        id: (Date.now() + 1).toString(),
        type: "assistant",
        content: getAIResponse(inputValue),
        timestamp: new Date(),
      };
      setMessages((prev) => [...prev, aiResponse]);
      setIsTyping(false);
    }, 1500);
  };

  const getAIResponse = (input: string): string => {
    const lowerInput = input.toLowerCase();

    if (lowerInput.includes("agent") || lowerInput.includes("create")) {
      return "I can help you create a new agent! Here's what we'll need to configure:\n\n1. **Agent Type** - Service account, user agent, or API key\n2. **Authentication** - OAuth, SAML, or certificate-based\n3. **Permissions** - Role assignments and access policies\n4. **Integration** - Service connections and endpoints\n\nWould you like me to guide you through the agent creation process?";
    }

    if (lowerInput.includes("security")) {
      return "I can help you with security configurations! Here are some common tasks:\n\n• **Authentication Methods** - MFA requirements and SSO setup\n• **Authorization Rules** - Role-based access controls\n• **Monitoring Settings** - Audit and compliance configuration\n• **Permission Management** - Resource and scope assignments\n\nWhat would you like to configure?";
    }

    if (lowerInput.includes("log") || lowerInput.includes("monitor")) {
      return "I can help you analyze your logs and monitoring data:\n\n📊 **Current Status:**\n• Authentication events: 1,247 today\n• Failed login attempts: 23\n• Access violations: 2\n• System alerts: 0\n\nWould you like me to show you specific log entries or help set up monitoring alerts?";
    }

    return `I understand you're asking about: "${input}"\n\nI can help you with various IAM tasks. Here are some quick actions:\n\n• Create new agents or services\n• Configure authentication methods\n• Manage roles and permissions\n• Review logs and analytics\n• Troubleshoot access issues\n\nCould you provide more details about what you'd like to accomplish?`;
  };

  const quickActions = [
    { label: "Create Agent", icon: "🤖" },
    { label: "Setup Auth", icon: "🔐" },
    { label: "Manage Roles", icon: "🛡️" },
    { label: "View Logs", icon: "📊" },
  ];

  return (
    <div
      className="h-full bg-background border-l border-border flex shadow-2xl"
      style={{ width: `${width}px` }}
    >
      {/* Resize Handle */}
      <div
        className="w-1 bg-border hover:bg-primary/50 cursor-ew-resize flex items-center justify-center group transition-all duration-200 relative"
        onMouseDown={handleMouseDown}
      >
        <div className="absolute inset-y-0 -left-2 -right-2 flex items-center justify-center">
          <GripVertical className="w-3 h-3 text-foreground group-hover:text-primary transition-colors" />
        </div>
        {isResizing && (
          <div className="absolute -top-8 left-1/2 transform -translate-x-1/2 bg-popover text-popover-foreground px-2 py-1 rounded text-xs shadow-lg z-50 border">
            {width}px
          </div>
        )}
      </div>

      {/* Chat Interface */}
      <div className="flex-1 flex flex-col bg-background">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
          <div className="flex items-center gap-3">
            <div className="flex items-center justify-center w-7 h-7 rounded-md bg-primary/10">
              <Bot className="h-4 w-4 text-primary" />
            </div>
            <div>
              <h3 className="font-medium text-sm">AI Assistant</h3>
              <p className="text-xs text-foreground">Always here to help</p>
            </div>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={onClose}
            className="h-7 w-7 p-0 hover:bg-muted rounded-md"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>

        {/* Messages */}
        <div className="flex-1 overflow-y-auto scrollbar-hide">
          <div className="p-4 space-y-4">
            {messages.map((message) => (
              <div
                key={message.id}
                className={`flex gap-3 ${
                  message.type === "user" ? "justify-end" : "justify-start"
                }`}
              >
                {message.type !== "user" && (
                  <div className="flex-shrink-0 w-6 h-6 rounded-md bg-primary/10 flex items-center justify-center mt-1">
                    {message.type === "system" ? (
                      <Sparkles className="h-3 w-3 text-primary" />
                    ) : (
                      <Bot className="h-3 w-3 text-primary" />
                    )}
                  </div>
                )}

                <div
                  className={`max-w-[80%] rounded-lg px-3 py-2 ${
                    message.type === "user"
                      ? "bg-primary text-primary-foreground"
                      : "bg-muted/50 border border-border"
                  }`}
                >
                  <div className="text-sm leading-relaxed whitespace-pre-wrap">
                    {message.content}
                  </div>

                  {message.type === "assistant" && (
                    <div className="flex items-center gap-1 mt-2 pt-2 border-t border-border/50">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-5 px-1.5 text-xs hover:bg-background/80"
                      >
                        <Copy className="h-3 w-3 mr-1" />
                        Copy
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-5 px-1.5 text-xs hover:bg-background/80"
                      >
                        <ThumbsUp className="h-3 w-3" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-5 px-1.5 text-xs hover:bg-background/80"
                      >
                        <ThumbsDown className="h-3 w-3" />
                      </Button>
                    </div>
                  )}
                </div>

                {message.type === "user" && (
                  <div className="flex-shrink-0 w-6 h-6 rounded-md bg-primary flex items-center justify-center mt-1">
                    <User className="h-3 w-3 text-primary-foreground" />
                  </div>
                )}
              </div>
            ))}

            {isTyping && (
              <div className="flex gap-3 justify-start">
                <div className="flex-shrink-0 w-6 h-6 rounded-md bg-primary/10 flex items-center justify-center mt-1">
                  <Bot className="h-3 w-3 text-primary" />
                </div>
                <div className="bg-muted/50 border border-border rounded-lg px-3 py-2">
                  <div className="flex items-center gap-1">
                    <div className="w-1.5 h-1.5 bg-muted-foreground/60 rounded-full animate-bounce"></div>
                    <div
                      className="w-1.5 h-1.5 bg-muted-foreground/60 rounded-full animate-bounce"
                      style={{ animationDelay: "0.1s" }}
                    ></div>
                    <div
                      className="w-1.5 h-1.5 bg-muted-foreground/60 rounded-full animate-bounce"
                      style={{ animationDelay: "0.2s" }}
                    ></div>
                  </div>
                </div>
              </div>
            )}
          </div>
          <div ref={messagesEndRef} />
        </div>

        {/* Quick Actions */}
        <div className="px-4 py-2 border-t border-border bg-background/95">
          <div className="grid grid-cols-2 gap-1.5 mb-3">
            {quickActions.map((action) => (
              <Button
                key={action.label}
                variant="outline"
                size="sm"
                className="h-7 text-xs justify-start hover:bg-muted/80 border-border/50"
                onClick={() => setInputValue(action.label)}
              >
                <span className="mr-1.5 text-xs">{action.icon}</span>
                {action.label}
              </Button>
            ))}
          </div>

          {/* Input */}
          <div className="flex gap-2">
            <div className="flex-1 relative">
              <Input
                ref={inputRef}
                value={inputValue}
                onChange={(e) => setInputValue(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && !e.shiftKey) {
                    e.preventDefault();
                    handleSendMessage();
                  }
                }}
                placeholder="Ask me anything..."
                className="h-8 pr-8 text-sm bg-background border-border/50 focus:border-primary/50 focus:ring-1 focus:ring-primary/20 rounded-md"
              />
              <Button
                onClick={handleSendMessage}
                disabled={!inputValue.trim() || isTyping}
                size="sm"
                className="absolute right-0.5 top-0.5 h-7 w-7 p-0 rounded-sm"
              >
                <Send className="h-3 w-3" />
              </Button>
            </div>
          </div>

          <p className="text-xs text-foreground mt-1.5 text-center">
            ⏎ to send • ⇧⏎ for new line
          </p>
        </div>
      </div>
    </div>
  );
}
