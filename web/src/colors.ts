// Hardcoded team palette, shared by player, admin and display.
export const teamColors: readonly string[] = [
  "#e6194b",
  "#3cb44b",
  "#4363d8",
  "#f58231",
  "#911eb4",
];

export function teamColor(team: number): string {
  const c = teamColors[team % teamColors.length];
  return c ?? "#666";
}
