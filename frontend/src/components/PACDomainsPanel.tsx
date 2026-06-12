import { useEffect, useState } from "react";
import {
  AddPACDomain,
  GetPACDomains,
  RemovePACDomain,
} from "wailsjs/go/main/App";
import { Input } from "./ui/input";
import { Button } from "./ui/button";
import { Trash } from "lucide-react";

export function PACDomainsPanel() {
  const [domains, setDomains] = useState<string[]>([]);
  const [input, setInput] = useState("");

  const load = async () => {
    const d = await GetPACDomains();
    setDomains(d ?? []);
  };

  useEffect(() => {
    load();
  }, []);

  const handleAdd = async () => {
    if (!input.trim()) return;
    await AddPACDomain(input.trim());
    setInput("");
    load();
  };

  const handleRemove = async (domain: string) => {
    await RemovePACDomain(domain);
    load();
  };

  return (
    <div className="flex flex-col gap-2 p-2">
      <div className="flex gap-2">
        <Input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="e.g. api.example.com"
          onKeyDown={(e) => e.key === "Enter" && handleAdd()}
        />
        <Button onClick={handleAdd}>Add</Button>
      </div>
      {domains.length === 0 && (
        <div className="text-muted-foreground text-sm">
          No domains. Import env config to auto-populate.
        </div>
      )}
      {domains.map((d) => (
        <div
          key={d}
          className="flex items-center justify-between px-2 py-1 border rounded text-sm font-mono"
        >
          <span>{d}</span>
          <Button variant="ghost" size="icon" onClick={() => handleRemove(d)}>
            <Trash className="w-4 h-4" />
          </Button>
        </div>
      ))}
    </div>
  );
}
