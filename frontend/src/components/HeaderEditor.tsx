import { useEffect, useState } from "react";
import { Input } from "./ui/input";
import { Trash } from "lucide-react";
import { Button } from "./ui/button";

type Props = {
  headers: Record<string, string>;
  onSave: (headers: Record<string, string>) => void;
};

type Row = {
  key: string;
  value: string;
};

export function HeaderEditor({ headers, onSave }: Props) {
  const [rows, setRows] = useState<Row[]>([]);

  useEffect(() => {
    const initialRows = Object.entries(headers).map(([key, value]) => ({
      key,
      value,
    }));
    setRows(initialRows);
  }, [headers]);

  const saveRowsHandler = () => {
    const headerRows = rows.filter((row) => row.key !== "" && row.value !== "");
    const result = Object.fromEntries(headerRows.map((r) => [r.key, r.value]));
    onSave(result);
  };

  const addRowHandler = () => {
    setRows([...rows, { key: "", value: "" }]);
  };

  const removeRowHandler = (idx: number) => {
    const newRows = [...rows];
    newRows.splice(idx, 1);
    setRows(newRows);
  };

  return (
    <div className="flex flex-col">
      <div className="flex mb-2">
        <div className="flex-auto w-34 me-1">Header Name</div>
        <div className="flex-auto w-66 me-1">Header Value</div>
        <div className="flex-none w-6" />
      </div>
      {rows.map((row, idx) => (
        <div key={idx} className="flex mb-2">
          <div className="flex-auto w-34 me-1">
            <Input
              value={row.key}
              placeholder="header name"
              onChange={(e) => {
                const updated = rows.map((r, i) =>
                  i === idx ? { ...r, key: e.target.value } : r,
                );
                setRows(updated);
              }}
            />
          </div>
          <div className="flex-auto w-66 me-1">
            <Input
              value={row.value}
              placeholder="value"
              onChange={(e) => {
                const updated = rows.map((r, i) =>
                  i === idx ? { ...r, value: e.target.value } : r,
                );
                setRows(updated);
              }}
            />
          </div>
          <Button
            variant="destructive"
            className="flex-none w-8"
            onClick={() => removeRowHandler(idx)}
          >
            <Trash />
          </Button>
        </div>
      ))}
      <Button
        variant="secondary"
        className="w-full mb-2"
        onClick={addRowHandler}
      >
        Add Row
      </Button>
      <Button
        variant="default"
        className="w-full"
        onClick={() => saveRowsHandler()}
      >
        Save
      </Button>
    </div>
  );
}
