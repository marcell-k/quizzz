import type { TimerInfo } from "./protocol";

// Server sends endsAt for running timers. Clients count down locally.
export function remainingSeconds(timer: TimerInfo): number {
  const s =
    timer.endsAt > 0
      ? Math.ceil((timer.endsAt - Date.now()) / 1000)
      : timer.remaining;
  return s > 0 ? s : 0;
}

export function formatClock(seconds: number): string {
  const s = Math.max(0, Math.floor(seconds));
  const m = Math.floor(s / 60);
  return `${String(m).padStart(2, "0")}:${String(s % 60).padStart(2, "0")}`;
}

// Renders once immediately, then four times per second.
export function startTick(render: () => void): void {
  render();
  setInterval(render, 250);
}
