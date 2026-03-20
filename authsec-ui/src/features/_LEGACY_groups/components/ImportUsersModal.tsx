import React, { useState, useRef } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Progress } from "@/components/ui/progress";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { DownloadCloud, Upload, CheckCircle, AlertTriangle, ArrowRight } from "lucide-react";
import { toast } from "@/lib/toast";

interface CsvRow {
  email: string;
  name?: string;
  roles?: string;
}

interface ValidationError {
  row: number;
  message: string;
}

interface ImportUsersModalProps {
  isOpen: boolean;
  onClose: () => void;
  onImport: (users: CsvRow[]) => Promise<{ created: number; skipped: number }>;
  groupId: string;
  groupName: string;
}

export function ImportUsersModal({
  isOpen,
  onClose,
  onImport,
  groupId,
  groupName,
}: ImportUsersModalProps) {
  const [step, setStep] = useState<"upload" | "preview" | "validate" | "import" | "complete">(
    "upload"
  );
  const [file, setFile] = useState<File | null>(null);
  const [parsedData, setParsedData] = useState<CsvRow[]>([]);
  const [previewData, setPreviewData] = useState<CsvRow[]>([]);
  const [validationErrors, setValidationErrors] = useState<ValidationError[]>([]);
  const [progress, setProgress] = useState(0);
  const [importResult, setImportResult] = useState<{ created: number; skipped: number } | null>(
    null
  );
  const fileInputRef = useRef<HTMLInputElement>(null);

  const resetState = () => {
    setStep("upload");
    setFile(null);
    setParsedData([]);
    setPreviewData([]);
    setValidationErrors([]);
    setProgress(0);
    setImportResult(null);
  };

  const handleClose = () => {
    resetState();
    onClose();
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      const selectedFile = e.target.files[0];
      setFile(selectedFile);
      parseCSV(selectedFile);
    }
  };

  const parseCSV = (file: File) => {
    const reader = new FileReader();
    reader.onload = (event) => {
      if (!event.target?.result) return;

      const csv = event.target.result as string;
      const lines = csv.split("\n");
      const headers = lines[0].split(",").map((h) => h.trim().toLowerCase());

      // Validate required headers
      if (!headers.includes("email")) {
        toast.error("CSV must include an 'email' column");
        return;
      }

      const data: CsvRow[] = [];

      for (let i = 1; i < lines.length; i++) {
        if (!lines[i].trim()) continue; // Skip empty lines

        const values = lines[i].split(",").map((v) => v.trim());
        const row: CsvRow = {
          email: "",
        };

        headers.forEach((header, index) => {
          if (header === "email") row.email = values[index];
          if (header === "name") row.name = values[index];
          if (header === "roles") row.roles = values[index];
        });

        if (row.email) data.push(row);
      }

      setParsedData(data);
      setPreviewData(data.slice(0, 10)); // First 10 rows for preview
      setStep("preview");
    };

    reader.readAsText(file);
  };

  const validateUsers = () => {
    setStep("validate");
    const errors: ValidationError[] = [];

    // Simulate progress
    let currentProgress = 0;
    const interval = setInterval(() => {
      currentProgress += 10;
      setProgress(Math.min(currentProgress, 100));

      if (currentProgress >= 100) {
        clearInterval(interval);

        // Simulate validation (in a real app, you'd do actual validation)
        parsedData.forEach((row, index) => {
          if (!row.email.includes("@")) {
            errors.push({
              row: index + 1, // +1 because CSV rows start at 1 (header is row 0)
              message: "Invalid email format",
            });
          }
        });

        setValidationErrors(errors);
      }
    }, 100);
  };

  const downloadErrors = () => {
    if (!validationErrors.length) return;

    let csvContent = "row,email,error\n";
    validationErrors.forEach((error) => {
      const row = parsedData[error.row - 1]; // -1 because we stored 1-based indices
      csvContent += `${error.row},${row.email},"${error.message}"\n`;
    });

    const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.setAttribute("href", url);
    link.setAttribute("download", "import_errors.csv");
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  const handleImport = async () => {
    if (!parsedData.length) return;

    setStep("import");
    setProgress(0);

    try {
      // Simulate progress during import
      let currentProgress = 0;
      const interval = setInterval(() => {
        currentProgress += 5;
        setProgress(Math.min(currentProgress, 90));

        if (currentProgress >= 90) clearInterval(interval);
      }, 100);

      // Actual import
      const result = await onImport(parsedData);
      clearInterval(interval);
      setProgress(100);
      setImportResult(result);
      setStep("complete");
    } catch (error) {
      setProgress(0);
      toast.error("Import failed. Please try again.");
      setStep("preview");
    }
  };

  const downloadTemplate = () => {
    const csvContent = 'email,name,roles\nexample@domain.com,John Doe,"role1,role2"\n';
    const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.setAttribute("href", url);
    link.setAttribute(
      "download",
      `${groupName.toLowerCase().replace(/\s+/g, "_")}_import_template.csv`
    );
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && handleClose()}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>
            {step === "upload" && `Import Users to ${groupName}`}
            {step === "preview" && "Preview Import Data"}
            {step === "validate" && "Validating Data"}
            {step === "import" && "Importing Users"}
            {step === "complete" && "Import Complete"}
          </DialogTitle>
          <DialogDescription>
            {step === "upload" &&
              "Upload a CSV file with user data to bulk import users to this group."}
            {step === "preview" &&
              `Found ${parsedData.length} rows. Please review the first 10 entries below.`}
            {step === "validate" && "Validating user data..."}
            {step === "import" && `Importing users to ${groupName}...`}
            {step === "complete" &&
              importResult &&
              `Successfully imported ${importResult.created} users to ${groupName} with ${importResult.skipped} duplicates skipped.`}
          </DialogDescription>
        </DialogHeader>

        {step === "upload" && (
          <div className="flex flex-col items-center justify-center space-y-4 py-8">
            <div
              className="border-2 border-dashed border-gray-300 dark:border-gray-700 rounded-lg p-12 text-center hover:border-primary/50 transition-colors flex flex-col items-center justify-center cursor-pointer"
              onClick={() => fileInputRef.current?.click()}
            >
              <Upload className="h-8 w-8 mb-4 text-foreground" />
              <h3 className="text-lg font-medium mb-1">Drag & drop your CSV file</h3>
              <p className="text-sm text-foreground mb-4">or click to browse files</p>
              <p className="text-xs text-foreground">
                Your CSV should include: email (required), name, roles
              </p>
              <Input
                ref={fileInputRef}
                type="file"
                accept=".csv"
                className="hidden"
                onChange={handleFileChange}
              />
            </div>
            <div className="text-sm text-foreground">
              Need a template?{" "}
              <button onClick={downloadTemplate} className="text-primary hover:underline">
                Download CSV template
              </button>
            </div>
          </div>
        )}

        {step === "preview" && (
          <div className="space-y-6">
            <div className="border rounded-lg overflow-auto max-h-72">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Email</TableHead>
                    <TableHead>Name</TableHead>
                    <TableHead>Roles</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {previewData.map((row, index) => (
                    <TableRow key={index}>
                      <TableCell className="font-medium">{row.email}</TableCell>
                      <TableCell>
                        {row.name || <span className="text-foreground">(not provided)</span>}
                      </TableCell>
                      <TableCell>
                        {row.roles || <span className="text-foreground">(none)</span>}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>

            <div className="flex items-center justify-between">
              <div className="text-sm text-foreground">
                Showing 10 of {parsedData.length} rows
              </div>
              <Button variant="outline" onClick={() => resetState()}>
                Upload different file
              </Button>
            </div>
          </div>
        )}

        {(step === "validate" || step === "import") && (
          <div className="space-y-6 py-8">
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label>
                  {step === "validate" ? "Validating" : "Importing"} {parsedData.length} users
                </Label>
                <span className="text-sm text-foreground">{progress}%</span>
              </div>
              <Progress value={progress} className="h-2" />
            </div>
            <p className="text-sm text-center text-foreground">
              {step === "validate" && "Checking for valid email formats, duplicates, and errors..."}
              {step === "import" && `Adding users to ${groupName} and assigning roles...`}
            </p>
          </div>
        )}

        {validationErrors.length > 0 && step === "validate" && progress === 100 && (
          <div className="space-y-4 border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-4 rounded-lg">
            <div className="flex items-start">
              <AlertTriangle className="h-5 w-5 text-red-500 mr-2 mt-0.5" />
              <div>
                <h4 className="font-semibold text-sm text-red-600 dark:text-red-400">
                  Found {validationErrors.length} errors
                </h4>
                <p className="text-xs text-red-600 dark:text-red-400 mt-1">
                  Please fix these issues and upload again, or continue with valid entries only.
                </p>
              </div>
            </div>
            <Button size="sm" onClick={downloadErrors} className="flex items-center gap-2">
              <DownloadCloud className="h-4 w-4" />
              Download errors CSV
            </Button>
          </div>
        )}

        {step === "complete" && importResult && (
          <div className="space-y-6 py-4">
            <div className="rounded-lg border p-6 flex flex-col items-center justify-center">
              <div className="h-12 w-12 rounded-full bg-green-100 dark:bg-green-900/30 flex items-center justify-center mb-4">
                <CheckCircle className="h-6 w-6 text-green-600 dark:text-green-500" />
              </div>
              <h3 className="text-xl font-semibold mb-1">Import Complete</h3>
              <p className="text-foreground text-center">
                {importResult.created} users have been added to the {groupName} group.
                {importResult.skipped > 0 && ` ${importResult.skipped} duplicates were skipped.`}
              </p>
              <div className="grid grid-cols-2 gap-4 w-full max-w-xs mt-6">
                <div className="flex flex-col items-center p-3 border rounded-lg">
                  <span className="text-xl font-bold">{importResult.created}</span>
                  <span className="text-xs text-foreground">Added</span>
                </div>
                <div className="flex flex-col items-center p-3 border rounded-lg">
                  <span className="text-xl font-bold">{importResult.skipped}</span>
                  <span className="text-xs text-foreground">Skipped</span>
                </div>
              </div>
            </div>
          </div>
        )}

        <DialogFooter className="gap-2">
          {step === "upload" && (
            <Button variant="outline" onClick={handleClose}>
              Cancel
            </Button>
          )}

          {step === "preview" && (
            <>
              <Button variant="outline" onClick={handleClose}>
                Cancel
              </Button>
              <Button
                onClick={validateUsers}
                className="bg-gradient-to-r from-primary to-primary/80 hover:from-primary/90 hover:to-primary/70 shadow-lg shadow-primary/25"
              >
                Validate {parsedData.length} rows
                <ArrowRight className="ml-2 h-4 w-4" />
              </Button>
            </>
          )}

          {step === "validate" && progress === 100 && (
            <>
              <Button variant="outline" onClick={handleClose}>
                Cancel
              </Button>
              <Button
                disabled={validationErrors.length === parsedData.length}
                onClick={handleImport}
                className="bg-gradient-to-r from-primary to-primary/80 hover:from-primary/90 hover:to-primary/70 shadow-lg shadow-primary/25"
              >
                Import {parsedData.length - validationErrors.length} users
                <ArrowRight className="ml-2 h-4 w-4" />
              </Button>
            </>
          )}

          {step === "complete" && (
            <Button
              onClick={handleClose}
              className="bg-gradient-to-r from-primary to-primary/80 hover:from-primary/90 hover:to-primary/70 shadow-lg shadow-primary/25"
            >
              Done
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
