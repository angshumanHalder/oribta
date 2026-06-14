import {
  AddEnvironment,
  DeleteEnvironment,
  UpdateEnvironment,
} from "wailsjs/go/main/App";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "./ui/tabs";
import { useState } from "react";
import { profiles } from "wailsjs/go/models";
import { Trash } from "lucide-react";
import { HeaderEditor } from "./HeaderEditor";
import { RewriteRulesEditor } from "./RewriteRulesEditor";
import { EnvMappingPanel } from "./EnvMappingPanel";
import { PACDomainsPanel } from "./PACDomainsPanel";

type Props = {
  activeEnv: profiles.Environment | null;
  activeEnvName: string;
  onEnvChange: (name: string) => Promise<void>;
  onEnvsChange: () => void;
};

export function ConfigPanel({
  activeEnv,
  activeEnvName,
  onEnvsChange,
  onEnvChange,
}: Props) {
  const [newEnvName, setNewEnvName] = useState("");

  const handleDeleteEnv = async () => {
    await DeleteEnvironment(activeEnvName);
    onEnvsChange();
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
    onEnvsChange();
    await onEnvChange(newEnvName);
    setNewEnvName("");
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
      onEnvsChange();
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
      onEnvsChange();
    } catch (err) {
      console.error("Unable to save rules", err);
    }
  };

  return (
    <div className="flex flex-col h-full">
      <div className="flex gap-2 items-center mb-2">
        <Button
          variant="destructive"
          size="sm"
          disabled={!activeEnv}
          onClick={() => handleDeleteEnv()}
        >
          Delete Active Env <Trash />
        </Button>
        <Input
          value={newEnvName}
          onChange={(e) => setNewEnvName(e.target.value)}
          placeholder="New env name"
          className="h-7 text-xs"
        />
        <Button size="sm" variant="default" onClick={handleAddEnv}>
          Add
        </Button>
      </div>
      <Tabs
        defaultValue="headers"
        className="flex flex-col flex-1 overflow-hidden p-2"
      >
        <TabsList>
          <TabsTrigger value="headers">Headers</TabsTrigger>
          <TabsTrigger value="env-mapping">Mapping & Rules</TabsTrigger>
          <TabsTrigger value="pac-domains">PAC Domains</TabsTrigger>
        </TabsList>
        <TabsContent value="headers" className="flex-1 overflow-auto mt-0">
          <h3 className="text-sm font-medium text-foreground mb-2">Headers</h3>
          {activeEnv && (
            <HeaderEditor
              headers={activeEnv?.Headers}
              onSave={handleSaveHeaders}
            />
          )}
        </TabsContent>
        <TabsContent value="env-mapping" className="flex-1 overflow-auto mt-0">
          <div className="mb-2">
            <EnvMappingPanel onMappingApplied={onEnvsChange} />
          </div>
          <div>
            <h3 className="text-sm font-medium text-foreground mb-2">Rules</h3>
            {activeEnv && (
              <RewriteRulesEditor
                rules={activeEnv?.RewriteRules}
                onSave={handleSaveRules}
              />
            )}
          </div>
        </TabsContent>
        <TabsContent value="pac-domains" className="flex-1 overflow-auto mt-0">
          <PACDomainsPanel />
        </TabsContent>
      </Tabs>
    </div>
  );
}
