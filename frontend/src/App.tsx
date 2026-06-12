import { useEffect, useState } from "react";
import "./App.css";
import { profiles } from "wailsjs/go/models";
import {
  GetActiveEnv,
  GetEnvironments,
  GetProxyAddr,
  OpenInChrome,
  SetActiveEnv,
  StartRecording,
  StopRecording,
} from "wailsjs/go/main/App";
import { EnvSelector } from "./components/EnvSelector";
import { Button } from "./components/ui/button";
import { Circle, Globe, Square } from "lucide-react";
import { RequestLog } from "./components/RequestLog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "./components/ui/tabs";
import { MockManager } from "./components/MockManager";
import { ConfigPanel } from "./components/ConfigPanel";
import { TestOutputDialog } from "./components/TestOutputDialog";
import { WebSocketLog } from "./components/WebSocketLog";

function App() {
  const [envs, setEnvs] = useState<profiles.Environment[]>([]);
  const [activeEnvName, setActiveEnvName] = useState("");
  const [activeEnv, setActiveEnv] = useState<profiles.Environment | null>(null);
  const [proxyAddr, setProxyAddr] = useState("");
  const [isRecording, setIsRecording] = useState(false);
  const [testOutput, setTestOutput] = useState("");

  const loadEnvs = async () => {
    try {
      const proxyAddr = await GetProxyAddr();
      setProxyAddr(proxyAddr);
      const envs = await GetEnvironments();
      setEnvs(envs);
      const activeEnv = await GetActiveEnv();
      if (activeEnv) {
        setActiveEnv(activeEnv);
        setActiveEnvName(activeEnv.Name);
      } else {
        setActiveEnv(null);
        setActiveEnvName("");
      }
    } catch (err) {
      console.error("Failed to read config", err);
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

  const handleOpenChrome = async () => {
    try {
      await OpenInChrome();
    } catch (err) {
      console.error("Failed to open chrome", err);
    }
  };

  const handleToggleRecording = async () => {
    if (isRecording) {
      const output = await StopRecording();
      setTestOutput(output);
    } else {
      await StartRecording();
    }
    setIsRecording(!isRecording);
  };

  useEffect(() => {
    loadEnvs();
  }, []);

  return (
    <div className="flex flex-col h-screen bg-background text-foreground">
      {/* Top bar */}
      <div className="flex items-center px-4 py-2 border-b border-border">
        <EnvSelector
          activeEnv={activeEnvName}
          environments={envs}
          onEnvChange={handleEnvChange}
        />
        <div className="flex-1 text-sm text-muted-foreground font-mono ms-4">
          Proxy: {proxyAddr}
        </div>
        <Button
          variant={isRecording ? "destructive" : "outline"}
          onClick={handleToggleRecording}
        >
          {isRecording ? <Square /> : <Circle />}
          {isRecording ? "Stop" : "Record"}
        </Button>
        <Button
          className="cursor-pointer"
          variant="outline"
          onClick={handleOpenChrome}
        >
          <Globe /> Open in Chrome
        </Button>
      </div>
      {/* Log panel */}
      <Tabs
        className="flex-1 overflow-hidden flex flex-col p-2"
        defaultValue="request-log"
      >
        <TabsList className="shrink-0">
          <TabsTrigger value="request-log">Requests</TabsTrigger>
          <TabsTrigger value="ws-log">WebSockets</TabsTrigger>
          <TabsTrigger value="mocks">Mocks</TabsTrigger>
          <TabsTrigger value="config">Config</TabsTrigger>
        </TabsList>
        <TabsContent
          value="request-log"
          className="flex-1 overflow-hidden mt-0"
        >
          <RequestLog />
        </TabsContent>
        <TabsContent value="ws-log" className="flex-1 overflow-hidden mt-0">
          <WebSocketLog />
        </TabsContent>
        <TabsContent value="mocks" className="flex-1 overflow-hidden mt-0">
          <MockManager />
        </TabsContent>
        <TabsContent value="config" className="flex-1 overflow-hidden mt-0">
          <ConfigPanel
            activeEnv={activeEnv}
            activeEnvName={activeEnvName}
            onEnvChange={handleEnvChange}
            onEnvsChange={loadEnvs}
          />
        </TabsContent>
      </Tabs>
      <TestOutputDialog
        open={!!testOutput}
        content={testOutput}
        onOpenChange={(open: boolean) => {
          if (!open) setTestOutput("");
        }}
      />
    </div>
  );
}

export default App;
