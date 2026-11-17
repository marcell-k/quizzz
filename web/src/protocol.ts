// Keep in sync with server/hub.go wire types.

export type JoinMsg = { type: "join"; team: number; name: string; token?: string };
export type BuzzMsg = { type: "buzz" };
export type CancelBuzzMsg = { type: "cancel_buzz" };
export type SelectMsg = { type: "select"; topic: number; q: number };
export type PlusMsg = { type: "plus"; team: number };
export type MinusMsg = { type: "minus"; team: number };
export type SetScoreMsg = { type: "set_score"; team: number; value: number };
export type UndoMsg = { type: "undo" };
export type NewGameMsg = { type: "new_game" };
export type NoneMsg = { type: "none" };
export type LoadQuizMsg = { type: "load_quiz"; file: string };
export type TimerMsg = { type: "timer"; action: "pause" | "resume" };
export type StartLightningMsg = { type: "start_lightning" };
export type EndGameMsg = { type: "end_game" };

export interface Rules {
  reopenOnWrong: boolean;
  lockoutTeam: boolean;
  subtractPoints: boolean;
  autoOpen: boolean;
}

export interface Settings {
  title: string;
  teams: string[];
  topics: number;
  questions: number;
  points: number[];
  lightningPoints: number;
  gameMinutes: number;
  lightningMinutes: number;
  rules: Rules;
  lastQuiz: string;
}

export type SetupMsg = { type: "setup"; settings: Settings };
export type OpenBuzzMsg = { type: "open_buzz" };
export type ResumeMsg = { type: "resume" };

export type ClientMsg =
  | JoinMsg
  | BuzzMsg
  | CancelBuzzMsg
  | SelectMsg
  | PlusMsg
  | MinusMsg
  | SetScoreMsg
  | UndoMsg
  | NewGameMsg
  | NoneMsg
  | LoadQuizMsg
  | TimerMsg
  | StartLightningMsg
  | EndGameMsg
  | SetupMsg
  | OpenBuzzMsg
  | ResumeMsg;

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

export interface ResultInfo {
  team: number;
  rank: number;
  score: number;
}

export interface TimerInfo {
  started: boolean;
  paused: boolean;
  remaining: number;
  endsAt: number;
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
  timer: TimerInfo;
  undoCount: number;
  results: ResultInfo[] | null;
  lightningPoints: number;
  lightningImage: string;
  lockedPlayers: number[];
  lockedTeams: boolean[];
  resumeAvailable: boolean;
  resumeWarning: string;
  settings: Settings;
};
export type JoinedMsg = { type: "joined"; id: number; token: string; team: number; name: string };
export type TooLateMsg = { type: "too_late" };
export type ErrorMsg = { type: "error"; msg: string };
export type ServerMsg = StateMsg | JoinedMsg | TooLateMsg | ErrorMsg;
