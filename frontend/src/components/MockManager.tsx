import { Trash } from "lucide-react";
import { useEffect, useState } from "react";
import { GetMocks, SetMocks } from "wailsjs/go/main/App";
import { Button } from "./ui/button";
import { Switch } from "./ui/switch";

export function MockManager() {
  const [mocks, setMocks] = useState<
    {
      Method: string;
      Path: string;
      Body: string;
      Enabled: boolean;
      Status: number;
    }[]
  >([]);

  const loadMocks = async () => {
    try {
      const mocks = await GetMocks();
      setMocks(mocks ?? []);
    } catch (err) {
      console.error("unable to load mocks", err);
    }
  };

  const handleToggle = async (index: number) => {
    const updated = mocks.map((m, i) =>
      i === index ? { ...m, Enabled: !m.Enabled } : m,
    );

    setMocks(updated);
    await SetMocks(updated);
  };

  const handleDelete = async (index: number) => {
    const updated = mocks.filter((_, i) => index !== i);
    setMocks(updated);
    await SetMocks(updated);
  };

  useEffect(() => {
    loadMocks();
  }, []);

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="grid grid-cols-5 px-4 py-2 text-xs text-muted-foreground border-b border-border">
        <div>Method</div>
        <div>Path</div>
        <div>Status</div>
        <div>Enabled</div>
        <div>Actions</div>
      </div>
      {/* Rows */}
      <div className="flex-1 overflow-y-auto">
        {mocks.map((m, i) => (
          <div
            key={i}
            className="grid grid-cols-5 px-4 py-2 text-sm border-b border-border hover:bg-muted/50 items-center"
          >
            <div>{m.Method}</div>
            <div className="overflow-x-hidden text-ellipsis">{m.Path}</div>
            <div>{m.Status}</div>
            <div>
              <Switch
                size="sm"
                checked={m.Enabled}
                onCheckedChange={() => handleToggle(i)}
              />
            </div>
            <div>
              <Button
                variant="destructive"
                className="cursor-pointer"
                onClick={() => handleDelete(i)}
              >
                <Trash />
              </Button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
