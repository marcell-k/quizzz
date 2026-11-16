import type { ClientMsg, ServerMsg } from "./protocol";

export interface Conn {
  send(m: ClientMsg): void;
}

// Reconnects every 1 s after close.
export function connect(
  path: string,
  onMsg: (m: ServerMsg) => void,
  onOpen: () => void,
  onClose: () => void,
): Conn {
  let ws: WebSocket | undefined;
  const open = (): void => {
    const s = new WebSocket(`ws://${location.host}${path}`);
    ws = s;
    s.onopen = () => onOpen();
    s.onmessage = (e: MessageEvent) => {
      onMsg(JSON.parse(String(e.data)) as ServerMsg);
    };
    s.onclose = () => {
      onClose();
      setTimeout(open, 1000);
    };
  };
  open();
  return {
    send(m: ClientMsg): void {
      if (ws && ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify(m));
    },
  };
}
