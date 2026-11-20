// Default team palette, shared by player, admin and display.
// ui/theme.css may override it with --team-1 .. --team-10.
export const teamColors: readonly string[] = [
  "#e6194b",
  "#3cb44b",
  "#4363d8",
  "#f58231",
  "#911eb4",
  "#42d4f4",
  "#f032e6",
  "#9a6324",
  "#808080",
  "#c9a400",
];

export function teamColor(team: number): string {
  const i = team % teamColors.length;
  const v = getComputedStyle(document.documentElement)
    .getPropertyValue(`--team-${i + 1}`)
    .trim();
  if (v !== "") return v;
  return teamColors[i] ?? "#666";
}
