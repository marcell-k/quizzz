// Keep in sync with server/hub.go wire types.

export type JoinMsg = { type: "join"; team: number; name: string; token?: string };
export type BuzzMsg = { type: "buzz" };
export type CancelBuzzMsg = { type: "cancel_buzz" };
export type ClientMsg = JoinMsg | BuzzMsg | CancelBuzzMsg;

export type BuzzerState = "locked" | "open" | "buzzed";

export interface PlayerInfo {
  id: number;
  team: number;
  name: string;
  online: boolean;
}

export type StateMsg = {
  type: "state";
  teams: string[];
  players: PlayerInfo[];
  buzzer: BuzzerState;
  buzzedBy: number;
};
export type JoinedMsg = { type: "joined"; id: number; token: string; team: number; name: string };
export type TooLateMsg = { type: "too_late" };
export type ErrorMsg = { type: "error"; msg: string };
export type ServerMsg = StateMsg | JoinedMsg | TooLateMsg | ErrorMsg;
