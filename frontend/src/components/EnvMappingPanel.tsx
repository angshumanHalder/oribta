import { useState } from "react";
import { Input } from "./ui/input";
import { Button } from "./ui/button";
import {
  ApplyEnvMapping,
  GetEnvConfigNames,
  ImportEnvConfig,
  OpenFilePicker,
} from "wailsjs/go/main/App";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "./ui/select";

type Props = { onMappingApplied: () => void };

export function EnvMappingPanel({ onMappingApplied }: Props) {
  const [filePath, setFilePath] = useState("");
  const [envNames, setEnvNames] = useState<string[]>([]);
  const [fromEnv, setFromEnv] = useState("");
  const [toEnv, setToEnv] = useState("");

  const handleImportFile = async () => {
    try {
      const filePath = await OpenFilePicker();
      if (filePath) {
        await ImportEnvConfig(filePath);
        setFilePath(filePath);
        const envs = await GetEnvConfigNames();
        setEnvNames(envs);
      }
    } catch (err) {
      console.error("cannot import file");
    }
  };

  const handleApplyEnvMapping = async () => {
    await ApplyEnvMapping(fromEnv, toEnv);
    onMappingApplied();
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="border-b pb-2 mb-2">
        <h3>Config File</h3>
        <div className="flex items-center">
          <div className="me-2">{filePath ? filePath : "No File Selected"}</div>
          <Button className="cursor-pointer" onClick={() => handleImportFile()}>
            Import
          </Button>
        </div>
      </div>
      <div className="border-b mb-2">
        <h3 className="mb-2">Environment Mapping</h3>
        <div className="mb-4 flex items-center justify-start">
          <div className="me-4">
            <div className="mb-2">From</div>
            <Select value={fromEnv} onValueChange={(v) => setFromEnv(v!)}>
              <SelectTrigger>
                <SelectValue placeholder="Select environment" />
              </SelectTrigger>
              <SelectContent>
                {envNames.map((name) => (
                  <SelectItem key={name} value={name}>
                    {name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div>
            <div className="mb-2">To</div>
            <Select value={toEnv} onValueChange={(v) => setToEnv(v!)}>
              <SelectTrigger>
                <SelectValue placeholder="Select environment" />
              </SelectTrigger>
              <SelectContent>
                {envNames.map((name) => (
                  <SelectItem key={name} value={name}>
                    {name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
        <Button
          className="w-full cursor-pointer"
          onClick={() => handleApplyEnvMapping()}
        >
          Apply Mapping
        </Button>
      </div>
    </div>
  );
}
