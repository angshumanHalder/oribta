import { useEffect, useState } from "react";
import { Dialog, DialogContent } from "./ui/dialog";
import { Input } from "./ui/input";
import { Textarea } from "./ui/textarea";
import { Button } from "./ui/button";

type Props = {
  open: boolean;
  method: string;
  path: string;
  onClose: () => void;
  status: number;
  initialBody?: string;
  onSave: (body: string, status: number) => void;
};

export const MockEditor = ({
  open,
  method,
  path,
  status,
  initialBody = "",
  onSave,
  onClose,
}: Props) => {
  const [localStatus, setLocalStatus] = useState(status);
  const [body, setBody] = useState(initialBody);
  const [jsonError, setJsonError] = useState<string | null>(null);

  useEffect(() => {
    setLocalStatus(status);
    setBody(initialBody);
    setJsonError(null);
  }, [open]);

  const handleBodyChange = (val: string) => {
    setBody(val);
    try {
      JSON.parse(val);
      setJsonError(null);
    } catch (e) {
      setJsonError("Invalid JSON");
    }
  };

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="flex flex-col overflow-hidden max-w-3xl sm:max-w-3xl max-h-[90vh]">
        <div className="flex items-center gap-2 shrink-0">
          <span className="font-mono text-xs font-semibold">{method}</span>
          <span className="text-xs text-muted-foreground truncate">{path}</span>
        </div>
        <div className="flex flex-col gap-2">
          {jsonError && (
            <p className="text-xs text-destructive shrink-0">{jsonError}</p>
          )}
          <Input
            value={localStatus}
            type="number"
            onChange={(e) => setLocalStatus(parseInt(e.target.value))}
            placeholder="http status"
            className="shrink-0"
          />
          <Textarea
            value={body}
            onChange={(e) => handleBodyChange(e.target.value)}
            className="font-mono text-xs max-h-100 overflow-y-auto min-h-20 resize-none"
            placeholder='{"key": "value"}'
          />
          <Button
            className="shrink-0"
            disabled={!!jsonError || body.trim() === ""}
            onClick={() => onSave(body, localStatus)}
          >
            Save
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
};
