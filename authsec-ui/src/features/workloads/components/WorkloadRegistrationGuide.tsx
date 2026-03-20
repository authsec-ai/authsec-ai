import { Card, CardContent } from "../../../components/ui/card";
import { Badge } from "../../../components/ui/badge";
import { CheckCircle, ArrowRight } from "lucide-react";

export function WorkloadRegistrationGuide() {
  return (
    <div className="space-y-6">
      {/* Step 2 */}
      <div className="flex gap-4">
        <div className="flex-shrink-0">
          <Badge className="h-8 w-8 rounded-full flex items-center justify-center bg-blue-600 hover:bg-blue-600">
            1
          </Badge>
        </div>
        <div className="flex-1">
          <h3 className="font-semibold text-lg mb-2">
            Click "Register Workload" Button
          </h3>
          <p className="text-foreground mb-3">
            Look for the blue "Register Workload" button in the top-right corner
            of the page with a plus icon.
          </p>
          <div className="my-3">
            <img
              src="/workload-img1.png"
              alt="Register Workload button in the top-right corner"
              className="h-85 w-150 rounded-lg border border-muted shadow-sm"
            />
          </div>
          <div className="flex items-center gap-2 text-sm text-foreground">
            <ArrowRight className="h-4 w-4" />
            <span>This will open the workload registration form</span>
          </div>
        </div>
      </div>

      {/* Step 3 */}
      <div className="flex gap-4">
        <div className="flex-shrink-0">
          <Badge className="h-8 w-8 rounded-full flex items-center justify-center bg-blue-600 hover:bg-blue-600">
            2
          </Badge>
        </div>
        <div className="flex-1">
          <h3 className="font-semibold text-lg mb-2">
            Fill in Workload Details
          </h3>
          <p className="text-foreground mb-3">
            Complete the registration form with your workload information:
          </p>
          <ul className="space-y-2">
            <li className="flex items-start gap-2">
              <CheckCircle className="h-4 w-4 text-green-500 mt-0.5 flex-shrink-0" />
              <div>
                <span className="text-sm font-medium">SPIFFE ID</span>
                <span className="text-sm text-foreground">
                  {" "}
                  - Unique identifier for your workload (e.g.,
                  spiffe://example.com/workload/app)
                </span>
              </div>
            </li>
            <li className="flex items-start gap-2">
              <CheckCircle className="h-4 w-4 text-green-500 mt-0.5 flex-shrink-0" />
              <div>
                <span className="text-sm font-medium">Parent ID</span>
                <span className="text-sm text-foreground">
                  {" "}
                  - Parent SPIFFE ID (for hierarchical identities)
                </span>
              </div>
            </li>
            <li className="flex items-start gap-2">
              <CheckCircle className="h-4 w-4 text-green-500 mt-0.5 flex-shrink-0" />
              <div>
                <span className="text-sm font-medium">Selectors</span>
                <span className="text-sm text-foreground">
                  {" "}
                  - Workload attestation selectors (e.g., k8s:namespace,
                  k8s:pod-name)
                </span>
              </div>
            </li>
            {/* <li className="flex items-start gap-2">
              <CheckCircle className="h-4 w-4 text-green-500 mt-0.5 flex-shrink-0" />
              <div>
                <span className="text-sm font-medium">TTL</span>
                <span className="text-sm text-foreground"> - Certificate time-to-live duration</span>
              </div>
            </li> */}
          </ul>
        </div>
      </div>

      {/* Step 4 */}
      <div className="flex gap-4">
        <div className="flex-shrink-0">
          <Badge className="h-8 w-8 rounded-full flex items-center justify-center bg-blue-600 hover:bg-blue-600">
            3
          </Badge>
        </div>
        <div className="flex-1">
          <h3 className="font-semibold text-lg mb-2">Submit and Verify</h3>
          <p className="text-foreground mb-3">
            Click the "Register" button to complete the process. Your workload
            will appear in the table once registered successfully.
          </p>
          <div className="my-3">
            <img
              src="/workload-img2.png"
              alt="Register Workload button in the top-right corner"
              className="h-85 w-150 rounded-lg border border-muted shadow-sm"
            />
          </div>
          <Card className="bg-green-50 dark:bg-green-950/20 border-green-200 dark:border-green-900">
            <CardContent className="p-4">
              <div className="flex items-start gap-2 text-sm">
                <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5 flex-shrink-0" />
                <span className="text-green-900 dark:text-green-100">
                  <strong>Success!</strong> You can now use this workload
                  identity for secure communication between services.
                </span>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Additional Info */}
      <Card className="bg-blue-50 dark:bg-blue-950/20 border-blue-200 dark:border-blue-900">
        <CardContent className="p-4">
          <h4 className="font-semibold text-sm mb-2 text-blue-900 dark:text-blue-100">
            What happens next?
          </h4>
          <ul className="space-y-1 text-sm text-blue-800 dark:text-blue-200">
            <li className="flex items-start gap-2">
              <span className="font-mono text-xs">•</span>
              <span>
                The SPIRE server will automatically issue an X.509-SVID
                certificate
              </span>
            </li>
            <li className="flex items-start gap-2">
              <span className="font-mono text-xs">•</span>
              <span>
                Your workload can retrieve the certificate using the SPIRE agent
              </span>
            </li>
            <li className="flex items-start gap-2">
              <span className="font-mono text-xs">•</span>
              <span>Certificates will auto-rotate before expiration</span>
            </li>
          </ul>
        </CardContent>
      </Card>
    </div>
  );
}
