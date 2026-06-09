import { useEffect, useState } from "react";
import { Input } from "./ui/input";
import { Trash } from "lucide-react";
import { Button } from "./ui/button";

type Props = {
  rules: Array<{ From: string; To: string }>;
  onSave: (rules: Array<{ From: string; To: string }>) => void;
};

type Row = {
  From: string;
  To: string;
};

export function RewriteRulesEditor({ rules, onSave }: Props) {
  const [rows, setRows] = useState<Row[]>([]);

  useEffect(() => {
    setRows(rules);
  }, [rules]);

  const saveRowsHandler = () => {
    const result = rows.filter((row) => row.From !== "" && row.To !== "");
    onSave(result);
  };

  const addRowHandler = () => {
    setRows([...rows, { From: "", To: "" }]);
  };

  const removeRowHandler = (idx: number) => {
    const newRows = [...rows];
    newRows.splice(idx, 1);
    setRows(newRows);
  };

  return (
    <div className="flex flex-col">
      <div className="flex mb-2">
        <div className="flex-auto w-34 me-1">From</div>
        <div className="flex-auto w-66 me-1">To</div>
        <div className="flex-none w-6" />
      </div>
      {rows.map((row, idx) => (
        <div key={idx} className="flex mb-2">
          <div className="flex-auto w-34 me-1">
            <Input
              value={row.From}
              placeholder="From (e.g. http://localhost:8080)"
              onChange={(e) => {
                const updated = rows.map((r, i) =>
                  i === idx ? { ...r, From: e.target.value } : r,
                );
                setRows(updated);
              }}
            />
          </div>
          <div className="flex-auto w-66 me-1">
            <Input
              value={row.To}
              placeholder="To (e.g. http://localhost:8080)"
              onChange={(e) => {
                const updated = rows.map((r, i) =>
                  i === idx ? { ...r, To: e.target.value } : r,
                );
                setRows(updated);
              }}
            />
          </div>
          <Button
            variant="destructive"
            className="flex-none w-8 cursor-pointer"
            onClick={() => removeRowHandler(idx)}
          >
            <Trash />
          </Button>
        </div>
      ))}
      <Button
        variant="secondary"
        className="w-full mb-2 cursor-pointer"
        onClick={addRowHandler}
      >
        Add Row
      </Button>
      <Button
        variant="default"
        className="w-full cursor-pointer"
        onClick={() => saveRowsHandler()}
      >
        Save
      </Button>
    </div>
  );
}
