import type { ReactNode } from "react";
import { useState } from "react";
import type { LucideIcon } from "lucide-react";
import {
  BookOpen,
  Boxes,
  Check,
  ClipboardCopy,
  Code,
  ExternalLink,
  FileCode2,
  FolderGit2,
  LifeBuoy,
  Lock,
  ShieldCheck,
  Sparkles,
  TerminalSquare,
  Workflow,
} from "lucide-react";

import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

const quickLinks = [
  {
    label: "Open SDK Repository",
    href: "https://github.com/authsec-ai/auth-manager/tree/main/sdk",
    icon: FolderGit2,
  },
  {
    label: "Readme Reference",
    href: "https://github.com/authsec-ai/auth-manager/blob/main/sdk/README.md",
    icon: BookOpen,
  },
  {
    label: "RBAC Architecture",
    href: "https://github.com/authsec-ai/auth-manager/blob/main/docs/RBAC_ARCHITECTURE.md",
    icon: Workflow,
  },
  {
    label: "Request Enterprise Support",
    href: "mailto:support@authsec.example.com",
    icon: LifeBuoy,
  },
];

const journey = [
  {
    id: "overview",
    title: "Understand the benefits",
    blurb: "Why AuthSec SDK matters for authentication, authorization, and token hygiene.",
    icon: ShieldCheck,
  },
  {
    id: "environment",
    title: "Prepare the environment",
    blurb: "Line up tenant secrets, Vault paths, and baseline infrastructure.",
    icon: TerminalSquare,
  },
  {
    id: "quickstart",
    title: "Install and bootstrap",
    blurb: "Set up Python tooling and drop the SDK starter code into a service.",
    icon: Code,
  },
  {
    id: "endpoints",
    title: "Wire token endpoints",
    blurb: "Understand the server APIs backing generation, verification, and OIDC exchange.",
    icon: FileCode2,
  },
  {
    id: "authorization",
    title: "Mirror authorization logic",
    blurb: "Reflect server-side decisions in your UI without introducing drift.",
    icon: Lock,
  },
  {
    id: "testing",
    title: "Validate and monitor",
    blurb: "Run smoke tests and scripts so regressions surface before reaching tenants.",
    icon: Code,
  },
  {
    id: "resources",
    title: "Share supporting docs",
    blurb: "Hand off deeper references and escalation channels for customer teams.",
    icon: BookOpen,
  },
];

const stepHighlights = [
  {
    tag: "AuthN",
    title: "Server-verified Tokens",
    description:
      "SDK talks to /auth/user/verifyToken to ensure JWT claims still match tenant DB state.",
    icon: ShieldCheck,
  },
  {
    tag: "AuthZ",
    title: "Layered Permission Checks",
    description:
      "Checks structured perms first, falls back to wildcard scopes, and optionally enforces resource lists.",
    icon: Lock,
  },
  {
    tag: "Secure by Default",
    title: "Dual Signing Secrets",
    description:
      "Supports default and SDK-agent secrets so third-party integrations stay segregated.",
    icon: Sparkles,
  },
  {
    tag: "Minimal",
    title: "Requests + PyJWT Only",
    description:
      "Lean Python footprint makes embedding the client into existing services straightforward.",
    icon: Code,
  },
];

const starterChecklist = [
  "Tenant base URL, default secret, and SDK secret confirmed",
  "Vault namespace created with client credential path",
  "Python 3.10+ environment available for SDK installation",
  "Downstream service identified for token injection",
];

const environmentScript = `# env_vars_linux.sh
export AUTHMGR_BASE_URL="https://auth.example.com"
export JWT_DEF_SECRET="<tenant-default-secret>"
export JWT_SDK_SECRET="<tenant-sdk-secret>"
export VAULT_ADDR="https://vault.example.com"
export HYDRA_ADMIN_URL="https://hydra-admin.example.com"`;

const installScript = `python3 -m venv .authsec
source .authsec/bin/activate
pip install -r sdk/requirements.txt`;

const bootstrapCode = `from minimal import AuthSecClient

client = AuthSecClient("https://auth.example.com")

# Option 1: Generate through AuthSec (requires Vault + tenant DB wiring)
token = client.generate_token(
    tenant_id="demo-tenant",
    project_id="demo-project",
    client_id="demo-client",
    email_id="user@example.com",
)

# Option 2: Set an existing JWT (from portal or CI secret)
client.set_token(token)

claims = client.verify_token()  # Raises on mismatch with tenant DB
print("Roles:", claims.get("roles"))

if client.authorize("invoice", "read"):
    print("User may read invoices")

response = client.request("GET", "/api/v1/invoices")
print("Status:", response.status_code)`;

const generatePayload = `{
  "tenant_id": "demo-tenant",
  "project_id": "demo-project",
  "client_id": "demo-client",
  "email_id": "user@example.com",
  "secret_id": "optional-sdk-secret"
}`;

const verifyPayload = `{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI..."
}`;

const oidcPayload = `{
  "oidc_token": "hydra-issued-token"
}`;

const claimsExample = `{
  "tenant_id": "demo-tenant",
  "project_id": "demo-project",
  "client_id": "demo-client",
  "email_id": "user@example.com",
  "scopes": ["invoice:read", "user:read"],
  "perms": [
    {"r": "invoice", "a": ["read", "write"]},
    {"r": "user", "a": ["read"]}
  ],
  "resources": ["invoice", "user"],
  "roles": ["finance_manager"],
  "token_type": "sdk-agent"
}`;

const authorizeHelper = `def authorize(client, resource, action, require_resource_list=False):
    claims = client._claims()  # cached decode, no signature verification
    if claims is None:
        return False
    if require_resource_list and resource not in claims.get("resources", []):
        return False

    # structured perms first
    for perm in claims.get("perms", []):
        if perm.get("r") == resource and action in perm.get("a", []):
            return True

    # fallback to wildcard scopes
    needed = f"{resource}:{action}"
    scopes = claims.get("scopes", []) + claims.get("scope", "").split()
    for scope in scopes:
        res, _, act = scope.partition(":")
        if (res in (resource, "*")) and (act in (action, "*")):
            return True

    return False`;

const testingScript = `# inside repo root
make sdk-venv
make sdk-test          # offline flow (uses mock token)
AUTHMGR_BASE_URL=https://auth.example.com make sdk-int  # requires running server`;

const verificationScript = `AUTHMGR_BASE_URL=https://auth.example.com \
AUTH_TOKEN="<pasted-jwt>" \
python sdk/verify_capabilities.py`;

interface SectionCardProps {
  id: string;
  icon: LucideIcon;
  title: string;
  description?: string;
  children: ReactNode;
  className?: string;
}

function SectionCard({
  id,
  icon: Icon,
  title,
  description,
  children,
  className,
}: SectionCardProps) {
  return (
    <Card id={id} className={cn("scroll-mt-28 border border-border/70 shadow-sm", className)}>
      <CardHeader className="flex flex-row items-start gap-4 pb-0">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
          <Icon className="h-5 w-5" />
        </div>
        <div className="space-y-1">
          <CardTitle className="text-xl font-semibold leading-tight">{title}</CardTitle>
          {description ? (
            <CardDescription className="text-sm text-foreground">
              {description}
            </CardDescription>
          ) : null}
        </div>
      </CardHeader>
      <CardContent className="space-y-5 pt-6">{children}</CardContent>
    </Card>
  );
}

function CopyButton({ value }: { value: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    if (typeof navigator === "undefined" || !navigator.clipboard) {
      return;
    }
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setTimeout(() => setCopied(false), 1400);
    } catch {
      setCopied(false);
    }
  };

  return (
    <Button
      variant="outline"
      size="sm"
      onClick={handleCopy}
      className="pointer-events-auto absolute right-3 top-3 gap-1.5 rounded-md border border-border bg-background/80 text-xs font-medium text-foreground hover:bg-muted"
    >
      {copied ? (
        <Check className="h-3.5 w-3.5 text-emerald-500" />
      ) : (
        <ClipboardCopy className="h-3.5 w-3.5" />
      )}
      {copied ? "Copied" : "Copy"}
    </Button>
  );
}

function CodeBlock({ value }: { value: string }) {
  return (
    <pre className="relative max-w-full overflow-x-auto-hidden rounded-xl border border-border/60 bg-background/95 p-5 text-sm leading-6 shadow-sm">
      <CopyButton value={value} />
      <code className="block w-full break-words whitespace-pre-wrap text-xs text-foreground sm:text-sm">
        {value}
      </code>
    </pre>
  );
}

export function SDKManagerPage() {
  return (
    <div className="bg-muted/10">
      <main className="mx-auto grid w-full   grid-cols-1 gap-6 px-6 py-10 lg:grid-cols-[minmax(0,1fr)_18rem] lg:px-12">
        <div className="lg:hidden">
          <Card className="border border-border/60 shadow-sm">
            <CardHeader className="pb-4">
              <p className="text-xs font-semibold uppercase tracking-[0.32em] text-foreground">
                Integration flow
              </p>
            </CardHeader>
            <CardContent className="flex w-full snap-x gap-3 overflow-x-auto-hidden pb-4">
              {journey.map((step, index) => (
                <a
                  key={step.id}
                  href={`#${step.id}`}
                  className="min-w-[220px] snap-start rounded-xl border border-border/60 bg-card px-4 py-3 text-left transition-colors hover:border-primary/40 hover:bg-card/80"
                >
                  <div className="flex items-center gap-3">
                    <span className="flex h-8 w-8 items-center justify-center rounded-full border border-border text-sm font-semibold text-foreground">
                      {index + 1}
                    </span>
                    <div>
                      <p className="text-sm font-semibold text-foreground">{step.title}</p>
                      <p className="text-xs text-foreground">{step.blurb}</p>
                    </div>
                  </div>
                </a>
              ))}
            </CardContent>
          </Card>
        </div>

        <div className="space-y-6">
          <SectionCard
            id="overview"
            icon={ShieldCheck}
            title="Why use the AuthSec SDK?"
            description="Security-first authentication and authorization helpers built for enterprise tenants."
          >
            <div className="grid gap-4 md:grid-cols-2">
              {stepHighlights.map((highlight) => {
                const Icon = highlight.icon;
                return (
                  <div
                    key={highlight.title}
                    className="rounded-xl border border-border/70 bg-card/70 p-4 shadow-sm"
                  >
                    <div className="flex items-center gap-3">
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                        <Icon className="h-5 w-5" />
                      </div>
                      <div>
                        <span className="text-xs font-semibold uppercase tracking-[0.2em] text-primary">
                          {highlight.tag}
                        </span>
                        <p className="text-sm font-semibold text-foreground">{highlight.title}</p>
                      </div>
                    </div>
                    <p className="mt-3 text-sm text-foreground">{highlight.description}</p>
                  </div>
                );
              })}
            </div>
            <div className="rounded-xl border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-700 dark:border-emerald-500/40 dark:bg-emerald-500/10 dark:text-emerald-200">
              <strong className="font-semibold">Tip:</strong> You can drop the SDK into any service
              that can run Python 3.10+ and pip.
            </div>
          </SectionCard>

          <SectionCard
            id="environment"
            icon={TerminalSquare}
            title="1. Prepare your tenant environment"
            description="Confirm configuration before distributing SDK instructions to teams."
          >
            <div className="space-y-4">
              <h3 className="text-sm font-semibold uppercase tracking-[0.18em] text-foreground">
                Required environment variables
              </h3>
              <CodeBlock value={environmentScript} />
              <p className="text-sm text-foreground">
                Use unique values per tenant so SDK-issued tokens cannot be forged across
                environments.
              </p>
            </div>
            <div className="space-y-3">
              <h3 className="text-sm font-semibold uppercase tracking-[0.18em] text-foreground">
                Vault setup (optional but recommended)
              </h3>
              <ul className="list-disc space-y-2 pl-6 text-sm text-foreground">
                <li>
                  Store client credentials at{" "}
                  <code className="font-mono text-xs">
                    kv/secret/&lt;tenant_id&gt;/&lt;project_id&gt;/&lt;client_id&gt;
                  </code>
                  .
                </li>
                <li>
                  Include <code className="font-mono text-xs">secret_id</code> for SDK agents that
                  should sign their own tokens.
                </li>
                <li>Grant SDK callers Vault read-only access to their namespace.</li>
              </ul>
            </div>
            <div className="rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-700 dark:border-amber-500/40 dark:bg-amber-500/10 dark:text-amber-200">
              <strong className="font-semibold">Remember:</strong> Rotate{" "}
              <code className="font-mono text-xs">JWT_SDK_SECRET</code> if you suspect leakage.
              Tokens minted with the old secret are rejected after rotation.
            </div>
          </SectionCard>

          <SectionCard
            id="quickstart"
            icon={Code}
            title="2. Install SDK & Quick Start"
            description="Bootstrap the Python client and validate connectivity."
          >
            <div className="space-y-4">
              <h3 className="text-sm font-semibold uppercase tracking-[0.18em] text-foreground">
                Install dependencies
              </h3>
              <CodeBlock value={installScript} />
            </div>
            <div className="space-y-4">
              <h3 className="text-sm font-semibold uppercase tracking-[0.18em] text-foreground">
                Minimal bootstrap code
              </h3>
              <CodeBlock value={bootstrapCode} />
            </div>
            <div className="rounded-xl border border-border/70 bg-card/70 p-4 text-sm text-foreground">
              <strong className="font-semibold text-foreground">Works best with:</strong> Python
              3.10+, requests &gt;= 2.32.0, PyJWT &gt;= 2.10.0.
            </div>
          </SectionCard>

          <SectionCard
            id="endpoints"
            icon={FileCode2}
            title="3. Token lifecycle endpoints"
            description="Surface the server APIs the SDK wraps so integrators can troubleshoot quickly."
          >
            <div className="space-y-5">
              <Card className="border border-border/70 shadow-sm">
                <CardHeader className="flex flex-col gap-2 pb-0">
                  <div className="flex items-center justify-between gap-2">
                    <CardTitle className="text-lg font-semibold text-foreground">
                      POST /auth/user/generateToken
                    </CardTitle>
                    <Badge
                      variant="outline"
                      className="border-primary/30 bg-primary/10 text-primary"
                    >
                      User token
                    </Badge>
                  </div>
                  <CardDescription className="text-xs text-foreground">
                    Mint a JWT after validating tenant, project, and client credentials (optionally
                    secret_id).
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4 pt-4 text-sm text-foreground">
                  <CodeBlock value={generatePayload} />
                  <ul className="list-disc space-y-2 pl-6">
                    <li>
                      Success returns{" "}
                      <code className="font-mono text-xs">
                        {'{"access_token": "...", "token_type": "Bearer", "expires_in": 86400}'}
                      </code>
                      .
                    </li>
                    <li>
                      Include <code className="font-mono text-xs">secret_id</code> for SDK agents
                      that require the SDK-specific signing secret.
                    </li>
                    <li>
                      Response claim highlights: <code className="font-mono text-xs">perms</code>,{" "}
                      <code className="font-mono text-xs">scopes</code>,
                      <code className="font-mono text-xs">resources</code>,{" "}
                      <code className="font-mono text-xs">roles</code>,{" "}
                      <code className="font-mono text-xs">token_type</code>.
                    </li>
                  </ul>
                </CardContent>
              </Card>

              <Card className="border border-border/70 shadow-sm">
                <CardHeader className="flex flex-col gap-2 pb-0">
                  <div className="flex items-center justify-between gap-2">
                    <CardTitle className="text-lg font-semibold text-foreground">
                      POST /auth/user/verifyToken
                    </CardTitle>
                    <Badge
                      variant="outline"
                      className="border-emerald-300/40 bg-emerald-100 text-emerald-700 dark:border-emerald-500/40 dark:bg-emerald-500/10 dark:text-emerald-200"
                    >
                      Validation
                    </Badge>
                  </div>
                  <CardDescription className="text-xs text-foreground">
                    Re-validate token signatures and confirm claims align with tenant state.
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4 pt-4 text-sm text-foreground">
                  <CodeBlock value={verifyPayload} />
                  <ul className="list-disc space-y-2 pl-6">
                    <li>
                      Use in admin consoles or CI pipelines to confirm permissions remain current.
                    </li>
                    <li>
                      Returns decoded claims plus <code className="font-mono text-xs">valid</code>,
                      <code className="font-mono text-xs">issued_at</code>,{" "}
                      <code className="font-mono text-xs">expires_at</code>.
                    </li>
                  </ul>
                </CardContent>
              </Card>

              <Card className="border border-border/70 shadow-sm">
                <CardHeader className="flex flex-col gap-2 pb-0">
                  <div className="flex items-center justify-between gap-2">
                    <CardTitle className="text-lg font-semibold text-foreground">
                      POST /auth/user/oidcToken
                    </CardTitle>
                    <Badge
                      variant="outline"
                      className="border-blue-300/40 bg-blue-100 text-blue-700 dark:border-blue-500/40 dark:bg-blue-500/10 dark:text-blue-200"
                    >
                      OIDC
                    </Badge>
                  </div>
                  <CardDescription className="text-xs text-foreground">
                    Exchange an IdP token (via Hydra) for an AuthSec JWT enriched with tenant
                    permissions.
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4 pt-4 text-sm text-foreground">
                  <CodeBlock value={oidcPayload} />
                  <ul className="list-disc space-y-2 pl-6">
                    <li>
                      Hydrates scopes, roles, resources, and metadata after confirming token
                      activity.
                    </li>
                    <li>
                      Returned JWT uses <code className="font-mono text-xs">token_type</code> ={" "}
                      <code className="font-mono text-xs">"oidc"</code>.
                    </li>
                  </ul>
                </CardContent>
              </Card>
            </div>
          </SectionCard>

          <SectionCard
            id="authorization"
            icon={Lock}
            title="4. Client authorization helpers"
            description="Mirror the SDK logic in your UI to match backend enforcement."
          >
            <div className="space-y-4">
              <h3 className="text-sm font-semibold uppercase tracking-[0.18em] text-foreground">
                Claims your UI should expect
              </h3>
              <CodeBlock value={claimsExample} />
            </div>
            <div className="space-y-4">
              <h3 className="text-sm font-semibold uppercase tracking-[0.18em] text-foreground">
                Local helper implementation
              </h3>
              <CodeBlock value={authorizeHelper} />
            </div>
            <div className="rounded-xl border border-border/70 bg-card/60 p-4 text-sm text-foreground">
              <strong className="font-semibold text-foreground">UI usage ideas:</strong>
              <ul className="mt-2 list-disc space-y-2 pl-6">
                <li>
                  Hide navigation items unless{" "}
                  <code className="font-mono text-xs">authorize("invoice","read")</code> succeeds.
                </li>
                <li>
                  Disable destructive buttons unless{" "}
                  <code className="font-mono text-xs">
                    authorize_all([("invoice","write"),("invoice","delete")])
                  </code>{" "}
                  returns true.
                </li>
                <li>
                  Use <code className="font-mono text-xs">has_role("admin")</code> for
                  coarse-grained admin portals.
                </li>
              </ul>
            </div>
          </SectionCard>

          <SectionCard
            id="testing"
            icon={Code}
            title="5. Validate integration"
            description="Use smoke tests and interactive scripts to keep integrations healthy."
          >
            <div className="space-y-4">
              <h3 className="text-sm font-semibold uppercase tracking-[0.18em] text-foreground">
                Run SDK smoke tests
              </h3>
              <CodeBlock value={testingScript} />
            </div>
            <div className="space-y-4">
              <h3 className="text-sm font-semibold uppercase tracking-[0.18em] text-foreground">
                Interactive verification script
              </h3>
              <CodeBlock value={verificationScript} />
              <p className="text-sm text-foreground">
                Outputs server-side verification status plus representative AuthZ allow/deny checks.
              </p>
            </div>
            <div className="rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-700 dark:border-amber-500/40 dark:bg-amber-500/10 dark:text-amber-200">
              <strong className="font-semibold">CI gate:</strong> Re-run{" "}
              <code className="font-mono text-xs">verify_capabilities.py</code> whenever permissions
              change to catch stale roles before deployment.
            </div>
          </SectionCard>

          <SectionCard id="resources" icon={BookOpen} title="6. Additional resources & support">
            <div className="grid gap-4 md:grid-cols-2">
              {[
                {
                  label: "RBAC architecture deep dive",
                  href: "https://github.com/authsec-ai/auth-manager/blob/main/docs/RBAC_ARCHITECTURE.md",
                },
                {
                  label: "Example usage script",
                  href: "https://github.com/authsec-ai/auth-manager/blob/main/sdk/example_usage.py",
                },
                {
                  label: "SDK regression tests",
                  href: "https://github.com/authsec-ai/auth-manager/blob/main/sdk/test_minimal.py",
                },
                {
                  label: "Top-level README summary",
                  href: "https://github.com/authsec-ai/auth-manager/blob/main/README.md#python-sdk",
                },
              ].map((link) => (
                <a
                  key={link.label}
                  href={link.href}
                  target="_blank"
                  rel="noreferrer"
                  className="flex items-center justify-between rounded-xl border border-border/70 bg-card/60 px-4 py-3 text-sm font-medium text-foreground transition-colors hover:border-primary/30 hover:bg-card"
                >
                  <span className="flex items-center gap-2">
                    <ExternalLink className="h-4 w-4 text-foreground" />
                    {link.label}
                  </span>
                  <span className="text-xs text-foreground">View</span>
                </a>
              ))}
            </div>
            <div className="flex flex-col items-start justify-between gap-3 rounded-xl border border-primary/30 bg-primary/10 px-4 py-4 text-sm text-foreground md:flex-row md:items-center">
              <strong className="font-semibold">
                Need help onboarding a tenant or building a language-specific SDK?
              </strong>
              <Button
                asChild
                variant="secondary"
                className="border-border bg-background hover:bg-muted"
              >
                <a href="mailto:support@authsec.example.com">Contact AuthSec Support</a>
              </Button>
            </div>
            <div className="rounded-xl border border-border/70 bg-card/60 p-4 text-sm leading-6 text-foreground">
              Crafted for AuthSec IAM enterprise tenants. Documentation stays synchronized with the
              source of truth in the AuthSec repository. For security notices and change logs,
              subscribe to the tenant bulletin or email
              <a
                href="mailto:security@authsec.example.com"
                className="ml-1 font-medium text-primary hover:underline"
              >
                security@authsec.example.com
              </a>
              .
            </div>
          </SectionCard>
        </div>

        <aside className="hidden lg:block">
          <div className="sticky top-0 pt-6">
            <Card className="border border-border/60 shadow-sm">
              <CardHeader className="pb-4">
                <p className="text-xs font-semibold uppercase tracking-[0.32em] text-foreground">
                  Integration flow
                </p>
              </CardHeader>
              <CardContent className="space-y-3 max-h-[calc(100vh-8rem)] overflow-y-auto pr-2">
                {journey.map((step, index) => (
                  <a
                    key={step.id}
                    href={`#${step.id}`}
                    className="group block rounded-lg border border-transparent px-3 py-3 transition-colors hover:border-primary/30 hover:bg-muted"
                  >
                    <div className="flex items-start gap-3">
                      <span className="flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-full border border-border text-xs font-semibold text-foreground transition-colors group-hover:border-primary group-hover:text-primary">
                        {index + 1}
                      </span>
                      <div className="space-y-1">
                        <p className="text-sm font-semibold text-foreground">{step.title}</p>
                        <p className="text-xs text-foreground">{step.blurb}</p>
                      </div>
                    </div>
                  </a>
                ))}
              </CardContent>
            </Card>
          </div>
        </aside>
      </main>
    </div>
  );
}
