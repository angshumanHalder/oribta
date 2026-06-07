import { useEffect, useState } from "react";
import {
  AddEnvironment,
  DeleteEnvironment,
  GetActiveEnv,
  GetEnvironments,
  GetProxyAddr,
  SetActiveEnv,
  UpdateEnvironment,
} from "wailsjs/go/main/App";
import { profiles } from "../../wailsjs/go/models";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import { EnvSelector } from "./EnvSelector";
import { HeaderEditor } from "./HeaderEditor";
import { RewriteRulesEditor } from "./RewriteRulesEditor";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "./ui/dialog";
import { RefreshCw, Trash } from "lucide-react";
import { Button } from "./ui/button";
import { Input } from "./ui/input";

export function SettingsPanel() {
  const [isOpen, setIsOpen] = useState(false);
  const [envs, setEnvs] = useState<profiles.Environment[]>([]);
  const [activeEnvName, setActiveEnvName] = useState("");
  const [activeEnv, setActiveEnv] = useState<profiles.Environment | null>(null);
  const [proxyAddr, setProxyAddr] = useState("");
  const [newEnvName, setNewEnvName] = useState("");

  useEffect(() => {
    if (!(window as any).runtime) return;
    const unsub = EventsOn("open-settings", () => {
      setIsOpen(true);
      loadData();
    });
    return () => unsub?.();
  }, []);

  const loadData = async () => {
    try {
      const [envs, activeEnv, proxyAddr] = await Promise.all([
        GetEnvironments(),
        GetActiveEnv(),
        GetProxyAddr(),
      ]);
      if (activeEnv !== null) {
        setActiveEnvName(activeEnv.Name);
        setActiveEnv(activeEnv);
      }
      setEnvs(envs);
      setProxyAddr(proxyAddr);
    } catch (err) {
      console.error("Unable to initialize", err);
    }
  };

  const handleEnvChange = async (name: string) => {
    try {
      await SetActiveEnv(name);
      const activeEnv = await GetActiveEnv();
      if (activeEnv !== null) {
        setActiveEnvName(activeEnv.Name);
        setActiveEnv(activeEnv);
      }
    } catch (err) {
      console.error("Unable to set environment", err);
    }
  };

  const handleSaveHeaders = async (headers: Record<string, string>) => {
    try {
      if (!activeEnvName) {
        return;
      }
      const updated = new profiles.Environment({
        ...activeEnv,
        Headers: headers,
      });
      await UpdateEnvironment(updated);
      setActiveEnv(updated);
      setActiveEnvName(updated.Name);
    } catch (err) {
      console.error("Unable to update headers", err);
    }
  };

  const handleSaveRules = async (rules: profiles.RewriteRule[]) => {
    try {
      if (!activeEnvName) {
        return;
      }
      const updated = new profiles.Environment({
        ...activeEnv,
        RewriteRules: rules,
      });
      await UpdateEnvironment(updated);
      setActiveEnv(updated);
      setActiveEnvName(updated.Name);
    } catch (err) {
      console.error("Unable to save rules", err);
    }
  };

  const handleAddEnv = async () => {
    if (!newEnvName) {
      return;
    }
    await AddEnvironment(
      new profiles.Environment({
        Name: newEnvName,
        Headers: {},
        RewriteRules: [],
      }),
    );
    await loadData();
    await handleEnvChange(newEnvName);
    setNewEnvName("");
  };

  const handleDeleteEnv = async () => {
    await DeleteEnvironment(activeEnvName);
    await loadData();
    setActiveEnv(null);
    setActiveEnvName("");
  };

  return (
    <Dialog
      open={isOpen}
      onOpenChange={(o) => {
        setIsOpen(o);
      }}
      disablePointerDismissal={true}
    >
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <div className="flex items-center justify-between pr-8">
            <DialogTitle>Settings</DialogTitle>
            <Button variant="ghost" size="icon" onClick={() => loadData()}>
              <RefreshCw />
            </Button>
          </div>
          <DialogDescription>Proxy: {proxyAddr}</DialogDescription>
        </DialogHeader>
        {/* Envs */}
        <div className="flex gap-2 items-center">
          <div className="flex-1">
            <EnvSelector
              environments={envs}
              activeEnv={activeEnvName}
              onEnvChange={handleEnvChange}
            />
          </div>
          <Button
            variant="destructive"
            disabled={!activeEnv}
            onClick={() => handleDeleteEnv()}
          >
            <Trash />
          </Button>
          <Input
            value={newEnvName}
            onChange={(e) => setNewEnvName(e.target.value)}
            placeholder="New env name"
          />
          <Button variant="default" onClick={handleAddEnv}>
            Add
          </Button>
        </div>
        <div className="overflow-y-auto max-h-[60vh] pr-4 -mr-4">
          {/* Headers */}
          <div className="mb-3">
            <h3 className="text-xl font-bold tracking-tight mb-2">Headers</h3>
            {activeEnv && (
              <HeaderEditor
                headers={activeEnv?.Headers}
                onSave={handleSaveHeaders}
              />
            )}
          </div>
          {/* Rules */}
          <div>
            <h3 className="text-xl font-bold tracking-tight mb-2">Rules</h3>
            {activeEnv && (
              <RewriteRulesEditor
                rules={activeEnv?.RewriteRules}
                onSave={handleSaveRules}
              />
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
