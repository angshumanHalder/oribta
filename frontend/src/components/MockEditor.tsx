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
  onSave: (body: string, status: number) => void;
};

export const MockEditor = ({
  open,
  method,
  path,
  status,
  onSave,
  onClose,
}: Props) => {
  const [localStatus, setLocalStatus] = useState(status);
  const [body, setBody] = useState("");
  const [jsonError, setJsonError] = useState<string | null>(null);

  useEffect(() => {
    setLocalStatus(status);
    setBody("");
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
      <DialogContent className="max-w-2xl">
        <div className="flex items-center">
          <div className="me-2">{method}</div>
          <div className="overflow-hidden">{path}</div>
        </div>
        <div className="flex flex-col">
          {jsonError && (
            <p className="text-xs text-destructive mb-1">{jsonError}</p>
          )}
          <Input
            value={localStatus}
            type="number"
            onChange={(e) => setLocalStatus(parseInt(e.target.value))}
            placeholder="http status"
            className="mb-2"
          />
          <Textarea
            value={body}
            onChange={(e) => handleBodyChange(e.target.value)}
            className="mb-2 font-mono text-xs min-h-75 resize-y"
            placeholder='{"key": "value"}'
          />
          <Button
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
