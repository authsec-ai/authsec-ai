import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";

export interface FAQModalProps {
  open: boolean;
  onClose: () => void;
  question: string;
  answer?: string;
  customContent?: React.ReactNode;
}

export function FAQModal({
  open,
  onClose,
  question,
  answer,
  customContent,
}: FAQModalProps) {
  return (
    <Dialog open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
      <DialogContent className="max-w-3xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="text-xl font-semibold text-slate-900 dark:text-white">
            {question}
          </DialogTitle>
        </DialogHeader>
        {customContent || (
          <DialogDescription className="text-base leading-relaxed text-slate-700 dark:text-slate-300 mt-4">
            {answer}
          </DialogDescription>
        )}
      </DialogContent>
    </Dialog>
  );
}
