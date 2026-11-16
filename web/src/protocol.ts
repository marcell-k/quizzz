// Keep in sync with server/hub.go wire types.

export type JoinMsg = { type: "join"; team: number; name: string; token?: string };
export type BuzzMsg = { type: "buzz" };
export type CancelBuzzMsg = { type: "cancel_buzz" };
export type SelectMsg = { type: "select"; topic: number; q: number };
export type PlusMsg = { type: "plus"; team: number };
export type MinusMsg = { type: "minus"; team: number };
export type NoneMsg = { type: "none" };
export type LoadQuizMsg = { type: "load_quiz"; file: string };

export type ClientMsg =
  | JoinMsg
  | BuzzMsg
  | CancelBuzzMsg
  | SelectMsg
  | PlusMsg
  | MinusMsg
  | NoneMsg
  | LoadQuizMsg;

export type BuzzerState = "locked" | "open" | "buzzed";
export type Round = "setup" | "board" | "lightning" | "ended";

export interface PlayerInfo {
  id: number;
  team: number;
  name: string;
  online: boolean;
}

export interface TeamInfo {
  name: string;
  score: number;
}

export interface CurrentInfo {
  topic: number;
  q: number;
  points: number;
  text: string;
  hasImage: boolean;
  image: string;
}

export type StateMsg = {
  type: "state";
  teams: TeamInfo[];
  players: PlayerInfo[];
  buzzer: BuzzerState;
  buzzedBy: number;
  round: Round;
  topics: string[];
  points: number[];
  used: boolean[][];
  current: CurrentInfo | null;
};
export type JoinedMsg = { type: "joined"; id: number; token: string; team: number; name: string };
export type TooLateMsg = { type: "too_late" };
export type ErrorMsg = { type: "error"; msg: string };
export type ServerMsg = StateMsg | JoinedMsg | TooLateMsg | ErrorMsg;
