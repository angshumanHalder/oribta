import { useEffect, useState } from "react";
import { EventsOn } from "wailsjs/runtime/runtime";

type WSFrame = {
  URL: string;
  Direction: string;
  MsgType: number;
  Payload: string;
};

export function WebSocketLog() {
  const [frames, setFrames] = useState<WSFrame[]>([]);

  useEffect(() => {
    const off = EventsOn("ws-frames", (frame: WSFrame) => {
      setFrames((prev) => [frame, ...prev]);
    });
    return () => off();
  }, []);

  return (
    <div className="flex flex-col h-full overflow-auto font-mono text-xs p-2">
      {frames.length === 0 && (
        <div className="text-muted-foreground">No websocket frames</div>
      )}
      {frames.map((f, i) => (
        <div
          key={i}
          className={`flex gap-2 py-1 border-b ${f.Direction === "send" ? "text-blue-400" : "text-green-400"}`}
        >
          <span>{f.Direction === "send" ? "↑" : "↓"}</span>
          <span className="truncate flex-1">{f.URL}</span>
          <span className="truncate max-w-xs">{f.Payload}</span>
        </div>
      ))}
    </div>
  );
}
