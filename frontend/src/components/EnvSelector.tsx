import { profiles } from "../../wailsjs/go/models";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "./ui/select";

type Props = {
  environments: profiles.Environment[];
  activeEnv: string;
  onEnvChange: (name: string) => void;
};

export function EnvSelector({ activeEnv, environments, onEnvChange }: Props) {
  return (
    <Select value={activeEnv} onValueChange={(val) => val && onEnvChange(val)}>
      <SelectTrigger className="w-full max-w-48">
        <SelectValue placeholder="Select a environment" />
      </SelectTrigger>
      <SelectContent>
        <SelectGroup>
          <SelectLabel>Environments</SelectLabel>
          {environments.map((env) => (
            <SelectItem key={env.Name} value={env.Name}>
              {env.Name}
            </SelectItem>
          ))}
        </SelectGroup>
      </SelectContent>
    </Select>
  );
}
