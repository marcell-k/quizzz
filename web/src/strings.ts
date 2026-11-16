export const t = {
  appTitle: "Kvíz",
  player: "Játékos",
  admin: "Tanár",
  display: "Kivetítő",
  help: "Súgó",
  connected: "Kapcsolódva",
  disconnected: "Nincs kapcsolat",
  scanToJoin: "Olvasd be a QR-kódot",
  name: "Név",
  join: "Csatlakozás",
  buzz: "JELENTKEZEM",
  buzzer: "Gomb",
  tooLate: "Elkéstél.",
  players: "Játékosok",
  online: "online",
  offline: "offline",
  cancelBuzz: "Jelentkezés törlése",
} as const;

const errors: Record<string, string> = {
  bad_team: "Hibás csapat.",
  bad_name: "Hibás név (1-30 karakter).",
  name_in_use: "Ez a név már használatban van a csapatban.",
  not_joined: "Még nem csatlakoztál.",
  already_joined: "Már csatlakoztál.",
  forbidden: "Nem engedélyezett.",
  bad_message: "Hibás üzenet.",
};

export function errorText(code: string): string {
  return errors[code] ?? code;
}
