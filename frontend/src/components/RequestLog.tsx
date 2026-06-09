import { CircleCheck, CircleX } from "lucide-react";
import { useEffect, useState } from "react";
import { EventsOn } from "wailsjs/runtime/runtime";
import { Button } from "./ui/button";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger,
} from "./ui/context-menu";
import { Input } from "./ui/input";
import { MockEditor } from "./MockEditor";
import { GetMocks, SetMocks } from "wailsjs/go/main/App";

type LogEntry = {
  Method: string;
  Path: string;
  Status: number;
  Latency: number;
  Mocked: boolean;
  ContentType: string;
};

const NON_API = [
  "text/html",
  "text/css",
  "text/javascript",
  "application/javascript",
  "image/",
  "font/",
  "audio/",
  "video/",
];

export function RequestLog() {
  const [entries, setEntries] = useState<LogEntry[]>([]);
  const [xhrOnly, setXhrOnly] = useState(false);
  const [search, setSearch] = useState("");
  const [mockDraft, setMockDraft] = useState<{
    method: string;
    path: string;
    status: number;
    body?: string;
  } | null>(null);

  useEffect(() => {
    if (!(window as any).runtime) return;
    const unsub = EventsOn("request-log", (entry: LogEntry) => {
      setEntries((prev: LogEntry[]) => [entry, ...prev]);
    });
    return () => unsub?.();
  }, []);

  const statusStyleHelper = (status: number) => {
    return status >= 400
      ? "text-destructive"
      : status >= 200
        ? "text-green-500"
        : "text-muted-foreground";
  };

  const handleMock = (entry: LogEntry) => {
    console.log("handlemock");
    setMockDraft({
      method: entry.Method,
      path: entry.Path,
      status: entry.Status,
    });
  };

  const handleSaveMock = async (body: string, status: number) => {
    if (!mockDraft) return;
    const existing = await GetMocks();
    const updated = [
      ...(existing ?? []),
      {
        Method: mockDraft.method,
        Path: mockDraft.path,
        Body: body,
        Status: status,
        Enabled: true,
      },
    ];
    await SetMocks(updated);
    setMockDraft(null);
  };

  const visible = entries
    .filter((e) => !xhrOnly || !NON_API.some((t) => e.ContentType.includes(t)))
    .filter((e) => e.Path.toLowerCase().includes(search.toLowerCase()));

  return (
    <div className="flex flex-col h-full">
      {/* Column headers */}
      <div className="grid grid-cols-5 px-4 py-2 text-xs text-muted-foreground border-b border-border items-center">
        <div className="flex items-center">
          <div className="me-1 cursor-pointer">Method</div>
          <Button
            variant={xhrOnly ? "secondary" : "outline"}
            size="xs"
            onClick={() => setXhrOnly((v) => !v)}
            className={
              xhrOnly
                ? "bg-[#87a987] text-[#1a1a2e] hover:bg-[#87a987]/90 border-[#87a987] cursor-pointer me-1"
                : "cusor-pointer me-1"
            }
          >
            XHR
          </Button>
          <Button variant="outline" size="xs" onClick={() => setEntries([])}>
            Clear
          </Button>
        </div>
        <div>
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Filter path..."
            className="bg-transparent text-xs border border-border rounded px-2 py-0.5 w-48 focus:outline-none focus:border-[#87a987]
            text-foreground placeholder:text-muted-foreground me-1 flex-1/3"
          />
        </div>
        <div>Status</div>
        <div>Latency</div>
        <div>Mocked</div>
      </div>
      {/* Rows */}
      <div className="flex-1 overflow-y-auto">
        {visible.map((e, i) => (
          <ContextMenu key={i}>
            <ContextMenuTrigger>
              <div className="grid grid-cols-5 px-4 py-2 text-sm border-b border-border hover:bg-muted/50">
                <span className="p-1">{e.Method}</span>
                <span
                  className="overflow-hidden text-ellipsis p-1"
                  title={e.Path}
                >
                  {e.Path}
                </span>
                <span className={`${statusStyleHelper(e.Status)} p-1`}>
                  {e.Status}
                </span>
                <span className="p-1">{e.Latency}ms</span>
                <span className="p-1">
                  {e.Mocked ? (
                    <CircleCheck style={{ color: "#87a987" }} />
                  ) : (
                    <CircleX style={{ color: "#c4746e" }} />
                  )}
                </span>
              </div>
            </ContextMenuTrigger>
            <ContextMenuContent>
              <ContextMenuItem onClick={() => handleMock(e)}>
                Mock this endpoint
              </ContextMenuItem>
            </ContextMenuContent>
          </ContextMenu>
        ))}
      </div>
      {mockDraft && (
        <MockEditor
          method={mockDraft.method}
          onClose={() => setMockDraft(null)}
          onSave={(body: string, status: number) => {
            handleSaveMock(body, status);
          }}
          open={mockDraft !== null}
          path={mockDraft.path}
          status={mockDraft.status}
        />
      )}
    </div>
  );
}
